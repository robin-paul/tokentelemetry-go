import React, { useEffect, useState, useRef } from 'react';
import {
  ArrowLeft,
  Bot,
  User,
  Wrench,
  Sparkles,
  Coins,
  Cpu,
  Layers,
  Play,
  Pause,
  ChevronLeft,
  ChevronRight,
  Search,
  Brain,
  Copy,
  Check,
  FileCode,
  SlidersHorizontal,
} from 'lucide-react';
import { apiFetch } from '../lib/api';
import { formatCost, formatDuration, formatTokens, formatDate } from '../lib/format';
import { getAgentMeta } from '../lib/agents';
import type { Session, MessageTurn } from '../lib/types';
import { UserTurnCard } from './session/UserTurnCard';
import { AssistantTurnCard } from './session/AssistantTurnCard';
import { ReasoningCard } from './session/ReasoningCard';

interface SessionDetailProps {
  sessionId?: string;
}

type TurnFilterCategory = 'all' | 'user' | 'assistant' | 'reasoning' | 'tools';

export const SessionDetail: React.FC<SessionDetailProps> = ({ sessionId: propSessionId }) => {
  const [session, setSession] = useState<Session | null>(null);
  const [activeStep, setActiveStep] = useState<number>(0);
  const [loading, setLoading] = useState(true);
  const [isPlaying, setIsPlaying] = useState(false);
  const [categoryFilter, setCategoryFilter] = useState<TurnFilterCategory>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [copiedId, setCopiedId] = useState(false);
  const [showRawJson, setShowRawJson] = useState(false);

  const timerRef = useRef<NodeJS.Timeout | null>(null);
  const turnRefs = useRef<(HTMLDivElement | null)[]>([]);

  useEffect(() => {
    let id = propSessionId;
    if ((!id || id === '[id]') && typeof window !== 'undefined') {
      const parts = window.location.pathname.split('/').filter(Boolean);
      if (parts[0] === 'sessions' && parts[1]) {
        id = decodeURIComponent(parts[1]);
      }
    }
    if (id && id !== '[id]') {
      apiFetch<Session>(`/api/sessions/${encodeURIComponent(id)}`)
        .then((data) => {
          setSession(data);
          if (data.turns && data.turns.length > 0) {
            setActiveStep(data.turns.length - 1);
          }
        })
        .catch((e) => console.error('Failed to load session details', e))
        .finally(() => setLoading(false));
    }
  }, [propSessionId]);

  const turns = session?.turns || [];

  // Playback timer loop (600ms per step)
  useEffect(() => {
    if (isPlaying) {
      timerRef.current = setInterval(() => {
        setActiveStep((prev) => {
          if (prev >= turns.length - 1) {
            setIsPlaying(false);
            return prev;
          }
          return prev + 1;
        });
      }, 600);
    } else if (timerRef.current) {
      clearInterval(timerRef.current);
    }
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
    };
  }, [isPlaying, turns.length]);

  const handleCopyId = () => {
    if (!session) return;
    navigator.clipboard.writeText(session.session_id);
    setCopiedId(true);
    setTimeout(() => setCopiedId(false), 2000);
  };

  const handleStepSeek = (index: number) => {
    setActiveStep(index);
    if (turnRefs.current[index]) {
      turnRefs.current[index]?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    }
  };

  if (loading) {
    return (
      <div className="p-12 text-center space-y-3">
        <div className="inline-block w-6 h-6 border-2 border-cyan-500 border-t-transparent rounded-full animate-spin" />
        <div className="text-xs text-gray-500 font-mono">Loading deep session inspector...</div>
      </div>
    );
  }

  if (!session) {
    return (
      <div className="p-12 text-center space-y-3">
        <div className="text-sm text-gray-400">Session not found.</div>
        <a href="/sessions" className="text-xs text-cyan-400 hover:underline">
          Return to sessions catalog
        </a>
      </div>
    );
  }

  const meta = getAgentMeta(session.agent_name);
  const subagents = session.subagent_runs || [];

  // Filter turn calculations
  const userTurnsCount = turns.filter((t) => t.role === 'user').length;
  const assistantTurnsCount = turns.filter((t) => t.role === 'assistant').length;
  const reasoningTurnsCount = turns.filter((t) => !!t.thinking || !!t.reasoning_effort).length;
  const toolTurnsCount = turns.filter(
    (t) => (t.tools_invoked && t.tools_invoked.length > 0) || (t.tool_calls && t.tool_calls.length > 0)
  ).length;

  const filteredTurns = turns.filter((turn) => {
    // 1. Category Filter
    if (categoryFilter === 'user' && turn.role !== 'user') return false;
    if (categoryFilter === 'assistant' && turn.role !== 'assistant') return false;
    if (categoryFilter === 'reasoning' && !turn.thinking && !turn.reasoning_effort) return false;
    if (categoryFilter === 'tools') {
      const hasLegacy = turn.tools_invoked && turn.tools_invoked.length > 0;
      const hasTools = turn.tool_calls && turn.tool_calls.length > 0;
      if (!hasLegacy && !hasTools) return false;
    }

    // 2. Search query filter
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      const matchContent = turn.content?.toLowerCase().includes(q);
      const matchThinking = turn.thinking?.toLowerCase().includes(q);
      const matchModel = turn.model_name?.toLowerCase().includes(q);
      const matchTools =
        turn.tools_invoked?.some((t) => t.toLowerCase().includes(q)) ||
        turn.tool_calls?.some((tc) => tc.name.toLowerCase().includes(q));
      if (!matchContent && !matchThinking && !matchModel && !matchTools) {
        return false;
      }
    }

    return true;
  });

  return (
    <div className="space-y-6">
      {/* Header & Breadcrumbs */}
      <div>
        <a
          href="/sessions"
          className="inline-flex items-center gap-1.5 text-xs text-gray-400 hover:text-white mb-3 transition-colors"
        >
          <ArrowLeft className="w-3.5 h-3.5" /> Back to Sessions
        </a>

        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-2.5 flex-wrap">
              <span
                className="px-2.5 py-1 rounded text-xs font-semibold"
                style={{ color: meta.color, backgroundColor: meta.bg }}
              >
                {meta.label}
              </span>
              <div className="flex items-center gap-1.5 bg-white/5 px-2.5 py-1 rounded-lg border border-white/5">
                <h1 className="text-base font-bold text-white font-mono">{session.session_id}</h1>
                <button
                  type="button"
                  onClick={handleCopyId}
                  className="text-gray-400 hover:text-white transition-colors"
                  title="Copy session ID"
                >
                  {copiedId ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                </button>
              </div>
              {session.git_branch && (
                <span className="text-xs font-mono text-gray-400 bg-white/5 px-2 py-0.5 rounded border border-white/5">
                  branch: <span className="text-gray-200 font-semibold">{session.git_branch}</span>
                </span>
              )}
            </div>
            <p className="text-xs text-gray-400 mt-1.5">
              Project: <span className="text-white font-medium">{session.project_name}</span> • Started{' '}
              {formatDate(session.start_time)}
            </p>
          </div>

          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={() => setShowRawJson(!showRawJson)}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-white/5 hover:bg-white/10 border border-white/10 text-xs font-mono text-gray-300 hover:text-white transition-colors"
            >
              <FileCode className="w-3.5 h-3.5 text-cyan-400" />
              <span>{showRawJson ? 'Hide Raw JSON' : 'Raw Session JSON'}</span>
            </button>
            <div className="px-3.5 py-1.5 rounded-lg bg-[#11141a] border border-white/10 text-right">
              <div className="text-[10px] uppercase tracking-wider text-gray-400">Net Cost</div>
              <div className="text-sm font-bold text-emerald-400 tabular">{formatCost(session.net_cost_usd)}</div>
            </div>
            <div className="px-3.5 py-1.5 rounded-lg bg-[#11141a] border border-white/10 text-right">
              <div className="text-[10px] uppercase tracking-wider text-gray-400">Tokens</div>
              <div className="text-sm font-bold text-blue-400 tabular">
                {formatTokens(session.input_tokens + session.output_tokens)}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Metrics Summary Strip */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 p-4 rounded-xl bg-[#11141a] border border-white/10 text-xs">
        <div>
          <span className="text-gray-400 flex items-center gap-1">
            <Cpu className="w-3.5 h-3.5 text-cyan-400" /> Model
          </span>
          <div className="font-mono text-white font-medium mt-1 truncate">
            {session.model_resolved || session.model_raw}
          </div>
        </div>
        <div>
          <span className="text-gray-400 flex items-center gap-1">
            <Coins className="w-3.5 h-3.5 text-purple-400" /> Prompt Cache
          </span>
          <div className="text-purple-400 font-medium mt-1 tabular">{formatTokens(session.cache_read_tokens)} reads</div>
        </div>
        <div>
          <span className="text-gray-400 flex items-center gap-1">
            <Layers className="w-3.5 h-3.5 text-blue-400" /> Turns / Steps
          </span>
          <div className="text-white font-medium mt-1 tabular">{turns.length} message turns</div>
        </div>
        <div>
          <span className="text-gray-400 flex items-center gap-1">
            <Sparkles className="w-3.5 h-3.5 text-amber-400" /> Duration
          </span>
          <div className="text-white font-medium mt-1 tabular">{formatDuration(session.duration_seconds)}</div>
        </div>
      </div>

      {/* Raw JSON viewer modal / accordion */}
      {showRawJson && (
        <div className="p-4 rounded-xl bg-[#07090d] border border-white/10 space-y-2">
          <div className="flex items-center justify-between text-xs text-gray-400 font-mono">
            <span>Raw Session Payload</span>
            <span>{session.turns?.length || 0} turns serialized</span>
          </div>
          <pre className="p-4 rounded-lg bg-black/60 border border-white/5 text-cyan-300 text-xs font-mono max-h-96 overflow-y-auto leading-relaxed">
            {JSON.stringify(session, null, 2)}
          </pre>
        </div>
      )}

      {/* Turn Scrubber & Controls Panel */}
      {turns.length > 0 && (
        <div className="p-4 rounded-xl bg-[#11141a] border border-white/10 space-y-3">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs">
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => setIsPlaying(!isPlaying)}
                className="p-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white transition-colors"
                title={isPlaying ? 'Pause auto-replay' : 'Play auto-replay'}
              >
                {isPlaying ? <Pause className="w-4 h-4 text-amber-400" /> : <Play className="w-4 h-4 text-emerald-400" />}
              </button>
              <button
                type="button"
                onClick={() => handleStepSeek(Math.max(0, activeStep - 1))}
                disabled={activeStep <= 0}
                className="p-1.5 rounded-lg bg-white/5 hover:bg-white/10 text-gray-300 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                title="Previous turn"
              >
                <ChevronLeft className="w-4 h-4" />
              </button>
              <button
                type="button"
                onClick={() => handleStepSeek(Math.min(turns.length - 1, activeStep + 1))}
                disabled={activeStep >= turns.length - 1}
                className="p-1.5 rounded-lg bg-white/5 hover:bg-white/10 text-gray-300 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                title="Next turn"
              >
                <ChevronRight className="w-4 h-4" />
              </button>

              <span className="text-gray-400 font-medium ml-2">Timeline Scrubber</span>
            </div>

            <span className="text-blue-400 font-mono font-semibold">
              Step {activeStep + 1} of {turns.length}
            </span>
          </div>

          <input
            type="range"
            min={0}
            max={turns.length > 0 ? turns.length - 1 : 0}
            value={activeStep}
            onChange={(e) => handleStepSeek(parseInt(e.target.value, 10))}
            className="w-full accent-blue-500 cursor-pointer"
          />
        </div>
      )}

      {/* Filter and In-Trace Search Strip */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-3">
        {/* Category Pills */}
        <div className="flex items-center gap-1.5 overflow-x-auto pb-1 text-xs">
          <button
            type="button"
            onClick={() => setCategoryFilter('all')}
            className={`px-3 py-1.5 rounded-lg font-medium transition-all ${
              categoryFilter === 'all'
                ? 'bg-blue-600 text-white font-semibold shadow-sm'
                : 'bg-white/5 text-gray-400 hover:text-white hover:bg-white/10'
            }`}
          >
            All Turns ({turns.length})
          </button>
          <button
            type="button"
            onClick={() => setCategoryFilter('user')}
            className={`px-3 py-1.5 rounded-lg font-medium transition-all ${
              categoryFilter === 'user'
                ? 'bg-blue-600 text-white font-semibold shadow-sm'
                : 'bg-white/5 text-gray-400 hover:text-white hover:bg-white/10'
            }`}
          >
            User ({userTurnsCount})
          </button>
          <button
            type="button"
            onClick={() => setCategoryFilter('assistant')}
            className={`px-3 py-1.5 rounded-lg font-medium transition-all ${
              categoryFilter === 'assistant'
                ? 'bg-blue-600 text-white font-semibold shadow-sm'
                : 'bg-white/5 text-gray-400 hover:text-white hover:bg-white/10'
            }`}
          >
            Assistant ({assistantTurnsCount})
          </button>
          {reasoningTurnsCount > 0 && (
            <button
              type="button"
              onClick={() => setCategoryFilter('reasoning')}
              className={`px-3 py-1.5 rounded-lg font-medium transition-all flex items-center gap-1 ${
                categoryFilter === 'reasoning'
                  ? 'bg-amber-600 text-white font-semibold shadow-sm'
                  : 'bg-amber-500/10 text-amber-300 hover:bg-amber-500/20'
              }`}
            >
              <Brain className="w-3.5 h-3.5" />
              <span>Reasoning ({reasoningTurnsCount})</span>
            </button>
          )}
          {toolTurnsCount > 0 && (
            <button
              type="button"
              onClick={() => setCategoryFilter('tools')}
              className={`px-3 py-1.5 rounded-lg font-medium transition-all flex items-center gap-1 ${
                categoryFilter === 'tools'
                  ? 'bg-cyan-600 text-white font-semibold shadow-sm'
                  : 'bg-cyan-500/10 text-cyan-300 hover:bg-cyan-500/20'
              }`}
            >
              <Wrench className="w-3.5 h-3.5" />
              <span>Tools ({toolTurnsCount})</span>
            </button>
          )}
        </div>

        {/* Search Input */}
        <div className="relative min-w-[240px]">
          <Search className="w-3.5 h-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-gray-500" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search prompt, model, tools..."
            className="w-full pl-9 pr-3 py-1.5 text-xs bg-[#11141a] border border-white/10 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-cyan-500"
          />
        </div>
      </div>

      {/* Subagents Section if any */}
      {subagents.length > 0 && (
        <div className="p-4 rounded-xl bg-[#11141a] border border-white/10 space-y-3">
          <h3 className="text-xs font-semibold text-white uppercase tracking-wider flex items-center gap-1.5">
            <Bot className="w-4 h-4 text-purple-400" /> Spawned Subagents ({subagents.length})
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
            {subagents.map((sub) => (
              <div
                key={sub.id}
                className="p-2.5 rounded-lg bg-white/[0.03] border border-white/5 flex items-center justify-between text-xs"
              >
                <div>
                  <div className="font-semibold text-white">{sub.agent_type}</div>
                  <div className="text-[11px] font-mono text-gray-400">
                    {sub.child_session_id.slice(0, 14)}...
                  </div>
                </div>
                <div className="text-right">
                  <div className="text-emerald-400 font-medium">{formatCost(sub.cost_usd)}</div>
                  <div className="text-[11px] text-gray-500">{formatTokens(sub.tokens)} tok</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Chronological Turns Stream */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-gray-400">
            Conversation Turns ({filteredTurns.length} of {turns.length})
          </h3>
        </div>

        {filteredTurns.length === 0 ? (
          <div className="p-8 text-center text-xs text-gray-500 border border-white/5 rounded-xl bg-[#11141a]">
            No message turns match the current filter or search criteria.
          </div>
        ) : (
          <div className="space-y-4">
            {filteredTurns.map((turn, index) => {
              const originalIndex = turn.turn_index >= 0 ? turn.turn_index : index;
              const isUser = turn.role === 'user';
              const isActive = originalIndex === activeStep;

              return (
                <div
                  key={turn.id || originalIndex}
                  ref={(el) => {
                    turnRefs.current[originalIndex] = el;
                  }}
                >
                  {isUser ? (
                    <UserTurnCard
                      turn={turn}
                      isActive={isActive}
                      onClick={() => setActiveStep(originalIndex)}
                    />
                  ) : (
                    <AssistantTurnCard
                      turn={turn}
                      agentName={session.agent_name}
                      isActive={isActive}
                      onClick={() => setActiveStep(originalIndex)}
                    />
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};
