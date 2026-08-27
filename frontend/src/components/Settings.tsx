import React, { useEffect, useMemo, useState } from 'react';
import {
  Settings as SettingsIcon,
  Coins,
  Sparkles,
  Plus,
  Trash2,
  Edit2,
  X,
  Search,
  CheckCircle,
  AlertCircle,
} from 'lucide-react';
import { apiFetch } from '../lib/api';
import type { PricingOverride } from '../lib/types';

export const Settings: React.FC = () => {
  const [pricingData, setPricingData] = useState<any>(null);
  const [overrides, setOverrides] = useState<PricingOverride[]>([]);
  const [searchQuery, setSearchQuery] = useState('');

  // Form state
  const [modelPattern, setModelPattern] = useState('');
  const [inputCost, setInputCost] = useState('3.0');
  const [outputCost, setOutputCost] = useState('15.0');
  const [cacheReadCost, setCacheReadCost] = useState('0.3');
  const [cacheWriteCost, setCacheWriteCost] = useState('3.75');
  const [editingPattern, setEditingPattern] = useState<string | null>(null);

  const [saving, setSaving] = useState(false);
  const [statusMsg, setStatusMsg] = useState('');
  const [errorMsg, setErrorMsg] = useState('');

  const loadSettings = async () => {
    try {
      const p = await apiFetch<any>('/api/pricing');
      setPricingData(p);
      setOverrides(p?.overrides || []);
    } catch (e: any) {
      console.error('Failed to load settings', e);
      setErrorMsg(e?.message || 'Failed to load pricing configuration');
    }
  };

  useEffect(() => {
    loadSettings();
  }, []);

  const handleInputChange = (val: string) => {
    setInputCost(val);
    const num = parseFloat(val);
    if (!isNaN(num)) {
      setCacheReadCost((num * 0.1).toFixed(2));
      setCacheWriteCost((num * 1.25).toFixed(2));
    }
  };

  const handleStartEdit = (o: PricingOverride) => {
    setEditingPattern(o.model_pattern);
    setModelPattern(o.model_pattern);
    setInputCost(o.input_cost_per_m.toString());
    setOutputCost(o.output_cost_per_m.toString());
    setCacheReadCost(o.cache_read_cost_per_m?.toString() || (o.input_cost_per_m * 0.1).toFixed(2));
    setCacheWriteCost(o.cache_write_cost_per_m?.toString() || (o.input_cost_per_m * 1.25).toFixed(2));
    setStatusMsg('');
    setErrorMsg('');
  };

  const handleCancelEdit = () => {
    setEditingPattern(null);
    setModelPattern('');
    setInputCost('3.0');
    setOutputCost('15.0');
    setCacheReadCost('0.3');
    setCacheWriteCost('3.75');
    setErrorMsg('');
  };

  const handleSaveOverride = async (e: React.FormEvent) => {
    e.preventDefault();
    const pattern = modelPattern.trim();
    if (!pattern) return;

    setSaving(true);
    setErrorMsg('');
    setStatusMsg('');

    try {
      // If editing and changed pattern name, delete old first
      if (editingPattern && editingPattern !== pattern) {
        await apiFetch(`/api/pricing/override/${encodeURIComponent(editingPattern)}`, {
          method: 'DELETE',
        });
      }

      await apiFetch('/api/pricing/override', {
        method: 'POST',
        body: JSON.stringify({
          model_pattern: pattern,
          input_cost_per_m: parseFloat(inputCost) || 0,
          output_cost_per_m: parseFloat(outputCost) || 0,
          cache_read_cost_per_m: parseFloat(cacheReadCost) || (parseFloat(inputCost) || 0) * 0.1,
          cache_write_cost_per_m: parseFloat(cacheWriteCost) || (parseFloat(inputCost) || 0) * 1.25,
          source: 'user_override',
        }),
      });

      const wasEditing = !!editingPattern;
      handleCancelEdit();
      setStatusMsg(wasEditing ? `Pricing override for "${pattern}" updated.` : `Custom pricing override saved.`);
      setTimeout(() => setStatusMsg(''), 4000);
      await loadSettings();
    } catch (err: any) {
      setErrorMsg(err.message || 'Failed to save override');
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteOverride = async (pattern: string) => {
    if (editingPattern === pattern) {
      handleCancelEdit();
    }
    try {
      await apiFetch(`/api/pricing/override/${encodeURIComponent(pattern)}`, {
        method: 'DELETE',
      });
      setStatusMsg(`Pricing override for "${pattern}" removed.`);
      setTimeout(() => setStatusMsg(''), 3000);
      loadSettings();
    } catch (err: any) {
      setErrorMsg(err.message || 'Failed to delete override');
    }
  };

  const filteredOverrides = useMemo(() => {
    return overrides.filter((o) =>
      o.model_pattern.toLowerCase().includes(searchQuery.toLowerCase().trim())
    );
  }, [overrides, searchQuery]);

  return (
    <div className="space-y-8 max-w-4xl">
      <div>
        <h1 className="text-xl font-bold text-white flex items-center gap-2">
          <SettingsIcon className="w-5 h-5 text-blue-400" /> Settings & Pricing Engine
        </h1>
        <p className="text-xs text-gray-400 mt-1">Configure pricing rate overrides, custom models, and summarizer backends</p>
      </div>

      {statusMsg && (
        <div className="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-xs flex items-center gap-2">
          <CheckCircle className="w-4 h-4 shrink-0" /> {statusMsg}
        </div>
      )}

      {errorMsg && (
        <div className="p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-xs flex items-center gap-2">
          <AlertCircle className="w-4 h-4 shrink-0" /> {errorMsg}
        </div>
      )}

      {/* Pricing Overrides Section */}
      <div className="p-6 rounded-xl bg-[#11141a] border border-white/10 space-y-5">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-sm font-semibold text-white flex items-center gap-2">
              <Coins className="w-4 h-4 text-emerald-400" /> Custom Model Pricing Overrides
            </h2>
            <p className="text-xs text-gray-400 mt-0.5">
              User-defined rates take Tier 2 precedence over embedded catalog rates during session cost calculations.
            </p>
          </div>
          <span className="text-xs text-gray-400 bg-white/5 px-2.5 py-1 rounded-full">
            {overrides.length} override{overrides.length === 1 ? '' : 's'}
          </span>
        </div>

        {/* Add/Edit Override Form */}
        <form onSubmit={handleSaveOverride} className="p-4 rounded-lg bg-[#07090d] border border-white/10 space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-white flex items-center gap-1.5">
              {editingPattern ? (
                <>
                  <Edit2 className="w-3.5 h-3.5 text-amber-400" /> Editing Override: <span className="font-mono text-amber-300">{editingPattern}</span>
                </>
              ) : (
                <>
                  <Plus className="w-3.5 h-3.5 text-blue-400" /> Add New Model Rate Override
                </>
              )}
            </span>
            {editingPattern && (
              <button
                type="button"
                onClick={handleCancelEdit}
                className="text-[11px] text-gray-400 hover:text-white flex items-center gap-1 px-2 py-0.5 rounded hover:bg-white/5"
              >
                <X className="w-3 h-3" /> Cancel Edit
              </button>
            )}
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3">
            <div className="md:col-span-2">
              <label className="block text-[11px] text-gray-400 mb-1">Model Pattern (exact or prefix match)</label>
              <input
                type="text"
                placeholder="Model Pattern (e.g. gpt-4o-custom-e2e, claude-3-7-sonnet)"
                value={modelPattern}
                onChange={(e) => setModelPattern(e.target.value)}
                className="w-full bg-[#11141a] border border-white/10 rounded-lg px-3 py-2 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-blue-500 font-mono"
                required
              />
            </div>
            <div>
              <label className="block text-[11px] text-gray-400 mb-1">Input ($ / 1M tokens)</label>
              <input
                type="number"
                step="0.001"
                min="0"
                placeholder="Input $/1M"
                value={inputCost}
                onChange={(e) => handleInputChange(e.target.value)}
                className="w-full bg-[#11141a] border border-white/10 rounded-lg px-3 py-2 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-blue-500"
                required
              />
            </div>
            <div>
              <label className="block text-[11px] text-gray-400 mb-1">Output ($ / 1M tokens)</label>
              <input
                type="number"
                step="0.001"
                min="0"
                placeholder="Output $/1M"
                value={outputCost}
                onChange={(e) => setOutputCost(e.target.value)}
                className="w-full bg-[#11141a] border border-white/10 rounded-lg px-3 py-2 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-blue-500"
                required
              />
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
            <div>
              <label className="block text-[11px] text-gray-400 mb-1">Cache Read ($ / 1M tokens, default 10%)</label>
              <input
                type="number"
                step="0.001"
                min="0"
                placeholder="Cache Read $/1M"
                value={cacheReadCost}
                onChange={(e) => setCacheReadCost(e.target.value)}
                className="w-full bg-[#11141a] border border-white/10 rounded-lg px-3 py-1.5 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-blue-500"
              />
            </div>
            <div>
              <label className="block text-[11px] text-gray-400 mb-1">Cache Write ($ / 1M tokens, default 125%)</label>
              <input
                type="number"
                step="0.001"
                min="0"
                placeholder="Cache Write $/1M"
                value={cacheWriteCost}
                onChange={(e) => setCacheWriteCost(e.target.value)}
                className="w-full bg-[#11141a] border border-white/10 rounded-lg px-3 py-1.5 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-blue-500"
              />
            </div>
          </div>

          <div className="pt-2 flex justify-end gap-2">
            {editingPattern && (
              <button
                type="button"
                onClick={handleCancelEdit}
                className="px-4 py-2 rounded-lg bg-white/5 hover:bg-white/10 text-gray-300 font-medium text-xs transition-colors"
              >
                Cancel
              </button>
            )}
            <button
              type="submit"
              disabled={saving}
              className={`flex items-center justify-center gap-2 px-5 py-2 rounded-lg font-medium text-xs transition-colors disabled:opacity-50 ${
                editingPattern ? 'bg-amber-600 hover:bg-amber-500 text-white' : 'bg-blue-600 hover:bg-blue-500 text-white'
              }`}
            >
              {editingPattern ? (
                <>
                  <Edit2 className="w-3.5 h-3.5" /> Update Override
                </>
              ) : (
                <>
                  <Plus className="w-3.5 h-3.5" /> Add Rate Override
                </>
              )}
            </button>
          </div>
        </form>

        {/* Active Overrides Table with Search Filter */}
        <div className="space-y-3 pt-2">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <h3 className="text-xs font-semibold text-gray-300">Active Rate Overrides</h3>
            {overrides.length > 0 && (
              <div className="relative w-full sm:w-64">
                <Search className="w-3.5 h-3.5 text-gray-500 absolute left-2.5 top-2" />
                <input
                  type="text"
                  placeholder="Filter overrides..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-full bg-[#07090d] border border-white/10 rounded-lg pl-8 pr-3 py-1 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-blue-500"
                />
              </div>
            )}
          </div>

          <div className="overflow-x-auto rounded-lg border border-white/5">
            <table className="w-full text-left text-xs">
              <thead>
                <tr className="border-b border-white/10 bg-white/[0.02] text-gray-400 font-medium">
                  <th className="py-2.5 px-3">Model Pattern</th>
                  <th className="py-2.5 px-3 text-right">Input ($/1M)</th>
                  <th className="py-2.5 px-3 text-right">Output ($/1M)</th>
                  <th className="py-2.5 px-3 text-right">Cache Read</th>
                  <th className="py-2.5 px-3 text-right">Cache Write</th>
                  <th className="py-2.5 px-3 text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {filteredOverrides.map((o) => (
                  <tr key={o.model_pattern} className="hover:bg-white/[0.02] transition-colors">
                    <td className="py-2.5 px-3 font-mono text-white text-[11px] font-medium">{o.model_pattern}</td>
                    <td className="py-2.5 px-3 text-right font-medium text-white tabular">${o.input_cost_per_m.toFixed(2)}</td>
                    <td className="py-2.5 px-3 text-right font-medium text-white tabular">${o.output_cost_per_m.toFixed(2)}</td>
                    <td className="py-2.5 px-3 text-right text-gray-400 tabular">${o.cache_read_cost_per_m?.toFixed(2) || '0.00'}</td>
                    <td className="py-2.5 px-3 text-right text-gray-400 tabular">${o.cache_write_cost_per_m?.toFixed(2) || '0.00'}</td>
                    <td className="py-2.5 px-3 text-right">
                      <div className="flex items-center justify-end gap-1.5">
                        <button
                          onClick={() => handleStartEdit(o)}
                          title="Edit override"
                          className="p-1 rounded text-gray-400 hover:text-amber-400 hover:bg-amber-500/10 transition-colors"
                        >
                          <Edit2 className="w-3.5 h-3.5" />
                        </button>
                        <button
                          onClick={() => handleDeleteOverride(o.model_pattern)}
                          title="Delete override"
                          className="p-1 rounded text-gray-400 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
                {filteredOverrides.length === 0 && (
                  <tr>
                    <td colSpan={6} className="py-8 text-center text-gray-500 text-xs">
                      {searchQuery
                        ? `No pricing overrides matching "${searchQuery}"`
                        : 'No custom overrides configured. Standard embedded rates are active.'}
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
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
