export interface AgentMeta {
  name: string;
  label: string;
  color: string;
  bg: string;
}

export const AGENT_REGISTRY: Record<string, AgentMeta> = {
  claude: { name: 'claude', label: 'Claude Code', color: '#f97316', bg: 'rgba(249, 115, 22, 0.12)' },
  codex: { name: 'codex', label: 'OpenAI Codex', color: '#a855f7', bg: 'rgba(168, 85, 247, 0.12)' },
  gemini: { name: 'gemini', label: 'Gemini CLI', color: '#06b6d4', bg: 'rgba(6, 182, 212, 0.12)' },
  antigravity: { name: 'antigravity', label: 'Antigravity', color: '#10b981', bg: 'rgba(16, 185, 129, 0.12)' },
  cursor: { name: 'cursor', label: 'Cursor', color: '#60a5fa', bg: 'rgba(96, 165, 250, 0.12)' },
  copilot: { name: 'copilot', label: 'GitHub Copilot', color: '#6366f1', bg: 'rgba(99, 102, 241, 0.12)' },
  hermes: { name: 'hermes', label: 'Hermes', color: '#eab308', bg: 'rgba(234, 179, 8, 0.12)' },
  qwen: { name: 'qwen', label: 'Qwen Code', color: '#3b82f6', bg: 'rgba(59, 130, 246, 0.12)' },
  grok: { name: 'grok', label: 'Grok Build', color: '#d4d4d8', bg: 'rgba(212, 212, 216, 0.12)' },
  pi: { name: 'pi', label: 'Pi Agent', color: '#ec4899', bg: 'rgba(236, 72, 153, 0.12)' },
  cline: { name: 'cline', label: 'Cline', color: '#14b8a6', bg: 'rgba(20, 184, 166, 0.12)' },
  muse: { name: 'muse', label: 'Meta Muse', color: '#2563eb', bg: 'rgba(37, 99, 235, 0.12)' },
  prime: { name: 'prime', label: 'Prime Agent', color: '#84cc16', bg: 'rgba(132, 204, 22, 0.12)' },
  dsh: { name: 'dsh', label: 'DeepSeek Harness', color: '#4d6bfe', bg: 'rgba(77, 107, 254, 0.12)' },
  smallcode: { name: 'smallcode', label: 'SmallCode', color: '#8b5cf6', bg: 'rgba(139, 92, 246, 0.12)' },
  windsurf: { name: 'windsurf', label: 'Windsurf', color: '#0ea5e9', bg: 'rgba(14, 165, 233, 0.12)' },
  vibe: { name: 'vibe', label: 'Mistral Vibe', color: '#f472b6', bg: 'rgba(244, 114, 182, 0.12)' },
  ollama: { name: 'ollama', label: 'Ollama', color: '#94a3b8', bg: 'rgba(148, 163, 184, 0.12)' },
  opencode: { name: 'opencode', label: 'OpenCode', color: '#10b981', bg: 'rgba(16, 185, 129, 0.12)' },
};

export function getAgentMeta(agentName: string | undefined): AgentMeta {
  if (!agentName) {
    return { name: 'unknown', label: 'Unknown Agent', color: '#64748b', bg: 'rgba(100, 116, 139, 0.12)' };
  }
  const lower = agentName.toLowerCase();
  for (const [k, v] of Object.entries(AGENT_REGISTRY)) {
    if (lower.includes(k)) return v;
  }
  return { name: agentName, label: agentName, color: '#64748b', bg: 'rgba(100, 116, 139, 0.12)' };
}
