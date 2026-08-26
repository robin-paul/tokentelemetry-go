import React, { useState } from 'react';
import ReactMarkdown, { defaultUrlTransform } from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Copy, Check, Code, FileText } from 'lucide-react';
import { HighlightText } from './TurnSearchInput';

interface ResponseBodyProps {
  content: string;
  defaultMode?: 'md' | 'raw';
  searchQuery?: string;
}

export const ResponseBody: React.FC<ResponseBodyProps> = ({
  content,
  defaultMode = 'md',
  searchQuery = '',
}) => {
  const [mode, setMode] = useState<'md' | 'raw'>(defaultMode);
  const [copiedCodeId, setCopiedCodeId] = useState<string | null>(null);

  const handleCopy = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopiedCodeId(id);
    setTimeout(() => setCopiedCodeId(null), 2000);
  };

  if (!content) {
    return <span className="text-gray-500 italic text-xs">No response content</span>;
  }

  return (
    <div className="relative group/body">
      {/* Mode toggle button */}
      <div className="absolute top-0 right-0 z-10 flex items-center gap-1.5">
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            setMode(mode === 'md' ? 'raw' : 'md');
          }}
          className="inline-flex items-center gap-1 text-[10px] font-mono px-2 py-0.5 rounded bg-white/5 hover:bg-white/10 text-gray-400 hover:text-white border border-white/10 transition-colors"
          title={mode === 'md' ? 'Switch to raw text' : 'Switch to parsed Markdown'}
        >
          {mode === 'md' ? (
            <>
              <Code className="w-3 h-3 text-cyan-400" />
              <span>View Raw</span>
            </>
          ) : (
            <>
              <FileText className="w-3 h-3 text-emerald-400" />
              <span>View MD</span>
            </>
          )}
        </button>
      </div>

      {mode === 'md' ? (
        <div className="prose prose-invert prose-sm max-w-none text-gray-200 text-sm leading-relaxed pt-2">
          <ReactMarkdown
            remarkPlugins={[remarkGfm]}
            urlTransform={(url) => {
              if (url.startsWith('file://')) {
                return `/artifacts?path=${encodeURIComponent(decodeURIComponent(url.slice(7)))}`;
              }
              return defaultUrlTransform(url);
            }}
            components={{
              // Custom code block with syntax container and 1-click Copy Code action
              code({ node, className, children, ...props }) {
                const match = /language-(\w+)/.exec(className || '');
                const codeString = String(children).replace(/\n$/, '');
                const isInline = !match && !codeString.includes('\n');
                const blockId = `code-${codeString.slice(0, 16)}-${codeString.length}`;

                if (isInline) {
                  return (
                    <code
                      className="px-1.5 py-0.5 rounded bg-white/10 text-cyan-300 font-mono text-xs font-normal"
                      {...props}
                    >
                      {searchQuery ? (
                        <HighlightText text={codeString} query={searchQuery} />
                      ) : (
                        children
                      )}
                    </code>
                  );
                }

                const lang = match ? match[1] : '';

                return (
                  <div className="relative my-3 rounded-lg overflow-hidden border border-white/10 bg-[#07090d]">
                    <div className="flex items-center justify-between px-3 py-1.5 bg-white/[0.03] border-b border-white/5 text-[11px] font-mono text-gray-400">
                      <span className="text-cyan-400/80 font-semibold">{lang || 'text'}</span>
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleCopy(codeString, blockId);
                        }}
                        className="inline-flex items-center gap-1 text-[10px] px-2 py-0.5 rounded hover:bg-white/10 text-gray-400 hover:text-white transition-colors"
                      >
                        {copiedCodeId === blockId ? (
                          <>
                            <Check className="w-3 h-3 text-emerald-400" />
                            <span className="text-emerald-400">Copied!</span>
                          </>
                        ) : (
                          <>
                            <Copy className="w-3 h-3" />
                            <span>Copy Code</span>
                          </>
                        )}
                      </button>
                    </div>
                    <pre className="p-3.5 overflow-x-auto text-xs font-mono text-gray-200 leading-normal m-0 bg-transparent">
                      <code className={className} {...props}>
                        {searchQuery ? (
                          <HighlightText text={codeString} query={searchQuery} />
                        ) : (
                          children
                        )}
                      </code>
                    </pre>
                  </div>
                );
              },
              table({ children }) {
                return (
                  <div className="overflow-x-auto my-3">
                    <table className="min-w-full divide-y divide-white/10 text-xs text-gray-300 border border-white/10 rounded">
                      {children}
                    </table>
                  </div>
                );
              },
              th({ children }) {
                return (
                  <th className="px-3 py-2 text-left font-semibold text-white bg-white/5 border-b border-white/10">
                    {children}
                  </th>
                );
              },
              td({ children }) {
                return <td className="px-3 py-2 border-b border-white/5">{children}</td>;
              },
              blockquote({ children }) {
                return (
                  <blockquote className="border-l-2 border-cyan-500/50 pl-3 italic text-gray-400 my-2">
                    {children}
                  </blockquote>
                );
              },
              a({ href, children }) {
                return (
                  <a
                    href={href}
                    target="_blank"
                    rel="noreferrer"
                    className="text-cyan-400 hover:text-cyan-300 underline underline-offset-2 transition-colors"
                  >
                    {children}
                  </a>
                );
              },
            }}
          >
            {content}
          </ReactMarkdown>
        </div>
      ) : (
        <pre className="mt-2 text-gray-300 whitespace-pre-wrap text-xs font-mono bg-[#07090d] p-4 rounded-xl border border-white/10 overflow-x-auto leading-relaxed">
          {searchQuery ? <HighlightText text={content} query={searchQuery} /> : content}
        </pre>
      )}
    </div>
  );
};
