import React, { useState, useRef, useEffect } from 'react';
import { User, Bot, Wrench, Brain, Layers, Filter, Check } from 'lucide-react';

export type TurnFilterCategory = 'all' | 'user' | 'assistant' | 'reasoning' | 'tools';

export interface CategoryCounts {
  all: number;
  user: number;
  assistant: number;
  reasoning: number;
  tools: number;
}

export interface StepFilterPopoverProps {
  activeCategory: TurnFilterCategory;
  onSelectCategory: (category: TurnFilterCategory) => void;
  counts: CategoryCounts;
  className?: string;
}

export const StepFilterPopover: React.FC<StepFilterPopoverProps> = ({
  activeCategory,
  onSelectCategory,
  counts,
  className = '',
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const popoverRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (popoverRef.current && !popoverRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [isOpen]);

  const categories: { id: TurnFilterCategory; label: string; icon: React.ReactNode; count: number; colorClass: string }[] = [
    {
      id: 'all',
      label: 'All Turns',
      icon: <Layers className="w-3.5 h-3.5" />,
      count: counts.all,
      colorClass: 'text-blue-400',
    },
    {
      id: 'user',
      label: 'User',
      icon: <User className="w-3.5 h-3.5" />,
      count: counts.user,
      colorClass: 'text-blue-400',
    },
    {
      id: 'assistant',
      label: 'Assistant',
      icon: <Bot className="w-3.5 h-3.5" />,
      count: counts.assistant,
      colorClass: 'text-cyan-400',
    },
    {
      id: 'reasoning',
      label: 'Reasoning',
      icon: <Brain className="w-3.5 h-3.5" />,
      count: counts.reasoning,
      colorClass: 'text-amber-400',
    },
    {
      id: 'tools',
      label: 'Tools',
      icon: <Wrench className="w-3.5 h-3.5" />,
      count: counts.tools,
      colorClass: 'text-emerald-400',
    },
  ];

  return (
    <div className={`relative inline-block ${className}`} ref={popoverRef}>
      {/* Inline quick-pills on larger screens */}
      <div className="flex items-center gap-1.5 overflow-x-auto pb-1 text-xs">
        {categories.map((cat) => {
          if (cat.id !== 'all' && cat.id !== 'user' && cat.id !== 'assistant' && cat.count === 0) {
            return null;
          }
          const isSelected = activeCategory === cat.id;

          let activeStyle = 'bg-blue-600 text-white font-semibold shadow-sm';
          if (cat.id === 'reasoning') {
            activeStyle = 'bg-amber-600 text-white font-semibold shadow-sm';
          } else if (cat.id === 'tools') {
            activeStyle = 'bg-emerald-600 text-white font-semibold shadow-sm';
          }

          let inactiveStyle = 'bg-white/5 text-gray-400 hover:text-white hover:bg-white/10';
          if (cat.id === 'reasoning') {
            inactiveStyle = 'bg-amber-500/10 text-amber-300 hover:bg-amber-500/20';
          } else if (cat.id === 'tools') {
            inactiveStyle = 'bg-emerald-500/10 text-emerald-300 hover:bg-emerald-500/20';
          }

          return (
            <button
              key={cat.id}
              type="button"
              onClick={() => onSelectCategory(cat.id)}
              className={`px-3 py-1.5 rounded-lg font-medium transition-all flex items-center gap-1.5 whitespace-nowrap ${
                isSelected ? activeStyle : inactiveStyle
              }`}
            >
              {cat.icon}
              <span>
                {cat.label} ({cat.count})
              </span>
            </button>
          );
        })}

        {/* Dropdown filter popover button for compact view */}
        <button
          type="button"
          onClick={() => setIsOpen(!isOpen)}
          className={`p-1.5 rounded-lg border transition-colors ${
            isOpen || activeCategory !== 'all'
              ? 'bg-blue-500/20 text-blue-300 border-blue-500/30'
              : 'bg-white/5 text-gray-400 hover:text-white border-white/10'
          }`}
          title="Filter category options"
          aria-label="Filter category options"
        >
          <Filter className="w-3.5 h-3.5" />
        </button>
      </div>

      {/* Popover Menu Dropdown */}
      {isOpen && (
        <div className="absolute left-0 mt-2 w-56 rounded-xl bg-[#11141a] border border-white/10 shadow-2xl p-1.5 z-30 space-y-1 backdrop-blur-xl">
          <div className="px-2.5 py-1.5 text-[10px] font-bold uppercase tracking-wider text-gray-400 border-b border-white/5">
            Filter Turn Categories
          </div>
          {categories.map((cat) => (
            <button
              key={cat.id}
              type="button"
              onClick={() => {
                onSelectCategory(cat.id);
                setIsOpen(false);
              }}
              className={`w-full flex items-center justify-between px-2.5 py-2 rounded-lg text-xs transition-colors text-left ${
                activeCategory === cat.id
                  ? 'bg-blue-600/20 text-blue-300 font-semibold'
                  : 'text-gray-300 hover:bg-white/5 hover:text-white'
              }`}
            >
              <div className="flex items-center gap-2">
                <span className={cat.colorClass}>{cat.icon}</span>
                <span>{cat.label}</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-[11px] font-mono text-gray-400">({cat.count})</span>
                {activeCategory === cat.id && <Check className="w-3.5 h-3.5 text-blue-400" />}
              </div>
            </button>
          ))}
        </div>
      )}
    </div>
  );
};
