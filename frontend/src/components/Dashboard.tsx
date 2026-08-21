import React, { useEffect, useState } from 'react';
import {
  Coins,
  Cpu,
  Layers,
  Sparkles,
  Zap,
  ArrowRight,
  TrendingUp,
  FolderGit2,
} from 'lucide-react';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import { apiFetch, subscribeEvents } from '../lib/api';
import { formatCost, formatDuration, formatTokens, formatDate } from '../lib/format';
import { getAgentMeta } from '../lib/agents';
import type { DailySummary, Session, StatsOverview } from '../lib/types';

export const Dashboard: React.FC = () => {
  const [stats, setStats] = useState<StatsOverview | null>(null);
  const [recentSessions, setRecentSessions] = useState<Session[]>([]);
  const [dailyData, setDailyData] = useState<DailySummary[]>([]);
  const [loading, setLoading] = useState(true);

  const loadData = async () => {
    try {
      const [statsRes, recentRes, dailyRes] = await Promise.all([
        apiFetch<StatsOverview>('/api/stats'),
        apiFetch<Session[]>('/api/recent?limit=15'),
        apiFetch<DailySummary[]>('/api/stats/daily'),
      ]);
      setStats(statsRes);
      setRecentSessions(recentRes || []);
      setDailyData(dailyRes || []);
    } catch (e) {
      console.error('Failed to load dashboard data', e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 15000);

    const unsubscribe = subscribeEvents((event) => {
      if (event.type === 'session.created' || event.type === 'session.updated') {
        loadData();
      }
    });

    return () => {
      clearInterval(interval);
      unsubscribe();
    };
  }, []);

  const chartData = dailyData.slice(-14).map((d) => ({
    date: d.date.slice(5),
    tokens: d.total_input_tokens + d.total_output_tokens,
    cost: d.total_cost_usd,
  }));

  return (
    <div className="space-y-8">
      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Total Tokens */}
        <div className="p-5 rounded-xl bg-[#11141a] border border-white/10 relative overflow-hidden">
          <div className="flex items-center justify-between text-gray-400 mb-3">
            <span className="text-xs font-semibold uppercase tracking-wider">Total Tokens</span>
            <div className="p-2 rounded-lg bg-blue-500/10 text-blue-400">
              <Cpu className="w-4 h-4" />
            </div>
          </div>
          <div className="text-2xl font-bold tabular text-white">
            {formatTokens(stats?.total_tokens)}
          </div>
          <div className="text-xs text-gray-400 mt-2 flex items-center gap-1.5">
            <span className="text-blue-400 font-medium">{formatTokens(stats?.total_input_tokens)} in</span>
            <span>•</span>
            <span className="text-purple-400 font-medium">{formatTokens(stats?.total_output_tokens)} out</span>
          </div>
        </div>

        {/* Net Billable Spend */}
        <div className="p-5 rounded-xl bg-[#11141a] border border-white/10 relative overflow-hidden">
          <div className="flex items-center justify-between text-gray-400 mb-3">
            <span className="text-xs font-semibold uppercase tracking-wider">Net Billable Cost</span>
            <div className="p-2 rounded-lg bg-emerald-500/10 text-emerald-400">
              <Coins className="w-4 h-4" />
            </div>
          </div>
          <div className="text-2xl font-bold tabular text-emerald-400">
            {formatCost(stats?.net_cost_usd)}
          </div>
          <div className="text-xs text-gray-400 mt-2">
            Gross un-cached: <span className="line-through">{formatCost(stats?.gross_cost_usd)}</span>
          </div>
        </div>

        {/* Active Sessions */}
        <div className="p-5 rounded-xl bg-[#11141a] border border-white/10 relative overflow-hidden">
          <div className="flex items-center justify-between text-gray-400 mb-3">
            <span className="text-xs font-semibold uppercase tracking-wider">Indexed Sessions</span>
            <div className="p-2 rounded-lg bg-purple-500/10 text-purple-400">
              <Layers className="w-4 h-4" />
            </div>
          </div>
          <div className="text-2xl font-bold tabular text-white">
            {stats?.total_sessions?.toLocaleString() || '0'}
          </div>
          <div className="text-xs text-gray-400 mt-2">
            Across <span className="text-white font-medium">{stats?.active_projects || 0}</span> workspace repositories
          </div>
        </div>

        {/* Connected Agents */}
        <div className="p-5 rounded-xl bg-[#11141a] border border-white/10 relative overflow-hidden">
          <div className="flex items-center justify-between text-gray-400 mb-3">
            <span className="text-xs font-semibold uppercase tracking-wider">Active Ecosystem</span>
            <div className="p-2 rounded-lg bg-amber-500/10 text-amber-400">
              <Sparkles className="w-4 h-4" />
            </div>
          </div>
          <div className="text-2xl font-bold tabular text-white">
            {stats?.active_agents || 0} Agents
          </div>
          <div className="text-xs text-emerald-400 mt-2 flex items-center gap-1">
            <Zap className="w-3.5 h-3.5" />
            <span>Zero-latency local SQLite engine</span>
          </div>
        </div>
      </div>

      {/* 14-Day Trends Chart */}
      <div className="p-6 rounded-xl bg-[#11141a] border border-white/10">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h2 className="text-base font-semibold text-white flex items-center gap-2">
              <TrendingUp className="w-4 h-4 text-blue-400" />
              14-Day Token Consumption Trends
            </h2>
            <p className="text-xs text-gray-400 mt-0.5">Aggregated daily prompt, completion, and prompt cache tokens</p>
          </div>
        </div>

        <div className="h-64 w-full">
          {chartData.length > 0 ? (
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="tokenGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.4} />
                    <stop offset="95%" stopColor="#3b82f6" stopOpacity={0.0} />
                  </linearGradient>
                </defs>
                <XAxis dataKey="date" stroke="#4b5563" fontSize={11} tickLine={false} />
                <YAxis
                  stroke="#4b5563"
                  fontSize={11}
                  tickLine={false}
                  tickFormatter={(v) => formatTokens(v)}
                />
                <Tooltip
                  contentStyle={{ backgroundColor: '#181c25', borderColor: 'rgba(255,255,255,0.1)', borderRadius: '8px', fontSize: '12px' }}
                  formatter={(val: any) => [formatTokens(val), 'Tokens']}
                />
                <Area
                  type="monotone"
                  dataKey="tokens"
                  stroke="#3b82f6"
                  strokeWidth={2}
                  fillOpacity={1}
                  fill="url(#tokenGradient)"
                />
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <div className="h-full flex items-center justify-center text-xs text-gray-500">
              No historical trend data recorded yet.
            </div>
          )}
        </div>
      </div>

      {/* Live Recent Sessions Feed */}
      <div className="p-6 rounded-xl bg-[#11141a] border border-white/10">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-base font-semibold text-white">Live Activity Feed</h2>
            <p className="text-xs text-gray-400 mt-0.5">Real-time stream of coding agent transcripts and turns</p>
          </div>
          <a
            href="/sessions"
            className="text-xs font-medium text-blue-400 hover:text-blue-300 flex items-center gap-1 transition-colors"
          >
            View all sessions <ArrowRight className="w-3.5 h-3.5" />
          </a>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="border-b border-white/10 text-gray-400 font-medium">
                <th className="pb-3 pr-4">Agent</th>
                <th className="pb-3 px-4">Project</th>
                <th className="pb-3 px-4">Model</th>
                <th className="pb-3 px-4 text-right">Tokens</th>
                <th className="pb-3 px-4 text-right">Net Cost</th>
                <th className="pb-3 px-4 text-right">Duration</th>
                <th className="pb-3 pl-4 text-right">Time</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5">
              {recentSessions.map((s) => {
                const meta = getAgentMeta(s.agent_name);
                const totalTok = s.input_tokens + s.output_tokens + s.cache_read_tokens;

                return (
                  <tr
                    key={s.id}
                    onClick={() => (window.location.href = `/sessions/${s.id}`)}
                    className="hover:bg-white/[0.03] cursor-pointer transition-colors group"
                  >
                    <td className="py-3 pr-4">
                      <span
                        className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium"
                        style={{ color: meta.color, backgroundColor: meta.bg }}
                      >
                        {meta.label}
                      </span>
                    </td>
                    <td className="py-3 px-4 text-white font-medium max-w-[200px] truncate">
                      <div className="flex items-center gap-1.5">
                        <FolderGit2 className="w-3.5 h-3.5 text-gray-500" />
                        <span className="truncate">{s.project_name || 'root'}</span>
                      </div>
                    </td>
                    <td className="py-3 px-4 text-gray-400 font-mono text-[11px]">
                      {s.model_resolved || s.model_raw || 'unknown'}
                    </td>
                    <td className="py-3 px-4 text-right font-medium text-white tabular">
                      {formatTokens(totalTok)}
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
              {recentSessions.length === 0 && !loading && (
                <tr>
                  <td colSpan={7} className="py-8 text-center text-gray-500">
                    No agent sessions recorded. Run an agent or scan directories to populate data.
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
