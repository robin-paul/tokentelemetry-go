import React from 'react';
import { Play, Pause, ChevronLeft, ChevronRight, RotateCcw } from 'lucide-react';

export interface PlaybackControlsProps {
  isPlaying: boolean;
  onTogglePlay: () => void;
  onPrevStep: () => void;
  onNextStep: () => void;
  canPrev?: boolean;
  canNext?: boolean;
  onReset?: () => void;
  className?: string;
}

export const PlaybackControls: React.FC<PlaybackControlsProps> = ({
  isPlaying,
  onTogglePlay,
  onPrevStep,
  onNextStep,
  canPrev = true,
  canNext = true,
  onReset,
  className = '',
}) => {
  return (
    <div className={`flex items-center gap-1.5 ${className}`}>
      {onReset && (
        <button
          type="button"
          onClick={onReset}
          className="p-1.5 rounded-lg bg-white/5 hover:bg-white/10 text-gray-400 hover:text-white transition-colors"
          title="Reset to beginning"
          aria-label="Reset to beginning"
        >
          <RotateCcw className="w-3.5 h-3.5" />
        </button>
      )}

      <button
        type="button"
        onClick={onTogglePlay}
        className={`p-1.5 rounded-lg transition-colors flex items-center justify-center ${
          isPlaying
            ? 'bg-amber-500/20 text-amber-300 hover:bg-amber-500/30 border border-amber-500/30'
            : 'bg-emerald-500/20 text-emerald-300 hover:bg-emerald-500/30 border border-emerald-500/30'
        }`}
        title={isPlaying ? 'Pause auto-replay (600ms)' : 'Play auto-replay (600ms)'}
        aria-label={isPlaying ? 'Pause playback' : 'Start playback'}
      >
        {isPlaying ? <Pause className="w-4 h-4 text-amber-400" /> : <Play className="w-4 h-4 text-emerald-400" />}
      </button>

      <button
        type="button"
        onClick={onPrevStep}
        disabled={!canPrev}
        className="p-1.5 rounded-lg bg-white/5 hover:bg-white/10 text-gray-300 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
        title="Previous turn"
        aria-label="Previous turn"
      >
        <ChevronLeft className="w-4 h-4" />
      </button>

      <button
        type="button"
        onClick={onNextStep}
        disabled={!canNext}
        className="p-1.5 rounded-lg bg-white/5 hover:bg-white/10 text-gray-300 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
        title="Next turn"
        aria-label="Next turn"
      >
        <ChevronRight className="w-4 h-4" />
      </button>
    </div>
  );
};
