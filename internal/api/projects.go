package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

var (
	canonicalRepoCacheMu sync.RWMutex
	canonicalRepoCache   = make(map[string]string)
	worktreePathRegex    = regexp.MustCompile(`[/\\]\.(?:claude|grok|cursor|agents?)[/\\]worktrees[/\\]([^/\\]+)[/\\]?$`)
)

func canonicalRepo(path string) string {
	if path == "" {
		return ""
	}
	cleanPath := filepath.Clean(path)

	canonicalRepoCacheMu.RLock()
	cached, ok := canonicalRepoCache[cleanPath]
	canonicalRepoCacheMu.RUnlock()
	if ok {
		return cached
	}

	result := cleanPath
	cur := cleanPath
	for i := 0; i < 20; i++ {
		gitPath := filepath.Join(cur, ".git")
		info, err := os.Stat(gitPath)
		if err == nil {
			if !info.IsDir() {
				// .git is a file (worktree pointer)
				content, err := os.ReadFile(gitPath)
				if err == nil {
					txt := strings.TrimSpace(string(content))
					if strings.HasPrefix(txt, "gitdir:") {
						gitdir := strings.TrimSpace(strings.TrimPrefix(txt, "gitdir:"))
						if !filepath.IsAbs(gitdir) {
							gitdir = filepath.Clean(filepath.Join(cur, gitdir))
						}
						// gitdir is typically <repo>/.git/worktrees/<name>
						parent := filepath.Dir(gitdir)
						grandparent := filepath.Dir(parent)
						if filepath.Base(parent) == "worktrees" && filepath.Base(grandparent) == ".git" {
							result = filepath.Dir(grandparent)
							break
						}
					}
				}
				result = cur
				break
			} else {
				// .git is a directory (main repository checkout)
				result = cur
				break
			}
		}

		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}

	if result == cleanPath {
		if loc := worktreePathRegex.FindStringIndex(cleanPath); loc != nil {
			result = cleanPath[:loc[0]]
		}
	}

	canonicalRepoCacheMu.Lock()
	canonicalRepoCache[cleanPath] = result
	canonicalRepoCacheMu.Unlock()

	return result
}

func repoWorktreePaths(repo string) []string {
	var paths []string
	wtDir := filepath.Join(repo, ".git", "worktrees")
	entries, err := os.ReadDir(wtDir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		gdFile := filepath.Join(wtDir, entry.Name(), "gitdir")
		content, err := os.ReadFile(gdFile)
		if err != nil {
			continue
		}
		wp := strings.TrimSpace(string(content))
		wp = strings.TrimSuffix(wp, "/.git")
		wp = strings.TrimSuffix(wp, "\\.git")
		if wp != "" {
			paths = append(paths, filepath.Clean(wp))
		}
	}
	return paths
}

