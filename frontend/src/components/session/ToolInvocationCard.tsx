import React, { useState } from 'react';
import { Wrench, Terminal, Copy, Check, ChevronDown, ChevronUp, Code } from 'lucide-react';
import type { ToolCall, ToolResult } from '../../lib/types';

interface ToolInvocationCardProps {
  toolCall?: ToolCall;
  toolResult?: ToolResult;
  toolName?: string;
}

export const ToolInvocationCard: React.FC<ToolInvocationCardProps> = ({
  toolCall,
  toolResult,
  toolName: propName,
}) => {
  const [copied, setCopied] = useState(false);
  const [openArgs, setOpenArgs] = useState(false);
  const name = toolCall?.name || toolResult?.name || propName || 'tool_use';

  const handleCopyArgs = (e: React.MouseEvent) => {
    e.stopPropagation();
    const argsStr = toolCall?.args_json || JSON.stringify(toolCall?.args || {}, null, 2);
    navigator.clipboard.writeText(argsStr);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const renderResultContent = (content: unknown) => {
    if (content === undefined || content === null) {
      return null;
    }
    const text = typeof content === 'string' ? content : JSON.stringify(content, null, 2);

    // If text contains git unified diff lines (+/-)
    const isDiff = text.includes('\n+') || text.includes('\n-');
    if (isDiff) {
      const lines = text.split('\n');
      return (
        <div className="space-y-0.5">
          {lines.map((line, idx) => {
            let lineClass = 'text-gray-300';
            if (line.startsWith('+')) {
              lineClass = 'text-emerald-400 bg-emerald-500/10 px-1 rounded-sm';
            } else if (line.startsWith('-')) {
              lineClass = 'text-rose-400 bg-rose-500/10 px-1 rounded-sm';
            } else if (line.startsWith('@@')) {
              lineClass = 'text-cyan-400 font-semibold';
            }
            return (
              <div key={idx} className={lineClass}>
                {line}
              </div>
            );
          })}
        </div>
      );
    }

    return <span>{text}</span>;
  };

  return (
    <div className="space-y-2 my-2.5">
      {/* Tool Call Card */}
      <div className="bg-white/[0.02] border border-white/10 rounded-xl p-3.5 hover:border-white/20 transition-all">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded bg-cyan-500/10 text-cyan-400 border border-cyan-500/20 text-xs font-mono font-semibold">
              <Code className="w-3.5 h-3.5" />
              {name}
            </span>
            {toolCall?.id && (
              <span className="text-[10px] font-mono text-gray-500">id: {toolCall.id}</span>
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
            <div className="absolute top-2 right-2">
              <button
                type="button"
                onClick={handleCopyArgs}
                className="inline-flex items-center gap-1 text-[10px] font-mono px-2 py-0.5 rounded bg-white/10 hover:bg-white/20 text-gray-300 transition-colors"
              >
                {copied ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                <span>{copied ? 'Copied' : 'Copy'}</span>
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
        <div className="bg-[#07090d] border border-white/10 rounded-xl p-3.5 space-y-2 ml-3">
          <div className="flex items-center justify-between text-xs">
            <div className="flex items-center gap-1.5 font-bold uppercase tracking-wider text-emerald-400 text-[11px]">
              <Terminal className="w-3.5 h-3.5" /> Tool Output
            </div>
            <span className="text-[10px] font-mono text-gray-500">
              {typeof toolResult.content === 'string'
                ? `${toolResult.content.length.toLocaleString()} chars`
                : 'JSON output'}
            </span>
          </div>
          <div className="bg-black/40 text-emerald-400 p-3 rounded-lg text-xs font-mono overflow-x-auto max-h-72 overflow-y-auto leading-relaxed border border-white/5">
            {renderResultContent(toolResult.content)}
          </div>
        </div>
      )}
    </div>
  );
};
