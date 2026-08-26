import React, { useState, useMemo } from 'react';
import {
  Cpu,
  Wrench,
  FileCode,
  LayoutGrid,
  Copy,
  Check,
  ChevronRight,
  ChevronLeft,
  Bot,
  Terminal,
  FileText,
  Image as ImageIcon,
  Maximize2,
  ExternalLink,
  Download,
  FolderGit2,
  Clock,
  Coins,
  Zap,
} from 'lucide-react';
import { formatCost, formatTokens, formatDuration, formatDate } from '../../lib/format';
import { getAgentMeta } from '../../lib/agents';
import type { Session, MessageTurn, SessionArtifact, PublishedArtifact } from '../../lib/types';
import type { LightboxArtifact } from './ArtifactLightboxModal';

export type InspectorTab = 'context' | 'tools' | 'artifacts' | 'raw';

export interface ToolHistogramItem {
  name: string;
  count: number;
  totalDurationMs: number;
  avgDurationMs: number;
  errorCount: number;
  firstTurnIndex: number;
}

interface InspectorSidebarProps {
  session: Session;
  activeTurn?: MessageTurn | null;
  activeStep?: number | null;
  isOpen: boolean;
  onToggleOpen: () => void;
  onJumpToTurn: (turnIndex: number) => void;
  onOpenArtifact: (artifact: LightboxArtifact) => void;
}

