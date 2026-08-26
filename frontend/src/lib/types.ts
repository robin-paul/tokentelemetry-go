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
}

export interface ToolResult {
  id?: string;
  name?: string;
  content?: unknown;
  is_error?: boolean;
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
}

export interface Session {
  id: string;
  session_id: string;
  agent_name: string;
  project_name: string;
  file_path: string;
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

export interface ProjectSummary {
  project_name: string;
  session_count: number;
  total_tokens: number;
  total_cost_usd: number;
  last_active: string;
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
