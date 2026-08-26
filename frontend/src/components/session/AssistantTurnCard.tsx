import React from 'react';
import { Bot, Clock, Cpu } from 'lucide-react';
import { formatCost, formatTokens, formatDate } from '../../lib/format';
import { getAgentMeta } from '../../lib/agents';
import type { MessageTurn } from '../../lib/types';
import { ResponseBody } from './ResponseBody';
import { ReasoningCard } from './ReasoningCard';
import { ToolInvocationCard } from './ToolInvocationCard';

interface AssistantTurnCardProps {
  turn: MessageTurn;
  agentName: string;
  isActive?: boolean;
  onClick?: () => void;
}

export const AssistantTurnCard: React.FC<AssistantTurnCardProps> = ({
  turn,
  agentName,
  isActive = false,
  onClick,
}) => {
  const meta = getAgentMeta(agentName);
  const totalTokens = turn.input_tokens + turn.output_tokens;

  // Extract tools
  const toolCalls = turn.tool_calls || [];
  const toolResults = turn.tool_results || [];

  // Fallback for legacy tools_invoked
  let legacyTools = turn.tools_invoked || [];
  if (legacyTools.length === 0 && turn.tools_invoked_json) {
    try {
      legacyTools = JSON.parse(turn.tools_invoked_json);
    } catch {}
  }

  return (
    <div
      onClick={onClick}
      className={`bg-[#11141a] border rounded-xl p-5 relative overflow-hidden transition-all cursor-pointer ${
        isActive
          ? 'border-cyan-500/60 ring-2 ring-cyan-500/20 shadow-lg'
          : 'border-white/10 hover:border-white/20'
      }`}
    >
      {/* Left accent bar dynamically tinted to agent identity */}
      <div
        className="absolute top-0 left-0 w-1.5 h-full"
        style={{ backgroundColor: meta.color || '#06b6d4' }}
      />

      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 mb-3 pl-1">
        <div className="flex items-center gap-2">
          <div
            className="w-6 h-6 rounded-md flex items-center justify-center"
            style={{ color: meta.color, backgroundColor: meta.bg }}
          >
            <Bot className="w-3.5 h-3.5" />
          </div>
          <span
            className="text-[11px] font-black uppercase tracking-[0.16em]"
            style={{ color: meta.color }}
          >
            {meta.label} Response
          </span>
          <span className="text-[11px] font-mono text-gray-500">
            Turn #{turn.turn_index + 1}
          </span>
          {turn.model_name && (
            <span className="inline-flex items-center gap-1 text-[10px] font-mono text-gray-400 bg-white/5 px-2 py-0.5 rounded border border-white/5">
              <Cpu className="w-3 h-3 text-gray-500" />
              {turn.model_name}
            </span>
          )}
        </div>

        <div className="flex items-center gap-3 text-[11px] font-mono text-gray-400">
          {totalTokens > 0 && (
            <span className="text-gray-300">
              {formatTokens(totalTokens)} tok
              {turn.cache_read_tokens > 0 && (
                <span className="text-purple-400 ml-1">
                  ({formatTokens(turn.cache_read_tokens)} cached)
                </span>
              )}
            </span>
          )}
          {turn.cost_usd > 0 && (
            <span className="text-emerald-400 font-semibold">{formatCost(turn.cost_usd)}</span>
          )}
          {turn.timestamp && (
            <span className="flex items-center gap-1 text-gray-500">
              <Clock className="w-3 h-3" />
              {formatDate(turn.timestamp)}
            </span>
          )}
        </div>
      </div>

      {/* Body Section */}
      <div className="pl-1 space-y-3">
        {/* Reasoning / Thinking block if present */}
        {(turn.thinking || turn.reasoning_effort) && (
          <ReasoningCard
            thinking={turn.thinking}
            reasoningEffort={turn.reasoning_effort}
          />
        )}

        {/* Primary Markdown Text Response */}
        {turn.content && <ResponseBody content={turn.content} defaultMode="md" />}

        {/* Tool Invocations */}
        {toolCalls.length > 0 ? (
          <div className="space-y-2 pt-1">
            {toolCalls.map((tc, idx) => {
              const matchedResult =
                toolResults.find((r) => r.id === tc.id || r.name === tc.name) ||
                toolResults[idx];
              return (
                <ToolInvocationCard
                  key={tc.id || idx}
                  toolCall={tc}
                  toolResult={matchedResult}
                />
              );
            })}
          </div>
        ) : (
          legacyTools.length > 0 && (
            <div className="space-y-1.5 pt-1">
              {legacyTools.map((tName, idx) => (
                <ToolInvocationCard key={idx} toolName={tName} />
              ))}
            </div>
          )
        )}
      </div>
    </div>
  );
};
