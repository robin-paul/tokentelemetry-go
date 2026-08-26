import React, { useState } from 'react';
import {
  Wrench,
  Terminal,
  Copy,
  Check,
  ChevronDown,
  ChevronUp,
  Code,
  AlertTriangle,
  Clock,
  Maximize2,
  Minimize2,
} from 'lucide-react';
import type { ToolCall, ToolResult } from '../../lib/types';
import { HighlightText } from './TurnSearchInput';

interface ToolInvocationCardProps {
  toolCall?: ToolCall;
  toolResult?: ToolResult;
  toolName?: string;
  searchQuery?: string;
}

export const ToolInvocationCard: React.FC<ToolInvocationCardProps> = ({
  toolCall,
  toolResult,
  toolName: propName,
  searchQuery = '',
}) => {
  const [copiedArgs, setCopiedArgs] = useState(false);
  const [copiedOutput, setCopiedOutput] = useState(false);
  const [openArgs, setOpenArgs] = useState(true);
  const [expandedOutput, setExpandedOutput] = useState(false);

  const name = toolCall?.name || toolResult?.name || propName || 'tool_use';
  const isError = Boolean(toolResult?.is_error);
  const durationMs = toolCall?.duration_ms ?? toolResult?.duration_ms;

  const handleCopyArgs = (e: React.MouseEvent) => {
    e.stopPropagation();
    const argsStr = toolCall?.args_json || JSON.stringify(toolCall?.args || {}, null, 2);
    navigator.clipboard.writeText(argsStr);
    setCopiedArgs(true);
    setTimeout(() => setCopiedArgs(false), 2000);
  };

  const handleCopyOutput = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (toolResult?.content === undefined || toolResult?.content === null) return;
    const outStr =
      typeof toolResult.content === 'string'
        ? toolResult.content
        : JSON.stringify(toolResult.content, null, 2);
    navigator.clipboard.writeText(outStr);
    setCopiedOutput(true);
    setTimeout(() => setCopiedOutput(false), 2000);
  };

  const renderResultContent = (content: unknown) => {
    if (content === undefined || content === null) {
      return null;
    }
    const text = typeof content === 'string' ? content : JSON.stringify(content, null, 2);

    // If text contains git unified diff lines (+/-)
    const isDiff = text.includes('\n+') || text.includes('\n-') || text.startsWith('+') || text.startsWith('-');
    if (isDiff) {
      const lines = text.split('\n');
      return (
        <div className="space-y-0.5 font-mono text-[11px] leading-relaxed">
          {lines.map((line, idx) => {
            let lineClass = 'text-gray-300';
            if (line.startsWith('+')) {
              lineClass = 'text-emerald-400 bg-emerald-500/10 px-1.5 py-0.5 rounded-sm border-l-2 border-emerald-500';
            } else if (line.startsWith('-')) {
              lineClass = 'text-rose-400 bg-rose-500/10 px-1.5 py-0.5 rounded-sm border-l-2 border-rose-500';
            } else if (line.startsWith('@@')) {
              lineClass = 'text-cyan-400 font-semibold bg-cyan-500/10 px-1.5 py-0.5 rounded-sm';
            }
            return (
              <div key={idx} className={lineClass}>
                {searchQuery ? <HighlightText text={line} query={searchQuery} /> : line}
              </div>
            );
          })}
        </div>
      );
    }

    return searchQuery ? <HighlightText text={text} query={searchQuery} /> : <span>{text}</span>;
  };

  const outputString =
    toolResult?.content !== undefined && toolResult?.content !== null
      ? typeof toolResult.content === 'string'
        ? toolResult.content
        : JSON.stringify(toolResult.content, null, 2)
      : '';
  const outputLineCount = outputString ? outputString.split('\n').length : 0;
  const isLongOutput = outputLineCount > 12 || outputString.length > 800;

  return (
    <div className="space-y-2 my-2.5" data-testid="tool-invocation-card">
      {/* Tool Call Card */}
      <div className="bg-white/[0.02] border border-white/10 rounded-xl p-3.5 hover:border-white/20 transition-all">
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded bg-cyan-500/10 text-cyan-400 border border-cyan-500/20 text-xs font-mono font-semibold">
              <Code className="w-3.5 h-3.5" />
              {searchQuery ? <HighlightText text={name} query={searchQuery} /> : name}
            </span>
            {toolCall?.id && (
              <span className="text-[10px] font-mono text-gray-500">id: {toolCall.id}</span>
            )}
            {durationMs !== undefined && (
              <span className="inline-flex items-center gap-1 text-[10px] font-mono text-amber-400 bg-amber-500/10 px-1.5 py-0.5 rounded border border-amber-500/20">
                <Clock className="w-3 h-3" />
                {durationMs >= 1000 ? `${(durationMs / 1000).toFixed(2)}s` : `${durationMs.toFixed(0)}ms`}
              </span>
            )}
            {isError && (
              <span className="inline-flex items-center gap-1 text-[10px] font-mono text-rose-400 bg-rose-500/10 px-2 py-0.5 rounded border border-rose-500/30 font-semibold">
                <AlertTriangle className="w-3 h-3" /> Error
              </span>
            )}
          </div>

          {(toolCall?.args || toolCall?.args_json) && (
            <button
              type="button"
              onClick={() => setOpenArgs(!openArgs)}
              className="inline-flex items-center gap-1 text-[11px] font-mono text-gray-400 hover:text-white px-2 py-0.5 rounded hover:bg-white/5 transition-colors"
            >
              <span>arguments</span>
              {openArgs ? <ChevronUp className="w-3.5 h-3.5" /> : <ChevronDown className="w-3.5 h-3.5" />}
            </button>
          )}
        </div>

        {/* Expandable Arguments */}
        {openArgs && (toolCall?.args || toolCall?.args_json) && (
          <div className="mt-2.5 relative">
            <div className="absolute top-2 right-2 z-10">
              <button
                type="button"
                onClick={handleCopyArgs}
                className="inline-flex items-center gap-1 text-[10px] font-mono px-2 py-0.5 rounded bg-white/10 hover:bg-white/20 text-gray-300 transition-colors"
              >
                {copiedArgs ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                <span>{copiedArgs ? 'Copied' : 'Copy'}</span>
              </button>
            </div>
            <pre className="bg-[#07090d] text-cyan-300 p-3 rounded-lg text-xs font-mono overflow-x-auto border border-white/5 leading-relaxed">
              {toolCall.args_json || JSON.stringify(toolCall.args, null, 2)}
            </pre>
          </div>
        )}
      </div>

      {/* Tool Result Card */}
      {toolResult && toolResult.content !== undefined && (
        <div
          className={`border rounded-xl p-3.5 space-y-2 ml-3 transition-all ${
            isError
              ? 'bg-rose-950/20 border-rose-500/30 text-rose-300'
              : 'bg-[#07090d] border-white/10 hover:border-emerald-500/30'
          }`}
        >
          <div className="flex items-center justify-between text-xs">
            <div className="flex items-center gap-2">
              <div
                className={`flex items-center gap-1.5 font-bold uppercase tracking-wider text-[11px] ${
                  isError ? 'text-rose-400' : 'text-emerald-400'
                }`}
              >
                <Terminal className="w-3.5 h-3.5" />
                <span>{isError ? 'Tool Error Output' : 'Tool Output'}</span>
              </div>
              <span className="text-[10px] font-mono text-gray-500">
                {outputString.length.toLocaleString()} chars
                {outputLineCount > 1 && ` · ${outputLineCount} lines`}
              </span>
            </div>

            <div className="flex items-center gap-2">
              {isLongOutput && (
                <button
                  type="button"
                  onClick={() => setExpandedOutput(!expandedOutput)}
                  className="inline-flex items-center gap-1 text-[10px] font-mono text-gray-400 hover:text-white px-2 py-0.5 rounded bg-white/5 hover:bg-white/10 transition-colors"
                >
                  {expandedOutput ? (
                    <>
                      <Minimize2 className="w-3 h-3" />
                      <span>Collapse</span>
                    </>
                  ) : (
                    <>
                      <Maximize2 className="w-3 h-3" />
                      <span>Expand</span>
                    </>
                  )}
                </button>
              )}
              <button
                type="button"
                onClick={handleCopyOutput}
                className="inline-flex items-center gap-1 text-[10px] font-mono px-2 py-0.5 rounded bg-white/10 hover:bg-white/20 text-gray-300 transition-colors"
              >
                {copiedOutput ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                <span>{copiedOutput ? 'Copied' : 'Copy'}</span>
              </button>
            </div>
          </div>

          <div
            className={`p-3 rounded-lg text-xs font-mono overflow-x-auto leading-relaxed border ${
              isError
                ? 'bg-black/60 text-rose-300 border-rose-500/20'
                : 'bg-black/40 text-emerald-400 border-white/5'
            } ${expandedOutput ? 'max-h-none' : 'max-h-72 overflow-y-auto'}`}
          >
            {renderResultContent(toolResult.content)}
          </div>
        </div>
      )}
    </div>
  );
};
