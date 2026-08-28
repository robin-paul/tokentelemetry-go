package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

var (
	ErrNotFound = errors.New("record not found")
)

// UpsertSession inserts or updates a session record.
func (d *DB) UpsertSession(ctx context.Context, s *models.Session) error {
	query := `
	INSERT INTO sessions (
		id, session_id, agent_name, project_name, file_path, machine_id,
		created_at, updated_at, start_time, end_time, duration_seconds,
		model_raw, model_resolved, input_tokens, output_tokens,
		cache_read_tokens, cache_creation_tokens, gross_cost_usd, net_cost_usd,
		electricity_cost_usd, hardware_profile, status, git_branch,
		is_subagent, parent_session_id, subagent_type
	) VALUES (
		?, ?, ?, ?, ?, ?,
		?, ?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?
	)
	ON CONFLICT(id) DO UPDATE SET
		session_id = excluded.session_id,
		agent_name = excluded.agent_name,
		project_name = excluded.project_name,
		file_path = excluded.file_path,
		machine_id = excluded.machine_id,
		updated_at = excluded.updated_at,
		start_time = excluded.start_time,
		end_time = excluded.end_time,
		duration_seconds = excluded.duration_seconds,
		model_raw = excluded.model_raw,
		model_resolved = excluded.model_resolved,
		input_tokens = excluded.input_tokens,
		output_tokens = excluded.output_tokens,
		cache_read_tokens = excluded.cache_read_tokens,
		cache_creation_tokens = excluded.cache_creation_tokens,
		gross_cost_usd = excluded.gross_cost_usd,
		net_cost_usd = excluded.net_cost_usd,
		electricity_cost_usd = excluded.electricity_cost_usd,
		hardware_profile = excluded.hardware_profile,
		status = excluded.status,
		git_branch = excluded.git_branch,
		is_subagent = excluded.is_subagent,
		parent_session_id = excluded.parent_session_id,
		subagent_type = excluded.subagent_type;
	`
	s.ProjectName = models.CanonicalProject(s.ProjectName)
	now := time.Now().UTC()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now
	if s.MachineID == "" {
		s.MachineID = "local"
	}

	return d.WithTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query,
			s.ID, s.SessionID, s.AgentName, s.ProjectName, s.FilePath, s.MachineID,
			s.CreatedAt, s.UpdatedAt, s.StartTime, s.EndTime, s.DurationSeconds,
			s.ModelRaw, s.ModelResolved, s.InputTokens, s.OutputTokens,
			s.CacheReadTokens, s.CacheCreationTokens, s.GrossCostUSD, s.NetCostUSD,
			s.ElectricityCostUSD, s.HardwareProfile, s.Status, s.GitBranch,
			s.IsSubagent, s.ParentSessionID, s.SubagentType,
		)
		return err
	})
}

