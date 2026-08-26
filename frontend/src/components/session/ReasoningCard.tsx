import React, { useState } from 'react';
import { Brain, ChevronDown, ChevronUp, Lock } from 'lucide-react';

interface ReasoningCardProps {
  thinking?: string;
  reasoningEffort?: string;
  defaultExpanded?: boolean;
}

export const ReasoningCard: React.FC<ReasoningCardProps> = ({
  thinking = '',
  reasoningEffort,
  defaultExpanded = true,
}) => {
  const [expanded, setExpanded] = useState(defaultExpanded);

  if (!thinking && !reasoningEffort) {
    return null;
  }

  const isSealed = thinking.includes('sig:') || (!thinking && !!reasoningEffort);

  return (
    <div className="bg-amber-500/5 border border-amber-500/20 rounded-xl border-l-4 border-l-amber-500/60 overflow-hidden transition-all my-2">
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center justify-between p-3.5 hover:bg-amber-500/10 transition-colors text-left"
      >
        <div className="flex items-center gap-2">
          <Brain className="w-4 h-4 text-amber-400" />
          <span className="text-xs font-bold text-amber-300 uppercase tracking-wider">
            Reasoning & Thoughts
          </span>
          {reasoningEffort && (
            <span className="px-2 py-0.5 rounded text-[10px] font-mono text-amber-300 bg-amber-500/10 border border-amber-500/30">
              effort: {reasoningEffort}
            </span>
          )}
          {isSealed && (
            <span className="inline-flex items-center gap-1 text-[10px] font-mono text-amber-300 bg-amber-500/20 px-2 py-0.5 rounded border border-amber-500/30">
              <Lock className="w-3 h-3" /> sealed
            </span>
          )}
        </div>

        <div className="flex items-center gap-2 text-xs text-amber-400/70 font-mono">
          {thinking && <span>{thinking.length.toLocaleString()} chars</span>}
          {expanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
        </div>
      </button>

      {expanded && (
        <div className="px-4 pb-4 pt-1 border-t border-amber-500/10 text-xs font-mono text-amber-200/90 italic leading-relaxed whitespace-pre-wrap max-h-96 overflow-y-auto">
          {isSealed && !thinking ? (
            <div className="text-gray-400 italic">
              Extended thinking is sealed by the provider — session log contains effort telemetry only.
            </div>
          ) : (
            thinking
          )}
        </div>
      )}
    </div>
  );
};
