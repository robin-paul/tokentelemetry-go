import React, { useEffect, useState } from 'react';
import { Search, FolderGit2, ChevronLeft, ChevronRight } from 'lucide-react';
import { apiFetch, subscribeEvents } from '../lib/api';
import { formatCost, formatDuration, formatTokens, formatDate } from '../lib/format';
import { getAgentMeta } from '../lib/agents';
import type { Session } from '../lib/types';

export const SessionList: React.FC = () => {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [search, setSearch] = useState('');
  const [selectedAgent, setSelectedAgent] = useState('');
  const [page, setPage] = useState(1);
  const [agents, setAgents] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchSessions = async () => {
    setLoading(true);
    try {
      const q = new URLSearchParams({
        page: page.toString(),
        limit: '30',
        search,
        agent: selectedAgent,
      });
      const data = await apiFetch<Session[]>(`/api/sessions?${q.toString()}`);
      setSessions(data || []);
    } catch (e) {
      console.error('Failed to fetch sessions', e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    apiFetch<string[]>('/agents')
      .then((res) => setAgents(res || []))
      .catch(() => {});
  }, []);

  useEffect(() => {
    fetchSessions();

    const unsubscribe = subscribeEvents((event) => {
      if (event.type === 'session.created' || event.type === 'session.updated') {
        fetchSessions();
        apiFetch<string[]>('/agents')
          .then((res) => setAgents(res || []))
          .catch(() => {});
      }
    });

    return () => {
      unsubscribe();
    };
  }, [page, selectedAgent]);

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setPage(1);
    fetchSessions();
  };

  return (
    <div className="space-y-6">
      {/* Search & Agent Filter Bar */}
      <div className="flex flex-col md:flex-row gap-4 items-start md:items-center justify-between">
        <form onSubmit={handleSearchSubmit} className="relative w-full md:w-80">
          <Search className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            placeholder="Search sessions or projects..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full bg-[#11141a] border border-white/10 rounded-lg pl-9 pr-4 py-2 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-blue-500 transition-colors"
          />
        </form>

        {/* Agent Filter Pills */}
        <div className="flex flex-wrap gap-1.5 items-center">
          <button
            onClick={() => { setSelectedAgent(''); setPage(1); }}
            className={`px-2.5 py-1 rounded-md text-xs font-medium transition-colors ${
              selectedAgent === ''
                ? 'bg-blue-600/30 text-blue-400 border border-blue-500/40'
                : 'bg-white/5 text-gray-400 hover:text-white border border-transparent'
            }`}
          >
            All Agents
          </button>
          {agents.map((ag) => {
            const meta = getAgentMeta(ag);
            const isSelected = selectedAgent === ag;
            return (
              <button
                key={ag}
                onClick={() => { setSelectedAgent(isSelected ? '' : ag); setPage(1); }}
                className={`px-2.5 py-1 rounded-md text-xs font-medium transition-colors ${
                  isSelected
                    ? 'border'
                    : 'bg-white/5 text-gray-400 hover:text-white border border-transparent'
                }`}
                style={isSelected ? { color: meta.color, backgroundColor: meta.bg, borderColor: meta.color } : {}}
              >
                {meta.label}
              </button>
            );
          })}
        </div>
      </div>

      {/* Sessions Table */}
      <div className="rounded-xl bg-[#11141a] border border-white/10 overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="border-b border-white/10 text-gray-400 font-medium bg-white/[0.02]">
                <th className="py-3 px-4">Agent</th>
                <th className="py-3 px-4">Session ID / Project</th>
                <th className="py-3 px-4">Model</th>
                <th className="py-3 px-4 text-right">Prompt / Compl</th>
                <th className="py-3 px-4 text-right">Cache Reads</th>
                <th className="py-3 px-4 text-right">Net Cost</th>
                <th className="py-3 px-4 text-right">Duration</th>
                <th className="py-3 px-4 text-right">Timestamp</th>
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
                    <td className="py-3 px-4">
                      <span
                        className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium"
                        style={{ color: meta.color, backgroundColor: meta.bg }}
                      >
                        {meta.label}
                      </span>
                    </td>
                    <td className="py-3 px-4">
                      <div className="font-mono text-white font-medium text-[11px]">
                        {s.session_id.slice(0, 16)}...
                      </div>
                      <div className="text-[11px] text-gray-400 flex items-center gap-1 mt-0.5">
                        <FolderGit2 className="w-3 h-3" />
                        <span>{s.project_name || 'root'}</span>
                      </div>
                    </td>
                    <td className="py-3 px-4 text-gray-300 font-mono text-[11px]">
                      {s.model_resolved || s.model_raw || 'unknown'}
                    </td>
                    <td className="py-3 px-4 text-right font-medium text-white tabular">
                      {formatTokens(s.input_tokens)} / {formatTokens(s.output_tokens)}
                    </td>
                    <td className="py-3 px-4 text-right text-purple-400 tabular">
                      {formatTokens(s.cache_read_tokens)}
                    </td>
                    <td className="py-3 px-4 text-right font-medium text-emerald-400 tabular">
                      {formatCost(s.net_cost_usd)}
                    </td>
                    <td className="py-3 px-4 text-right text-gray-400 tabular">
                      {formatDuration(s.duration_seconds)}
                    </td>
                    <td className="py-3 px-4 text-right text-gray-500 text-[11px] tabular">
                      {formatDate(s.start_time)}
                    </td>
                  </tr>
                );
              })}
              {sessions.length === 0 && !loading && (
                <tr>
                  <td colSpan={8} className="py-12 text-center text-gray-500">
                    No sessions match the selected query.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination Footer */}
        <div className="p-3 border-t border-white/10 flex items-center justify-between text-xs text-gray-400 bg-white/[0.01]">
          <div>Page {page}</div>
          <div className="flex items-center gap-2">
            <button
              disabled={page <= 1}
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              className="p-1.5 rounded bg-white/5 hover:bg-white/10 disabled:opacity-30 disabled:pointer-events-none transition-colors"
            >
              <ChevronLeft className="w-4 h-4" />
            </button>
            <button
              disabled={sessions.length < 30}
              onClick={() => setPage((p) => p + 1)}
              className="p-1.5 rounded bg-white/5 hover:bg-white/10 disabled:opacity-30 disabled:pointer-events-none transition-colors"
            >
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