func enrichProjectList(projects []models.ProjectSummary) []models.ProjectSummary {
	projectByPath := make(map[string]*models.ProjectSummary)
	for i := range projects {
		p := &projects[i]
		p.Name = filepath.Base(filepath.Clean(p.ProjectName))
		p.Path = p.ProjectName
		if _, err := os.Stat(p.Path); err == nil {
			p.Status = "active"
		} else {
			p.Status = "missing"
		}
		p.CanonicalRepo = canonicalRepo(p.Path)
		if p.CanonicalRepo != "" && p.CanonicalRepo != p.Path {
			p.IsWorktree = true
			p.ParentPath = p.CanonicalRepo
			p.ParentName = filepath.Base(p.CanonicalRepo)
			p.WorktreeName = filepath.Base(p.Path)
		} else {
			p.IsWorktree = false
		}
		projectByPath[p.Path] = p
	}

	// Recovery pass for deleted worktrees registered in git
	wtToRepo := make(map[string]string)
	for _, p := range projects {
		if p.CanonicalRepo != "" {
			for _, wp := range repoWorktreePaths(p.CanonicalRepo) {
				wtToRepo[wp] = p.CanonicalRepo
			}
		}
	}
	for _, p := range projects {
		if !p.IsWorktree {
			if repo, ok := wtToRepo[p.Path]; ok && repo != p.Path {
				p.CanonicalRepo = repo
				p.IsWorktree = true
				p.ParentPath = repo
				p.ParentName = filepath.Base(repo)
				p.WorktreeName = filepath.Base(p.Path)
			}
		}
	}

	// Group worktrees under canonical repos
	groups := make(map[string][]*models.ProjectSummary)
	for i := range projects {
		p := &projects[i]
		if p.IsWorktree && p.CanonicalRepo != "" {
			groups[p.CanonicalRepo] = append(groups[p.CanonicalRepo], p)
		}
	}

	var synthesized []*models.ProjectSummary
	for repo, children := range groups {
		sort.Slice(children, func(i, j int) bool {
			return children[i].TotalTokens > children[j].TotalTokens
		})

		var wtSummaries []models.WorktreeSummary
		for _, c := range children {
			wtName := c.WorktreeName
			if wtName == "" {
				wtName = c.Name
			}
			wtSummaries = append(wtSummaries, models.WorktreeSummary{
				Name:         wtName,
				Path:         c.Path,
				SessionCount: c.SessionCount,
				TotalTokens:  c.TotalTokens,
				TotalCostUSD: c.TotalCostUSD,
				Agents:       c.Agents,
				Status:       c.Status,
			})
		}

		parent, exists := projectByPath[repo]
		if !exists {
			status := "missing"
			if _, err := os.Stat(repo); err == nil {
				status = "active"
			}
			synth := &models.ProjectSummary{
				ProjectName:   repo,
				Name:          filepath.Base(repo),
				Path:          repo,
				SessionCount:  0,
				TotalTokens:   0,
				TotalCostUSD:  0,
				Agents:        []string{},
				MCPTools:      []string{},
				Status:        status,
				CanonicalRepo: repo,
				IsWorktree:    false,
				Synthesized:   true,
			}
			synthesized = append(synthesized, synth)
			parent = synth
			projectByPath[repo] = synth
		}

		parent.IsRepoRoot = true
		parent.Worktrees = wtSummaries

		// Calculate aggregate
		agg := &models.ProjectAggregate{
			SessionCount:            parent.SessionCount,
			SubagentCount:           parent.SubagentCount,
			PlanCount:               parent.PlanCount,
			ConfiguredSubagentCount: parent.ConfiguredSubagentCount,
			TotalTokens:             parent.TotalTokens,
			TotalCostUSD:            parent.TotalCostUSD,
			WorktreeCount:           len(children),
		}
		agentsSet := make(map[string]bool)
		for _, a := range parent.Agents {
			agentsSet[a] = true
		}
		for _, c := range children {
			agg.SessionCount += c.SessionCount
			agg.SubagentCount += c.SubagentCount
			agg.PlanCount += c.PlanCount
			agg.ConfiguredSubagentCount += c.ConfiguredSubagentCount
			agg.TotalTokens += c.TotalTokens
			agg.TotalCostUSD += c.TotalCostUSD
			for _, a := range c.Agents {
				agentsSet[a] = true
			}
		}
		for a := range agentsSet {
			agg.Agents = append(agg.Agents, a)
		}
		sort.Strings(agg.Agents)
		parent.Aggregate = agg

		if parent.Synthesized {
			parent.Agents = agg.Agents
		}
	}

	result := make([]models.ProjectSummary, 0, len(projects)+len(synthesized))
	for _, p := range projects {
		result = append(result, p)
	}
	for _, p := range synthesized {
		result = append(result, *p)
	}

	return result
}

