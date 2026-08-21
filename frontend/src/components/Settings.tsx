import React, { useEffect, useState } from 'react';
import { Settings as SettingsIcon, Coins, ShieldCheck, Sparkles, Plus, Trash2 } from 'lucide-react';
import { apiFetch } from '../lib/api';
import type { PricingOverride } from '../lib/types';

export const Settings: React.FC = () => {
  const [pricingData, setPricingData] = useState<any>(null);
  const [overrides, setOverrides] = useState<PricingOverride[]>([]);
  const [newModel, setNewModel] = useState('');
  const [newInput, setNewInput] = useState('3.0');
  const [newOutput, setNewOutput] = useState('15.0');
  const [saving, setSaving] = useState(false);
  const [statusMsg, setStatusMsg] = useState('');

  const loadSettings = async () => {
    try {
      const p = await apiFetch<any>('/api/pricing');
      setPricingData(p);
      setOverrides(p?.overrides || []);
    } catch (e) {
      console.error('Failed to load settings', e);
    }
  };

  useEffect(() => {
    loadSettings();
  }, []);

  const handleAddOverride = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newModel) return;
    setSaving(true);
    try {
      await apiFetch('/api/pricing/override', {
        method: 'POST',
        body: JSON.stringify({
          model_pattern: newModel,
          input_cost_per_m: parseFloat(newInput) || 0,
          output_cost_per_m: parseFloat(newOutput) || 0,
          cache_read_cost_per_m: (parseFloat(newInput) || 0) * 0.1,
          cache_write_cost_per_m: (parseFloat(newInput) || 0) * 1.25,
          source: 'user_override',
        }),
      });
      setNewModel('');
      setStatusMsg('Custom pricing override saved.');
      setTimeout(() => setStatusMsg(''), 3000);
      loadSettings();
    } catch (err: any) {
      alert(err.message);
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteOverride = async (pattern: string) => {
    try {
      await apiFetch(`/api/pricing/override/${encodeURIComponent(pattern)}`, {
        method: 'DELETE',
      });
      loadSettings();
    } catch (err: any) {
      alert(err.message);
    }
  };

  return (
    <div className="space-y-8 max-w-4xl">
      <div>
        <h1 className="text-xl font-bold text-white flex items-center gap-2">
          <SettingsIcon className="w-5 h-5 text-blue-400" /> Settings & Pricing Engine
        </h1>
        <p className="text-xs text-gray-400 mt-1">Configure pricing rate overrides, budgets, and summarizer backends</p>
      </div>

      {statusMsg && (
        <div className="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-xs flex items-center gap-2">
          <ShieldCheck className="w-4 h-4" /> {statusMsg}
        </div>
      )}

      {/* Pricing Overrides Section */}
      <div className="p-6 rounded-xl bg-[#11141a] border border-white/10 space-y-4">
        <h2 className="text-sm font-semibold text-white flex items-center gap-2">
          <Coins className="w-4 h-4 text-emerald-400" /> Custom Model Pricing Overrides
        </h2>
        <p className="text-xs text-gray-400">
          User-defined rates take Tier 2 precedence over embedded models.dev tables during session cost calculations.
        </p>

        {/* Add Override Form */}
        <form onSubmit={handleAddOverride} className="grid grid-cols-1 md:grid-cols-4 gap-3 pt-2">
          <input
            type="text"
            placeholder="Model Pattern (e.g. claude-sonnet-4-6)"
            value={newModel}
            onChange={(e) => setNewModel(e.target.value)}
            className="md:col-span-2 bg-[#07090d] border border-white/10 rounded-lg px-3 py-2 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-blue-500"
            required
          />
          <input
            type="number"
            step="0.01"
            placeholder="Input $/1M"
            value={newInput}
            onChange={(e) => setNewInput(e.target.value)}
            className="bg-[#07090d] border border-white/10 rounded-lg px-3 py-2 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-blue-500"
            required
          />
          <input
            type="number"
            step="0.01"
            placeholder="Output $/1M"
            value={newOutput}
            onChange={(e) => setNewOutput(e.target.value)}
            className="bg-[#07090d] border border-white/10 rounded-lg px-3 py-2 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-blue-500"
            required
          />
          <button
            type="submit"
            disabled={saving}
            className="md:col-span-4 flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white font-medium text-xs transition-colors disabled:opacity-50"
          >
            <Plus className="w-4 h-4" /> Add Rate Override
          </button>
        </form>

        {/* Active Overrides Table */}
        <div className="pt-4 border-t border-white/5">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="border-b border-white/10 text-gray-400 font-medium">
                <th className="pb-2">Model Pattern</th>
                <th className="pb-2 text-right">Input ($/1M)</th>
                <th className="pb-2 text-right">Output ($/1M)</th>
                <th className="pb-2 text-right">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5">
              {overrides.map((o) => (
                <tr key={o.model_pattern}>
                  <td className="py-2.5 font-mono text-white text-[11px]">{o.model_pattern}</td>
                  <td className="py-2.5 text-right font-medium text-white tabular">${o.input_cost_per_m.toFixed(2)}</td>
                  <td className="py-2.5 text-right font-medium text-white tabular">${o.output_cost_per_m.toFixed(2)}</td>
                  <td className="py-2.5 text-right">
                    <button
                      onClick={() => handleDeleteOverride(o.model_pattern)}
                      className="p-1 rounded text-gray-400 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </td>
                </tr>
              ))}
              {overrides.length === 0 && (
                <tr>
                  <td colSpan={4} className="py-6 text-center text-gray-500 text-xs">
                    No custom overrides configured. Standard embedded rates are active.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* AI Summarizer Config Section */}
      <div className="p-6 rounded-xl bg-[#11141a] border border-white/10 space-y-3">
        <h2 className="text-sm font-semibold text-white flex items-center gap-2">
          <Sparkles className="w-4 h-4 text-purple-400" /> Trace Summarizer Settings
        </h2>
        <p className="text-xs text-gray-400">
          Configure local or remote LLM endpoints for automatic trace summarization and step reduction.
        </p>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3 pt-2">
          <div className="p-3 rounded-lg bg-[#07090d] border border-white/5 space-y-1">
            <div className="font-semibold text-xs text-white">Ollama (Local)</div>
            <div className="text-[11px] text-gray-400">http://localhost:11434 (llama3.2)</div>
          </div>
          <div className="p-3 rounded-lg bg-[#07090d] border border-white/5 space-y-1">
            <div className="font-semibold text-xs text-white">Claude CLI</div>
            <div className="text-[11px] text-gray-400">Native CLI Pipe (claude-3-5-haiku)</div>
          </div>
          <div className="p-3 rounded-lg bg-[#07090d] border border-white/5 space-y-1">
            <div className="font-semibold text-xs text-white">OpenAI Compatible</div>
            <div className="text-[11px] text-gray-400">Custom Base URL + Bearer API Key</div>
          </div>
        </div>
      </div>
    </div>
  );
};
