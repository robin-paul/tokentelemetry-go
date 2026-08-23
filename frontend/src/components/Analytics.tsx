import React, { useEffect, useState } from 'react';
import { LineChart, Trophy, BarChart3, PieChart } from 'lucide-react';
import {
  AreaChart,
  Area,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  PieChart as RePieChart,
  Pie,
  Cell,
} from 'recharts';
import { apiFetch } from '../lib/api';
import { formatCost, formatTokens } from '../lib/format';
import { getAgentMeta } from '../lib/agents';
import type { LeaderboardEntry } from '../lib/types';

export const Analytics: React.FC = () => {
  const [analyticsData, setAnalyticsData] = useState<any>(null);
  const [modelLeaderboard, setModelLeaderboard] = useState<LeaderboardEntry[]>([]);
  const [agentLeaderboard, setAgentLeaderboard] = useState<LeaderboardEntry[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      apiFetch<any>('/api/analytics'),
      apiFetch<{ models: LeaderboardEntry[]; agents: LeaderboardEntry[] }>('/api/leaderboard?limit=10'),
    ])
      .then(([analytics, leaderboards]) => {
        setAnalyticsData(analytics);
        setModelLeaderboard(leaderboards?.models || []);
        setAgentLeaderboard(leaderboards?.agents || []);
      })
      .catch((e) => console.error('Failed to load analytics data', e))
      .finally(() => setLoading(false));
  }, []);

  const byDay = analyticsData?.by_day || [];
  const byAgent = analyticsData?.by_agent || {};

  const pieData = Object.entries(byAgent).map(([name, data]: [string, any]) => ({
    name: getAgentMeta(name).label,
    value: data.total || 0,
    color: getAgentMeta(name).color,
  }));

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-xl font-bold text-white flex items-center gap-2">
          <LineChart className="w-5 h-5 text-blue-400" /> Token & Cost Analytics
        </h1>
        <p className="text-xs text-gray-400 mt-1">Multi-dimensional consumption, trends, and model cost telemetry</p>
      </div>

      {/* Top Trends & Agent Breakdown Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Token Volume Timeline (2 Cols) */}
        <div className="lg:col-span-2 p-6 rounded-xl bg-[#11141a] border border-white/10">
          <h2 className="text-sm font-semibold text-white mb-4">Token Consumption Volume</h2>
          <div className="h-64 w-full">
            {byDay.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={byDay}>
                  <defs>
                    <linearGradient id="analyticsGradient" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#8b5cf6" stopOpacity={0.4} />
                      <stop offset="95%" stopColor="#8b5cf6" stopOpacity={0.0} />
                    </linearGradient>
                  </defs>
                  <XAxis dataKey="date" stroke="#4b5563" fontSize={11} tickLine={false} />
                  <YAxis stroke="#4b5563" fontSize={11} tickLine={false} tickFormatter={(v) => formatTokens(v)} />
                  <Tooltip
                    contentStyle={{ backgroundColor: '#181c25', borderColor: 'rgba(255,255,255,0.1)', borderRadius: '8px', fontSize: '12px' }}
                    formatter={(val: any) => [formatTokens(val), 'Tokens']}
                  />
                  <Area type="monotone" dataKey="total" stroke="#8b5cf6" strokeWidth={2} fillOpacity={1} fill="url(#analyticsGradient)" />
                </AreaChart>
              </ResponsiveContainer>
            ) : (
              <div className="h-full flex items-center justify-center text-xs text-gray-500">No time-series data</div>
            )}
          </div>
        </div>

        {/* Agent Share Donut (1 Col) */}
        <div className="p-6 rounded-xl bg-[#11141a] border border-white/10">
          <h2 className="text-sm font-semibold text-white mb-4 flex items-center gap-2">
            <PieChart className="w-4 h-4 text-purple-400" /> Agent Token Share
          </h2>
          <div className="h-64 w-full">
            {pieData.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <RePieChart>
                  <Pie data={pieData} dataKey="value" nameKey="name" cx="50%" cy="50%" innerRadius={50} outerRadius={80} paddingAngle={4}>
                    {pieData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip
                    contentStyle={{ backgroundColor: '#181c25', borderColor: 'rgba(255,255,255,0.1)', borderRadius: '8px', fontSize: '12px' }}
                    formatter={(val: any) => [formatTokens(val), 'Tokens']}
                  />
                </RePieChart>
              </ResponsiveContainer>
            ) : (
              <div className="h-full flex items-center justify-center text-xs text-gray-500">No agent data</div>
            )}
          </div>
        </div>
      </div>

      {/* Leaderboards Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Model Leaderboard */}
        <div className="p-6 rounded-xl bg-[#11141a] border border-white/10">
          <h2 className="text-sm font-semibold text-white mb-4 flex items-center gap-2">
            <Trophy className="w-4 h-4 text-amber-400" /> Top Models by Consumption
          </h2>
          <div className="space-y-3">
            {modelLeaderboard.map((m, idx) => (
              <div key={m.name} className="flex items-center justify-between p-2.5 rounded-lg bg-white/[0.02] border border-white/5 text-xs">
                <div className="flex items-center gap-2.5">
                  <span className="font-bold text-gray-500 w-4">#{idx + 1}</span>
                  <span className="font-mono text-white font-medium">{m.name}</span>
                </div>
                <div className="text-right">
                  <div className="text-white font-medium tabular">{formatTokens(m.total_tokens)}</div>
                  <div className="text-[11px] text-emerald-400 tabular">{formatCost(m.total_cost_usd)}</div>
                </div>
              </div>
            ))}
            {modelLeaderboard.length === 0 && <div className="text-xs text-gray-500">No models recorded</div>}
          </div>
        </div>

        {/* Agent Leaderboard */}
        <div className="p-6 rounded-xl bg-[#11141a] border border-white/10">
          <h2 className="text-sm font-semibold text-white mb-4 flex items-center gap-2">
            <BarChart3 className="w-4 h-4 text-blue-400" /> Agent Activity Leaderboard
          </h2>
          <div className="space-y-3">
            {agentLeaderboard.map((a, idx) => {
              const meta = getAgentMeta(a.name);
              return (
                <div key={a.name} className="flex items-center justify-between p-2.5 rounded-lg bg-white/[0.02] border border-white/5 text-xs">
                  <div className="flex items-center gap-2.5">
                    <span className="font-bold text-gray-500 w-4">#{idx + 1}</span>
                    <span className="px-2 py-0.5 rounded font-medium text-[11px]" style={{ color: meta.color, backgroundColor: meta.bg }}>
                      {meta.label}
                    </span>
                  </div>
                  <div className="text-right">
                    <div className="text-white font-medium tabular">{formatTokens(a.total_tokens)}</div>
                    <div className="text-[11px] text-gray-400">{a.session_count} sessions</div>
                  </div>
                </div>
              );
            })}
            {agentLeaderboard.length === 0 && <div className="text-xs text-gray-500">No agents recorded</div>}
          </div>
        </div>
      </div>
    </div>
  );
};
