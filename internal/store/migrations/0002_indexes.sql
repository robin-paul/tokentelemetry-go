-- Core Performance Indexes
CREATE INDEX IF NOT EXISTS idx_sessions_agent_start ON sessions(agent_name, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_project_start ON sessions(project_name, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_parent ON sessions(parent_session_id) WHERE is_subagent = 1;
CREATE INDEX IF NOT EXISTS idx_sessions_updated ON sessions(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_message_turns_session ON message_turns(session_id, turn_index);
CREATE INDEX IF NOT EXISTS idx_daily_summaries_date ON daily_summaries(date DESC);
