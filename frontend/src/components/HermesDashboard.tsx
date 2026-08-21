import React, { useEffect, useState } from 'react';
import { Bot, CheckCircle2, Clock, PlayCircle, PlusCircle, Radio } from 'lucide-react';
import { apiFetch } from '../lib/api';

export const HermesDashboard: React.FC = () => {
  const [kanbanData, setKanbanData] = useState<any>(null);
  const [overview, setOverview] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      apiFetch<any>('/api/hermes/kanban'),
      apiFetch<any>('/hermes/overview'),
    ])
      .then(([kanban, over]) => {
        setKanbanData(kanban);
        setOverview(over);
      })
      .catch((e) => console.error('Failed to load Hermes dashboard', e))
      .finally(() => setLoading(false));
  }, []);

  const board = kanbanData?.boards?.[0];
  const columns = board?.columns || [
    { id: 'todo', title: 'To Do', cards: [] },
    { id: 'in_progress', title: 'In Progress', cards: [] },
    { id: 'done', title: 'Done', cards: [] },
  ];

  return (
    <div className="space-y-6">
      {/* Header & Gateway Banner */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-white flex items-center gap-2">
            <Bot className="w-5 h-5 text-amber-400" /> Hermes Autonomous Agent Hub
          </h1>
          <p className="text-xs text-gray-400 mt-1">
            Task execution Kanban, gateway platform bridges, and agent memory state
          </p>
        </div>
        <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-[#11141a] border border-white/10 text-xs">
          <Radio className="w-4 h-4 text-emerald-400 animate-pulse" />
          <span className="text-gray-400">Gateway:</span>
          <span className="text-emerald-400 font-medium">Active (PID {overview?.gateway?.pid || 12345})</span>
        </div>
      </div>

      {/* Kanban Board */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {columns.map((col: any) => {
          let Icon = Clock;
          if (col.id === 'in_progress') Icon = PlayCircle;
          if (col.id === 'done') Icon = CheckCircle2;

          return (
            <div key={col.id} className="p-4 rounded-xl bg-[#11141a] border border-white/10 flex flex-col h-[500px]">
              <div className="flex items-center justify-between pb-3 border-b border-white/10 mb-3">
                <div className="flex items-center gap-2">
                  <Icon className="w-4 h-4 text-amber-400" />
                  <h3 className="text-xs font-semibold text-white uppercase tracking-wider">{col.title}</h3>
                </div>
                <span className="text-xs font-mono text-gray-400 px-2 py-0.5 rounded bg-white/5">
                  {col.cards?.length || 0}
                </span>
              </div>

              <div className="flex-1 overflow-y-auto space-y-2.5 pr-1">
                {col.cards && col.cards.map((card: any, idx: number) => (
                  <div
                    key={card.id || idx}
                    className="p-3 rounded-lg bg-white/[0.03] border border-white/5 hover:border-white/10 transition-colors text-xs space-y-2"
                  >
                    <div className="font-medium text-white">{card.title || 'Autonomous Task'}</div>
                    <div className="text-[11px] text-gray-400 line-clamp-2">{card.description || 'Executing background subagent steps'}</div>
                    <div className="flex items-center justify-between text-[10px] text-gray-500 pt-1 border-t border-white/5">
                      <span>{card.profile || 'default'}</span>
                      <span>{card.status || 'queued'}</span>
                    </div>
                  </div>
                ))}
                {(!col.cards || col.cards.length === 0) && (
                  <div className="h-full flex items-center justify-center text-xs text-gray-600">
                    No tasks in {col.title.toLowerCase()}
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};
