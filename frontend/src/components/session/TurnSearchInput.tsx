import React from 'react';
import { Search, X } from 'lucide-react';

export interface TurnSearchInputProps {
  value: string;
  onChange: (query: string) => void;
  placeholder?: string;
  matchCount?: number;
  onClear?: () => void;
  className?: string;
}

export const TurnSearchInput: React.FC<TurnSearchInputProps> = ({
  value,
  onChange,
  placeholder = 'Search prompt, model, tools, thinking...',
  matchCount,
  onClear,
  className = '',
}) => {
  const handleClear = () => {
    onChange('');
    if (onClear) onClear();
  };

  return (
    <div className={`relative flex items-center ${className}`}>
      <Search className="w-3.5 h-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 pointer-events-none" />
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="w-full pl-9 pr-16 py-1.5 text-xs bg-[#11141a] border border-white/10 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-cyan-500 transition-colors"
        aria-label="Search within active trace"
      />

      <div className="absolute right-2.5 top-1/2 -translate-y-1/2 flex items-center gap-1.5">
        {value.trim() && matchCount !== undefined && (
          <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-cyan-500/10 text-cyan-400 border border-cyan-500/20">
            {matchCount} {matchCount === 1 ? 'match' : 'matches'}
          </span>
        )}

        {value && (
          <button
            type="button"
            onClick={handleClear}
            className="p-1 rounded text-gray-400 hover:text-white hover:bg-white/10 transition-colors"
            title="Clear search"
            aria-label="Clear search"
          >
            <X className="w-3 h-3" />
          </button>
        )}
      </div>
    </div>
  );
};

export interface HighlightTextProps {
  text?: string;
  query?: string;
  className?: string;
}

export const HighlightText: React.FC<HighlightTextProps> = ({
  text = '',
  query = '',
  className = '',
}) => {
  if (!query || !query.trim() || !text) {
    return <span className={className}>{text}</span>;
  }

  const cleanQuery = query.trim();
  const escaped = cleanQuery.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const regex = new RegExp(`(${escaped})`, 'gi');
  const parts = text.split(regex);

  return (
    <span className={className}>
      {parts.map((part, idx) => {
        if (part.toLowerCase() === cleanQuery.toLowerCase()) {
          return (
            <mark
              key={idx}
              className="bg-yellow-500/30 text-yellow-200 px-0.5 rounded font-medium"
            >
              {part}
            </mark>
          );
        }
        return part;
      })}
    </span>
  );
};
