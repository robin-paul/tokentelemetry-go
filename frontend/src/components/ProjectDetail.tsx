import React, { useEffect, useState } from 'react';
import { ArrowLeft, FolderGit2, Coins, Cpu, Layers, Clock } from 'lucide-react';
import { apiFetch } from '../lib/api';
import { formatCost, formatDuration, formatTokens, formatDate } from '../lib/format';
import { getAgentMeta } from '../lib/agents';
import type { ProjectSummary, Session } from '../lib/types';

interface ProjectDetailProps {
  projectPath?: string;
}

export const ProjectDetail: React.FC<ProjectDetailProps> = ({ projectPath: propPath }) => {
  const [project, setProject] = useState<ProjectSummary | null>(null);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let p = propPath;
    if ((!p || p.startsWith('[')) && typeof window !== 'undefined') {
      const parts = window.location.pathname.split('/').filter(Boolean);
      if (parts[0] === 'projects' && parts[1]) {
        p = decodeURIComponent(parts.slice(1).join('/'));
      }
    }
    if (p && !p.startsWith('[')) {
      apiFetch<{ project: ProjectSummary; sessions: Session[] }>(`/api/projects/${encodeURIComponent(p)}`)
        .then((res) => {
          setProject(res.project);
          setSessions(res.sessions || []);
        })
        .catch((e) => console.error('Failed to load project detail', e))
        .finally(() => setLoading(false));
    }
  }, [propPath]);

  if (loading) {
    return <div className="p-8 text-center text-xs text-gray-500">Loading project data...</div>;
  }

  if (!project) {
    return (
      <div className="p-8 text-center space-y-3">
        <div className="text-sm text-gray-400">Project not found.</div>
        <a href="/projects" className="text-xs text-blue-400 hover:underline">
          Return to projects catalog
        </a>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <a
          href="/projects"
          className="inline-flex items-center gap-1.5 text-xs text-gray-400 hover:text-white mb-3 transition-colors"
        >
          <ArrowLeft className="w-3.5 h-3.5" /> Back to Projects
        </a>
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-blue-500/10 border border-blue-500/20 flex items-center justify-center text-blue-400">
            <FolderGit2 className="w-5 h-5" />
          </div>
          <div>
            <h1 className="text-xl font-bold text-white">{project.project_name}</h1>
            <p className="text-xs text-gray-400">Workspace repository telemetry</p>
          </div>
        </div>
      </div>

      {/* Summary KPI Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div className="p-4 rounded-xl bg-[#11141a] border border-white/10">
          <span className="text-xs text-gray-400 flex items-center gap-1.5"><Cpu className="w-3.5 h-3.5 text-blue-400" /> Total Tokens</span>
          <div className="text-lg font-bold text-white tabular mt-1">{formatTokens(project.total_tokens)}</div>
        </div>
        <div className="p-4 rounded-xl bg-[#11141a] border border-white/10">
          <span className="text-xs text-gray-400 flex items-center gap-1.5"><Coins className="w-3.5 h-3.5 text-emerald-400" /> Net Cost</span>
          <div className="text-lg font-bold text-emerald-400 tabular mt-1">{formatCost(project.total_cost_usd)}</div>
        </div>
        <div className="p-4 rounded-xl bg-[#11141a] border border-white/10">
          <span className="text-xs text-gray-400 flex items-center gap-1.5"><Layers className="w-3.5 h-3.5 text-purple-400" /> Sessions</span>
          <div className="text-lg font-bold text-white tabular mt-1">{project.session_count}</div>
        </div>
        <div className="p-4 rounded-xl bg-[#11141a] border border-white/10">
          <span className="text-xs text-gray-400 flex items-center gap-1.5"><Clock className="w-3.5 h-3.5 text-amber-400" /> Last Active</span>
          <div className="text-xs text-gray-300 font-medium mt-2">{formatDate(project.last_active)}</div>
        </div>
      </div>

      {/* Project Session Activity List */}
      <div className="p-5 rounded-xl bg-[#11141a] border border-white/10">
        <h2 className="text-sm font-semibold text-white mb-4">Project Session Activity</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="border-b border-white/10 text-gray-400 font-medium">
                <th className="pb-3 pr-4">Agent</th>
                <th className="pb-3 px-4">Session ID</th>
                <th className="pb-3 px-4">Model</th>
                <th className="pb-3 px-4 text-right">Tokens</th>
                <th className="pb-3 px-4 text-right">Net Cost</th>
                <th className="pb-3 px-4 text-right">Duration</th>
                <th className="pb-3 pl-4 text-right">Time</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5">
              {sessions.map((s) => {
                const meta = getAgentMeta(s.agent_name);
                return (
                  <tr
                    key={s.id}
                    onClick={() => (window.location.href = `/sessions/${s.id}`)}
                    className="hover:bg-white/[0.03] cursor-pointer transition-colors"
                  >
                    <td className="py-3 pr-4">
                      <span
                        className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium"
                        style={{ color: meta.color, backgroundColor: meta.bg }}
                      >
                        {meta.label}
                      </span>
                    </td>
                    <td className="py-3 px-4 font-mono text-white text-[11px]">
                      {s.session_id.slice(0, 16)}...
                    </td>
                    <td className="py-3 px-4 text-gray-400 font-mono text-[11px]">
                      {s.model_resolved || s.model_raw}
                    </td>
                    <td className="py-3 px-4 text-right font-medium text-white tabular">
                      {formatTokens(s.input_tokens + s.output_tokens)}
                    </td>
                    <td className="py-3 px-4 text-right font-medium text-emerald-400 tabular">
                      {formatCost(s.net_cost_usd)}
                    </td>
                    <td className="py-3 px-4 text-right text-gray-400 tabular">
                      {formatDuration(s.duration_seconds)}
                    </td>
                    <td className="py-3 pl-4 text-right text-gray-500 text-[11px] tabular">
                      {formatDate(s.start_time)}
                    </td>
                  </tr>
                );
              })}
              {sessions.length === 0 && (
                <tr>
                  <td colSpan={7} className="py-8 text-center text-gray-500">
                    No sessions found for this project.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};
