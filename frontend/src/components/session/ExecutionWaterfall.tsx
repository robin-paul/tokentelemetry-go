import React, { useMemo, useState, useRef, useEffect } from 'react';
import {
  ListMusic,
  ChevronDown,
  ChevronUp,
  Wrench,
  AlertTriangle,
  Clock,
  Sparkles,
} from 'lucide-react';
import type { MessageTurn } from '../../lib/types';

export interface WaterfallToolItem {
  id: string;
  name: string;
  turnIndex: number;
  stepNumber: number;
  startTime: number;
  endTime: number;
  durationMs: number;
  isError: boolean;
  argsPreview?: string;
}

interface ExecutionWaterfallProps {
  turns: MessageTurn[];
  activeStep: number | null;
  onSelectStep: (stepIndex: number) => void;
}

export const ExecutionWaterfall: React.FC<ExecutionWaterfallProps> = ({
  turns,
  activeStep,
  onSelectStep,
}) => {
  const [isOpen, setIsOpen] = useState(true);
  const rowRefs = useRef<Record<number, HTMLDivElement | null>>({});

  // Extract and normalize all tool invocations across turns
  const toolsData = useMemo<WaterfallToolItem[]>(() => {
    const items: WaterfallToolItem[] = [];

    turns.forEach((turn, tIdx) => {
      const turnIndex = tIdx;
      const baseTime = turn.timestamp ? new Date(turn.timestamp).getTime() : 1000 + turnIndex * 1000;
      const turnBase = isNaN(baseTime) ? 1000 + turnIndex * 1000 : baseTime;

      // 1. Structured tool_calls
      if (turn.tool_calls && turn.tool_calls.length > 0) {
        turn.tool_calls.forEach((tc, cIdx) => {
          const matchedResult = turn.tool_results?.find(
            (r) => r.id === tc.id || r.name === tc.name
          ) || turn.tool_results?.[cIdx];

          const durationMs =
            tc.duration_ms ??
            matchedResult?.duration_ms ??
            (matchedResult ? 450 : 250);

          const startTime = turnBase + cIdx * 300;
          const endTime = startTime + durationMs;
          const isError = Boolean(matchedResult?.is_error);

          let argsPreview = '';
          if (tc.args && typeof tc.args === 'object' && !Array.isArray(tc.args)) {
            argsPreview = Object.keys(tc.args).slice(0, 2).join(', ');
          }

          items.push({
            id: tc.id || `tool-${turnIndex}-${cIdx}`,
            name: tc.name || 'tool_use',
            turnIndex,
            stepNumber: turnIndex + 1,
            startTime,
            endTime,
            durationMs,
            isError,
            argsPreview,
          });
        });
      } else if (turn.tools_invoked && turn.tools_invoked.length > 0) {
        // 2. Legacy tools_invoked strings
        turn.tools_invoked.forEach((tName, cIdx) => {
          const startTime = turnBase + cIdx * 300;
          const durationMs = 300;
          items.push({
            id: `legacy-${turnIndex}-${cIdx}`,
            name: tName,
            turnIndex,
            stepNumber: turnIndex + 1,
            startTime,
            endTime: startTime + durationMs,
            durationMs,
            isError: false,
          });
        });
      } else if (turn.tools_invoked_json) {
        try {
          const parsed: string[] = JSON.parse(turn.tools_invoked_json);
          if (Array.isArray(parsed)) {
            parsed.forEach((tName, cIdx) => {
              const startTime = turnBase + cIdx * 300;
              const durationMs = 300;
              items.push({
                id: `json-${turnIndex}-${cIdx}`,
                name: tName,
                turnIndex,
                stepNumber: turnIndex + 1,
                startTime,
                endTime: startTime + durationMs,
                durationMs,
                isError: false,
              });
            });
          }
        } catch {}
      }
    });

    return items;
  }, [turns]);

  // Compute total timeline range for Gantt visualization
  const { minStart, totalSpan } = useMemo(() => {
    if (toolsData.length === 0) return { minStart: 0, totalSpan: 1 };
    const minStart = toolsData[0].startTime;
    const maxEnd = toolsData.reduce((acc, curr) => Math.max(acc, curr.endTime), toolsData[0].endTime);
    const span = Math.max(1, maxEnd - minStart);
    return { minStart, totalSpan: span };
  }, [toolsData]);

  // Auto-scroll active tool into view inside waterfall
  useEffect(() => {
    if (activeStep !== null && rowRefs.current[activeStep]) {
      rowRefs.current[activeStep]?.scrollIntoView({
        behavior: 'smooth',
        block: 'nearest',
      });
    }
  }, [activeStep]);

  if (toolsData.length === 0) {
    return null;
  }

  const totalToolDurationMs = toolsData.reduce((sum, t) => sum + t.durationMs, 0);

  return (
    <div
      data-testid="execution-waterfall"
      data-test="execution-waterfall"
      className="bg-[#11141a] border border-white/10 rounded-xl overflow-hidden shadow-xl"
    >
      {/* Header bar */}
      <div className="px-4 py-3 bg-white/[0.02] border-b border-white/5 flex items-center justify-between">
        <button
          type="button"
          onClick={() => setIsOpen(!isOpen)}
          className="flex items-center gap-2 text-left group transition-colors"
        >
          <div className="p-1 rounded-md bg-cyan-500/10 text-cyan-400 border border-cyan-500/20 group-hover:bg-cyan-500/20">
            <ListMusic className="w-4 h-4" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="text-xs font-bold text-white uppercase tracking-wider">
                Tool Execution Waterfall
              </span>
              <span className="text-[10px] font-mono text-cyan-400 bg-cyan-500/10 px-2 py-0.5 rounded border border-cyan-500/20">
                {toolsData.length} {toolsData.length === 1 ? 'call' : 'calls'}
              </span>
            </div>
          </div>
          <span className="text-gray-400 group-hover:text-white transition-colors ml-1">
            {isOpen ? <ChevronDown className="w-4 h-4" /> : <ChevronUp className="w-4 h-4" />}
          </span>
        </button>

        <div className="flex items-center gap-3 text-xs">
          <span className="text-[11px] font-mono text-gray-400 hidden sm:inline-block">
            Total tool time: <span className="text-amber-400 font-semibold">{totalToolDurationMs >= 1000 ? `${(totalToolDurationMs / 1000).toFixed(2)}s` : `${totalToolDurationMs.toFixed(0)}ms`}</span>
          </span>
          <button
            type="button"
            onClick={() => setIsOpen(!isOpen)}
            className="text-[10px] font-mono uppercase tracking-wider px-2.5 py-1 rounded bg-white/5 hover:bg-white/10 text-gray-300 transition-colors border border-white/5"
          >
            {isOpen ? 'Collapse' : 'Expand'}
          </button>
        </div>
      </div>

      {/* Waterfall Content */}
      {isOpen && (
        <div className="p-4 space-y-2 max-h-60 overflow-y-auto">
          {toolsData.map((item, idx) => {
            const leftPercent = ((item.startTime - minStart) / totalSpan) * 92;
            const widthPercent = Math.max(3, (item.durationMs / totalSpan) * 92);
            const isActive = activeStep === item.turnIndex;

            return (
              <div
                key={item.id || idx}
                ref={(el) => {
                  rowRefs.current[item.turnIndex] = el;
                }}
                onClick={() => onSelectStep(item.turnIndex)}
                className={`flex items-center gap-3 p-2 rounded-lg cursor-pointer transition-all ${
                  isActive
                    ? 'bg-cyan-500/10 ring-1 ring-cyan-500/40 border-l-2 border-cyan-400'
                    : 'bg-white/[0.02] hover:bg-white/[0.05] border border-transparent'
                }`}
              >
                {/* Tool label column */}
                <div className="w-44 shrink-0 flex items-center justify-between pr-2">
                  <div className="flex items-center gap-1.5 min-w-0">
                    <Wrench className={`w-3.5 h-3.5 shrink-0 ${item.isError ? 'text-rose-400' : 'text-cyan-400'}`} />
                    <span
                      className={`text-xs font-mono font-medium truncate ${
                        isActive ? 'text-cyan-300' : 'text-gray-200'
                      }`}
                      title={item.name}
                    >
                      {item.name}
                    </span>
                  </div>
                  <span className="text-[10px] font-mono text-gray-500 shrink-0 ml-1">
                    #{item.stepNumber}
                  </span>
                </div>

                {/* Duration chip */}
                <div className="w-16 shrink-0 text-right">
                  <span
                    className={`text-[10px] font-mono px-1.5 py-0.5 rounded border ${
                      item.isError
                        ? 'bg-rose-500/10 text-rose-400 border-rose-500/20'
                        : 'bg-amber-500/10 text-amber-400 border-amber-500/20'
                    }`}
                  >
                    {item.durationMs >= 1000
                      ? `${(item.durationMs / 1000).toFixed(1)}s`
                      : `${item.durationMs.toFixed(0)}ms`}
                  </span>
                </div>

                {/* Gantt Timeline Bar Lane */}
                <div className="flex-1 bg-black/40 h-4 rounded-md relative border border-white/5 overflow-hidden">
                  {/* Subtle Grid ticks */}
                  <div className="absolute inset-0 grid grid-cols-4 pointer-events-none opacity-10 divide-x divide-white" />

                  {/* Gantt Span Bar */}
                  <div
                    className={`absolute h-full rounded transition-all flex items-center px-1 text-[9px] font-mono font-bold uppercase tracking-wider text-white/90 ${
                      item.isError
                        ? 'bg-gradient-to-r from-rose-600 to-rose-400 ring-1 ring-rose-400/50'
                        : isActive
                        ? 'bg-gradient-to-r from-cyan-500 to-blue-500 ring-2 ring-cyan-400/60 shadow-[0_0_8px_rgba(6,182,212,0.5)]'
                        : 'bg-gradient-to-r from-cyan-600/60 to-blue-600/80 hover:from-cyan-500 hover:to-blue-500'
                    }`}
                    style={{
                      left: `${leftPercent}%`,
                      width: `${widthPercent}%`,
                    }}
                  >
                    {widthPercent > 15 && (
                      <span className="truncate drop-shadow-sm">{item.name}</span>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};