// SaveSessionWithTurnsAndSubagents atomically saves a session and replaces its message turns and subagent runs.
func (d *DB) SaveSessionWithTurnsAndSubagents(ctx context.Context, s *models.Session) error {
	return d.WithTx(ctx, func(tx *sql.Tx) error {
		// 1. Upsert session
		s.ProjectName = models.CanonicalProject(s.ProjectName)
		upsertSessionQuery := `
		INSERT INTO sessions (
			id, session_id, agent_name, project_name, file_path, machine_id,
			created_at, updated_at, start_time, end_time, duration_seconds,
			model_raw, model_resolved, input_tokens, output_tokens,
			cache_read_tokens, cache_creation_tokens, gross_cost_usd, net_cost_usd,
			electricity_cost_usd, hardware_profile, status, git_branch,
			is_subagent, parent_session_id, subagent_type
		) VALUES (
			?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?
		)
		ON CONFLICT(id) DO UPDATE SET
			session_id = excluded.session_id,
			agent_name = excluded.agent_name,
			project_name = excluded.project_name,
			file_path = excluded.file_path,
			machine_id = excluded.machine_id,
			updated_at = excluded.updated_at,
			start_time = excluded.start_time,
			end_time = excluded.end_time,
			duration_seconds = excluded.duration_seconds,
			model_raw = excluded.model_raw,
			model_resolved = excluded.model_resolved,
			input_tokens = excluded.input_tokens,
			output_tokens = excluded.output_tokens,
			cache_read_tokens = excluded.cache_read_tokens,
			cache_creation_tokens = excluded.cache_creation_tokens,
			gross_cost_usd = excluded.gross_cost_usd,
			net_cost_usd = excluded.net_cost_usd,
			electricity_cost_usd = excluded.electricity_cost_usd,
			hardware_profile = excluded.hardware_profile,
			status = excluded.status,
			git_branch = excluded.git_branch,
			is_subagent = excluded.is_subagent,
			parent_session_id = excluded.parent_session_id,
			subagent_type = excluded.subagent_type;
		`
		now := time.Now().UTC()
		if s.CreatedAt.IsZero() {
			s.CreatedAt = now
		}
		s.UpdatedAt = now
		if s.MachineID == "" {
			s.MachineID = "local"
		}

		_, err := tx.ExecContext(ctx, upsertSessionQuery,
			s.ID, s.SessionID, s.AgentName, s.ProjectName, s.FilePath, s.MachineID,
			s.CreatedAt, s.UpdatedAt, s.StartTime, s.EndTime, s.DurationSeconds,
			s.ModelRaw, s.ModelResolved, s.InputTokens, s.OutputTokens,
			s.CacheReadTokens, s.CacheCreationTokens, s.GrossCostUSD, s.NetCostUSD,
			s.ElectricityCostUSD, s.HardwareProfile, s.Status, s.GitBranch,
			s.IsSubagent, s.ParentSessionID, s.SubagentType,
		)
		if err != nil {
			return fmt.Errorf("failed to upsert session %s: %w", s.ID, err)
		}

		// 2. Replace message turns
		if len(s.Turns) > 0 {
			if _, err := tx.ExecContext(ctx, `DELETE FROM message_turns WHERE session_id = ?;`, s.ID); err != nil {
				return fmt.Errorf("failed to clean existing message turns: %w", err)
			}
			turnStmt, err := tx.PrepareContext(ctx, `
				INSERT INTO message_turns (
					id, session_id, turn_index, timestamp, role,
					model_name, content, thinking, reasoning_effort,
					input_tokens, output_tokens, cache_read_tokens,
					cache_creation_tokens, cost_usd, tools_invoked_json,
					tool_calls_json, tool_results_json, raw_payload_json
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
			`)
			if err != nil {
				return fmt.Errorf("failed to prepare message turn insert: %w", err)
			}
			defer turnStmt.Close()

			for _, t := range s.Turns {
				toolsJSON := t.ToolsInvokedJSON
				if toolsJSON == "" {
					if len(t.ToolsInvoked) > 0 {
						b, _ := json.Marshal(t.ToolsInvoked)
						toolsJSON = string(b)
					} else {
						toolsJSON = "[]"
					}
				}
				toolCallsJSON := t.ToolCallsJSON
				if toolCallsJSON == "" {
					if len(t.ToolCalls) > 0 {
						b, _ := json.Marshal(t.ToolCalls)
						toolCallsJSON = string(b)
					} else {
						toolCallsJSON = "[]"
					}
				}
				toolResultsJSON := t.ToolResultsJSON
				if toolResultsJSON == "" {
					if len(t.ToolResults) > 0 {
						b, _ := json.Marshal(t.ToolResults)
						toolResultsJSON = string(b)
					} else {
						toolResultsJSON = "[]"
					}
				}
				if _, err := turnStmt.ExecContext(ctx,
					t.ID, s.ID, t.TurnIndex, t.Timestamp, t.Role,
					t.ModelName, t.Content, t.Thinking, t.ReasoningEffort,
					t.InputTokens, t.OutputTokens, t.CacheReadTokens,
					t.CacheCreationTokens, t.CostUSD, toolsJSON,
					toolCallsJSON, toolResultsJSON, t.RawPayloadJSON,
				); err != nil {
					return fmt.Errorf("failed to insert message turn: %w", err)
				}
			}
		}

		// 3. Insert Subagent runs if any
		if len(s.SubagentRuns) > 0 {
			subStmt, err := tx.PrepareContext(ctx, `
				INSERT INTO subagent_runs (
					id, parent_session_id, child_session_id, agent_type, tokens, cost_usd, created_at
				) VALUES (?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(child_session_id) DO UPDATE SET
					parent_session_id = excluded.parent_session_id,
					tokens = excluded.tokens,
					cost_usd = excluded.cost_usd;
			`)
			if err != nil {
				return fmt.Errorf("failed to prepare subagent insert: %w", err)
			}
			defer subStmt.Close()

			for _, sub := range s.SubagentRuns {
				if sub.CreatedAt.IsZero() {
					sub.CreatedAt = now
				}
				if _, err := subStmt.ExecContext(ctx,
					sub.ID, s.ID, sub.ChildSessionID, sub.AgentType, sub.Tokens, sub.CostUSD, sub.CreatedAt,
				); err != nil {
					return fmt.Errorf("failed to insert subagent run: %w", err)
				}
			}
		}

		// If this session itself is a subagent, link to parent in subagent_runs
		if s.IsSubagent && s.ParentSessionID != "" {
			subStmt, err := tx.PrepareContext(ctx, `
				INSERT INTO subagent_runs (
					id, parent_session_id, child_session_id, agent_type, tokens, cost_usd, created_at
				) VALUES (?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(child_session_id) DO UPDATE SET
					parent_session_id = excluded.parent_session_id,
					tokens = excluded.tokens,
					cost_usd = excluded.cost_usd;
			`)
			if err == nil {
				agentType := s.SubagentType
				if agentType == "" {
					agentType = s.AgentName + "-subagent"
				}
				totTokens := s.InputTokens + s.OutputTokens
				subID := "subrun:" + s.ID
				runTime := s.StartTime
				if runTime.IsZero() {
					runTime = now
				}
				_, _ = subStmt.ExecContext(ctx,
					subID, s.ParentSessionID, s.ID, agentType, totTokens, s.NetCostUSD, runTime,
				)
				subStmt.Close()
			}
		}

		return nil
	})
}

