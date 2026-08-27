import React, { useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import {
  X,
  Download,
  ZoomIn,
  ZoomOut,
  RotateCcw,
  FileText,
  Image as ImageIcon,
  Video,
  Terminal,
  Copy,
  Check,
} from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { apiFetch } from '../../lib/api';
import type { SessionArtifact, PublishedArtifact } from '../../lib/types';

export type LightboxArtifact =
  | SessionArtifact
  | (PublishedArtifact & { content?: string; name?: string; type?: string });

interface ArtifactLightboxModalProps {
  artifact: LightboxArtifact;
  onClose: () => void;
}

export const ArtifactLightboxModal: React.FC<ArtifactLightboxModalProps> = ({
  artifact,
  onClose,
}) => {
  if (typeof document === 'undefined') return null;
  const [zoomScale, setZoomScale] = useState(1);
  const [fetchedContent, setFetchedContent] = useState<string | null>(
    'content' in artifact && typeof artifact.content === 'string' ? artifact.content : null
  );
  const [loading, setLoading] = useState(false);
  const [copied, setCopied] = useState(false);

  const name =
    ('name' in artifact && artifact.name) ||
    ('title' in artifact && artifact.title) ||
    ('file_name' in artifact && artifact.file_name) ||
    'Artifact Preview';

  const path = ('path' in artifact && artifact.path) || '';
  const url = ('url' in artifact && artifact.url) || (path ? `/artifacts?path=${encodeURIComponent(path)}` : '');

  // Detect artifact type
  const isImage =
    artifact.type === 'image' ||
    /\.(png|jpe?g|webp|gif|svg|bmp|ico)$/i.test(name) ||
    /\.(png|jpe?g|webp|gif|svg|bmp|ico)$/i.test(path);

  const isVideo =
    artifact.type === 'video' ||
    /\.(mp4|webm|mov|ogg)$/i.test(name) ||
    /\.(mp4|webm|mov|ogg)$/i.test(path);

  const isTerminalOrDiff =
    artifact.type === 'terminal' ||
    /\.(diff|patch|log)$/i.test(name) ||
    /\.(diff|patch|log)$/i.test(path);

  const isMarkdown =
    artifact.type === 'document' ||
    /\.(md|markdown|txt|json|sql|ya?ml|ts|tsx|js|go|py|sh)$/i.test(name) ||
    /\.(md|markdown|txt|json|sql|ya?ml|ts|tsx|js|go|py|sh)$/i.test(path) ||
    (!isImage && !isVideo && !isTerminalOrDiff);

  // Fetch document content if path provided and content not available
  useEffect(() => {
    if (path && (isMarkdown || isTerminalOrDiff) && fetchedContent === null) {
      setLoading(true);
      apiFetch<string>(`/artifacts?path=${encodeURIComponent(path)}`)
        .then((res) => {
          if (typeof res === 'string') {
            setFetchedContent(res);
          } else {
            setFetchedContent(JSON.stringify(res, null, 2));
          }
        })
        .catch((e) => {
          console.error('Failed to fetch artifact content', e);
          setFetchedContent(`Error loading artifact: ${e.message || String(e)}`);
        })
        .finally(() => setLoading(false));
    }
  }, [path, isMarkdown, isTerminalOrDiff, fetchedContent]);

  // Keyboard navigation & zoom shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      } else if (isImage) {
        if (e.key === '=' || e.key === '+') {
          setZoomScale((z) => Math.min(4, z + 0.25));
        } else if (e.key === '-') {
          setZoomScale((z) => Math.max(0.5, z - 0.25));
        } else if (e.key === '0') {
          setZoomScale(1);
        }
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [onClose, isImage]);

  const handleZoomIn = () => setZoomScale((z) => Math.min(4, z + 0.25));
  const handleZoomOut = () => setZoomScale((z) => Math.max(0.5, z - 0.25));
  const handleResetZoom = () => setZoomScale(1);

  const handleCopyContent = () => {
    if (fetchedContent) {
      navigator.clipboard.writeText(fetchedContent);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  if (typeof document === 'undefined') return null;

  return createPortal(
    <div
      data-testid="artifact-lightbox-modal"
      className="fixed inset-0 z-[120] flex flex-col bg-black/85 backdrop-blur-md p-4 sm:p-6"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-label={name}
    >
      {/* Lightbox Header */}
      <div
        className="flex items-center justify-between gap-4 mb-4 pb-3 border-b border-white/10 shrink-0"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center gap-2.5 min-w-0">
          <div className="p-1.5 rounded-lg bg-white/5 border border-white/10 text-cyan-400">
            {isImage ? (
              <ImageIcon className="w-4 h-4 text-emerald-400" />
            ) : isVideo ? (
              <Video className="w-4 h-4 text-blue-400" />
            ) : isTerminalOrDiff ? (
              <Terminal className="w-4 h-4 text-purple-400" />
            ) : (
              <FileText className="w-4 h-4 text-cyan-400" />
            )}
          </div>
          <div className="min-w-0">
            <h2 className="text-sm font-bold text-white font-mono truncate" title={name}>
              {name}
            </h2>
            {path && (
              <p className="text-[10px] font-mono text-gray-400 truncate max-w-md" title={path}>
                {path}
              </p>
            )}
          </div>
        </div>

        {/* Toolbar Controls */}
        <div className="flex items-center gap-2 shrink-0">
          {/* Zoom controls for image */}
          {isImage && (
            <div className="flex items-center gap-1 bg-white/5 border border-white/10 rounded-lg p-1 mr-2">
              <button
                type="button"
                onClick={handleZoomOut}
                disabled={zoomScale <= 0.5}
                title="Zoom Out (-)"
                className="p-1 text-gray-400 hover:text-white disabled:opacity-30 transition-colors"
              >
                <ZoomOut className="w-4 h-4" />
              </button>
              <span className="text-[11px] font-mono font-medium text-gray-300 w-12 text-center">
                {Math.round(zoomScale * 100)}%
              </span>
              <button
                type="button"
                onClick={handleZoomIn}
                disabled={zoomScale >= 4}
                title="Zoom In (+)"
                className="p-1 text-gray-400 hover:text-white disabled:opacity-30 transition-colors"
              >
                <ZoomIn className="w-4 h-4" />
              </button>
              <button
                type="button"
                onClick={handleResetZoom}
                title="Reset Zoom (0)"
                className="p-1 text-gray-400 hover:text-white transition-colors ml-1 border-l border-white/10 pl-1.5"
              >
                <RotateCcw className="w-3.5 h-3.5" />
              </button>
            </div>
          )}

          {/* Copy document content button */}
          {(isMarkdown || isTerminalOrDiff) && fetchedContent && (
            <button
              type="button"
              onClick={handleCopyContent}
              className="inline-flex items-center gap-1.5 text-xs font-mono px-3 py-1.5 rounded-lg bg-white/5 hover:bg-white/10 border border-white/10 text-gray-300 hover:text-white transition-colors"
            >
              {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
              <span>{copied ? 'Copied' : 'Copy'}</span>
            </button>
          )}

          {/* Download button */}
          {url && (
            <a
              href={url}
              download={name}
              className="inline-flex items-center gap-1 text-xs font-mono px-3 py-1.5 rounded-lg bg-white/5 hover:bg-white/10 border border-white/10 text-gray-300 hover:text-white transition-colors"
            >
              <Download className="w-3.5 h-3.5" />
              <span className="hidden sm:inline">Download</span>
            </a>
          )}

          {/* Close button */}
          <button
            type="button"
            onClick={onClose}
            aria-label="Close"
            className="p-1.5 rounded-lg bg-white/5 hover:bg-white/10 border border-white/10 text-gray-400 hover:text-white transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* Lightbox Body / Media Container */}
      <div
        className="flex-1 min-h-0 flex items-center justify-center overflow-auto rounded-xl bg-black/60 border border-white/5 p-4"
        onClick={(e) => e.stopPropagation()}
      >
        {isImage && url && (
          <div className="w-full h-full flex items-center justify-center overflow-auto">
            <img
              src={url}
              alt={name}
              style={{
                transform: `scale(${zoomScale})`,
                transformOrigin: 'center center',
                transition: 'transform 0.15s ease-out',
              }}
              className="max-w-full max-h-full object-contain rounded-lg shadow-2xl select-none"
            />
          </div>
        )}

        {isVideo && url && (
          <video controls autoPlay className="max-w-full max-h-full rounded-lg shadow-2xl bg-black">
            <source src={url} type="video/mp4" />
            Your browser does not support the video tag.
          </video>
        )}

        {(isMarkdown || isTerminalOrDiff) && (
          <div className="w-full max-w-4xl h-full overflow-y-auto rounded-xl bg-[#0e1117] border border-white/10 p-6 shadow-2xl">
            {loading ? (
              <div className="p-12 text-center space-y-3">
                <div className="inline-block w-6 h-6 border-2 border-cyan-500 border-t-transparent rounded-full animate-spin" />
                <div className="text-xs text-gray-400 font-mono">Loading artifact content...</div>
              </div>
            ) : isTerminalOrDiff ? (
              <pre className="text-xs font-mono leading-relaxed text-emerald-400 whitespace-pre-wrap">
                {fetchedContent || 'No output recorded'}
              </pre>
            ) : (
              <div className="prose prose-invert prose-sm max-w-none text-gray-200 text-xs leading-relaxed [&_pre]:bg-black/60 [&_pre]:border [&_pre]:border-white/10 [&_pre]:p-4 [&_pre]:rounded-lg">
                <ReactMarkdown
                  remarkPlugins={[remarkGfm]}
                  urlTransform={(u) =>
                    u.startsWith('file://')
                      ? `/artifacts?path=${encodeURIComponent(decodeURIComponent(u.slice(7)))}`
                      : u
                  }
                >
                  {fetchedContent || '# Document Preview\n\nNo text content available for this artifact.'}
                </ReactMarkdown>
              </div>
            )}
          </div>
        )}
      </div>
    </div>,
    document.body
  );
};
