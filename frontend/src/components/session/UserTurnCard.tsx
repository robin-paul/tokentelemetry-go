import React from 'react';
import { User, Clock } from 'lucide-react';
import { formatDate } from '../../lib/format';
import type { MessageTurn } from '../../lib/types';
import { ResponseBody } from './ResponseBody';

interface UserTurnCardProps {
  turn: MessageTurn;
  isActive?: boolean;
  onClick?: () => void;
}

export const UserTurnCard: React.FC<UserTurnCardProps> = ({
  turn,
  isActive = false,
  onClick,
}) => {
  const content = turn.content || '';

  return (
    <div
      onClick={onClick}
      className={`bg-[#11141a] border rounded-xl p-5 relative overflow-hidden transition-all cursor-pointer ${
        isActive
          ? 'border-blue-500/60 ring-2 ring-blue-500/20 shadow-lg'
          : 'border-white/10 hover:border-white/20'
      }`}
    >
      {/* Left blue accent indicator bar */}
      <div className="absolute top-0 left-0 w-1.5 h-full bg-blue-600" />

      {/* Header */}
      <div className="flex items-center justify-between gap-2 mb-3 pl-1">
        <div className="flex items-center gap-2">
          <div className="w-6 h-6 rounded-md bg-blue-500/20 text-blue-400 flex items-center justify-center">
            <User className="w-3.5 h-3.5" />
          </div>
          <span className="text-[11px] font-black uppercase tracking-[0.16em] text-blue-400">
            User Prompt
          </span>
          <span className="text-[11px] font-mono text-gray-500">
            Turn #{turn.turn_index + 1}
          </span>
        </div>

        {turn.timestamp && (
          <div className="flex items-center gap-1 text-[11px] font-mono text-gray-400">
            <Clock className="w-3 h-3 text-gray-500" />
            <span>{formatDate(turn.timestamp)}</span>
          </div>
        )}
      </div>

      {/* Content */}
      <div className="pl-1">
        {content ? (
          <ResponseBody content={content} defaultMode="md" />
        ) : (
          <span className="text-xs text-gray-500 italic">No prompt text recorded</span>
        )}
      </div>
    </div>
  );
};