// GetExistingSessionIDs checks a slice of sessions and returns a map of session IDs that already exist in the database.
func (d *DB) GetExistingSessionIDs(ctx context.Context, sessions []models.Session) (map[string]bool, error) {
	if len(sessions) == 0 {
		return map[string]bool{}, nil
	}
	existing := make(map[string]bool)
	for i := 0; i < len(sessions); i += 50 {
		end := i + 50
		if end > len(sessions) {
			end = len(sessions)
		}
		chunk := sessions[i:end]
		placeholders := make([]string, len(chunk))
		args := make([]interface{}, len(chunk))
		for j, s := range chunk {
			placeholders[j] = "?"
			id := s.ID
			if id == "" {
				id = s.FilePath
			}
			args[j] = id
		}
		query := fmt.Sprintf("SELECT id FROM sessions WHERE id IN (%s);", strings.Join(placeholders, ","))
		rows, err := d.readerDB.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing session IDs: %w", err)
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err == nil {
				existing[id] = true
			}
		}
		rows.Close()
	}
	return existing, nil
}

// GetSession retrieves a single session by primary ID.
func (d *DB) GetSession(ctx context.Context, id string) (*models.Session, error) {
	query := `
	SELECT
		id, session_id, agent_name, project_name, file_path, machine_id,
		created_at, updated_at, start_time, end_time, duration_seconds,
		model_raw, model_resolved, input_tokens, output_tokens,
		cache_read_tokens, cache_creation_tokens, gross_cost_usd, net_cost_usd,
		electricity_cost_usd, hardware_profile, status, git_branch,
		is_subagent, parent_session_id, subagent_type
	FROM sessions
	WHERE id = ? OR session_id = ?
	LIMIT 1;
	`
	row := d.readerDB.QueryRowContext(ctx, query, id, id)

	var s models.Session
	var isSubagentInt int
	err := row.Scan(
		&s.ID, &s.SessionID, &s.AgentName, &s.ProjectName, &s.FilePath, &s.MachineID,
		&s.CreatedAt, &s.UpdatedAt, &s.StartTime, &s.EndTime, &s.DurationSeconds,
		&s.ModelRaw, &s.ModelResolved, &s.InputTokens, &s.OutputTokens,
		&s.CacheReadTokens, &s.CacheCreationTokens, &s.GrossCostUSD, &s.NetCostUSD,
		&s.ElectricityCostUSD, &s.HardwareProfile, &s.Status, &s.GitBranch,
		&isSubagentInt, &s.ParentSessionID, &s.SubagentType,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	s.IsSubagent = isSubagentInt == 1
	return &s, nil
}

// GetSessionDetail fetches a full session along with all message turns and subagent runs.
func (d *DB) GetSessionDetail(ctx context.Context, id string) (*models.Session, error) {
	s, err := d.GetSession(ctx, id)
	if err != nil {
		return nil, err
	}

	// 1. Fetch turns
	turns, err := d.GetMessageTurns(ctx, s.ID)
	if err != nil {
		return nil, err
	}
	s.Turns = turns

	// 2. Fetch subagent runs
	subagents, err := d.GetSubagentRuns(ctx, s.ID)
	if err != nil {
		return nil, err
	}
	s.SubagentRuns = subagents

	return s, nil
}

// GetMessageTurns fetches all message turns for a session in sequential order.
func (d *DB) GetMessageTurns(ctx context.Context, sessionID string) ([]models.MessageTurn, error) {
	query := `
	SELECT
		id, session_id, turn_index, timestamp, role,
		model_name, COALESCE(content, ''), COALESCE(thinking, ''), COALESCE(reasoning_effort, ''),
		input_tokens, output_tokens, cache_read_tokens,
		cache_creation_tokens, cost_usd, COALESCE(tools_invoked_json, '[]'),
		COALESCE(tool_calls_json, '[]'), COALESCE(tool_results_json, '[]'), COALESCE(raw_payload_json, '')
	FROM message_turns
	WHERE session_id = ?
	ORDER BY turn_index ASC;
	`
	rows, err := d.readerDB.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query message turns: %w", err)
	}
	defer rows.Close()

	var turns []models.MessageTurn
	for rows.Next() {
		var t models.MessageTurn
		if err := rows.Scan(
			&t.ID, &t.SessionID, &t.TurnIndex, &t.Timestamp, &t.Role,
			&t.ModelName, &t.Content, &t.Thinking, &t.ReasoningEffort,
			&t.InputTokens, &t.OutputTokens, &t.CacheReadTokens,
			&t.CacheCreationTokens, &t.CostUSD, &t.ToolsInvokedJSON,
			&t.ToolCallsJSON, &t.ToolResultsJSON, &t.RawPayloadJSON,
		); err != nil {
			return nil, fmt.Errorf("failed to scan message turn: %w", err)
		}
		if t.ToolsInvokedJSON != "" && t.ToolsInvokedJSON != "[]" {
			_ = json.Unmarshal([]byte(t.ToolsInvokedJSON), &t.ToolsInvoked)
		}
		if t.ToolCallsJSON != "" && t.ToolCallsJSON != "[]" {
			_ = json.Unmarshal([]byte(t.ToolCallsJSON), &t.ToolCalls)
		}
		if t.ToolResultsJSON != "" && t.ToolResultsJSON != "[]" {
			_ = json.Unmarshal([]byte(t.ToolResultsJSON), &t.ToolResults)
		}
		turns = append(turns, t)
	}
	return turns, rows.Err()
}

