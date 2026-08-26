import React from 'react';
import { User, Bot, Wrench, Brain } from 'lucide-react';
import type { MessageTurn } from '../../lib/types';
import { formatTokens } from '../../lib/format';
import type { TurnFilterCategory } from './StepFilterPopover';

export interface StepIndexProps {
  turns: MessageTurn[];
  activeStep: number;
  revealedCount?: number;
  onSelectStep: (index: number) => void;
  categoryFilter?: TurnFilterCategory;
  className?: string;
}

export const StepIndex: React.FC<StepIndexProps> = ({
  turns,
  activeStep,
  revealedCount,
  onSelectStep,
  categoryFilter = 'all',
  className = '',
}) => {
  if (turns.length === 0) {
    return null;
  }

  const getStepIcon = (turn: MessageTurn) => {
    if (turn.role === 'user') {
      return <User className="w-3.5 h-3.5 text-blue-400" />;
    }
    if (turn.thinking || turn.reasoning_effort) {
      return <Brain className="w-3.5 h-3.5 text-amber-400" />;
    }
    if ((turn.tool_calls && turn.tool_calls.length > 0) || (turn.tools_invoked && turn.tools_invoked.length > 0)) {
      return <Wrench className="w-3.5 h-3.5 text-emerald-400" />;
    }
    return <Bot className="w-3.5 h-3.5 text-cyan-400" />;
  };

  const getStepLabel = (turn: MessageTurn) => {
    if (turn.role === 'user') {
      return turn.content ? turn.content.slice(0, 32).trim() + (turn.content.length > 32 ? '...' : '') : 'User Prompt';
    }
    if (turn.tool_calls && turn.tool_calls.length > 0) {
      return `Tool: ${turn.tool_calls[0].name}`;
    }
    if (turn.tools_invoked && turn.tools_invoked.length > 0) {
      return `Tool: ${turn.tools_invoked[0]}`;
    }
    if (turn.thinking) {
      return 'Reasoning & Thoughts';
    }
    if (turn.content) {
      return turn.content.slice(0, 32).trim() + (turn.content.length > 32 ? '...' : '');
    }
    return 'Assistant Response';
  };

  return (
    <div className={`rounded-xl bg-[#11141a] border border-white/10 p-3 space-y-2 ${className}`}>
      <div className="flex items-center justify-between px-1 pb-1 border-b border-white/5 text-[11px] font-semibold uppercase tracking-wider text-gray-400">
        <span>Step Index ({turns.length})</span>
        <span className="font-mono text-[10px] text-blue-400 font-normal">
          Active: Step {activeStep + 1}
        </span>
      </div>

      <div className="flex items-center gap-1.5 overflow-x-auto py-1 custom-scrollbar">
        {turns.map((turn, index) => {
          const isActive = index === activeStep;
          const isRevealed = revealedCount === undefined || index < revealedCount;
          const totalTokens = turn.input_tokens + turn.output_tokens;

          return (
            <button
              key={turn.id || index}
              type="button"
              data-test="step-index-btn"
              data-testid="step-index-btn"
              data-step={index + 1}
              onClick={() => onSelectStep(index)}
              className={`flex-shrink-0 flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-mono transition-all text-left ${
                isActive
                  ? 'bg-blue-600/30 text-blue-200 border border-blue-500/60 ring-1 ring-blue-500/40 shadow-sm'
                  : isRevealed
                  ? 'bg-white/5 text-gray-400 hover:text-white hover:bg-white/10 border border-transparent'
                  : 'bg-white/[0.02] text-gray-600 hover:text-gray-400 border border-transparent'
              }`}
              title={`Step ${index + 1}: ${getStepLabel(turn)}`}
            >
              <span className="opacity-80">{getStepIcon(turn)}</span>
              <span className="font-bold">#{index + 1}</span>
              <span className="hidden md:inline text-[11px] text-gray-300 font-sans max-w-[120px] truncate">
                {getStepLabel(turn)}
              </span>
              {totalTokens > 0 && (
                <span className="text-[10px] text-gray-500 ml-1 hidden lg:inline">
                  {formatTokens(totalTokens)}
                </span>
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
};
