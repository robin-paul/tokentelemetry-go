import React, { useEffect, useMemo, useState } from 'react';
import {
  Search,
  FolderGit2,
  ChevronLeft,
  ChevronRight,
  SlidersHorizontal,
  ArrowUpDown,
  X,
  RotateCcw,
  Layers,
  Coins,
} from 'lucide-react';
import { apiFetch, subscribeEvents } from '../lib/api';
import { formatCost, formatDuration, formatTokens, formatDate } from '../lib/format';
import { getAgentMeta } from '../lib/agents';
import { useDebounce } from '../lib/useDebounce';
import type { Session } from '../lib/types';

interface PricingResponse {
  models?: Record<string, any>;
}

export const SessionList: React.FC = () => {
  // 1. Filter States
  const [search, setSearch] = useState('');
  const debouncedSearch = useDebounce(search, 300);

  const [selectedAgent, setSelectedAgent] = useState('');
  const [selectedModel, setSelectedModel] = useState('');
  const [datePreset, setDatePreset] = useState<'all' | 'today' | '7d' | '30d'>('all');
  const [minCost, setMinCost] = useState<string>('');
  const [maxCost, setMaxCost] = useState<string>('');
  const [minTokens, setMinTokens] = useState<string>('');
  const [maxTokens, setMaxTokens] = useState<string>('');
  const [sortBy, setSortBy] = useState<string>('start_time');
  const [sortOrder, setSortOrder] = useState<'desc' | 'asc'>('desc');
  const [page, setPage] = useState(1);
  const [showFiltersPanel, setShowFiltersPanel] = useState(false);

  // 2. Data States
  const [sessions, setSessions] = useState<Session[]>([]);
  const [agents, setAgents] = useState<string[]>([]);
  const [modelsList, setModelsList] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [totalPages, setTotalPages] = useState(1);
  const [totalCount, setTotalCount] = useState<number>(0);
  const [isInitialMount, setIsInitialMount] = useState(true);

  // 3. Initialize Filters from URL / SessionStorage
  useEffect(() => {
    if (typeof window === 'undefined') return;

    const urlParams = new URLSearchParams(window.location.search);
    const hasUrlParams = urlParams.toString().length > 0;

    let initSearch = '';
    let initAgent = '';
    let initModel = '';
    let initDate = 'all';
    let initMinCost = '';
    let initMaxCost = '';
    let initMinTokens = '';
    let initMaxTokens = '';
    let initSortBy = 'start_time';
    let initSortOrder: 'desc' | 'asc' = 'desc';
    let initPage = 1;

    if (hasUrlParams) {
      initSearch = urlParams.get('q') || urlParams.get('search') || '';
      initAgent = urlParams.get('agent') || '';
      initModel = urlParams.get('model') || '';
      initDate = (urlParams.get('date_preset') as any) || 'all';
      initMinCost = urlParams.get('min_cost') || '';
      initMaxCost = urlParams.get('max_cost') || '';
      initMinTokens = urlParams.get('min_tokens') || '';
      initMaxTokens = urlParams.get('max_tokens') || '';
      initSortBy = urlParams.get('sort_by') || 'start_time';
      initSortOrder = urlParams.get('sort_order') === 'asc' ? 'asc' : 'desc';
      initPage = parseInt(urlParams.get('page') || '1', 10) || 1;
    } else {
      try {
        const stored = sessionStorage.getItem('tt_session_filters');
        if (stored) {
          const parsed = JSON.parse(stored);
          initSearch = parsed.search || '';
          initAgent = parsed.agent || '';
          initModel = parsed.model || '';
          initDate = parsed.datePreset || 'all';
          initMinCost = parsed.minCost || '';
          initMaxCost = parsed.maxCost || '';
          initMinTokens = parsed.minTokens || '';
          initMaxTokens = parsed.maxTokens || '';
          initSortBy = parsed.sortBy || 'start_time';
          initSortOrder = parsed.sortOrder || 'desc';
          initPage = parsed.page || 1;
        }
      } catch {}
    }

    setSearch(initSearch);
    setSelectedAgent(initAgent);
    setSelectedModel(initModel);
    setDatePreset(initDate as any);
    setMinCost(initMinCost);
    setMaxCost(initMaxCost);
    setMinTokens(initMinTokens);
    setMaxTokens(initMaxTokens);
    setSortBy(initSortBy);
    setSortOrder(initSortOrder);
    setPage(initPage);
    setIsInitialMount(false);
  }, []);

  // 4. Fetch Agents and Models Catalogs
  useEffect(() => {
    apiFetch<string[]>('/agents')
      .then((res) => setAgents(res || []))
      .catch(() => {});

    apiFetch<PricingResponse>('/api/pricing')
      .then((res) => {
        if (res && res.models) {
          setModelsList(Object.keys(res.models).sort());
        }
      })
      .catch(() => {});
  }, []);

  // 5. Query Builder & URL / Storage Sync
  const fetchSessions = async () => {
    setLoading(true);
    try {
      const q = new URLSearchParams();
      q.set('page', page.toString());
      q.set('limit', '30');
      q.set('format', 'paginated');

      if (debouncedSearch) {
        q.set('q', debouncedSearch);
      }
      if (selectedAgent) {
        q.set('agent', selectedAgent);
      }
      if (selectedModel) {
        q.set('model', selectedModel);
      }

      // Date Range Calculation
      const now = new Date();
      if (datePreset === 'today') {
        const startOfDay = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate(), 0, 0, 0));
        q.set('since', startOfDay.toISOString());
      } else if (datePreset === '7d') {
        const past7 = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
        q.set('since', past7.toISOString());
      } else if (datePreset === '30d') {
        const past30 = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
        q.set('since', past30.toISOString());
      }

      if (minCost) q.set('min_cost', minCost);
      if (maxCost) q.set('max_cost', maxCost);
      if (minTokens) q.set('min_tokens', minTokens);
      if (maxTokens) q.set('max_tokens', maxTokens);

      if (sortBy) q.set('sort_by', sortBy);
      if (sortOrder) q.set('sort_order', sortOrder);

      const resp = await apiFetch<{
        sessions: Session[];
        pagination: { page: number; page_size: number; total: number; total_pages: number };
      }>(`/api/sessions?${q.toString()}`);

      setSessions(resp.sessions || []);
      setTotalPages(resp.pagination?.total_pages || 1);
      setTotalCount(resp.pagination?.total || 0);

      // Persist to URL and SessionStorage
      if (!isInitialMount && typeof window !== 'undefined') {
        const browserParams = new URLSearchParams();
        if (debouncedSearch) browserParams.set('q', debouncedSearch);
        if (selectedAgent) browserParams.set('agent', selectedAgent);
        if (selectedModel) browserParams.set('model', selectedModel);
        if (datePreset !== 'all') browserParams.set('date_preset', datePreset);
        if (minCost) browserParams.set('min_cost', minCost);
        if (maxCost) browserParams.set('max_cost', maxCost);
        if (minTokens) browserParams.set('min_tokens', minTokens);
        if (maxTokens) browserParams.set('max_tokens', maxTokens);
        if (sortBy !== 'start_time') browserParams.set('sort_by', sortBy);
        if (sortOrder !== 'desc') browserParams.set('sort_order', sortOrder);
        if (page > 1) browserParams.set('page', page.toString());

        const newUrl = browserParams.toString()
          ? `${window.location.pathname}?${browserParams.toString()}`
          : window.location.pathname;
        window.history.replaceState({}, '', newUrl);

        sessionStorage.setItem(
          'tt_session_filters',
          JSON.stringify({
            search: debouncedSearch,
            agent: selectedAgent,
            model: selectedModel,
            datePreset,
            minCost,
            maxCost,
            minTokens,
            maxTokens,
            sortBy,
            sortOrder,
            page,
          })
        );
      }
    } catch (e) {
      console.error('Failed to fetch sessions', e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!isInitialMount) {
      fetchSessions();
    }
  }, [
    debouncedSearch,
    selectedAgent,
    selectedModel,
    datePreset,
    minCost,
    maxCost,
    minTokens,
    maxTokens,
    sortBy,
    sortOrder,
    page,
    isInitialMount,
  ]);

  // Real-time Event Subscription
  useEffect(() => {
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
  }, [debouncedSearch, selectedAgent, selectedModel, datePreset, minCost, maxCost, minTokens, maxTokens, sortBy, sortOrder, page]);

  const resetAllFilters = () => {
    setSearch('');
    setSelectedAgent('');
    setSelectedModel('');
    setDatePreset('all');
    setMinCost('');
    setMaxCost('');
    setMinTokens('');
    setMaxTokens('');
    setSortBy('start_time');
    setSortOrder('desc');
    setPage(1);
  };

  const hasActiveFilters = useMemo(() => {
    return Boolean(
      search ||
      selectedAgent ||
      selectedModel ||
      datePreset !== 'all' ||
      minCost ||
      maxCost ||
      minTokens ||
      maxTokens ||
      sortBy !== 'start_time' ||
      sortOrder !== 'desc'
    );
  }, [search, selectedAgent, selectedModel, datePreset, minCost, maxCost, minTokens, maxTokens, sortBy, sortOrder]);

  return (
    <div className="space-y-4">
      {/* 1. Top Controls Bar */}
      <div className="flex flex-col lg:flex-row gap-3 items-stretch lg:items-center justify-between">
        {/* Search Input */}
        <div className="relative flex-1 max-w-md">
          <Search className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            placeholder="Search session ID, branch, project, keywords..."
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              setPage(1);
            }}
            className="w-full bg-[#11141a] border border-white/10 rounded-lg pl-9 pr-8 py-2 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-blue-500 transition-colors"
          />
          {search && (
            <button
              onClick={() => {
                setSearch('');
                setPage(1);
              }}
              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          )}
        </div>

        {/* Dropdowns & Filter Toggles */}
        <div className="flex flex-wrap items-center gap-2">
          {/* Model Filter Dropdown */}
          <div className="relative">
            <select
              value={selectedModel}
              onChange={(e) => {
                setSelectedModel(e.target.value);
                setPage(1);
              }}
              className="bg-[#11141a] border border-white/10 rounded-lg px-3 py-2 text-xs text-gray-300 focus:outline-none focus:border-blue-500 cursor-pointer appearance-none pr-7"
            >
              <option value="">All Models</option>
              {modelsList.map((m) => (
                <option key={m} value={m}>
                  {m}
                </option>
              ))}
            </select>
            <div className="pointer-events-none absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 text-[10px]">
              ▼
            </div>
          </div>

          {/* Date Range Preset Selector */}
          <div className="flex items-center rounded-lg bg-[#11141a] border border-white/10 p-0.5 text-xs">
            <button
              onClick={() => { setDatePreset('all'); setPage(1); }}
              className={`px-2.5 py-1 rounded-md transition-colors ${
                datePreset === 'all' ? 'bg-white/10 text-white font-medium' : 'text-gray-400 hover:text-white'
              }`}
            >
              All Time
            </button>
            <button
              onClick={() => { setDatePreset('today'); setPage(1); }}
              className={`px-2.5 py-1 rounded-md transition-colors ${
                datePreset === 'today' ? 'bg-white/10 text-white font-medium' : 'text-gray-400 hover:text-white'
              }`}
            >
              Today
            </button>
            <button
              onClick={() => { setDatePreset('7d'); setPage(1); }}
              className={`px-2.5 py-1 rounded-md transition-colors ${
                datePreset === '7d' ? 'bg-white/10 text-white font-medium' : 'text-gray-400 hover:text-white'
              }`}
            >
              7 Days
            </button>
            <button
              onClick={() => { setDatePreset('30d'); setPage(1); }}
              className={`px-2.5 py-1 rounded-md transition-colors ${
                datePreset === '30d' ? 'bg-white/10 text-white font-medium' : 'text-gray-400 hover:text-white'
              }`}
            >
              30 Days
            </button>
          </div>

          {/* Sort Selector */}
          <div className="flex items-center gap-1 bg-[#11141a] border border-white/10 rounded-lg px-2 py-1 text-xs">
            <span className="text-gray-500 text-[11px]">Sort:</span>
            <select
              value={sortBy}
              onChange={(e) => {
                setSortBy(e.target.value);
                setPage(1);
              }}
              className="bg-transparent text-gray-300 focus:outline-none cursor-pointer text-xs"
            >
              <option value="start_time" className="bg-[#11141a]">Recent</option>
              <option value="cost" className="bg-[#11141a]">Net Cost</option>
              <option value="tokens" className="bg-[#11141a]">Total Tokens</option>
              <option value="duration" className="bg-[#11141a]">Duration</option>
              {search && <option value="relevance" className="bg-[#11141a]">Relevance</option>}
            </select>
            <button
              onClick={() => setSortOrder(sortOrder === 'desc' ? 'asc' : 'desc')}
              title={`Toggle sort order (current: ${sortOrder.toUpperCase()})`}
              className="p-1 hover:bg-white/10 rounded text-gray-400 hover:text-white transition-colors"
            >
              <ArrowUpDown className="w-3.5 h-3.5" />
            </button>
          </div>

          {/* Advanced Numeric Filters Toggle */}
          <button
            onClick={() => setShowFiltersPanel(!showFiltersPanel)}
            className={`flex items-center gap-1.5 px-3 py-2 rounded-lg border text-xs font-medium transition-colors ${
              showFiltersPanel || minCost || maxCost || minTokens || maxTokens
                ? 'bg-blue-600/20 border-blue-500/40 text-blue-400'
                : 'bg-[#11141a] border-white/10 text-gray-400 hover:text-white'
            }`}
          >
            <SlidersHorizontal className="w-3.5 h-3.5" />
            <span>Range Filters</span>
          </button>

          {/* Reset Filters */}
          {hasActiveFilters && (
            <button
              onClick={resetAllFilters}
              title="Reset all filters"
              className="flex items-center gap-1 px-2.5 py-2 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 hover:bg-red-500/20 text-xs font-medium transition-colors"
            >
              <RotateCcw className="w-3.5 h-3.5" />
              <span>Reset</span>
            </button>
          )}
        </div>
      </div>

      {/* 2. Agent Pills Filter Bar */}
      <div className="flex flex-wrap gap-1.5 items-center">
        <button
          onClick={() => { setSelectedAgent(''); setPage(1); }}
          className={`px-2.5 py-1 rounded-md text-xs font-medium transition-colors ${
            selectedAgent === ''
              ? 'bg-blue-600/30 text-blue-400 border border-blue-500/40'
              : 'bg-white/5 text-gray-400 hover:text-white border border-transparent'
          }`}
        >
          All Agents ({totalCount})
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

      {/* 3. Expandable Range Bounds Panel */}
      {showFiltersPanel && (
        <div className="p-4 rounded-xl bg-[#11141a] border border-white/10 grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4 text-xs">
          <div>
            <label className="text-gray-400 font-medium block mb-1.5 flex items-center gap-1">
              <Coins className="w-3.5 h-3.5 text-emerald-400" />
              <span>Min Cost ($ USD)</span>
            </label>
            <input
              type="number"
              step="0.01"
              min="0"
              placeholder="e.g. 0.05"
              value={minCost}
              onChange={(e) => { setMinCost(e.target.value); setPage(1); }}
              className="w-full bg-black/40 border border-white/10 rounded-lg px-3 py-1.5 text-white placeholder-gray-600 focus:outline-none focus:border-blue-500"
            />
          </div>

          <div>
            <label className="text-gray-400 font-medium block mb-1.5 flex items-center gap-1">
              <Coins className="w-3.5 h-3.5 text-emerald-400" />
              <span>Max Cost ($ USD)</span>
            </label>
            <input
              type="number"
              step="0.01"
              min="0"
              placeholder="e.g. 5.00"
              value={maxCost}
              onChange={(e) => { setMaxCost(e.target.value); setPage(1); }}
              className="w-full bg-black/40 border border-white/10 rounded-lg px-3 py-1.5 text-white placeholder-gray-600 focus:outline-none focus:border-blue-500"
            />
          </div>

          <div>
            <label className="text-gray-400 font-medium block mb-1.5 flex items-center gap-1">
              <Layers className="w-3.5 h-3.5 text-purple-400" />
              <span>Min Tokens (Total)</span>
            </label>
            <input
              type="number"
              step="1000"
              min="0"
              placeholder="e.g. 10000"
              value={minTokens}
              onChange={(e) => { setMinTokens(e.target.value); setPage(1); }}
              className="w-full bg-black/40 border border-white/10 rounded-lg px-3 py-1.5 text-white placeholder-gray-600 focus:outline-none focus:border-blue-500"
            />
          </div>

          <div>
            <label className="text-gray-400 font-medium block mb-1.5 flex items-center gap-1">
              <Layers className="w-3.5 h-3.5 text-purple-400" />
              <span>Max Tokens (Total)</span>
            </label>
            <input
              type="number"
              step="1000"
              min="0"
              placeholder="e.g. 500000"
              value={maxTokens}
              onChange={(e) => { setMaxTokens(e.target.value); setPage(1); }}
              className="w-full bg-black/40 border border-white/10 rounded-lg px-3 py-1.5 text-white placeholder-gray-600 focus:outline-none focus:border-blue-500"
            />
          </div>
        </div>
      )}

      {/* 4. Sessions Table */}
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
                      <div className="text-[11px] text-gray-400 flex items-center gap-1.5 mt-0.5">
                        <FolderGit2 className="w-3 h-3 text-gray-500" />
                        <span>{s.project_name || 'root'}</span>
                        {s.git_branch && (
                          <span className="text-[10px] px-1.5 py-0.2 bg-white/5 rounded text-gray-400 font-mono">
                            {s.git_branch}
                          </span>
                        )}
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

        {/* 5. Pagination Footer */}
        <div className="p-3 border-t border-white/10 flex items-center justify-between text-xs text-gray-400 bg-white/[0.01]">
          <div>
            Page {page} of {totalPages || 1} ({totalCount} total sessions)
          </div>
          <div className="flex items-center gap-2">
            <button
              disabled={page <= 1}
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              className="p-1.5 rounded bg-white/5 hover:bg-white/10 disabled:opacity-30 disabled:pointer-events-none transition-colors"
            >
              <ChevronLeft className="w-4 h-4" />
            </button>
            <button
              disabled={page >= totalPages}
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