// GetSubagentRuns fetches all subagent runs linked to a parent session.
func (d *DB) GetSubagentRuns(ctx context.Context, parentSessionID string) ([]models.SubagentRun, error) {
	altID := parentSessionID
	if strings.Contains(parentSessionID, ":") {
		parts := strings.SplitN(parentSessionID, ":", 2)
		altID = parts[1]
	} else {
		altID = "dsh:" + parentSessionID
	}
	query := `
	SELECT
		id, parent_session_id, child_session_id, agent_type, tokens, cost_usd, created_at
	FROM subagent_runs
	WHERE parent_session_id = ? OR parent_session_id = ?
	ORDER BY created_at ASC;
	`
	rows, err := d.readerDB.QueryContext(ctx, query, parentSessionID, altID)
	if err != nil {
		return nil, fmt.Errorf("failed to query subagents: %w", err)
	}
	defer rows.Close()

	var runs []models.SubagentRun
	for rows.Next() {
		var r models.SubagentRun
		if err := rows.Scan(
			&r.ID, &r.ParentSessionID, &r.ChildSessionID, &r.AgentType, &r.Tokens, &r.CostUSD, &r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan subagent run: %w", err)
		}
		if child, err := d.GetSessionDetail(ctx, r.ChildSessionID); err == nil && child != nil && child.DSH != nil && child.DSH.Sandbox != nil {
			r.Sandbox = child.DSH.Sandbox
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

// Allowed sort field mapping (Whitelisted column expressions)
var sortColumnMap = map[models.SortField]string{
	models.SortByStartTime:    "s.start_time",
	models.SortByEndTime:      "s.end_time",
	models.SortByUpdatedAt:    "s.updated_at",
	models.SortByCost:         "s.net_cost_usd",
	models.SortByTokens:       "(s.input_tokens + s.output_tokens)",
	models.SortByInputTokens:  "s.input_tokens",
	models.SortByOutputTokens: "s.output_tokens",
	models.SortByDuration:     "s.duration_seconds",
	models.SortByRelevance:    "rank",
}

// SanitizeFTSQuery transforms raw user search input into a safe FTS5 MATCH expression.
func SanitizeFTSQuery(input string, scope string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	// Remove dangerous characters that break FTS parser
	re := regexp.MustCompile(`[^\w\s\-\.\_\/\*\"]+`)
	clean := re.ReplaceAllString(input, " ")

	tokens := strings.Fields(clean)
	if len(tokens) == 0 {
		return ""
	}

	var terms []string
	for _, t := range tokens {
		t = strings.TrimSpace(t)
		if t == "" || t == "*" || t == "-" || strings.EqualFold(t, "AND") || strings.EqualFold(t, "OR") || strings.EqualFold(t, "NOT") {
			continue
		}
		// Handle phrase or wildcard term
		if strings.HasPrefix(t, "\"") && strings.HasSuffix(t, "\"") && len(t) > 2 {
			terms = append(terms, t)
		} else {
			termClean := strings.Trim(t, `"*`)
			if termClean == "" {
				continue
			}
			if strings.HasSuffix(t, "*") {
				terms = append(terms, `"`+termClean+`"*`)
			} else {
				terms = append(terms, `"`+termClean+`"*`)
			}
		}
	}

	if len(terms) == 0 {
		return ""
	}

	matchExpr := strings.Join(terms, " ")
	if scope != "" && scope != "all" {
		switch scope {
		case "project", "project_name":
			return fmt.Sprintf("{project_name} : %s", matchExpr)
		case "agent", "agent_name":
			return fmt.Sprintf("{agent_name} : %s", matchExpr)
		case "model":
			return fmt.Sprintf("{model_resolved} : %s", matchExpr)
		case "session_id":
			return fmt.Sprintf("{session_id} : %s", matchExpr)
		case "branch", "git_branch":
			return fmt.Sprintf("{git_branch} : %s", matchExpr)
		}
	}

	return matchExpr
}

func buildSessionFilterQuery(params models.FilterParams) (fromSQL string, whereSQL string, args []interface{}) {
	var where []string
	fromSQL = "sessions s"

	// 1. FTS5 Full-Text Search Integration
	ftsQuery := SanitizeFTSQuery(params.Search, params.SearchScope)
	if ftsQuery != "" {
		fromSQL = "sessions_fts fts JOIN sessions s ON s.rowid = fts.rowid"
		where = append(where, "sessions_fts MATCH ?")
		args = append(args, ftsQuery)
	}

	// 2. Agents Filter (Multi-select)
	agents := params.Agents
	if len(agents) == 0 && params.Agent != "" {
		for _, a := range strings.Split(params.Agent, ",") {
			if trimmed := strings.TrimSpace(a); trimmed != "" {
				agents = append(agents, trimmed)
			}
		}
	}
	if len(agents) > 0 {
		placeholders := make([]string, len(agents))
		for i, a := range agents {
			placeholders[i] = "?"
			args = append(args, a)
		}
		where = append(where, fmt.Sprintf("s.agent_name IN (%s)", strings.Join(placeholders, ",")))
	}

	// 3. Projects Filter (Multi-select)
	projects := params.Projects
	if len(projects) == 0 && params.Project != "" {
		for _, p := range strings.Split(params.Project, ",") {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				projects = append(projects, trimmed)
			}
		}
	}
	if len(projects) > 0 {
		placeholders := make([]string, len(projects))
		for i, p := range projects {
			placeholders[i] = "?"
			args = append(args, models.CanonicalProject(p))
		}
		where = append(where, fmt.Sprintf("s.project_name IN (%s)", strings.Join(placeholders, ",")))
	}

	// 4. Models Filter (Raw or Resolved)
	modelsList := params.Models
	if len(modelsList) == 0 && params.Model != "" {
		for _, m := range strings.Split(params.Model, ",") {
			if trimmed := strings.TrimSpace(m); trimmed != "" {
				modelsList = append(modelsList, trimmed)
			}
		}
	}
	if len(modelsList) > 0 {
		placeholders := make([]string, len(modelsList))
		for i, m := range modelsList {
			placeholders[i] = "?"
			args = append(args, m)
		}
		inList := strings.Join(placeholders, ",")
		where = append(where, fmt.Sprintf("(s.model_resolved IN (%s) OR s.model_raw IN (%s))", inList, inList))
		for _, m := range modelsList {
			args = append(args, m)
		}
	}

	// 5. Machine IDs Filter
	machineIDs := params.MachineIDs
	if len(machineIDs) == 0 && params.MachineID != "" {
		for _, m := range strings.Split(params.MachineID, ",") {
			if trimmed := strings.TrimSpace(m); trimmed != "" {
				machineIDs = append(machineIDs, trimmed)
			}
		}
	}
	if len(machineIDs) > 0 {
		placeholders := make([]string, len(machineIDs))
		for i, m := range machineIDs {
			placeholders[i] = "?"
			args = append(args, m)
		}
		where = append(where, fmt.Sprintf("s.machine_id IN (%s)", strings.Join(placeholders, ",")))
	}

	// 6. Status Filter
	if params.Status != "" && params.Status != "all" {
		where = append(where, "s.status = ?")
		args = append(args, params.Status)
	}

	// 7. Git Branch Filter
	if params.GitBranch != "" {
		if strings.Contains(params.GitBranch, "*") {
			where = append(where, "s.git_branch LIKE ?")
			args = append(args, strings.ReplaceAll(params.GitBranch, "*", "%"))
		} else {
			where = append(where, "s.git_branch = ?")
			args = append(args, params.GitBranch)
		}
	}

	// 8. Subagent Filter
	if params.IsSubagent != nil {
		if *params.IsSubagent {
			where = append(where, "s.is_subagent = 1")
		} else {
			where = append(where, "s.is_subagent = 0")
		}
	}
	if params.ParentSessionID != "" {
		where = append(where, "s.parent_session_id = ?")
		args = append(args, params.ParentSessionID)
	}
	if len(params.SubagentTypes) > 0 {
		placeholders := make([]string, len(params.SubagentTypes))
		for i, st := range params.SubagentTypes {
			placeholders[i] = "?"
			args = append(args, st)
		}
		where = append(where, fmt.Sprintf("s.subagent_type IN (%s)", strings.Join(placeholders, ",")))
	}

	// 9. Tools Invocation Filter (Subquery Check)
	if len(params.Tools) > 0 {
		for _, t := range params.Tools {
			where = append(where, `EXISTS (
				SELECT 1 FROM message_turns mt
				WHERE mt.session_id = s.id AND mt.tools_invoked_json LIKE ?
			)`)
			args = append(args, "%\""+t+"\"%")
		}
	}

	// 10. Temporal Range
	if !params.StartDate.IsZero() {
		where = append(where, "s.start_time >= ?")
		args = append(args, params.StartDate)
	}
	if !params.EndDate.IsZero() {
		where = append(where, "s.start_time <= ?")
		args = append(args, params.EndDate)
	}

	// 11. Numeric Range (Cost, Tokens, Duration)
	if params.MinCostUSD != nil {
		where = append(where, "s.net_cost_usd >= ?")
		args = append(args, *params.MinCostUSD)
	}
	if params.MaxCostUSD != nil {
		where = append(where, "s.net_cost_usd <= ?")
		args = append(args, *params.MaxCostUSD)
	}
	if params.MinTokens != nil {
		where = append(where, "(s.input_tokens + s.output_tokens) >= ?")
		args = append(args, *params.MinTokens)
	}
	if params.MaxTokens != nil {
		where = append(where, "(s.input_tokens + s.output_tokens) <= ?")
		args = append(args, *params.MaxTokens)
	}
	if params.MinInputTokens != nil {
		where = append(where, "s.input_tokens >= ?")
		args = append(args, *params.MinInputTokens)
	}
	if params.MaxInputTokens != nil {
		where = append(where, "s.input_tokens <= ?")
		args = append(args, *params.MaxInputTokens)
	}
	if params.MinOutputTokens != nil {
		where = append(where, "s.output_tokens >= ?")
		args = append(args, *params.MinOutputTokens)
	}
	if params.MaxOutputTokens != nil {
		where = append(where, "s.output_tokens <= ?")
		args = append(args, *params.MaxOutputTokens)
	}
	if params.MinDurationSec != nil {
		where = append(where, "s.duration_seconds >= ?")
		args = append(args, *params.MinDurationSec)
	}
	if params.MaxDurationSec != nil {
		where = append(where, "s.duration_seconds <= ?")
		args = append(args, *params.MaxDurationSec)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	return fromSQL, whereClause, args
}

// ListSessions queries sessions with multi-criteria filtering, full-text search, and pagination.
func (d *DB) ListSessions(ctx context.Context, params models.FilterParams) ([]models.Session, int64, error) {
	fromTable, whereSQL, args := buildSessionFilterQuery(params)

	// 1. Total Count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s %s;", fromTable, whereSQL)
	var total int64
	if err := d.readerDB.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count sessions: %w", err)
	}

	if total == 0 {
		return []models.Session{}, 0, nil
	}

	// 2. Paginated Query
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	page := params.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	// 3. Sorting
	sortCol, ok := sortColumnMap[params.SortBy]
	if !ok {
		sortCol = "s.start_time"
	}
	sortDir := "DESC"
	if params.SortOrder == models.SortOrderAsc {
		sortDir = "ASC"
	}

	if params.SortBy == models.SortByRelevance && strings.Contains(fromTable, "sessions_fts") {
		sortCol = "rank"
		sortDir = "ASC"
	}

	query := fmt.Sprintf(`
	SELECT
		s.id, s.session_id, s.agent_name, s.project_name, s.file_path, s.machine_id,
		s.created_at, s.updated_at, s.start_time, s.end_time, s.duration_seconds,
		s.model_raw, s.model_resolved, s.input_tokens, s.output_tokens,
		s.cache_read_tokens, s.cache_creation_tokens, s.gross_cost_usd, s.net_cost_usd,
		s.electricity_cost_usd, s.hardware_profile, s.status, s.git_branch,
		s.is_subagent, s.parent_session_id, s.subagent_type
	FROM %s
	%s
	ORDER BY %s %s
	LIMIT ? OFFSET ?;
	`, fromTable, whereSQL, sortCol, sortDir)

	queryArgs := append(args, limit, offset)
	rows, err := d.readerDB.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var s models.Session
		var isSubagentInt int
		if err := rows.Scan(
			&s.ID, &s.SessionID, &s.AgentName, &s.ProjectName, &s.FilePath, &s.MachineID,
			&s.CreatedAt, &s.UpdatedAt, &s.StartTime, &s.EndTime, &s.DurationSeconds,
			&s.ModelRaw, &s.ModelResolved, &s.InputTokens, &s.OutputTokens,
			&s.CacheReadTokens, &s.CacheCreationTokens, &s.GrossCostUSD, &s.NetCostUSD,
			&s.ElectricityCostUSD, &s.HardwareProfile, &s.Status, &s.GitBranch,
			&isSubagentInt, &s.ParentSessionID, &s.SubagentType,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan session row: %w", err)
		}
		s.IsSubagent = isSubagentInt == 1
		sessions = append(sessions, s)
	}

	return sessions, total, rows.Err()
}

// GetRecentSessions returns the most recent N sessions.
func (d *DB) GetRecentSessions(ctx context.Context, limit int) ([]models.Session, error) {
	if limit <= 0 {
		limit = 20
	}
	sessions, _, err := d.ListSessions(ctx, models.FilterParams{
		Page:  1,
		Limit: limit,
	})
	return sessions, err
}

// DeleteSession removes a session by ID (cascades to turns and subagent runs).
func (d *DB) DeleteSession(ctx context.Context, id string) error {
	return d.WithTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?;`, id)
		if err != nil {
			return fmt.Errorf("failed to delete session %s: %w", id, err)
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			return ErrNotFound
		}
		return nil
	})
}
