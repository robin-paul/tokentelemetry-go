import React from 'react';
import { PlaybackControls } from './PlaybackControls';
import { Layers } from 'lucide-react';

export interface TurnScrubberProps {
  activeStep: number;
  totalSteps: number;
  revealedCount: number;
  isPlaying: boolean;
  onSeek: (stepIndex: number) => void;
  onTogglePlay: () => void;
  onPrevStep: () => void;
  onNextStep: () => void;
  onReset?: () => void;
  className?: string;
}

export const TurnScrubber: React.FC<TurnScrubberProps> = ({
  activeStep,
  totalSteps,
  revealedCount,
  isPlaying,
  onSeek,
  onTogglePlay,
  onPrevStep,
  onNextStep,
  onReset,
  className = '',
}) => {
  const maxRange = totalSteps > 0 ? totalSteps - 1 : 0;
  const progressPercent = totalSteps > 1 ? (activeStep / maxRange) * 100 : 100;
  const revealedPercent = totalSteps > 1 ? (Math.min(revealedCount, totalSteps) / totalSteps) * 100 : 100;

  return (
    <div className={`p-4 rounded-xl bg-[#11141a] border border-white/10 space-y-3 ${className}`}>
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs">
        <div className="flex items-center gap-2">
          <PlaybackControls
            isPlaying={isPlaying}
            onTogglePlay={onTogglePlay}
            onPrevStep={onPrevStep}
            onNextStep={onNextStep}
            canPrev={activeStep > 0}
            canNext={activeStep < maxRange}
            onReset={onReset}
          />
          <div className="flex items-center gap-1.5 ml-2 text-gray-400 font-medium">
            <Layers className="w-3.5 h-3.5 text-blue-400" />
            <span>Timeline Scrubber</span>
          </div>
        </div>

        <div className="flex items-center gap-3 font-mono text-xs">
          {revealedCount < totalSteps && (
            <span className="text-[11px] text-gray-500 hidden sm:inline">
              (revealed {revealedCount}/{totalSteps})
            </span>
          )}
          <span
            data-test="scrubber-step-indicator"
            data-testid="scrubber-step-indicator"
            className="text-blue-400 font-semibold px-2 py-0.5 rounded bg-blue-500/10 border border-blue-500/20"
          >
            Step {totalSteps > 0 ? activeStep + 1 : 0} of {totalSteps}
          </span>
        </div>
      </div>

      <div className="relative pt-1">
        {/* Visual track background for high-water mark */}
        <div className="relative w-full h-2 rounded-lg bg-white/5 overflow-hidden">
          {/* Revealed high-water mark track */}
          <div
            className="absolute top-0 left-0 h-full bg-blue-500/20 transition-all duration-200 rounded-l-lg"
            style={{ width: `${revealedPercent}%` }}
          />
          {/* Active playhead track */}
          <div
            className="absolute top-0 left-0 h-full bg-blue-500 transition-all duration-100 rounded-l-lg"
            style={{ width: `${progressPercent}%` }}
          />
        </div>

        {/* Interactive range slider overlay */}
        <input
          type="range"
          min={0}
          max={maxRange}
          value={activeStep}
          onChange={(e) => onSeek(parseInt(e.target.value, 10))}
          className="absolute top-0 left-0 w-full h-4 opacity-0 cursor-pointer z-10"
          aria-label="Timeline scrubber"
        />
      </div>
    </div>
  );
};
