import React, { useEffect, useMemo, useState } from 'react';
import {
  LineChart,
  Trophy,
  BarChart3,
  PieChart,
  TrendingUp,
  ArrowDownToLine,
  ArrowUpFromLine,
  Zap,
  Cpu,
  Search,
  ArrowUpDown,
  Calendar,
  Layers,
} from 'lucide-react';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart as RePieChart,
  Pie,
  Cell,
} from 'recharts';
import { apiFetch, subscribeEvents } from '../lib/api';
import { formatCost, formatTokens } from '../lib/format';
import { getAgentMeta } from '../lib/agents';
import type { LeaderboardEntry } from '../lib/types';

type RangeKey = '7d' | '30d' | '90d' | 'all' | 'custom';
type SortField = 'volume' | 'cost' | 'sessions';
type SortOrder = 'desc' | 'asc';

const PRESETS: { key: RangeKey; label: string }[] = [
  { key: '7d', label: '7d' },
  { key: '30d', label: '30d' },
  { key: '90d', label: '90d' },
  { key: 'all', label: 'All' },
];

function ymd(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

function presetBounds(key: RangeKey): { from: string; to: string } {
  const now = new Date();
  const to = ymd(now);
  if (key === 'all') return { from: '', to: '' };
  const days = key === '7d' ? 7 : key === '30d' ? 30 : 90;
  const f = new Date(now);
  f.setDate(f.getDate() - (days - 1));
  return { from: ymd(f), to };
}

export const Analytics: React.FC = () => {
  const [range, setRange] = useState<RangeKey>('30d');
  const [customFrom, setCustomFrom] = useState('');
  const [customTo, setCustomTo] = useState('');
  const [analyticsData, setAnalyticsData] = useState<any>(null);
  const [modelLeaderboard, setModelLeaderboard] = useState<LeaderboardEntry[]>([]);
  const [agentLeaderboard, setAgentLeaderboard] = useState<LeaderboardEntry[]>([]);
  const [loading, setLoading] = useState(true);

  // Leaderboard sorting & filtering state
  const [modelSearch, setModelSearch] = useState('');
  const [modelSortField, setModelSortField] = useState<SortField>('volume');
  const [modelSortOrder, setModelSortOrder] = useState<SortOrder>('desc');

  const [agentSearch, setAgentSearch] = useState('');
  const [agentSortField, setAgentSortField] = useState<SortField>('volume');
  const [agentSortOrder, setAgentSortOrder] = useState<SortOrder>('desc');

  const activeBounds = useMemo(() => {
    if (range === 'custom') {
      return { from: customFrom, to: customTo };
    }
    return presetBounds(range);
  }, [range, customFrom, customTo]);

  const loadData = () => {
    const params = new URLSearchParams();
    if (activeBounds.from) params.set('from', activeBounds.from);
    if (activeBounds.to) params.set('to', activeBounds.to);
    const qs = params.toString() ? `?${params.toString()}` : '';

    const lbParams = new URLSearchParams();
    lbParams.set('limit', '50');
    if (activeBounds.from) lbParams.set('from', activeBounds.from);
    if (activeBounds.to) lbParams.set('to', activeBounds.to);

    Promise.all([
      apiFetch<any>(`/api/analytics${qs}`),
      apiFetch<{ models: LeaderboardEntry[]; agents: LeaderboardEntry[] }>(`/api/leaderboard?${lbParams.toString()}`),
    ])
      .then(([analytics, leaderboards]) => {
        setAnalyticsData(analytics);
        setModelLeaderboard(leaderboards?.models || []);
        setAgentLeaderboard(leaderboards?.agents || []);
      })
      .catch((e) => console.error('Failed to load analytics data', e))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    loadData();
    const unsubscribe = subscribeEvents((event) => {
      if (event.type === 'session.created' || event.type === 'session.updated') {
        loadData();
      }
    });
    return () => unsubscribe();
  }, [activeBounds.from, activeBounds.to]);

  const byDay = analyticsData?.by_day || [];
  const byAgent = analyticsData?.by_agent || {};
  const totalStats = analyticsData?.total || { input: 0, output: 0, cached: 0, cache_reads: 0, total: 0, cost: 0 };

  const rangeTotals = useMemo(() => {
    return byDay.reduce(
      (acc: { total: number; input: number; output: number; cached: number; cost: number }, d: any) => ({
        total: acc.total + (d.total || 0),
        input: acc.input + (d.input || 0),
        output: acc.output + (d.output || 0),
        cached: acc.cached + (d.cached || 0),
        cost: acc.cost + (d.cost || 0),
      }),
      { total: 0, input: 0, output: 0, cached: 0, cost: 0 }
    );
  }, [byDay]);

  const pieData = useMemo(() => {
    return Object.entries(byAgent)
      .map(([name, data]: [string, any]) => ({
        name: getAgentMeta(name).label,
        rawName: name,
        value: data.total || 0,
        cost: data.cost || 0,
        sessions: data.session_count || 0,
        color: getAgentMeta(name).color,
      }))
      .filter((d) => d.value > 0)
      .sort((a, b) => b.value - a.value);
  }, [byAgent]);

  // Filtered & Sorted Models
  const sortedModels = useMemo(() => {
    return modelLeaderboard
      .filter((m) => m.name.toLowerCase().includes(modelSearch.toLowerCase().trim()))
      .sort((a, b) => {
        let diff = 0;
        if (modelSortField === 'volume') {
          diff = a.total_tokens - b.total_tokens;
        } else if (modelSortField === 'cost') {
          diff = a.total_cost_usd - b.total_cost_usd;
        } else if (modelSortField === 'sessions') {
          diff = a.session_count - b.session_count;
        }
        return modelSortOrder === 'desc' ? -diff : diff;
      });
  }, [modelLeaderboard, modelSearch, modelSortField, modelSortOrder]);

  // Filtered & Sorted Agents
  const sortedAgents = useMemo(() => {
    return agentLeaderboard
      .filter((a) => a.name.toLowerCase().includes(agentSearch.toLowerCase().trim()) || getAgentMeta(a.name).label.toLowerCase().includes(agentSearch.toLowerCase().trim()))
      .sort((a, b) => {
        let diff = 0;
        if (agentSortField === 'volume') {
          diff = a.total_tokens - b.total_tokens;
        } else if (agentSortField === 'cost') {
          diff = a.total_cost_usd - b.total_cost_usd;
        } else if (agentSortField === 'sessions') {
          diff = a.session_count - b.session_count;
        }
        return agentSortOrder === 'desc' ? -diff : diff;
      });
  }, [agentLeaderboard, agentSearch, agentSortField, agentSortOrder]);

  const rangeLabel = useMemo(() => {
    const labels: Record<string, string> = {
      '7d': 'Last 7 Days',
      '30d': 'Last 30 Days',
      '90d': 'Last 90 Days',
      'all': 'All Time',
    };
    if (range === 'custom') {
      return (customFrom || customTo) ? `${customFrom || '…'} → ${customTo || '…'}` : 'Custom Range';
    }
    return labels[range] || range;
  }, [range, customFrom, customTo]);

  const effectiveTotalTokens = byDay.length > 0 ? rangeTotals.total : (totalStats.total || 0);
  const effectiveInputTokens = byDay.length > 0 ? rangeTotals.input : (totalStats.input || 0);
  const effectiveOutputTokens = byDay.length > 0 ? rangeTotals.output : (totalStats.output || 0);
  const effectiveCachedTokens = byDay.length > 0 ? rangeTotals.cached : (totalStats.cached || totalStats.cache_reads || 0);
  const effectiveCostUSD = byDay.length > 0 ? rangeTotals.cost : (totalStats.cost || 0);

  const cacheEfficiencyPct = (effectiveInputTokens + effectiveCachedTokens) > 0
    ? (effectiveCachedTokens / (effectiveInputTokens + effectiveCachedTokens)) * 100
    : 0;

  return (
    <div className="space-y-8">
      {/* Top Header & Range Controls */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-white flex items-center gap-2">
            <LineChart className="w-5 h-5 text-blue-400" /> Token & Cost Analytics
          </h1>
          <p className="text-xs text-gray-400 mt-1">Multi-dimensional consumption, stacked trends, and model cost telemetry</p>
        </div>

        {/* Date Range Presets Toolbar */}
        <div className="flex flex-wrap items-center gap-2 bg-[#11141a] p-1.5 rounded-xl border border-white/10">
          <span className="text-[11px] text-gray-400 px-2 flex items-center gap-1">
            <Calendar className="w-3.5 h-3.5 text-blue-400" /> Range:
          </span>
          <div className="flex items-center gap-1">
            {PRESETS.map((p) => (
              <button
                key={p.key}
                onClick={() => setRange(p.key)}
                className={`px-3 py-1 text-xs font-semibold rounded-lg transition-all ${
                  range === p.key
                    ? 'bg-blue-600 text-white shadow-sm'
                    : 'text-gray-400 hover:text-white hover:bg-white/5'
                }`}
              >
                {p.label}
              </button>
            ))}
            <button
              onClick={() => setRange('custom')}
              className={`px-3 py-1 text-xs font-semibold rounded-lg transition-all ${
                range === 'custom'
                  ? 'bg-blue-600 text-white shadow-sm'
                  : 'text-gray-400 hover:text-white hover:bg-white/5'
              }`}
            >
              Custom
            </button>
          </div>

          {range === 'custom' && (
            <div className="flex items-center gap-1.5 pl-2 border-l border-white/10">
              <input
                type="date"
                value={customFrom}
                onChange={(e) => setCustomFrom(e.target.value)}
                className="bg-[#07090d] border border-white/10 rounded-md px-2 py-0.5 text-xs text-white focus:outline-none focus:border-blue-500"
              />
              <span className="text-gray-500 text-xs">→</span>
              <input
                type="date"
                value={customTo}
                onChange={(e) => setCustomTo(e.target.value)}
                className="bg-[#07090d] border border-white/10 rounded-md px-2 py-0.5 text-xs text-white focus:outline-none focus:border-blue-500"
              />
            </div>
          )}
        </div>
      </div>

      {/* KPI Metric Summary Strip */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="p-4 rounded-xl bg-[#11141a] border border-white/10 space-y-1">
          <div className="flex items-center justify-between text-xs text-gray-400">
            <span>Total Tokens</span>
            <TrendingUp className="w-4 h-4 text-blue-400" />
          </div>
          <div className="text-xl font-bold text-white tabular">{formatTokens(effectiveTotalTokens)}</div>
          <div className="text-[11px] text-gray-500 flex items-center justify-between">
            <span>{rangeLabel}</span>
            <span className="text-emerald-400 font-medium tabular">{formatCost(effectiveCostUSD)}</span>
          </div>
        </div>

        <div className="p-4 rounded-xl bg-[#11141a] border border-white/10 space-y-1">
          <div className="flex items-center justify-between text-xs text-gray-400">
            <span>Prompt (Input)</span>
            <ArrowDownToLine className="w-4 h-4 text-blue-400" />
          </div>
          <div className="text-xl font-bold text-white tabular">{formatTokens(effectiveInputTokens)}</div>
          <div className="text-[11px] text-gray-500">
            {effectiveTotalTokens > 0 ? ((effectiveInputTokens / effectiveTotalTokens) * 100).toFixed(1) : 0}% of total
          </div>
        </div>

        <div className="p-4 rounded-xl bg-[#11141a] border border-white/10 space-y-1">
          <div className="flex items-center justify-between text-xs text-gray-400">
            <span>Completion (Output)</span>
            <ArrowUpFromLine className="w-4 h-4 text-emerald-400" />
          </div>
          <div className="text-xl font-bold text-white tabular">{formatTokens(effectiveOutputTokens)}</div>
          <div className="text-[11px] text-gray-500">
            {effectiveTotalTokens > 0 ? ((effectiveOutputTokens / effectiveTotalTokens) * 100).toFixed(1) : 0}% of total
          </div>
        </div>

        <div className="p-4 rounded-xl bg-[#11141a] border border-white/10 space-y-1">
          <div className="flex items-center justify-between text-xs text-gray-400">
            <span>Cache Efficiency</span>
            <Zap className="w-4 h-4 text-purple-400" />
          </div>
          <div className="text-xl font-bold text-white tabular">{cacheEfficiencyPct.toFixed(1)}%</div>
          <div className="text-[11px] text-gray-500">
            {formatTokens(effectiveCachedTokens)} cached tokens
          </div>
        </div>
      </div>

      {/* Top Trends & Agent Breakdown Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Stacked Token Volume Timeline (2 Cols) */}
        <div className="lg:col-span-2 p-6 rounded-xl bg-[#11141a] border border-white/10">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 mb-4">
            <div>
              <h2 className="text-sm font-semibold text-white flex items-center gap-2">
                <Layers className="w-4 h-4 text-blue-400" /> Token Consumption Volume
              </h2>
              <p className="text-[11px] text-gray-400 mt-0.5">Stacked prompt, completion, and cache token breakdown ({rangeLabel})</p>
            </div>
            {/* Chart Legend Chips */}
            <div className="flex items-center gap-3 text-[11px]">
              <span className="flex items-center gap-1.5 text-blue-400">
                <span className="w-2.5 h-2.5 rounded-full bg-blue-500 inline-block" /> Prompt
              </span>
              <span className="flex items-center gap-1.5 text-emerald-400">
                <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 inline-block" /> Completion
              </span>
              <span className="flex items-center gap-1.5 text-purple-400">
                <span className="w-2.5 h-2.5 rounded-full bg-purple-500 inline-block" /> Cache
              </span>
            </div>
          </div>

          <div className="h-64 w-full">
            {byDay.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={byDay} margin={{ top: 10, right: 10, bottom: 0, left: -10 }}>
                  <defs>
                    <linearGradient id="promptGradient" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.45} />
                      <stop offset="95%" stopColor="#3b82f6" stopOpacity={0.05} />
                    </linearGradient>
                    <linearGradient id="completionGradient" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#10b981" stopOpacity={0.45} />
                      <stop offset="95%" stopColor="#10b981" stopOpacity={0.05} />
                    </linearGradient>
                    <linearGradient id="cacheGradient" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#a855f7" stopOpacity={0.45} />
                      <stop offset="95%" stopColor="#a855f7" stopOpacity={0.05} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid stroke="#1f2430" strokeDasharray="3 3" vertical={false} />
                  <XAxis dataKey="date" stroke="#4b5563" fontSize={11} tickLine={false} />
                  <YAxis stroke="#4b5563" fontSize={11} tickLine={false} tickFormatter={(v) => formatTokens(v)} />
                  <Tooltip
                    contentStyle={{ backgroundColor: '#181c25', borderColor: 'rgba(255,255,255,0.1)', borderRadius: '8px', fontSize: '12px' }}
                    formatter={(val: any, name: string) => [formatTokens(val), name]}
                  />
                  <Area
                    type="monotone"
                    dataKey="input"
                    name="Prompt"
                    stackId="tokens"
                    stroke="#3b82f6"
                    strokeWidth={2}
                    fillOpacity={1}
                    fill="url(#promptGradient)"
                  />
                  <Area
                    type="monotone"
                    dataKey="output"
                    name="Completion"
                    stackId="tokens"
                    stroke="#10b981"
                    strokeWidth={2}
                    fillOpacity={1}
                    fill="url(#completionGradient)"
                  />
                  <Area
                    type="monotone"
                    dataKey="cached"
                    name="Cache"
                    stackId="tokens"
                    stroke="#a855f7"
                    strokeWidth={2}
                    fillOpacity={1}
                    fill="url(#cacheGradient)"
                  />
                </AreaChart>
              </ResponsiveContainer>
            ) : (
              <div className="h-full flex items-center justify-center text-xs text-gray-500">No time-series data for selected window</div>
            )}
          </div>
        </div>

        {/* Agent Share Donut (1 Col) */}
        <div className="p-6 rounded-xl bg-[#11141a] border border-white/10 flex flex-col">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-sm font-semibold text-white flex items-center gap-2">
              <PieChart className="w-4 h-4 text-purple-400" /> Agent Token Share
            </h2>
            <span className="text-[11px] text-gray-400">{pieData.length} agents</span>
          </div>

          <div className="h-44 w-full">
            {pieData.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <RePieChart>
                  <Pie data={pieData} dataKey="value" nameKey="name" cx="50%" cy="50%" innerRadius={44} outerRadius={68} paddingAngle={4}>
                    {pieData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip
                    contentStyle={{ backgroundColor: '#181c25', borderColor: 'rgba(255,255,255,0.1)', borderRadius: '8px', fontSize: '12px' }}
                    formatter={(val: any, _name: string, item: any) => [formatTokens(val), item?.payload?.name]}
                  />
                </RePieChart>
              </ResponsiveContainer>
            ) : (
              <div className="h-full flex items-center justify-center text-xs text-gray-500">No agent data</div>
            )}
          </div>

          <div className="mt-2 space-y-1.5 flex-1 overflow-y-auto max-h-24 pr-1">
            {pieData.map((a) => (
              <div key={a.name} className="flex items-center justify-between text-xs">
                <span className="flex items-center gap-1.5 text-gray-300 truncate">
                  <span className="w-2 h-2 rounded-full shrink-0" style={{ backgroundColor: a.color }} />
                  <span className="truncate">{a.name}</span>
                </span>
                <span className="tabular text-gray-400 shrink-0">
                  {effectiveTotalTokens > 0 ? ((a.value / effectiveTotalTokens) * 100).toFixed(1) : 0}%
                </span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Leaderboards Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Model Leaderboard */}
        <div className="p-6 rounded-xl bg-[#11141a] border border-white/10 space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <h2 className="text-sm font-semibold text-white flex items-center gap-2">
              <Trophy className="w-4 h-4 text-amber-400" /> Top Models by Consumption
            </h2>
            <div className="flex items-center gap-2">
              {/* Sort field buttons */}
              <div className="flex items-center bg-[#07090d] border border-white/10 rounded-lg p-0.5 text-[11px]">
                <button
                  onClick={() => setModelSortField('volume')}
                  className={`px-2 py-0.5 rounded font-medium transition-colors ${
                    modelSortField === 'volume' ? 'bg-blue-600 text-white' : 'text-gray-400 hover:text-white'
                  }`}
                >
                  Volume
                </button>
                <button
                  onClick={() => setModelSortField('cost')}
                  className={`px-2 py-0.5 rounded font-medium transition-colors ${
                    modelSortField === 'cost' ? 'bg-blue-600 text-white' : 'text-gray-400 hover:text-white'
                  }`}
                >
                  Cost
                </button>
                <button
                  onClick={() => setModelSortField('sessions')}
                  className={`px-2 py-0.5 rounded font-medium transition-colors ${
                    modelSortField === 'sessions' ? 'bg-blue-600 text-white' : 'text-gray-400 hover:text-white'
                  }`}
                >
                  Sessions
                </button>
              </div>

              {/* Sort order toggle */}
              <button
                onClick={() => setModelSortOrder(modelSortOrder === 'desc' ? 'asc' : 'desc')}
                title={`Sort ${modelSortOrder === 'desc' ? 'Ascending' : 'Descending'}`}
                className="p-1 rounded bg-[#07090d] border border-white/10 text-gray-400 hover:text-white transition-colors"
              >
                <ArrowUpDown className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>

          {/* Model Search Input */}
          <div className="relative">
            <Search className="w-3.5 h-3.5 text-gray-500 absolute left-3 top-2.5" />
            <input
              type="text"
              placeholder="Filter models..."
              value={modelSearch}
              onChange={(e) => setModelSearch(e.target.value)}
              className="w-full bg-[#07090d] border border-white/10 rounded-lg pl-8 pr-3 py-1.5 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-blue-500"
            />
          </div>

          {/* Model List */}
          <div className="space-y-2.5 max-h-96 overflow-y-auto pr-1">
            {sortedModels.map((m, idx) => (
              <div key={m.name} className="flex items-center justify-between p-3 rounded-lg bg-white/[0.02] border border-white/5 text-xs hover:border-white/10 transition-colors">
                <div className="flex items-center gap-3 min-w-0">
                  <span className="font-bold text-gray-500 w-5 shrink-0 text-center">#{idx + 1}</span>
                  <div className="min-w-0">
                    <div className="font-mono text-white font-medium truncate flex items-center gap-1.5">
                      <Cpu className="w-3.5 h-3.5 text-emerald-400 shrink-0" />
                      <span className="truncate">{m.name}</span>
                    </div>
                    <div className="text-[11px] text-gray-500 mt-0.5">{m.session_count} session{m.session_count === 1 ? '' : 's'}</div>
                  </div>
                </div>
                <div className="text-right shrink-0">
                  <div className="text-white font-medium tabular">{formatTokens(m.total_tokens)}</div>
                  <div className="text-[11px] text-emerald-400 tabular font-medium">{formatCost(m.total_cost_usd)}</div>
                </div>
              </div>
            ))}
            {sortedModels.length === 0 && (
              <div className="text-xs text-gray-500 py-6 text-center">
                {modelSearch ? `No models matching "${modelSearch}"` : 'No models recorded'}
              </div>
            )}
          </div>
        </div>

        {/* Agent Leaderboard */}
        <div className="p-6 rounded-xl bg-[#11141a] border border-white/10 space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <h2 className="text-sm font-semibold text-white flex items-center gap-2">
              <BarChart3 className="w-4 h-4 text-blue-400" /> Agent Activity Leaderboard
            </h2>
            <div className="flex items-center gap-2">
              {/* Sort field buttons */}
              <div className="flex items-center bg-[#07090d] border border-white/10 rounded-lg p-0.5 text-[11px]">
                <button
                  onClick={() => setAgentSortField('volume')}
                  className={`px-2 py-0.5 rounded font-medium transition-colors ${
                    agentSortField === 'volume' ? 'bg-blue-600 text-white' : 'text-gray-400 hover:text-white'
                  }`}
                >
                  Volume
                </button>
                <button
                  onClick={() => setAgentSortField('cost')}
                  className={`px-2 py-0.5 rounded font-medium transition-colors ${
                    agentSortField === 'cost' ? 'bg-blue-600 text-white' : 'text-gray-400 hover:text-white'
                  }`}
                >
                  Cost
                </button>
                <button
                  onClick={() => setAgentSortField('sessions')}
                  className={`px-2 py-0.5 rounded font-medium transition-colors ${
                    agentSortField === 'sessions' ? 'bg-blue-600 text-white' : 'text-gray-400 hover:text-white'
                  }`}
                >
                  Sessions
                </button>
              </div>

              {/* Sort order toggle */}
              <button
                onClick={() => setAgentSortOrder(agentSortOrder === 'desc' ? 'asc' : 'desc')}
                title={`Sort ${agentSortOrder === 'desc' ? 'Ascending' : 'Descending'}`}
                className="p-1 rounded bg-[#07090d] border border-white/10 text-gray-400 hover:text-white transition-colors"
              >
                <ArrowUpDown className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>

          {/* Agent Search Input */}
          <div className="relative">
            <Search className="w-3.5 h-3.5 text-gray-500 absolute left-3 top-2.5" />
            <input
              type="text"
              placeholder="Filter agents..."
              value={agentSearch}
              onChange={(e) => setAgentSearch(e.target.value)}
              className="w-full bg-[#07090d] border border-white/10 rounded-lg pl-8 pr-3 py-1.5 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-blue-500"
            />
          </div>

          {/* Agent List */}
          <div className="space-y-2.5 max-h-96 overflow-y-auto pr-1">
            {sortedAgents.map((a, idx) => {
              const meta = getAgentMeta(a.name);
              return (
                <div key={a.name} className="flex items-center justify-between p-3 rounded-lg bg-white/[0.02] border border-white/5 text-xs hover:border-white/10 transition-colors">
                  <div className="flex items-center gap-3 min-w-0">
                    <span className="font-bold text-gray-500 w-5 shrink-0 text-center">#{idx + 1}</span>
                    <div className="min-w-0">
                      <div className="flex items-center gap-2">
                        <span
                          className="px-2 py-0.5 rounded font-medium text-[11px]"
                          style={{ color: meta.color, backgroundColor: meta.bg }}
                        >
                          {meta.label}
                        </span>
                      </div>
                      <div className="text-[11px] text-gray-500 mt-1">{a.session_count} session{a.session_count === 1 ? '' : 's'}</div>
                    </div>
                  </div>
                  <div className="text-right shrink-0">
                    <div className="text-white font-medium tabular">{formatTokens(a.total_tokens)}</div>
                    <div className="text-[11px] text-emerald-400 tabular font-medium">{formatCost(a.total_cost_usd)}</div>
                  </div>
                </div>
              );
            })}
            {sortedAgents.length === 0 && (
              <div className="text-xs text-gray-500 py-6 text-center">
                {agentSearch ? `No agents matching "${agentSearch}"` : 'No agents recorded'}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
