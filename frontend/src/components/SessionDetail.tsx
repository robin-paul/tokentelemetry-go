import React, { useEffect, useState, useRef } from 'react';
import {
  ArrowLeft,
  Bot,
  Sparkles,
  Coins,
  Cpu,
  Layers,
  Copy,
  Check,
  PanelRight,
} from 'lucide-react';
import { apiFetch } from '../lib/api';
import { formatCost, formatDuration, formatTokens, formatDate } from '../lib/format';
import { getAgentMeta } from '../lib/agents';
import type { Session, MessageTurn } from '../lib/types';
import { UserTurnCard } from './session/UserTurnCard';
import { AssistantTurnCard } from './session/AssistantTurnCard';
import { TurnScrubber } from './session/TurnScrubber';
import { StepIndex } from './session/StepIndex';
import { StepFilterPopover, type TurnFilterCategory } from './session/StepFilterPopover';
import { TurnSearchInput } from './session/TurnSearchInput';
import { ExecutionWaterfall } from './session/ExecutionWaterfall';
import { InspectorSidebar } from './session/InspectorSidebar';
import { ArtifactLightboxModal, type LightboxArtifact } from './session/ArtifactLightboxModal';

interface SessionDetailProps {
  sessionId?: string;
}

export const SessionDetail: React.FC<SessionDetailProps> = ({ sessionId: propSessionId }) => {
  const [session, setSession] = useState<Session | null>(null);
  const [activeStep, setActiveStep] = useState<number>(0);
  const [revealedCount, setRevealedCount] = useState<number>(1);
  const [loading, setLoading] = useState(true);
  const [isPlaying, setIsPlaying] = useState(false);
  const [categoryFilter, setCategoryFilter] = useState<TurnFilterCategory>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [copiedId, setCopiedId] = useState(false);
  const [isSidebarOpen, setIsSidebarOpen] = useState(true);
  const [activeLightboxArtifact, setActiveLightboxArtifact] = useState<LightboxArtifact | null>(null);

  const timerRef = useRef<NodeJS.Timeout | null>(null);
  const turnRefs = useRef<(HTMLDivElement | null)[]>([]);
  const rafScrollHandleRef = useRef<number | null>(null);

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
            const lastIdx = data.turns.length - 1;
            setActiveStep(lastIdx);
            setRevealedCount(data.turns.length);
          }
        })
        .catch((e) => console.error('Failed to load session details', e))
        .finally(() => setLoading(false));
    }
  }, [propSessionId]);

  const turns = session?.turns || [];

  // Cleanup animation frame and timers on unmount
  useEffect(() => {
    return () => {
      if (rafScrollHandleRef.current !== null) {
        cancelAnimationFrame(rafScrollHandleRef.current);
      }
      if (timerRef.current) {
        clearInterval(timerRef.current);
      }
    };
  }, []);

  // RAF-debounced smooth scrolling to target turn
  const scrollToTurn = (index: number) => {
    if (rafScrollHandleRef.current !== null) {
      cancelAnimationFrame(rafScrollHandleRef.current);
    }
    rafScrollHandleRef.current = requestAnimationFrame(() => {
      const el = turnRefs.current[index];
      if (el) {
        el.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
      }
      rafScrollHandleRef.current = null;
    });
  };

  // Playback timer loop (600ms per step)
  useEffect(() => {
    if (isPlaying) {
      timerRef.current = setInterval(() => {
        setActiveStep((prev) => {
          if (prev >= turns.length - 1) {
            setIsPlaying(false);
            return prev;
          }
          const next = prev + 1;
          setRevealedCount((hwm) => Math.max(hwm, next + 1));
          scrollToTurn(next);
          return next;
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
    const clampedIndex = Math.max(0, Math.min(turns.length - 1, index));
    setActiveStep(clampedIndex);
    setRevealedCount((prev) => Math.max(prev, clampedIndex + 1));
    scrollToTurn(clampedIndex);
  };

  const handleTogglePlay = () => {
    if (!isPlaying && activeStep >= turns.length - 1 && turns.length > 1) {
      // If at the end, restart from step 0
      handleStepSeek(0);
      setIsPlaying(true);
    } else {
      setIsPlaying(!isPlaying);
    }
  };

  const handlePrevStep = () => {
    if (activeStep > 0) {
      handleStepSeek(activeStep - 1);
    }
  };

  const handleNextStep = () => {
    if (activeStep < turns.length - 1) {
      handleStepSeek(activeStep + 1);
    }
  };

  const handleReset = () => {
    handleStepSeek(0);
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

  const categoryCounts = {
    all: turns.length,
    user: userTurnsCount,
    assistant: assistantTurnsCount,
    reasoning: reasoningTurnsCount,
    tools: toolTurnsCount,
  };

  const allToolResults = useMemo(() => {
    return turns.flatMap((t) => t.tool_results || []);
  }, [turns]);

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
    <div className="space-y-6" data-testid="session-detail-view">
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
              data-testid="toggle-sidebar-button"
              onClick={() => setIsSidebarOpen(!isSidebarOpen)}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-white/5 hover:bg-white/10 border border-white/10 text-xs font-mono text-gray-300 hover:text-white transition-colors"
            >
              <PanelRight className="w-3.5 h-3.5 text-cyan-400" />
              <span>{isSidebarOpen ? 'Hide Inspector' : 'Show Inspector'}</span>
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

      {/* Turn Scrubber & Controls Panel */}
      {turns.length > 0 && (
        <TurnScrubber
          activeStep={activeStep}
          totalSteps={turns.length}
          revealedCount={revealedCount}
          isPlaying={isPlaying}
          onSeek={handleStepSeek}
          onTogglePlay={handleTogglePlay}
          onPrevStep={handlePrevStep}
          onNextStep={handleNextStep}
          onReset={handleReset}
        />
      )}

      {/* Step Index Strip */}
      {turns.length > 0 && (
        <StepIndex
          turns={turns}
          activeStep={activeStep}
          revealedCount={revealedCount}
          onSelectStep={handleStepSeek}
          categoryFilter={categoryFilter}
        />
      )}

      {/* Filter and In-Trace Search Strip */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-3">
        {/* Step Filter Popover / Category Pills */}
        <StepFilterPopover
          activeCategory={categoryFilter}
          onSelectCategory={setCategoryFilter}
          counts={categoryCounts}
        />

        {/* In-trace Keyword Search */}
        <TurnSearchInput
          value={searchQuery}
          onChange={setSearchQuery}
          matchCount={searchQuery.trim() ? filteredTurns.length : undefined}
          className="min-w-[260px]"
        />
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

      {/* Main Content Area: Conversation Stream + Inspector Sidebar */}
      <div className="flex flex-col lg:flex-row items-start gap-6">
        {/* Left/Center Conversation Stream */}
        <div className="flex-1 min-w-0 space-y-4 w-full">
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
                        turnNumber={originalIndex + 1}
                        isActive={isActive}
                        searchQuery={searchQuery}
                        onClick={() => handleStepSeek(originalIndex)}
                      />
                    ) : (
                      <AssistantTurnCard
                        turn={turn}
                        turnNumber={originalIndex + 1}
                        agentName={session.agent_name}
                        isActive={isActive}
                        searchQuery={searchQuery}
                        allToolResults={allToolResults}
                        onClick={() => handleStepSeek(originalIndex)}
                      />
                    )}
                  </div>
                );
              })}
            </div>
          )}

          {/* Bottom Execution Waterfall Gantt Timeline */}
          {turns.length > 0 && (
            <div className="pt-2">
              <ExecutionWaterfall
                turns={turns}
                activeStep={activeStep}
                onSelectStep={handleStepSeek}
              />
            </div>
          )}
        </div>

        {/* Right Inspector Sidebar (Context, Tools, Artifacts, Raw) */}
        <InspectorSidebar
          session={session}
          activeTurn={turns[activeStep] || null}
          activeStep={activeStep}
          isOpen={isSidebarOpen}
          onToggleOpen={() => setIsSidebarOpen(!isSidebarOpen)}
          onJumpToTurn={handleStepSeek}
          onOpenArtifact={(art) => setActiveLightboxArtifact(art)}
        />
      </div>

      {/* Portalled Full-screen Artifact Lightbox Modal */}
      {activeLightboxArtifact && (
        <ArtifactLightboxModal
          artifact={activeLightboxArtifact}
          onClose={() => setActiveLightboxArtifact(null)}
        />
      )}
    </div>
  );
};
