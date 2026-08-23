-- Multi-machine collector ingestion schema update
ALTER TABLE sessions ADD COLUMN machine_id TEXT NOT NULL DEFAULT 'local';

CREATE INDEX IF NOT EXISTS idx_sessions_machine_agent ON sessions(machine_id, agent_name, start_time DESC);
