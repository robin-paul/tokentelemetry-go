import React, { useEffect, useMemo, useState } from 'react';
import {
  ArrowLeft,
  FolderGit2,
  GitBranch,
  Coins,
  Cpu,
  Layers,
  Clock,
  Activity,
  ClipboardList,
  Settings2,
  Search,
  ArrowUpRight,
  EyeOff,
  Eye,
  Tag,
  Puzzle,
  Users,
  BookOpen,
  Terminal,
  Wrench,
  FileText,
} from 'lucide-react';
import { apiFetch } from '../lib/api';
import { formatCost, formatDuration, formatTokens, formatDate } from '../lib/format';
import { getAgentMeta } from '../lib/agents';
import type { ProjectSummary, Session, WorktreeSummary } from '../lib/types';

interface ProjectDetailProps {
  projectPath?: string;
}

type TabKey = 'activity' | 'plans' | 'config';

interface PlanItem {
  sessionId: string;
  agent: string;
  timestamp: string;
  title: string;
  content: string;
}

interface ProjectConfigResponse {
  project: string;
  project_valid: boolean;
  skills: Array<{ name: string; description?: string; agent?: string; scope?: string }>;
  mcps: Array<{ name: string; command?: string; scope?: string }>;
  memory: Array<{ name: string; path?: string; scope?: string }>;
  commands: Array<{ name: string; scope?: string }>;
  subagents: Array<{ name: string; scope?: string }>;
  plugins: Array<{ name: string; scope?: string }>;
  counts: {
    skills: number;
    mcps: number;
    memory_files: number;
    commands: number;
    subagents: number;
    plugins: number;
  };
}