export const InspectorSidebar: React.FC<InspectorSidebarProps> = ({
  session,
  activeTurn,
  activeStep,
  isOpen,
  onToggleOpen,
  onJumpToTurn,
  onOpenArtifact,
}) => {
  const [activeTab, setActiveTab] = useState<InspectorTab>('context');
  const [rawViewMode, setRawViewMode] = useState<'turn' | 'session'>('turn');
  const [copiedId, setCopiedId] = useState(false);
  const [copiedRaw, setCopiedRaw] = useState(false);

  const turns = session.turns || [];
  const meta = getAgentMeta(session.agent_name);

  // 1. Tool Histogram aggregation
  const toolHistogram = useMemo<ToolHistogramItem[]>(() => {
    const map = new Map<string, { count: number; totalDurationMs: number; errorCount: number; firstTurnIndex: number }>();

    turns.forEach((turn, tIdx) => {
      const turnIndex = turn.turn_index >= 0 ? turn.turn_index : tIdx;

      if (turn.tool_calls && turn.tool_calls.length > 0) {
        turn.tool_calls.forEach((tc, cIdx) => {
          const name = tc.name || 'tool_use';
          const matchedResult = turn.tool_results?.find((r) => r.id === tc.id || r.name === tc.name) || turn.tool_results?.[cIdx];
          const dur = tc.duration_ms ?? matchedResult?.duration_ms ?? 300;
          const isErr = Boolean(matchedResult?.is_error);

          const existing = map.get(name) || { count: 0, totalDurationMs: 0, errorCount: 0, firstTurnIndex: turnIndex };
          existing.count += 1;
          existing.totalDurationMs += dur;
          if (isErr) existing.errorCount += 1;
          map.set(name, existing);
        });
      } else if (turn.tools_invoked && turn.tools_invoked.length > 0) {
        turn.tools_invoked.forEach((name) => {
          const existing = map.get(name) || { count: 0, totalDurationMs: 0, errorCount: 0, firstTurnIndex: turnIndex };
          existing.count += 1;
          existing.totalDurationMs += 300;
          map.set(name, existing);
        });
      }
    });

    const list: ToolHistogramItem[] = [];
    map.forEach((val, name) => {
      list.push({
        name,
        count: val.count,
        totalDurationMs: val.totalDurationMs,
        avgDurationMs: val.count > 0 ? Math.round(val.totalDurationMs / val.count) : 0,
        errorCount: val.errorCount,
        firstTurnIndex: val.firstTurnIndex,
      });
    });

    return list.sort((a, b) => b.count - a.count);
  }, [turns]);

  const maxToolCount = toolHistogram[0]?.count || 1;

  // 2. Artifact aggregation (from session and turns)
  const artifacts = useMemo<SessionArtifact[]>(() => {
    const list = [...(session.artifacts || [])];
    return list;
  }, [session.artifacts]);

  const publishedArtifacts = session.published_artifacts || [];

  const handleCopyId = () => {
    navigator.clipboard.writeText(session.session_id);
    setCopiedId(true);
    setTimeout(() => setCopiedId(false), 2000);
  };

  const handleCopyRaw = () => {
    const dataToCopy = rawViewMode === 'turn' ? activeTurn || turns[0] || session : session;
    navigator.clipboard.writeText(JSON.stringify(dataToCopy, null, 2));
    setCopiedRaw(true);
    setTimeout(() => setCopiedRaw(false), 2000);
  };

  if (!isOpen) {
    return (
      <div className="shrink-0">
        <button
          type="button"
          data-testid="open-inspector-sidebar-button"
          onClick={onToggleOpen}
          className="h-full px-2 py-4 flex flex-col items-center gap-3 bg-[#11141a] hover:bg-white/5 border border-white/10 rounded-xl text-gray-400 hover:text-white transition-colors"
          title="Open Inspector Sidebar"
        >
          <ChevronLeft className="w-4 h-4 text-cyan-400" />
          <span className="text-[10px] font-bold uppercase tracking-[0.2em] [writing-mode:vertical-rl] rotate-180">
            Inspector
          </span>
        </button>
      </div>
    );
  }

  return (
    <aside
      data-testid="inspector-sidebar"
      className="w-80 sm:w-96 shrink-0 bg-[#11141a] border border-white/10 rounded-xl flex flex-col h-full overflow-hidden shadow-2xl"
    >
      {/* Header with tab buttons */}
      <div className="flex items-center justify-between border-b border-white/10 bg-white/[0.02]">
        <div className="flex items-center flex-1">
          <button
            type="button"
            data-testid="inspector-tab-context"
            onClick={() => setActiveTab('context')}
            className={`flex-1 py-3 px-2 flex items-center justify-center gap-1.5 text-[11px] font-bold uppercase tracking-wider transition-colors border-b-2 ${
              activeTab === 'context'
                ? 'border-cyan-400 text-cyan-400 bg-cyan-500/5'
                : 'border-transparent text-gray-400 hover:text-white hover:bg-white/5'
            }`}
          >
            <Cpu className="w-3.5 h-3.5" />
            <span>Context</span>
          </button>

          <button
            type="button"
            data-testid="inspector-tab-tools"
            onClick={() => setActiveTab('tools')}
            className={`flex-1 py-3 px-2 flex items-center justify-center gap-1.5 text-[11px] font-bold uppercase tracking-wider transition-colors border-b-2 ${
              activeTab === 'tools'
                ? 'border-cyan-400 text-cyan-400 bg-cyan-500/5'
                : 'border-transparent text-gray-400 hover:text-white hover:bg-white/5'
            }`}
          >
            <Wrench className="w-3.5 h-3.5" />
            <span>Tools ({toolHistogram.length})</span>
          </button>

          <button
            type="button"
            data-testid="inspector-tab-artifacts"
            onClick={() => setActiveTab('artifacts')}
            className={`flex-1 py-3 px-2 flex items-center justify-center gap-1.5 text-[11px] font-bold uppercase tracking-wider transition-colors border-b-2 ${
              activeTab === 'artifacts'
                ? 'border-cyan-400 text-cyan-400 bg-cyan-500/5'
                : 'border-transparent text-gray-400 hover:text-white hover:bg-white/5'
            }`}
          >
            <LayoutGrid className="w-3.5 h-3.5" />
            <span>Artifacts</span>
          </button>

          <button
            type="button"
            data-testid="inspector-tab-raw"
            onClick={() => setActiveTab('raw')}
            className={`flex-1 py-3 px-2 flex items-center justify-center gap-1.5 text-[11px] font-bold uppercase tracking-wider transition-colors border-b-2 ${
              activeTab === 'raw'
                ? 'border-cyan-400 text-cyan-400 bg-cyan-500/5'
                : 'border-transparent text-gray-400 hover:text-white hover:bg-white/5'
            }`}
          >
            <FileCode className="w-3.5 h-3.5" />
            <span>Raw</span>
          </button>
        </div>

        <button
          type="button"
          data-testid="close-inspector-sidebar-button"
          onClick={onToggleOpen}
          aria-label="Close Inspector"
          className="p-3 text-gray-400 hover:text-white hover:bg-white/5 border-l border-white/10 transition-colors"
        >
          <ChevronRight className="w-4 h-4" />
        </button>
      </div>

      {/* Tab Panels Body */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* TAB 1: CONTEXT */}
        {activeTab === 'context' && (
          <div className="space-y-4 text-xs" data-testid="context-panel">
            {/* Session ID Pill */}
            <div className="space-y-1">
              <div className="text-[10px] font-bold uppercase tracking-wider text-gray-400">
                Session ID
              </div>
              <div className="flex items-center justify-between gap-2 p-2 rounded-lg bg-black/40 border border-white/10 font-mono text-[11px]">
                <span className="text-white truncate" title={session.session_id}>
                  {session.session_id}
                </span>
                <button
                  type="button"
                  onClick={handleCopyId}
                  className="p-1 rounded text-gray-400 hover:text-white hover:bg-white/10 transition-colors"
                  title="Copy session ID"
                >
                  {copiedId ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                </button>
              </div>
            </div>

            {/* Agent and Model info */}
            <div className="grid grid-cols-2 gap-2">
              <div className="p-2.5 rounded-lg bg-white/[0.02] border border-white/5 space-y-1">
                <div className="text-[10px] uppercase tracking-wider text-gray-400">Agent Ecosystem</div>
                <div className="font-semibold" style={{ color: meta.color }}>
                  {meta.label}
                </div>
              </div>

              <div className="p-2.5 rounded-lg bg-white/[0.02] border border-white/5 space-y-1">
                <div className="text-[10px] uppercase tracking-wider text-gray-400">Model Name</div>
                <div className="font-mono text-white font-medium truncate" title={session.model_resolved || session.model_raw}>
                  {session.model_resolved || session.model_raw}
                </div>
              </div>
            </div>

            {/* Project and Workspace Environment */}
            <div className="space-y-2 pt-2 border-t border-white/5">
              <div className="text-[10px] font-bold uppercase tracking-wider text-gray-400">
                Environment & Worktree
              </div>

              <div className="space-y-1.5 bg-white/[0.02] p-3 rounded-lg border border-white/5 font-mono text-[11px]">
                <div className="flex items-center justify-between">
                  <span className="text-gray-400 flex items-center gap-1">
                    <FolderGit2 className="w-3 h-3 text-cyan-400" /> Project:
                  </span>
                  <span className="text-white font-medium">{session.project_name}</span>
                </div>

                {session.git_branch && (
                  <div className="flex items-center justify-between">
                    <span className="text-gray-400">Git Branch:</span>
                    <span className="text-cyan-300 font-semibold">{session.git_branch}</span>
                  </div>
                )}

                {session.machine_id && (
                  <div className="flex items-center justify-between">
                    <span className="text-gray-400">Machine:</span>
                    <span className="text-gray-300 truncate max-w-[160px]">{session.machine_id}</span>
                  </div>
                )}

                {session.hardware_profile && (
                  <div className="flex items-center justify-between">
                    <span className="text-gray-400">Hardware:</span>
                    <span className="text-gray-300">{session.hardware_profile}</span>
                  </div>
                )}
              </div>
            </div>

            {/* Financial and Token Footprint Breakdown */}
            <div className="space-y-2 pt-2 border-t border-white/5">
              <div className="text-[10px] font-bold uppercase tracking-wider text-gray-400 flex items-center gap-1">
                <Coins className="w-3.5 h-3.5 text-emerald-400" /> Spend & Power Attribution
              </div>

              <div className="grid grid-cols-2 gap-2 font-mono text-[11px]">
                <div className="p-2 rounded-lg bg-black/40 border border-white/5">
                  <div className="text-[10px] text-gray-500 uppercase">Gross Cost</div>
                  <div className="text-gray-200 font-semibold">{formatCost(session.gross_cost_usd)}</div>
                </div>
                <div className="p-2 rounded-lg bg-black/40 border border-white/5">
                  <div className="text-[10px] text-gray-500 uppercase">Net Cost (Prompt Cached)</div>
                  <div className="text-emerald-400 font-bold">{formatCost(session.net_cost_usd)}</div>
                </div>
                <div className="p-2 rounded-lg bg-black/40 border border-white/5">
                  <div className="text-[10px] text-gray-500 uppercase">Electricity Cost</div>
                  <div className="text-amber-400">{formatCost(session.electricity_cost_usd)}</div>
                </div>
                <div className="p-2 rounded-lg bg-black/40 border border-white/5">
                  <div className="text-[10px] text-gray-500 uppercase">Cache Read Hit</div>
                  <div className="text-purple-400">{formatTokens(session.cache_read_tokens)}</div>
                </div>
              </div>
            </div>

            {/* Subagents Rollup */}
            {session.subagent_runs && session.subagent_runs.length > 0 && (
              <div className="space-y-2 pt-2 border-t border-white/5">
                <div className="text-[10px] font-bold uppercase tracking-wider text-gray-400 flex items-center gap-1">
                  <Bot className="w-3.5 h-3.5 text-purple-400" /> Spawned Subagents ({session.subagent_runs.length})
                </div>
                <div className="space-y-1.5">
                  {session.subagent_runs.map((sub) => (
                    <div
                      key={sub.id}
                      className="p-2 rounded-lg bg-black/40 border border-white/5 flex items-center justify-between font-mono text-[11px]"
                    >
                      <div>
                        <div className="text-white font-semibold">{sub.agent_type}</div>
                        <div className="text-[10px] text-gray-500">{sub.child_session_id.slice(0, 16)}...</div>
                      </div>
                      <div className="text-right">
                        <div className="text-emerald-400 font-bold">{formatCost(sub.cost_usd)}</div>
                        <div className="text-[10px] text-gray-400">{formatTokens(sub.tokens)} tok</div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {/* TAB 2: TOOLS HISTOGRAM */}
        {activeTab === 'tools' && (
          <div className="space-y-3" data-testid="tools-panel">
            <div className="flex items-center justify-between text-xs text-gray-400">
              <span className="font-semibold uppercase tracking-wider text-[10px]">Tool Summary Histogram</span>
              <span className="font-mono text-[11px]">{toolHistogram.length} unique tools</span>
            </div>

            {toolHistogram.length === 0 ? (
              <div className="p-6 text-center text-xs text-gray-500 border border-white/5 rounded-xl bg-black/40">
                No tool invocations recorded in this session.
              </div>
            ) : (
              <div className="space-y-2">
                {toolHistogram.map((item) => {
                  const percent = Math.round((item.count / maxToolCount) * 100);

                  return (
                    <button
                      key={item.name}
                      type="button"
                      onClick={() => onJumpToTurn(item.firstTurnIndex)}
                      className="w-full text-left p-3 rounded-xl bg-white/[0.02] hover:bg-white/[0.06] border border-white/5 hover:border-cyan-500/30 transition-all group"
                      title={`Jump to first invocation of ${item.name} (Turn #${item.firstTurnIndex + 1})`}
                    >
                      <div className="flex items-center justify-between mb-1.5">
                        <span className="text-xs font-mono font-bold text-white group-hover:text-cyan-400 transition-colors truncate">
                          {item.name}
                        </span>
                        <div className="flex items-center gap-2 font-mono text-xs">
                          {item.errorCount > 0 && (
                            <span className="text-rose-400 font-semibold text-[10px]">
                              {item.errorCount} err
                            </span>
                          )}
                          <span className="text-cyan-400 font-bold tabular-nums">×{item.count}</span>
                        </div>
                      </div>

                      {/* Histogram Bar */}
                      <div className="h-1.5 bg-black/60 rounded-full overflow-hidden mb-1.5 border border-white/5">
                        <div
                          className="h-full bg-gradient-to-r from-cyan-500 to-blue-500 rounded-full transition-all group-hover:from-cyan-400 group-hover:to-blue-400"
                          style={{ width: `${percent}%` }}
                        />
                      </div>

                      <div className="flex items-center justify-between text-[10px] font-mono text-gray-400">
                        <span>avg {item.avgDurationMs >= 1000 ? `${(item.avgDurationMs / 1000).toFixed(1)}s` : `${item.avgDurationMs}ms`}</span>
                        <span className="text-gray-500 group-hover:text-cyan-300 transition-colors">Jump to turn ▸</span>
                      </div>
                    </button>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {/* TAB 3: ARTIFACTS GALLERY */}
        {activeTab === 'artifacts' && (
          <div className="space-y-4" data-testid="artifacts-panel">
            {/* Published Pages */}
            {publishedArtifacts.length > 0 && (
              <div className="space-y-2">
                <div className="text-[10px] font-bold uppercase tracking-wider text-gray-400">
                  Published Pages & Apps
                </div>
                <div className="space-y-2">
                  {publishedArtifacts.map((pub, idx) => (
                    <a
                      key={idx}
                      href={pub.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="block p-3 rounded-xl bg-white/[0.02] hover:bg-white/[0.06] border border-white/5 hover:border-cyan-500/30 transition-colors group"
                    >
                      <div className="flex items-center justify-between gap-2">
                        <div className="font-semibold text-xs text-white group-hover:text-cyan-400 truncate">
                          {pub.title || pub.file_name || 'Published App'}
                        </div>
                        <ExternalLink className="w-3.5 h-3.5 text-gray-500 group-hover:text-cyan-400 shrink-0" />
                      </div>
                      {pub.description && (
                        <p className="text-[10px] text-gray-400 mt-1 line-clamp-2">{pub.description}</p>
                      )}
                    </a>
                  ))}
                </div>
              </div>
            )}

            {/* Session Artifacts list */}
            <div className="space-y-2">
              <div className="text-[10px] font-bold uppercase tracking-wider text-gray-400">
                Generated Documents, Plans & Media ({artifacts.length})
              </div>

              {artifacts.length === 0 ? (
                <div className="p-6 text-center text-xs text-gray-500 border border-white/5 rounded-xl bg-black/40">
                  No artifacts generated in this session.
                </div>
              ) : (
                <div className="space-y-2">
                  {artifacts.map((art, idx) => {
                    const isImg = art.type === 'image' || /\.(png|jpe?g|webp|svg)$/i.test(art.name);

                    return (
                      <div
                        key={idx}
                        className="p-3 rounded-xl bg-white/[0.02] border border-white/5 hover:border-white/10 transition-colors space-y-2"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <div className="flex items-center gap-2 min-w-0">
                            {isImg ? (
                              <ImageIcon className="w-4 h-4 text-emerald-400 shrink-0" />
                            ) : art.type === 'terminal' ? (
                              <Terminal className="w-4 h-4 text-purple-400 shrink-0" />
                            ) : (
                              <FileText className="w-4 h-4 text-cyan-400 shrink-0" />
                            )}
                            <span className="text-xs font-mono font-medium text-white truncate" title={art.name}>
                              {art.name}
                            </span>
                          </div>

                          <div className="flex items-center gap-1.5 shrink-0">
                            <button
                              type="button"
                              onClick={() => onOpenArtifact(art)}
                              className="p-1 rounded bg-white/5 hover:bg-white/10 text-gray-400 hover:text-white transition-colors"
                              title="Expand in Lightbox"
                            >
                              <Maximize2 className="w-3.5 h-3.5" />
                            </button>
                            <a
                              href={`/artifacts?path=${encodeURIComponent(art.path)}`}
                              download={art.name}
                              className="p-1 rounded bg-white/5 hover:bg-white/10 text-gray-400 hover:text-white transition-colors"
                              title="Download artifact"
                            >
                              <Download className="w-3.5 h-3.5" />
                            </a>
                          </div>
                        </div>

                        {/* Thumbnail preview if image */}
                        {isImg && (
                          <div
                            onClick={() => onOpenArtifact(art)}
                            className="cursor-zoom-in rounded-lg overflow-hidden border border-white/10 bg-black/40 max-h-32 flex items-center justify-center"
                          >
                            <img
                              src={`/artifacts?path=${encodeURIComponent(art.path)}`}
                              alt={art.name}
                              className="max-h-32 object-contain hover:scale-105 transition-transform"
                            />
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        )}

        {/* TAB 4: RAW JSON */}
        {activeTab === 'raw' && (
          <div className="space-y-3" data-testid="raw-panel">
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-1 bg-black/40 p-0.5 rounded-lg border border-white/10 text-[10px] font-mono">
                <button
                  type="button"
                  onClick={() => setRawViewMode('turn')}
                  className={`px-2.5 py-1 rounded-md transition-colors ${
                    rawViewMode === 'turn'
                      ? 'bg-cyan-500/20 text-cyan-400 font-bold border border-cyan-500/30'
                      : 'text-gray-400 hover:text-white'
                  }`}
                >
                  Active Turn
                </button>
                <button
                  type="button"
                  onClick={() => setRawViewMode('session')}
                  className={`px-2.5 py-1 rounded-md transition-colors ${
                    rawViewMode === 'session'
                      ? 'bg-cyan-500/20 text-cyan-400 font-bold border border-cyan-500/30'
                      : 'text-gray-400 hover:text-white'
                  }`}
                >
                  Full Session
                </button>
              </div>

              <button
                type="button"
                data-testid="copy-raw-json-button"
                onClick={handleCopyRaw}
                className="inline-flex items-center gap-1 text-[10px] font-mono px-2.5 py-1 rounded-lg bg-white/5 hover:bg-white/10 border border-white/10 text-gray-300 hover:text-white transition-colors"
              >
                {copiedRaw ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                <span>{copiedRaw ? 'Copied' : 'Copy'}</span>
              </button>
            </div>

            <pre className="p-3 rounded-xl bg-black/60 border border-white/5 text-cyan-300 text-[11px] font-mono max-h-[calc(100vh-280px)] overflow-auto leading-relaxed">
              {JSON.stringify(
                rawViewMode === 'turn' ? activeTurn || turns[activeStep ?? 0] || session : session,
                null,
                2
              )}
            </pre>
          </div>
        )}
      </div>
    </aside>
  );
};
