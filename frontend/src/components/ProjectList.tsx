import React, { useEffect, useState } from 'react';
import { FolderGit2, ArrowRight, Search } from 'lucide-react';
import { apiFetch } from '../lib/api';
import { formatCost, formatTokens, formatDate } from '../lib/format';
import type { ProjectSummary } from '../lib/types';

export const ProjectList: React.FC = () => {
  const [projects, setProjects] = useState<ProjectSummary[]>([]);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    apiFetch<ProjectSummary[]>('/api/projects')
      .then((data) => setProjects(data || []))
      .catch((e) => console.error('Failed to load projects', e))
      .finally(() => setLoading(false));
  }, []);

  const filtered = projects.filter((p) =>
    p.project_name.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-white">Project Workspaces</h1>
          <p className="text-xs text-gray-400 mt-1">Discovered git repositories and worktrees monitored for agent activity</p>
        </div>
        <div className="relative w-64">
          <Search className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            placeholder="Filter projects..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full bg-[#11141a] border border-white/10 rounded-lg pl-9 pr-4 py-2 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-blue-500"
          />
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {filtered.map((p) => (
          <a
            key={p.project_name}
            href={`/projects/${encodeURIComponent(p.project_name)}`}
            className="p-5 rounded-xl bg-[#11141a] border border-white/10 hover:border-blue-500/40 hover:bg-white/[0.02] transition-all group block"
          >
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2.5">
                <div className="w-8 h-8 rounded-lg bg-blue-500/10 border border-blue-500/20 flex items-center justify-center text-blue-400 group-hover:scale-105 transition-transform">
                  <FolderGit2 className="w-4 h-4" />
                </div>
                <div className="font-semibold text-sm text-white truncate max-w-[160px]">{p.project_name}</div>
              </div>
              <ArrowRight className="w-4 h-4 text-gray-500 group-hover:text-blue-400 group-hover:translate-x-0.5 transition-all" />
            </div>

            <div className="grid grid-cols-2 gap-2 pt-3 border-t border-white/5 text-xs">
              <div>
                <span className="text-gray-400">Total Tokens</span>
                <div className="font-semibold text-white tabular mt-0.5">{formatTokens(p.total_tokens)}</div>
              </div>
              <div>
                <span className="text-gray-400">Net Cost</span>
                <div className="font-semibold text-emerald-400 tabular mt-0.5">{formatCost(p.total_cost_usd)}</div>
              </div>
              <div className="col-span-2 text-[11px] text-gray-500 mt-2 flex items-center justify-between">
                <span>{p.session_count} indexed sessions</span>
                <span>Active {formatDate(p.last_active)}</span>
              </div>
            </div>
          </a>
        ))}
        {filtered.length === 0 && !loading && (
          <div className="col-span-full py-12 text-center text-gray-500 text-xs">
            No projects found matching the filter.
          </div>
        )}
      </div>
    </div>
  );
};
