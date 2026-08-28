export interface TokenUsage {
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_creation_tokens: number;
}

export interface ToolCall {
  id?: string;
  name: string;
  args?: Record<string, unknown>;
  args_json?: string;
  duration_ms?: number;
}

export interface ToolResult {
  id?: string;
  name?: string;
  content?: unknown;
  is_error?: boolean;
  duration_ms?: number;
}

export interface SessionArtifact {
  name: string;
  path: string;
  type: 'image' | 'video' | 'document' | 'terminal';
  size_bytes?: number;
  created_at?: string;
}

export interface PublishedArtifact {
  kind?: 'page' | 'document';
  url?: string;
  path?: string;
  title?: string;
  description?: string;
  favicon?: string;
  file_name?: string;
}

export interface MessageTurn {
  id: string;
  session_id: string;
  turn_index: number;
  timestamp: string;
  role: string;
  model_name: string;
  content?: string;
  thinking?: string;
  reasoning_effort?: string;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_creation_tokens: number;
  cost_usd: number;
  tools_invoked?: string[];
  tools_invoked_json?: string;
  tool_calls?: ToolCall[];
  tool_calls_json?: string;
  tool_results?: ToolResult[];
  tool_results_json?: string;
  raw_payload_json?: string;
}

export interface SubagentRun {
  id: string;
  parent_session_id: string;
  child_session_id: string;
  agent_type: string;
  tokens: number;
  cost_usd: number;
  created_at: string;
  sandbox?: {
    mode?: string;
    mode_source?: string;
    approval?: string;
    approval_source?: string;
    permission_preset?: string;
  };
}

export interface DSHMetrics {
  turns?: number;
  steps?: number;
  llm_ms?: number | null;
  tool_ms?: number | null;
  ttft_ms_avg?: number | null;
  output_tok_per_sec?: number | null;
  cache_hit_pct?: number | null;
}

export interface DSHContext {
  agent_preset?: string;
  preset_chain?: string[];
  sandbox?: Record<string, unknown>;
  metrics?: DSHMetrics;
  models_used?: string[];
  providers_used?: string[];
  skills_catalog?: string[];
  tools_available?: string[];
}

export interface Session {
  id: string;
  session_id: string;
  agent_name: string;
  project_name: string;
  file_path: string;
  machine_id?: string;
  created_at: string;
  updated_at: string;
  start_time: string;
  end_time: string;
  duration_seconds: number;
  model_raw: string;
  model_resolved: string;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_creation_tokens: number;
  gross_cost_usd: number;
  net_cost_usd: number;
  electricity_cost_usd: number;
  hardware_profile: string;
  status: string;
  git_branch?: string;
  is_subagent?: boolean;
  parent_session_id?: string;
  subagent_type?: string;
  turns?: MessageTurn[];
  subagent_runs?: SubagentRun[];
  artifacts?: SessionArtifact[];
  published_artifacts?: PublishedArtifact[];
  instructions?: string;
  system_prompt?: string;
  env?: Record<string, unknown>;
  cwd?: string;
  dsh?: DSHContext;
}

export interface StatsOverview {
  total_sessions: number;
  total_input_tokens: number;
  total_output_tokens: number;
  total_tokens: number;
  gross_cost_usd: number;
  net_cost_usd: number;
  active_agents: number;
  active_projects: number;
}

export interface DailySummary {
  date: string;
  agent_name: string;
  project_name: string;
  model_name: string;
  total_sessions: number;
  total_input_tokens: number;
  total_output_tokens: number;
  total_cache_read_tokens: number;
  total_cache_creation_tokens: number;
  total_cost_usd: number;
  total_duration_seconds: number;
}

export interface LeaderboardEntry {
  name: string;
  total_tokens: number;
  total_cost_usd: number;
  session_count: number;
}

export interface WorktreeSummary {
  name: string;
  path: string;
  session_count: number;
  total_tokens: number;
  total_cost_usd: number;
  agents: string[];
  status: string;
}

export interface ProjectAggregate {
  session_count: number;
  subagent_count: number;
  plan_count: number;
  configured_subagent_count: number;
  total_tokens: number;
  total_cost_usd: number;
  agents: string[];
  mcp_tools: string[];
  worktree_count: number;
}

export interface ProjectSummary {
  project_name: string;
  name?: string;
  path?: string;
  session_count: number;
  total_tokens: number;
  total_cost_usd: number;
  last_active: string;
  agents?: string[];
  mcp_tools?: string[];
  subagent_count?: number;
  configured_subagent_count?: number;
  plan_count?: number;
  status?: string;
  canonical_repo?: string;
  is_worktree?: boolean;
  worktree_name?: string;
  parent_path?: string;
  parent_name?: string;
  is_repo_root?: boolean;
  synthesized?: boolean;
  worktrees?: WorktreeSummary[];
  aggregate?: ProjectAggregate;
}

export interface PricingOverride {
  model_pattern: string;
  input_cost_per_m: number;
  output_cost_per_m: number;
  cache_read_cost_per_m: number;
  cache_write_cost_per_m: number;
  source: string;
  updated_at?: string;
}

export interface Budget {
  id: string;
  period: string;
  limit_type: string;
  limit_value: number;
  enabled: boolean;
  used?: number;
  fraction?: number;
}
