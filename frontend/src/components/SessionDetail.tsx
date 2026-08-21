import React, { useEffect, useState } from 'react';
import {
  ArrowLeft,
  Bot,
  User,
  Wrench,
  Sparkles,
  Coins,
  Cpu,
  Layers,
  ChevronRight,
} from 'lucide-react';
import { apiFetch } from '../lib/api';
import { formatCost, formatDuration, formatTokens, formatDate } from '../lib/format';
import { getAgentMeta } from '../lib/agents';
import type { Session } from '../lib/types';

interface SessionDetailProps {
  sessionId?: string;
}

export const SessionDetail: React.FC<SessionDetailProps> = ({ sessionId: propSessionId }) => {
  const [session, setSession] = useState<Session | null>(null);
  const [activeStep, setActiveStep] = useState<number>(0);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let id = propSessionId;
    if (!id && typeof window !== 'undefined') {
      const parts = window.location.pathname.split('/').filter(Boolean);
      if (parts[0] === 'sessions' && parts[1]) {
        id = parts[1];
      }
    }
    if (id) {
      apiFetch<Session>(`/api/sessions/${id}`)
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

  if (loading) {
    return <div className="p-8 text-center text-xs text-gray-500">Loading session trace...</div>;
  }

  if (!session) {
    return (
      <div className="p-8 text-center space-y-3">
        <div className="text-sm text-gray-400">Session not found.</div>
        <a href="/sessions" className="text-xs text-blue-400 hover:underline">
          Return to sessions list
        </a>
      </div>
    );
  }

  const meta = getAgentMeta(session.agent_name);
  const turns = session.turns || [];
  const subagents = session.subagent_runs || [];

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
            <div className="flex items-center gap-2.5">
              <span
                className="px-2.5 py-1 rounded text-xs font-semibold"
                style={{ color: meta.color, backgroundColor: meta.bg }}
              >
                {meta.label}
              </span>
              <h1 className="text-lg font-bold text-white font-mono">{session.session_id}</h1>
            </div>
            <p className="text-xs text-gray-400 mt-1">
              Project: <span className="text-white font-medium">{session.project_name}</span> • Started {formatDate(session.start_time)}
            </p>
          </div>
          <div className="flex items-center gap-3">
            <div className="px-3 py-1.5 rounded-lg bg-[#11141a] border border-white/10 text-right">
              <div className="text-[10px] uppercase tracking-wider text-gray-400">Net Cost</div>
              <div className="text-sm font-bold text-emerald-400 tabular">{formatCost(session.net_cost_usd)}</div>
            </div>
            <div className="px-3 py-1.5 rounded-lg bg-[#11141a] border border-white/10 text-right">
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
          <span className="text-gray-400 flex items-center gap-1"><Cpu className="w-3.5 h-3.5" /> Model</span>
          <div className="font-mono text-white font-medium mt-1 truncate">{session.model_resolved || session.model_raw}</div>
        </div>
        <div>
          <span className="text-gray-400 flex items-center gap-1"><Coins className="w-3.5 h-3.5" /> Prompt Cache</span>
          <div className="text-purple-400 font-medium mt-1 tabular">{formatTokens(session.cache_read_tokens)} reads</div>
        </div>
        <div>
          <span className="text-gray-400 flex items-center gap-1"><Layers className="w-3.5 h-3.5" /> Turns / Steps</span>
          <div className="text-white font-medium mt-1 tabular">{turns.length} message turns</div>
        </div>
        <div>
          <span className="text-gray-400 flex items-center gap-1"><Sparkles className="w-3.5 h-3.5" /> Duration</span>
          <div className="text-white font-medium mt-1 tabular">{formatDuration(session.duration_seconds)}</div>
        </div>
      </div>

      {/* Step Scrubber Slider */}
      {turns.length > 1 && (
        <div className="p-4 rounded-xl bg-[#11141a] border border-white/10 space-y-2">
          <div className="flex items-center justify-between text-xs">
            <span className="text-gray-400 font-medium">Timeline Scrubber</span>
            <span className="text-blue-400 font-mono">
              Step {activeStep + 1} of {turns.length}
            </span>
          </div>
          <input
            type="range"
            min={0}
            max={turns.length - 1}
            value={activeStep}
            onChange={(e) => setActiveStep(parseInt(e.target.value, 10))}
            className="w-full accent-blue-500 cursor-pointer"
          />
        </div>
      )}

      {/* Subagents Section if any */}
      {subagents.length > 0 && (
        <div className="p-4 rounded-xl bg-[#11141a] border border-white/10 space-y-3">
          <h3 className="text-xs font-semibold text-white uppercase tracking-wider flex items-center gap-1.5">
            <Bot className="w-4 h-4 text-purple-400" /> Spawned Subagents ({subagents.length})
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
            {subagents.map((sub) => (
              <div key={sub.id} className="p-2.5 rounded-lg bg-white/[0.03] border border-white/5 flex items-center justify-between text-xs">
                <div>
                  <div className="font-semibold text-white">{sub.agent_type}</div>
                  <div className="text-[11px] font-mono text-gray-400">{sub.child_session_id.slice(0, 14)}...</div>
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

      {/* Message Turns Waterfall */}
      <div className="space-y-3">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-gray-400">Chronological Turns</h3>
        <div className="space-y-3">
          {turns.map((turn, idx) => {
            const isUser = turn.role === 'user';
            const isHighlighted = idx <= activeStep;

            return (
              <div
                key={turn.id || idx}
                className={`p-4 rounded-xl border transition-all ${
                  isHighlighted ? 'bg-[#11141a] border-white/10' : 'bg-[#11141a]/40 border-white/5 opacity-50'
                }`}
              >
                <div className="flex items-center justify-between text-xs mb-2">
                  <div className="flex items-center gap-2">
                    <div
                      className={`w-6 h-6 rounded-md flex items-center justify-center ${
                        isUser ? 'bg-blue-500/20 text-blue-400' : 'bg-purple-500/20 text-purple-400'
                      }`}
                    >
                      {isUser ? <User className="w-3.5 h-3.5" /> : <Bot className="w-3.5 h-3.5" />}
                    </div>
                    <span className="font-semibold text-white uppercase tracking-wider text-[11px]">
                      {turn.role}
                    </span>
                    <span className="text-gray-500 text-[11px]">Turn #{turn.turn_index + 1}</span>
                  </div>
                  <div className="flex items-center gap-2 text-gray-400 font-mono text-[11px]">
                    <span>{formatTokens(turn.input_tokens + turn.output_tokens)} tok</span>
                    {turn.cost_usd > 0 && <span className="text-emerald-400">{formatCost(turn.cost_usd)}</span>}
                  </div>
                </div>

                {/* Tools invoked */}
                {turn.tools_invoked && turn.tools_invoked.length > 0 && (
                  <div className="flex flex-wrap gap-1.5 mt-2">
                    {turn.tools_invoked.map((tool, tIdx) => (
                      <span
                        key={tIdx}
                        className="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-amber-500/10 text-amber-400 border border-amber-500/20 text-[11px] font-mono"
                      >
                        <Wrench className="w-3 h-3" />
                        {tool}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
};