// GetProjects handles GET /api/projects and GET /projects.
func (s *Server) GetProjects(w http.ResponseWriter, r *http.Request) {
	includeHidden := r.URL.Query().Get("include_hidden") == "true"

	rawProjects, err := s.db.GetProjects(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	projects := enrichProjectList(rawProjects)

	s.mu.RLock()
	var filtered []models.ProjectSummary
	for _, p := range projects {
		if !includeHidden && (s.hiddenProjects[p.ProjectName] || s.hiddenProjects[p.Path]) {
			continue
		}
		filtered = append(filtered, p)
	}
	s.mu.RUnlock()

	if filtered == nil {
		filtered = []models.ProjectSummary{}
	}
	respondJSON(w, http.StatusOK, filtered)
}

// GetProjectDetail handles GET /api/projects/{name} and GET /projects/{name}.
func (s *Server) GetProjectDetail(w http.ResponseWriter, r *http.Request) {
	projectName := chi.URLParam(r, "*")
	if projectName == "" {
		projectName = chi.URLParam(r, "path")
	}
	if projectName == "" {
		projectName = chi.URLParam(r, "name")
	}
	projectName = strings.TrimPrefix(projectName, "/")

	rawProjects, err := s.db.GetProjects(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	enrichedList := enrichProjectList(rawProjects)
	var foundSummary *models.ProjectSummary
	for i := range enrichedList {
		p := &enrichedList[i]
		if p.ProjectName == projectName || p.Path == projectName || p.Name == projectName {
			foundSummary = p
			break
		}
	}

	if foundSummary == nil {
		// Try standard DB lookup
		summary, sessions, err := s.db.GetProjectDetail(r.Context(), projectName)
		if err != nil {
			respondError(w, http.StatusNotFound, "Project not found")
			return
		}
		if sessions == nil {
			sessions = []models.Session{}
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"project":  summary,
			"sessions": sessions,
		})
		return
	}

	// Fetch sessions for this project or all worktrees if synthesized root
	var sessions []models.Session
	if foundSummary.Synthesized && foundSummary.IsRepoRoot {
		for _, wt := range foundSummary.Worktrees {
			wtSessions, _, err := s.db.ListSessions(r.Context(), models.FilterParams{
				Project: wt.Path,
				Limit:   50,
			})
			if err == nil {
				sessions = append(sessions, wtSessions...)
			}
		}
	} else {
		dbSessions, _, err := s.db.ListSessions(r.Context(), models.FilterParams{
			Project: foundSummary.ProjectName,
			Limit:   100,
		})
		if err == nil {
			sessions = dbSessions
		}
	}

	if sessions == nil {
		sessions = []models.Session{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"project":  foundSummary,
		"sessions": sessions,
	})
}

// GetHiddenProjects handles GET /config/hidden.
func (s *Server) GetHiddenProjects(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var hidden []string
	for p := range s.hiddenProjects {
		hidden = append(hidden, p)
	}
	if hidden == nil {
		hidden = []string{}
	}
	respondJSON(w, http.StatusOK, hidden)
}

// HideProject handles POST /config/hide.
func (s *Server) HideProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Path == "" {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	s.mu.Lock()
	s.hiddenProjects[body.Path] = true
	var hidden []string
	for p := range s.hiddenProjects {
		hidden = append(hidden, p)
	}
	s.mu.Unlock()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok":     true,
		"hidden": hidden,
	})
}

// UnhideProject handles POST /config/unhide.
func (s *Server) UnhideProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Path == "" {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	s.mu.Lock()
	delete(s.hiddenProjects, body.Path)
	var hidden []string
	for p := range s.hiddenProjects {
		hidden = append(hidden, p)
	}
	s.mu.Unlock()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok":     true,
		"hidden": hidden,
	})
}

// GetAliases handles GET /config/aliases.
func (s *Server) GetAliases(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok":      true,
		"aliases": s.aliases,
	})
}

// SetAliases handles POST /config/aliases.
func (s *Server) SetAliases(w http.ResponseWriter, r *http.Request) {
	var updates map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	s.mu.Lock()
	for k, v := range updates {
		s.aliases[k] = v
	}
	aliases := s.aliases
	s.mu.Unlock()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok":      true,
		"aliases": aliases,
	})
}
