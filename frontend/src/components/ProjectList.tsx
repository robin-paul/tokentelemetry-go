import React, { useEffect, useMemo, useState } from 'react';
import {
  Folder,
  FolderGit2,
  GitBranch,
  Search,
  ArrowRight,
  ArrowUpDown,
  LayoutGrid,
  List as ListIcon,
  ChevronRight,
  Wrench,
} from 'lucide-react';
import { apiFetch } from '../lib/api';
import { formatCost, formatTokens, formatDate } from '../lib/format';
import { getAgentMeta } from '../lib/agents';
import type { ProjectSummary, WorktreeSummary } from '../lib/types';

type ViewMode = 'grid' | 'table';
type SortKey = 'sessions' | 'tokens' | 'cost' | 'name';

const SORTS: { key: SortKey; label: string }[] = [
  { key: 'sessions', label: 'Sessions' },
  { key: 'tokens', label: 'Tokens' },
  { key: 'cost', label: 'Net Cost' },
  { key: 'name', label: 'Name' },
];

export const ProjectList: React.FC = () => {
  const [projects, setProjects] = useState<ProjectSummary[]>([]);
  const [search, setSearch] = useState('');
  const [viewMode, setViewMode] = useState<ViewMode>('grid');
  const [sortKey, setSortKey] = useState<SortKey>('sessions');
  const [sortDesc, setSortDesc] = useState(true);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Restore preferred view mode from sessionStorage if available
    try {
      const savedView = sessionStorage.getItem('tt_projects_view_mode') as ViewMode;
      if (savedView === 'grid' || savedView === 'table') {
        setViewMode(savedView);
      }
    } catch {
      // Ignore storage errors
    }

    apiFetch<ProjectSummary[]>('/api/projects')
      .then((data) => setProjects(data || []))
      .catch((e) => console.error('Failed to load projects', e))
      .finally(() => setLoading(false));
  }, []);

  const handleSetViewMode = (mode: ViewMode) => {
    setViewMode(mode);
    try {
      sessionStorage.setItem('tt_projects_view_mode', mode);
    } catch {
      // Ignore storage errors
    }
  };

  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDesc(!sortDesc);
    } else {
      setSortKey(key);
      setSortDesc(true);
    }
  };

  const filtered = useMemo(() => {
    const q = search.toLowerCase().trim();
    const list = !q
      ? projects
      : projects.filter(
          (p) =>
            (p.name || p.project_name).toLowerCase().includes(q) ||
            (p.path || p.project_name).toLowerCase().includes(q)
        );

    return [...list].sort((a, b) => {
      let av: number | string = 0;
      let bv: number | string = 0;
      switch (sortKey) {
        case 'sessions':
          av = a.session_count;
          bv = b.session_count;
          break;
        case 'tokens':
          av = a.total_tokens;
          bv = b.total_tokens;
          break;
        case 'cost':
          av = a.total_cost_usd;
          bv = b.total_cost_usd;
          break;
        case 'name':
          av = (a.name || a.project_name).toLowerCase();
          bv = (b.name || b.project_name).toLowerCase();
          break;
      }
      if (av < bv) return sortDesc ? 1 : -1;
      if (av > bv) return sortDesc ? -1 : 1;
      return 0;
    });
  }, [projects, search, sortKey, sortDesc]);

  return (
    <div className="space-y-6 max-w-[1600px] mx-auto pb-12">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-xl font-bold text-white tracking-tight">Project Workspaces</h1>
            <span className="px-2 py-0.5 rounded-full text-[11px] font-semibold bg-white/10 text-gray-300 border border-white/10">
              {projects.length} {projects.length === 1 ? 'workspace' : 'workspaces'}
            </span>
          </div>
          <p className="text-xs text-gray-400 mt-1">
            Activity grouped by workspace path and git worktree repositories.
          </p>
        </div>
      </div>

      {/* Toolbar: Search, Sort, View Toggle */}
      <div className="flex flex-wrap items-center gap-3">
        {/* Search */}
        <div className="relative flex-1 min-w-[240px]">
          <Search className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
          <input
            type="text"
            data-test="project-search-input"
            data-testid="project-search-input"
            placeholder="Search by name or path..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full bg-[#11141a] border border-white/10 rounded-lg pl-9 pr-4 py-2 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-blue-500 transition-colors"
          />
        </div>

        {/* Sort Button Group */}
        <div className="flex items-center rounded-lg border border-white/10 bg-[#11141a] overflow-hidden p-0.5">
          <span className="pl-2.5 pr-1.5 text-[10px] font-semibold uppercase tracking-wider text-gray-500">
            Sort
          </span>
          {SORTS.map((s) => {
            const isActive = sortKey === s.key;
            return (
              <button
                key={s.key}
                type="button"
                data-test={`sort-${s.key}`}
                data-testid={`sort-${s.key}`}
                onClick={() => handleSort(s.key)}
                className={`h-7 px-2.5 rounded-md text-xs font-medium transition-colors flex items-center gap-1 ${
                  isActive
                    ? 'bg-blue-500/20 text-blue-400 font-semibold'
                    : 'text-gray-400 hover:text-white hover:bg-white/5'
                }`}
              >
                {s.label}
                {isActive && (
                  <ArrowUpDown
                    className={`w-3 h-3 transition-transform ${sortDesc ? 'rotate-180' : ''}`}
                  />
                )}
              </button>
            );
          })}
        </div>

        {/* View Mode Toggle */}
        <div className="flex items-center rounded-lg border border-white/10 bg-[#11141a] overflow-hidden p-0.5">
          <button
            type="button"
            data-test="view-mode-grid"
            data-testid="view-mode-grid"
            aria-label="Grid View"
            onClick={() => handleSetViewMode('grid')}
            className={`h-7 w-8 grid place-items-center rounded-md transition-colors ${
              viewMode === 'grid'
                ? 'bg-blue-500/20 text-blue-400'
                : 'text-gray-400 hover:text-white hover:bg-white/5'
            }`}
          >
            <LayoutGrid className="w-4 h-4" />
          </button>
          <button
            type="button"
            data-test="view-mode-table"
            data-testid="view-mode-table"
            aria-label="Table View"
            onClick={() => handleSetViewMode('table')}
            className={`h-7 w-8 grid place-items-center rounded-md transition-colors ${
              viewMode === 'table'
                ? 'bg-blue-500/20 text-blue-400'
                : 'text-gray-400 hover:text-white hover:bg-white/5'
            }`}
          >
            <ListIcon className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Content */}
      {loading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {Array.from({ length: 6 }).map((_, i) => (
            <div
              key={i}
              className="h-52 rounded-xl bg-[#11141a] border border-white/5 animate-pulse"
            />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <div className="py-16 text-center rounded-xl bg-[#11141a] border border-white/10 p-8 space-y-2">
          <div className="w-10 h-10 mx-auto rounded-full bg-white/5 flex items-center justify-center text-gray-400">
            <Search className="w-5 h-5" />
          </div>
          <div className="text-sm font-semibold text-white">
            {search ? `No projects match "${search}"` : 'No projects detected yet'}
          </div>
          <p className="text-xs text-gray-500 max-w-sm mx-auto">
            {search
              ? 'Try searching with a shorter keyword or path fragment.'
              : 'Once agent telemetry activity is indexed, project workspaces will appear here.'}
          </p>
        </div>
      ) : viewMode === 'grid' ? (
        <div
          className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4"
          data-test="projects-grid"
          data-testid="projects-grid"
        >
          {filtered.map((p) => (
            <ProjectCard key={p.path || p.project_name} project={p} />
          ))}
        </div>
      ) : (
        /* Table / List View */
        <div
          className="rounded-xl bg-[#11141a] border border-white/10 overflow-hidden"
          data-test="projects-table"
          data-testid="projects-table"
        >
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead>
                <tr className="border-b border-white/10 text-gray-400 font-medium bg-white/[0.02]">
                  <th className="py-3 px-5">Project Workspace</th>
                  <th className="py-3 px-4">Agents</th>
                  <th className="py-3 px-4 text-right">Sessions</th>
                  <th className="py-3 px-4 text-right">Subagents</th>
                  <th className="py-3 px-4 text-right">Plans</th>
                  <th className="py-3 px-4 text-right">Tokens</th>
                  <th className="py-3 px-5 text-right">Net Cost</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {filtered.map((p) => {
                  const projPath = p.path || p.project_name;
                  const projName = p.name || p.project_name;
                  const worktrees = p.worktrees || [];
                  const hasWorktrees = worktrees.length > 0;
                  const agents = p.agents || [];

                  return (
                    <tr
                      key={projPath}
                      onClick={() => (window.location.href = `/projects/${encodeURIComponent(projPath)}`)}
                      className="hover:bg-white/[0.03] cursor-pointer transition-colors group"
                      data-test={`project-row-${projName}`}
                      data-testid={`project-row-${projName}`}
                    >
                      <td className="py-3.5 px-5">
                        <div className="flex items-center gap-2">
                          {hasWorktrees ? (
                            <GitBranch className="w-3.5 h-3.5 text-blue-400 shrink-0" />
                          ) : (
                            <Folder className="w-3.5 h-3.5 text-gray-400 shrink-0" />
                          )}
                          <div className="font-semibold text-white group-hover:text-blue-400 transition-colors truncate max-w-[240px]">
                            {projName}
                          </div>
                          {hasWorktrees && (
                            <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-blue-500/10 text-blue-400 border border-blue-500/20 shrink-0">
                              +{worktrees.length} wt
                            </span>
                          )}
                        </div>
                        <div className="font-mono text-[11px] text-gray-500 truncate max-w-[320px] mt-0.5">
                          {p.is_worktree && p.parent_name
                            ? `⑂ worktree of ${p.parent_name} · ${projPath}`
                            : projPath}
                        </div>
                      </td>

                      <td className="py-3.5 px-4">
                        <div className="flex items-center gap-1 flex-wrap">
                          {agents.slice(0, 4).map((a) => {
                            const meta = getAgentMeta(a);
                            return (
                              <span
                                key={a}
                                title={meta.label}
                                className="px-1.5 py-0.5 rounded text-[10px] font-medium"
                                style={{ color: meta.color, backgroundColor: meta.bg }}
                              >
                                {meta.label}
                              </span>
                            );
                          })}
                          {agents.length > 4 && (
                            <span className="text-[10px] text-gray-500">+{agents.length - 4}</span>
                          )}
                        </div>
                      </td>

                      <td className="py-3.5 px-4 text-right font-semibold text-white tabular">
                        {p.session_count}
                      </td>

                      <td className="py-3.5 px-4 text-right tabular text-purple-400 font-medium">
                        {(p.configured_subagent_count ?? 0) + (p.subagent_count ?? 0)}
                      </td>

                      <td className="py-3.5 px-4 text-right tabular text-emerald-400 font-medium">
                        {p.plan_count || 0}
                      </td>

                      <td className="py-3.5 px-4 text-right tabular text-gray-300">
                        {formatTokens(p.total_tokens)}
                      </td>

                      <td className="py-3.5 px-5 text-right tabular text-emerald-400 font-semibold">
                        {formatCost(p.total_cost_usd)}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
};

/* ──────────────────────────────────────────────────────────────────────────
   Project Card Component
   ────────────────────────────────────────────────────────────────────────── */

interface ProjectCardProps {
  project: ProjectSummary;
}

const ProjectCard: React.FC<ProjectCardProps> = ({ project }) => {
  const [openWorktrees, setOpenWorktrees] = useState(false);
  const projPath = project.path || project.project_name;
  const projName = project.name || project.project_name;
  const worktrees = project.worktrees || [];
  const hasWorktrees = worktrees.length > 0;
  const agg = project.aggregate;
  const agents = project.agents || (agg ? agg.agents : []);
  const mcpTools = project.mcp_tools || (agg ? agg.mcp_tools : []);
  const subCount = (project.configured_subagent_count ?? 0) + (project.subagent_count ?? 0);

  return (
    <div
      data-test={`project-card-${projName}`}
      data-testid={`project-card-${projName}`}
      className="rounded-xl bg-[#11141a] border border-white/10 hover:border-blue-500/40 transition-all flex flex-col overflow-hidden group shadow-sm"
    >
      {/* Top Main Link */}
      <a
        href={`/projects/${encodeURIComponent(projPath)}`}
        className="p-5 pb-3 block hover:bg-white/[0.01] transition-colors"
      >
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-start gap-3 min-w-0">
            <div className="w-9 h-9 rounded-lg bg-blue-500/10 border border-blue-500/20 flex items-center justify-center text-blue-400 shrink-0 group-hover:scale-105 transition-transform">
              {hasWorktrees ? (
                <GitBranch className="w-4 h-4" />
              ) : (
                <FolderGit2 className="w-4 h-4" />
              )}
            </div>
            <div className="min-w-0">
              <div
                className="text-sm font-semibold text-white truncate group-hover:text-blue-400 transition-colors"
                title={projName}
              >
                {projName}
              </div>
              <div
                className="font-mono text-[10px] text-gray-500 truncate mt-0.5"
                title={projPath}
              >
                {projPath}
              </div>
            </div>
          </div>
          <ArrowRight className="w-4 h-4 text-gray-500 group-hover:text-blue-400 group-hover:translate-x-0.5 transition-all shrink-0 mt-1" />
        </div>
      </a>

      {/* Worktree Parent Badge (child cards) */}
      {project.is_worktree && project.parent_path && (
        <div className="px-5 pb-2.5">
          <a
            href={`/projects/${encodeURIComponent(project.parent_path)}`}
            className="inline-flex items-center gap-1 text-[11px] text-gray-400 hover:text-blue-400 transition-colors"
          >
            <GitBranch className="w-3 h-3 text-blue-400" /> worktree of{' '}
            <span className="font-medium text-gray-300">{project.parent_name}</span>
          </a>
        </div>
      )}

      {/* Agents Row */}
      {agents.length > 0 && (
        <div className="px-5 pb-3 flex items-center gap-1.5 flex-wrap">
          {agents.slice(0, 5).map((a) => {
            const meta = getAgentMeta(a);
            return (
              <span
                key={a}
                title={meta.label}
                className="px-2 py-0.5 rounded text-[10px] font-medium border"
                style={{
                  color: meta.color,
                  backgroundColor: meta.bg,
                  borderColor: `${meta.color}33`,
                }}
              >
                {meta.label}
              </span>
            );
          })}
          {agents.length > 5 && (
            <span className="text-[10px] text-gray-500">+{agents.length - 5}</span>
          )}
        </div>
      )}

      {/* MCP Tools Row */}
      {mcpTools.length > 0 && (
        <div className="px-5 pb-3 flex items-center gap-1.5 flex-wrap">
          <Wrench className="w-3 h-3 text-gray-500 shrink-0" />
          {mcpTools.slice(0, 3).map((t) => (
            <span
              key={t}
              className="px-1.5 py-0.2 rounded text-[10px] font-mono text-gray-400 bg-white/5 border border-white/5"
            >
              {t}
            </span>
          ))}
          {mcpTools.length > 3 && (
            <span className="text-[10px] text-gray-500">+{mcpTools.length - 3}</span>
          )}
        </div>
      )}

      {/* Worktrees Collapsible (parent cards) */}
      {hasWorktrees && (
        <div className="px-5 pb-3 pt-1 border-t border-white/5">
          <button
            type="button"
            data-test={`toggle-worktrees-${projName}`}
            data-testid={`toggle-worktrees-${projName}`}
            onClick={() => setOpenWorktrees(!openWorktrees)}
            className="w-full flex items-center justify-between text-[11px] font-medium text-gray-400 hover:text-white transition-colors py-1"
          >
            <span className="inline-flex items-center gap-1.5">
              <ChevronRight
                className={`w-3.5 h-3.5 text-blue-400 transition-transform ${
                  openWorktrees ? 'rotate-90' : ''
                }`}
              />
              <GitBranch className="w-3 h-3 text-blue-400" />
              <span>
                {worktrees.length} {worktrees.length === 1 ? 'worktree' : 'worktrees'}
              </span>
            </span>
            {agg && (
              <span className="text-[10px] font-semibold text-emerald-400 tabular">
                ∑ {formatCost(agg.total_cost_usd)}
              </span>
            )}
          </button>

          {openWorktrees && (
            <div className="mt-2 space-y-1 pl-3 border-l-2 border-blue-500/20">
              {worktrees.map((w: WorktreeSummary) => (
                <a
                  key={w.path}
                  href={`/projects/${encodeURIComponent(w.path)}`}
                  className="flex items-center justify-between gap-2 py-1 px-2 rounded-md text-[11px] text-gray-400 hover:text-white hover:bg-white/5 transition-colors"
                >
                  <span className="font-mono truncate" title={w.path}>
                    {w.name}
                  </span>
                  <span className="tabular text-gray-500 text-[10px] shrink-0">
                    {w.session_count}s · {formatCost(w.total_cost_usd)}
                  </span>
                </a>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Bottom KPI Stat Row */}
      <div className="mt-auto grid grid-cols-4 border-t border-white/5 divide-x divide-white/5 bg-white/[0.01]">
        <div className="py-2.5 px-2 text-center">
          <div className="text-xs font-semibold text-white tabular">{project.session_count}</div>
          <div className="text-[9px] uppercase tracking-wider text-gray-500 mt-0.5">Sessions</div>
        </div>
        <div className="py-2.5 px-2 text-center">
          <div className="text-xs font-semibold text-purple-400 tabular">{subCount}</div>
          <div className="text-[9px] uppercase tracking-wider text-gray-500 mt-0.5">Subs</div>
        </div>
        <div className="py-2.5 px-2 text-center">
          <div className="text-xs font-semibold text-emerald-400 tabular">
            {project.plan_count || 0}
          </div>
          <div className="text-[9px] uppercase tracking-wider text-gray-500 mt-0.5">Plans</div>
        </div>
        <div className="py-2.5 px-2 text-center" title={`${formatTokens(project.total_tokens)} tokens`}>
          <div className="text-xs font-semibold text-emerald-400 tabular">
            {formatCost(project.total_cost_usd)}
          </div>
          <div className="text-[9px] uppercase tracking-wider text-gray-500 mt-0.5">Net Cost</div>
        </div>
      </div>
    </div>
  );
};