export const ProjectDetail: React.FC<ProjectDetailProps> = ({ projectPath: propPath }) => {
  const [project, setProject] = useState<ProjectSummary | null>(null);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [config, setConfig] = useState<ProjectConfigResponse | null>(null);
  const [hiddenProjects, setHiddenProjects] = useState<string[]>([]);
  const [aliases, setAliases] = useState<Record<string, string>>({});
  const [newAlias, setNewAlias] = useState('');
  const [aliasSaved, setAliasSaved] = useState(false);
  const [activeTab, setActiveTab] = useState<TabKey>('activity');
  const [sessionSearch, setSessionSearch] = useState('');
  const [agentFilter, setAgentFilter] = useState('');
  const [loading, setLoading] = useState(true);

  // Determine decoded workspace path
  const targetPath = useMemo(() => {
    let p = propPath;
    if ((!p || p.startsWith('[')) && typeof window !== 'undefined') {
      const parts = window.location.pathname.split('/').filter(Boolean);
      if (parts[0] === 'projects' && parts[1]) {
        p = decodeURIComponent(parts.slice(1).join('/'));
      }
    }
    return p && !p.startsWith('[') ? p : '';
  }, [propPath]);

  // Sync tab with URL query parameter
  useEffect(() => {
    if (typeof window !== 'undefined') {
      const params = new URLSearchParams(window.location.search);
      const tabParam = params.get('tab') as TabKey;
      if (tabParam === 'activity' || tabParam === 'plans' || tabParam === 'config') {
        setActiveTab(tabParam);
      }
    }
  }, []);

  const handleTabChange = (tab: TabKey) => {
    setActiveTab(tab);
    if (typeof window !== 'undefined') {
      const url = new URL(window.location.href);
      url.searchParams.set('tab', tab);
      window.history.replaceState({}, '', url.toString());
    }
  };

  useEffect(() => {
    if (!targetPath) return;

    setLoading(true);
    Promise.all([
      apiFetch<{ project: ProjectSummary; sessions: Session[] }>(
        `/api/projects/${encodeURIComponent(targetPath)}`
      ),
      apiFetch<string[]>('/config/hidden').catch(() => [] as string[]),
      apiFetch<{ aliases: Record<string, string> }>('/config/aliases')
        .then((r) => r.aliases || {})
        .catch(() => ({} as Record<string, string>)),
      apiFetch<ProjectConfigResponse>(`/config?project=${encodeURIComponent(targetPath)}`).catch(
        () => null
      ),
    ])
      .then(([res, hiddenList, aliasesMap, configData]) => {
        setProject(res.project);
        setSessions(res.sessions || []);
        setHiddenProjects(hiddenList);
        setAliases(aliasesMap);
        setNewAlias(aliasesMap[targetPath] || aliasesMap[res.project.project_name] || '');
        setConfig(configData);
      })
      .catch((e) => console.error('Failed to load project detail', e))
      .finally(() => setLoading(false));
  }, [targetPath]);

  // Extract architectural plans from sessions
  const plans = useMemo<PlanItem[]>(() => {
    const list: PlanItem[] = [];
    for (const s of sessions) {
      if (s.status?.toLowerCase().includes('plan') || s.subagent_type?.toLowerCase() === 'plan') {
        list.push({
          sessionId: s.id,
          agent: s.agent_name,
          timestamp: s.start_time,
          title: `Architecture Plan from ${s.model_resolved || s.model_raw || 'Agent'}`,
          content:
            s.instructions ||
            `Session ${s.session_id.slice(0, 8)} executed in Plan mode with ${s.turns?.length || 0} turns.`,
        });
      }
      if (s.turns) {
        for (const t of s.turns) {
          if (
            t.tools_invoked?.some((name) => name.toLowerCase().includes('plan')) ||
            t.content?.includes('## Plan') ||
            t.content?.includes('# Implementation Plan')
          ) {
            list.push({
              sessionId: s.id,
              agent: s.agent_name,
              timestamp: t.timestamp ? new Date(t.timestamp).toISOString() : s.start_time,
              title: `Plan generated in turn #${t.turn_index}`,
              content: t.content || 'Plan artifact generated during turn execution.',
            });
          }
        }
      }
    }
    return list;
  }, [sessions]);

  // Filter sessions for the activity tab
  const filteredSessions = useMemo(() => {
    const q = sessionSearch.toLowerCase().trim();
    return sessions.filter((s) => {
      const matchQuery =
        !q ||
        s.session_id.toLowerCase().includes(q) ||
        s.agent_name.toLowerCase().includes(q) ||
        (s.model_resolved || s.model_raw).toLowerCase().includes(q);
      const matchAgent = !agentFilter || s.agent_name.toLowerCase() === agentFilter.toLowerCase();
      return matchQuery && matchAgent;
    });
  }, [sessions, sessionSearch, agentFilter]);

  const isHidden = useMemo(() => {
    if (!project) return false;
    const pPath = project.path || project.project_name;
    return hiddenProjects.includes(pPath) || hiddenProjects.includes(project.project_name);
  }, [project, hiddenProjects]);

  const handleToggleHide = async () => {
    if (!project) return;
    const pPath = project.path || project.project_name;
    try {
      if (isHidden) {
        const res = await apiFetch<{ hidden: string[] }>('/config/unhide', {
          method: 'POST',
          body: JSON.stringify({ path: pPath }),
        });
        setHiddenProjects(res.hidden || []);
      } else {
        const res = await apiFetch<{ hidden: string[] }>('/config/hide', {
          method: 'POST',
          body: JSON.stringify({ path: pPath }),
        });
        setHiddenProjects(res.hidden || []);
      }
    } catch (e) {
      console.error('Failed to toggle hide state', e);
    }
  };

  const handleSaveAlias = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!project) return;
    const pPath = project.path || project.project_name;
    try {
      const updated = { ...aliases, [pPath]: newAlias.trim() };
      await apiFetch<{ ok: boolean; aliases: Record<string, string> }>('/config/aliases', {
        method: 'POST',
        body: JSON.stringify({ [pPath]: newAlias.trim() }),
      });
      setAliases(updated);
      setAliasSaved(true);
      setTimeout(() => setAliasSaved(false), 2000);
    } catch (e) {
      console.error('Failed to save alias', e);
    }
  };

  if (loading) {
    return (
      <div className="p-12 text-center text-xs text-gray-500 max-w-[1600px] mx-auto space-y-3">
        <div className="w-8 h-8 mx-auto rounded-full border-2 border-blue-500/30 border-t-blue-500 animate-spin" />
        <div>Loading project workspace telemetry...</div>
      </div>
    );
  }

  if (!project) {
    return (
      <div className="p-12 text-center space-y-3 max-w-[1600px] mx-auto">
        <div className="text-sm font-semibold text-gray-300">Project workspace not found.</div>
        <a
          href="/projects"
          className="inline-flex items-center gap-1.5 text-xs text-blue-400 hover:underline"
        >
          <ArrowLeft className="w-3.5 h-3.5" /> Return to projects catalog
        </a>
      </div>
    );
  }

  const projName = project.name || project.project_name;
  const projPath = project.path || project.project_name;
  const worktrees = project.worktrees || [];
  const hasWorktrees = worktrees.length > 0;
  const agg = project.aggregate;
  const agents = project.agents || [];
  const displayAlias = aliases[projPath] || aliases[project.project_name];

  return (
    <div className="space-y-6 max-w-[1600px] mx-auto pb-16">
      {/* Breadcrumb & Navigation */}
      <nav aria-label="Breadcrumb" className="flex items-center gap-2 text-xs text-gray-400">
        <a
          href="/projects"
          className="hover:text-white transition-colors flex items-center gap-1"
          data-test="back-to-projects"
          data-testid="back-to-projects"
        >
          <ArrowLeft className="w-3.5 h-3.5" /> Projects
        </a>
        <span className="text-gray-600">/</span>
        {project.is_worktree && project.parent_path && (
          <>
            <a
              href={`/projects/${encodeURIComponent(project.parent_path)}`}
              className="hover:text-white transition-colors flex items-center gap-1 text-gray-400"
            >
              <GitBranch className="w-3 h-3 text-blue-400" />
              <span>{project.parent_name || 'Parent Repo'}</span>
            </a>
            <span className="text-gray-600">/</span>
          </>
        )}
        <span className="text-gray-300 font-semibold truncate max-w-[320px]">{projName}</span>
      </nav>

      {/* Project Header Card */}
      <div className="p-6 rounded-xl bg-[#11141a] border border-white/10 shadow-sm space-y-5">
        <div className="flex flex-wrap items-start justify-between gap-6">
          <div className="flex items-start gap-4 min-w-0">
            <div className="w-12 h-12 rounded-xl bg-blue-500/10 border border-blue-500/20 flex items-center justify-center text-blue-400 shrink-0">
              {hasWorktrees ? <GitBranch className="w-6 h-6" /> : <FolderGit2 className="w-6 h-6" />}
            </div>
            <div className="min-w-0">
              <div className="flex items-center gap-2.5">
                <h1 className="text-2xl font-bold text-white tracking-tight truncate">
                  {displayAlias ? `${displayAlias} (${projName})` : projName}
                </h1>
                {isHidden && (
                  <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-amber-500/10 text-amber-400 border border-amber-500/20">
                    Hidden
                  </span>
                )}
              </div>
              <div className="mt-1.5 inline-flex items-center gap-2 px-2.5 py-1 rounded-md font-mono text-[11px] text-gray-400 bg-black/40 border border-white/5 max-w-full truncate">
                <span className="truncate" title={projPath}>
                  {projPath}
                </span>
              </div>
              {agents.length > 0 && (
                <div className="mt-3 flex items-center gap-1.5 flex-wrap">
                  {agents.map((a) => {
                    const meta = getAgentMeta(a);
                    return (
                      <span
                        key={a}
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
                </div>
              )}
            </div>
          </div>

          {/* Quick KPI Stats Pill Group */}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-px rounded-lg overflow-hidden bg-white/10 border border-white/10 text-center">
            <div className="bg-[#11141a] py-3 px-4">
              <span className="text-[10px] uppercase tracking-wider text-gray-500 flex items-center justify-center gap-1">
                <Layers className="w-3 h-3 text-blue-400" /> Sessions
              </span>
              <div className="text-base font-bold text-white tabular mt-0.5">
                {project.session_count}
              </div>
            </div>
            <div className="bg-[#11141a] py-3 px-4">
              <span className="text-[10px] uppercase tracking-wider text-gray-500 flex items-center justify-center gap-1">
                <Users className="w-3 h-3 text-purple-400" /> Subs
              </span>
              <div className="text-base font-bold text-purple-400 tabular mt-0.5">
                {(project.configured_subagent_count ?? 0) + (project.subagent_count ?? 0)}
              </div>
            </div>
            <div className="bg-[#11141a] py-3 px-4">
              <span className="text-[10px] uppercase tracking-wider text-gray-500 flex items-center justify-center gap-1">
                <Cpu className="w-3 h-3 text-amber-400" /> Tokens
              </span>
              <div className="text-base font-bold text-gray-300 tabular mt-0.5">
                {formatTokens(project.total_tokens)}
              </div>
            </div>
            <div className="bg-[#11141a] py-3 px-4">
              <span className="text-[10px] uppercase tracking-wider text-gray-500 flex items-center justify-center gap-1">
                <Coins className="w-3 h-3 text-emerald-400" /> Net Cost
              </span>
              <div className="text-base font-bold text-emerald-400 tabular mt-0.5">
                {formatCost(project.total_cost_usd)}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Worktrees Strip (if this repository has linked worktrees) */}
      {hasWorktrees && agg && (
        <div
          className="p-5 rounded-xl bg-[#11141a] border border-white/10 space-y-3"
          data-test="worktree-strip"
          data-testid="worktree-strip"
        >
          <div className="flex items-center justify-between gap-3">
            <div className="flex items-center gap-2 text-xs font-semibold text-white">
              <GitBranch className="w-4 h-4 text-blue-400" />
              <span>
                {agg.worktree_count} {agg.worktree_count === 1 ? 'worktree' : 'worktrees'}
              </span>
              <span className="font-normal text-gray-500 text-[11px]">
                · this canonical repository + worktrees
              </span>
            </div>
            <div className="flex items-center gap-3 text-xs text-gray-400 tabular">
              <span>∑ {agg.session_count} sessions</span>
              <span>∑ {formatTokens(agg.total_tokens)} tokens</span>
              <span className="text-emerald-400 font-semibold">
                ∑ {formatCost(agg.total_cost_usd)}
              </span>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2">
            {/* Main repo direct contribution */}
            <div className="p-3 rounded-lg border border-blue-500/20 bg-blue-500/5 flex items-center justify-between gap-2">
              <div className="min-w-0">
                <div className="text-xs font-semibold text-white truncate flex items-center gap-1.5">
                  <FolderGit2 className="w-3.5 h-3.5 text-blue-400 shrink-0" />
                  <span>{projName}</span>
                  <span className="text-[10px] text-gray-500 font-normal">(direct)</span>
                </div>
                <div className="text-[10px] text-gray-500 mt-0.5 tabular">
                  {project.session_count} sessions · {formatTokens(project.total_tokens)} tok
                </div>
              </div>
              <span className="text-xs font-semibold text-emerald-400 tabular shrink-0">
                {formatCost(project.total_cost_usd)}
              </span>
            </div>

            {/* Child worktree cards */}
            {worktrees.map((w: WorktreeSummary) => (
              <a
                key={w.path}
                href={`/projects/${encodeURIComponent(w.path)}`}
                className="p-3 rounded-lg border border-white/5 bg-white/[0.02] hover:border-blue-500/30 hover:bg-white/[0.04] transition-colors flex items-center justify-between gap-2 group"
              >
                <div className="min-w-0">
                  <div className="text-xs font-semibold text-white group-hover:text-blue-400 transition-colors truncate flex items-center gap-1.5">
                    <GitBranch className="w-3.5 h-3.5 text-blue-400 shrink-0" />
                    <span className="truncate">{w.name}</span>
                  </div>
                  <div className="text-[10px] text-gray-500 mt-0.5 tabular">
                    {w.session_count} sessions · {formatTokens(w.total_tokens)} tok
                  </div>
                </div>
                <span className="text-xs font-semibold text-emerald-400 tabular shrink-0">
                  {formatCost(w.total_cost_usd)}
                </span>
              </a>
            ))}
          </div>
        </div>
      )}

      {/* Sub-Tabs Bar */}
      <div
        className="flex items-center gap-2 border-b border-white/10 pb-3"
        data-test="project-tabs"
        data-testid="project-tabs"
      >
        <button
          type="button"
          data-test="tab-activity"
          data-testid="tab-activity"
          onClick={() => handleTabChange('activity')}
          className={`flex items-center gap-2 px-4 py-2 rounded-lg text-xs font-medium transition-colors ${
            activeTab === 'activity'
              ? 'bg-blue-500/20 text-blue-400 font-semibold'
              : 'text-gray-400 hover:text-white hover:bg-white/5'
          }`}
        >
          <Activity className="w-3.5 h-3.5" />
          <span>Activity</span>
          <span className="px-1.5 py-0.2 rounded text-[10px] bg-white/10 text-gray-300">
            {sessions.length}
          </span>
        </button>

        <button
          type="button"
          data-test="tab-plans"
          data-testid="tab-plans"
          onClick={() => handleTabChange('plans')}
          className={`flex items-center gap-2 px-4 py-2 rounded-lg text-xs font-medium transition-colors ${
            activeTab === 'plans'
              ? 'bg-blue-500/20 text-blue-400 font-semibold'
              : 'text-gray-400 hover:text-white hover:bg-white/5'
          }`}
        >
          <ClipboardList className="w-3.5 h-3.5" />
          <span>Plans</span>
          {plans.length > 0 && (
            <span className="px-1.5 py-0.2 rounded text-[10px] bg-emerald-500/20 text-emerald-400">
              {plans.length}
            </span>
          )}
        </button>

        <button
          type="button"
          data-test="tab-config"
          data-testid="tab-config"
          onClick={() => handleTabChange('config')}
          className={`flex items-center gap-2 px-4 py-2 rounded-lg text-xs font-medium transition-colors ${
            activeTab === 'config'
              ? 'bg-blue-500/20 text-blue-400 font-semibold'
              : 'text-gray-400 hover:text-white hover:bg-white/5'
          }`}
        >
          <Settings2 className="w-3.5 h-3.5" />
          <span>Config</span>
        </button>
      </div>

      {/* Tab 1: Activity View */}
      {activeTab === 'activity' && (
        <div
          className="space-y-4"
          data-test="tab-content-activity"
          data-testid="tab-content-activity"
        >
          {/* Search and Agent Filter Bar */}
          <div className="flex flex-wrap items-center gap-3">
            <div className="relative flex-1 min-w-[240px]">
              <Search className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
              <input
                type="text"
                data-test="session-search-input"
                data-testid="session-search-input"
                placeholder="Search session ID, model, or instructions..."
                value={sessionSearch}
                onChange={(e) => setSessionSearch(e.target.value)}
                className="w-full bg-[#11141a] border border-white/10 rounded-lg pl-9 pr-4 py-2 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-blue-500"
              />
            </div>
            {agents.length > 0 && (
              <div className="flex items-center gap-1.5 flex-wrap">
                <button
                  type="button"
                  onClick={() => setAgentFilter('')}
                  className={`px-2.5 py-1.5 rounded-lg text-[11px] font-medium transition-colors ${
                    !agentFilter ? 'bg-blue-500/20 text-blue-400' : 'text-gray-400 hover:text-white bg-white/5'
                  }`}
                >
                  All Agents
                </button>
                {agents.map((a) => (
                  <button
                    key={a}
                    type="button"
                    onClick={() => setAgentFilter(agentFilter === a ? '' : a)}
                    className={`px-2.5 py-1.5 rounded-lg text-[11px] font-medium transition-colors ${
                      agentFilter === a
                        ? 'bg-blue-500/20 text-blue-400'
                        : 'text-gray-400 hover:text-white bg-white/5'
                    }`}
                  >
                    {getAgentMeta(a).label}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Sessions Table */}
          <div className="rounded-xl bg-[#11141a] border border-white/10 overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead>
                  <tr className="border-b border-white/10 text-gray-400 font-medium bg-white/[0.02]">
                    <th className="py-3 px-5">Agent</th>
                    <th className="py-3 px-4">Session ID</th>
                    <th className="py-3 px-4">Model</th>
                    <th className="py-3 px-4 text-right">Tokens</th>
                    <th className="py-3 px-4 text-right">Net Cost</th>
                    <th className="py-3 px-4 text-right">Duration</th>
                    <th className="py-3 px-5 text-right">Started</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-white/5">
                  {filteredSessions.map((s) => {
                    const meta = getAgentMeta(s.agent_name);
                    return (
                      <tr
                        key={s.id}
                        data-test={`session-row-${s.id}`}
                        data-testid={`session-row-${s.id}`}
                        onClick={() => (window.location.href = `/sessions/${s.id}`)}
                        className="hover:bg-white/[0.03] cursor-pointer transition-colors group"
                      >
                        <td className="py-3.5 px-5">
                          <span
                            className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium border"
                            style={{
                              color: meta.color,
                              backgroundColor: meta.bg,
                              borderColor: `${meta.color}33`,
                            }}
                          >
                            {meta.label}
                          </span>
                        </td>
                        <td className="py-3.5 px-4 font-mono text-white text-[11px] group-hover:text-blue-400 transition-colors" title={s.session_id}>
                          {s.session_id.length > 24 ? `${s.session_id.slice(0, 20)}...` : s.session_id}
                        </td>
                        <td className="py-3.5 px-4 text-gray-400 font-mono text-[11px]">
                          {s.model_resolved || s.model_raw}
                        </td>
                        <td className="py-3.5 px-4 text-right font-medium text-white tabular">
                          {formatTokens(s.input_tokens + s.output_tokens)}
                        </td>
                        <td className="py-3.5 px-4 text-right font-semibold text-emerald-400 tabular">
                          {formatCost(s.net_cost_usd)}
                        </td>
                        <td className="py-3.5 px-4 text-right text-gray-400 tabular">
                          {formatDuration(s.duration_seconds)}
                        </td>
                        <td className="py-3.5 px-5 text-right text-gray-500 text-[11px] tabular">
                          {formatDate(s.start_time)}
                        </td>
                      </tr>
                    );
                  })}
                  {filteredSessions.length === 0 && (
                    <tr>
                      <td colSpan={7} className="py-12 text-center text-gray-500 text-xs">
                        No sessions match the current search filters in this workspace.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {/* Tab 2: Plans View */}
      {activeTab === 'plans' && (
        <div
          className="space-y-4"
          data-test="tab-content-plans"
          data-testid="tab-content-plans"
        >
          {plans.length === 0 ? (
            <div className="py-16 text-center rounded-xl bg-[#11141a] border border-white/10 p-8 space-y-2">
              <div className="w-10 h-10 mx-auto rounded-full bg-white/5 flex items-center justify-center text-gray-400">
                <ClipboardList className="w-5 h-5" />
              </div>
              <div className="text-sm font-semibold text-white">No architectural plans detected</div>
              <p className="text-xs text-gray-500 max-w-sm mx-auto">
                Plans are extracted when agents produce structured planning documents, plan modes,
                or architecture markdown artifacts.
              </p>
            </div>
          ) : (
            <div className="space-y-4">
              {plans.map((plan, i) => {
                const meta = getAgentMeta(plan.agent);
                return (
                  <div
                    key={`${plan.sessionId}-${i}`}
                    className="rounded-xl bg-[#11141a] border border-white/10 overflow-hidden shadow-sm"
                  >
                    <div className="flex items-center justify-between gap-3 px-5 py-3.5 border-b border-white/10 bg-white/[0.01]">
                      <div className="flex items-center gap-2.5">
                        <span
                          className="px-2 py-0.5 rounded text-[10px] font-medium border"
                          style={{
                            color: meta.color,
                            backgroundColor: meta.bg,
                            borderColor: `${meta.color}33`,
                          }}
                        >
                          {meta.label}
                        </span>
                        <span className="text-xs font-semibold text-white">{plan.title}</span>
                      </div>
                      <span className="text-[11px] text-gray-500 tabular">
                        {formatDate(plan.timestamp)}
                      </span>
                    </div>

                    <div className="p-5 text-xs text-gray-300 font-mono whitespace-pre-wrap leading-relaxed max-h-80 overflow-y-auto bg-black/30">
                      {plan.content}
                    </div>

                    <div className="px-5 py-2.5 border-t border-white/5 bg-white/[0.01] flex justify-end">
                      <a
                        href={`/sessions/${plan.sessionId}`}
                        className="inline-flex items-center gap-1 text-[11px] font-medium text-blue-400 hover:text-blue-300 transition-colors"
                      >
                        Inspect Session <ArrowUpRight className="w-3 h-3" />
                      </a>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* Tab 3: Config View */}
      {activeTab === 'config' && (
        <div
          className="space-y-6"
          data-test="tab-content-config"
          data-testid="tab-content-config"
        >
          {/* Workspace Settings Card */}
          <div className="p-6 rounded-xl bg-[#11141a] border border-white/10 space-y-6">
            <h2 className="text-sm font-bold text-white flex items-center gap-2">
              <Settings2 className="w-4 h-4 text-blue-400" />
              <span>Workspace Preferences</span>
            </h2>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6 pt-2 border-t border-white/5">
              {/* Workspace Visibility Toggle */}
              <div className="space-y-2">
                <div className="text-xs font-semibold text-white flex items-center gap-1.5">
                  {isHidden ? <EyeOff className="w-4 h-4 text-amber-400" /> : <Eye className="w-4 h-4 text-gray-400" />}
                  <span>Workspace Catalog Visibility</span>
                </div>
                <p className="text-[11px] text-gray-500">
                  Hidden workspaces remain queryable and indexed, but are hidden from the primary
                  projects catalog.
                </p>
                <button
                  type="button"
                  data-test="toggle-hide-btn"
                  data-testid="toggle-hide-btn"
                  onClick={handleToggleHide}
                  className={`mt-2 px-3.5 py-1.5 rounded-lg text-xs font-medium transition-colors inline-flex items-center gap-1.5 ${
                    isHidden
                      ? 'bg-amber-500/20 text-amber-400 hover:bg-amber-500/30'
                      : 'bg-white/10 text-gray-300 hover:text-white hover:bg-white/15'
                  }`}
                >
                  {isHidden ? (
                    <>
                      <Eye className="w-3.5 h-3.5" /> Unhide Workspace
                    </>
                  ) : (
                    <>
                      <EyeOff className="w-3.5 h-3.5" /> Hide Workspace
                    </>
                  )}
                </button>
              </div>

              {/* Display Alias Setting */}
              <form onSubmit={handleSaveAlias} className="space-y-2">
                <div className="text-xs font-semibold text-white flex items-center gap-1.5">
                  <Tag className="w-4 h-4 text-blue-400" />
                  <span>Workspace Friendly Alias</span>
                </div>
                <p className="text-[11px] text-gray-500">
                  Assign a human-friendly name alias for this repository path across the dashboard.
                </p>
                <div className="flex items-center gap-2 mt-2">
                  <input
                    type="text"
                    data-test="alias-input"
                    data-testid="alias-input"
                    placeholder="e.g. Core Engine"
                    value={newAlias}
                    onChange={(e) => setNewAlias(e.target.value)}
                    className="flex-1 bg-black/40 border border-white/10 rounded-lg px-3 py-1.5 text-xs text-white placeholder-gray-600 focus:outline-none focus:border-blue-500"
                  />
                  <button
                    type="submit"
                    data-test="save-alias-btn"
                    data-testid="save-alias-btn"
                    className="px-3 py-1.5 rounded-lg bg-blue-500 text-white text-xs font-medium hover:bg-blue-600 transition-colors"
                  >
                    Save
                  </button>
                </div>
                {aliasSaved && (
                  <span className="text-[11px] text-emerald-400 font-medium">Alias saved!</span>
                )}
              </form>
            </div>
          </div>

          {/* Config Summary Tiles */}
          {config && (
            <div className="space-y-4">
              <h2 className="text-sm font-bold text-white flex items-center gap-2">
                <Wrench className="w-4 h-4 text-blue-400" />
                <span>Configured Agent Extensions & Skills</span>
              </h2>

              <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
                <div className="p-3.5 rounded-xl bg-[#11141a] border border-white/10 text-center">
                  <Puzzle className="w-4 h-4 mx-auto text-violet-400 mb-1" />
                  <div className="text-lg font-bold text-white tabular">
                    {config.counts.plugins}
                  </div>
                  <div className="text-[10px] uppercase tracking-wider text-gray-500">Plugins</div>
                </div>
                <div className="p-3.5 rounded-xl bg-[#11141a] border border-white/10 text-center">
                  <Users className="w-4 h-4 mx-auto text-purple-400 mb-1" />
                  <div className="text-lg font-bold text-white tabular">
                    {config.counts.subagents}
                  </div>
                  <div className="text-[10px] uppercase tracking-wider text-gray-500">
                    Subagents
                  </div>
                </div>
                <div className="p-3.5 rounded-xl bg-[#11141a] border border-white/10 text-center">
                  <BookOpen className="w-4 h-4 mx-auto text-cyan-400 mb-1" />
                  <div className="text-lg font-bold text-white tabular">
                    {config.counts.skills}
                  </div>
                  <div className="text-[10px] uppercase tracking-wider text-gray-500">Skills</div>
                </div>
                <div className="p-3.5 rounded-xl bg-[#11141a] border border-white/10 text-center">
                  <Terminal className="w-4 h-4 mx-auto text-emerald-400 mb-1" />
                  <div className="text-lg font-bold text-white tabular">
                    {config.counts.commands}
                  </div>
                  <div className="text-[10px] uppercase tracking-wider text-gray-500">Commands</div>
                </div>
                <div className="p-3.5 rounded-xl bg-[#11141a] border border-white/10 text-center">
                  <Wrench className="w-4 h-4 mx-auto text-emerald-400 mb-1" />
                  <div className="text-lg font-bold text-white tabular">{config.counts.mcps}</div>
                  <div className="text-[10px] uppercase tracking-wider text-gray-500">
                    MCP Servers
                  </div>
                </div>
                <div className="p-3.5 rounded-xl bg-[#11141a] border border-white/10 text-center">
                  <FileText className="w-4 h-4 mx-auto text-amber-400 mb-1" />
                  <div className="text-lg font-bold text-white tabular">
                    {config.counts.memory_files}
                  </div>
                  <div className="text-[10px] uppercase tracking-wider text-gray-500">Memory</div>
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
};
