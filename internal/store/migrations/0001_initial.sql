-- 1. Schema Migrations Tracker
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Sessions (Primary conversation / execution units)
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,                       -- UUID or deterministic hash (agent:filepath:id)
    session_id TEXT NOT NULL,                  -- Native session ID from agent transcript
    agent_name TEXT NOT NULL,                  -- 'claude_code', 'gemini_cli', 'codex', etc.
    project_name TEXT NOT NULL,                -- Resolved project name / folder basename
    file_path TEXT NOT NULL UNIQUE,            -- Absolute path to transcript file
    created_at TIMESTAMP NOT NULL,             -- Session start timestamp
    updated_at TIMESTAMP NOT NULL,             -- Last updated timestamp
    start_time TIMESTAMP NOT NULL,             -- First message timestamp
    end_time TIMESTAMP NOT NULL,               -- Last message timestamp
    duration_seconds REAL DEFAULT 0,           -- Total active wall-clock time
    model_raw TEXT NOT NULL,                   -- Raw model name from log (e.g. 'claude-3-7-sonnet-20250219')
    model_resolved TEXT NOT NULL,              -- Canonical model name for pricing lookup
    input_tokens INTEGER DEFAULT 0,            -- Total net prompt tokens
    output_tokens INTEGER DEFAULT 0,           -- Total completion tokens
    cache_read_tokens INTEGER DEFAULT 0,       -- Prompt cache hit tokens
    cache_creation_tokens INTEGER DEFAULT 0,   -- Prompt cache write tokens
    gross_cost_usd REAL DEFAULT 0,             -- Cost without cache discounts
    net_cost_usd REAL DEFAULT 0,               -- True billable cost with cache discounts
    electricity_cost_usd REAL DEFAULT 0,       -- Estimated hardware power cost
    hardware_profile TEXT DEFAULT 'default',   -- CPU/GPU profile identifier
    status TEXT DEFAULT 'completed',           -- 'active', 'completed', 'error'
    git_branch TEXT DEFAULT '',                -- Associated git branch name
    is_subagent BOOLEAN DEFAULT 0,             -- 1 if spawned by parent orchestrator
    parent_session_id TEXT DEFAULT '',         -- ID of parent orchestrator session
    subagent_type TEXT DEFAULT ''              -- Subagent role/type ('research', 'planner', etc.)
);

-- 3. Message Turns (Fine-grained turn-by-turn metrics)
CREATE TABLE IF NOT EXISTS message_turns (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    turn_index INTEGER NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    role TEXT NOT NULL,                        -- 'user', 'assistant', 'system', 'tool'
    model_name TEXT NOT NULL,
    input_tokens INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    cache_read_tokens INTEGER DEFAULT 0,
    cache_creation_tokens INTEGER DEFAULT 0,
    cost_usd REAL DEFAULT 0,
    tools_invoked_json TEXT DEFAULT '[]',      -- JSON array of tool names called
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

-- 4. Subagent Runs (Parent-Child Rollup Linkages)
CREATE TABLE IF NOT EXISTS subagent_runs (
    id TEXT PRIMARY KEY,
    parent_session_id TEXT NOT NULL,
    child_session_id TEXT NOT NULL UNIQUE,
    agent_type TEXT NOT NULL,
    tokens INTEGER DEFAULT 0,
    cost_usd REAL DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY (parent_session_id) REFERENCES sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (child_session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

-- 5. Daily Summaries (Pre-aggregated rollups for instant dashboard queries)
CREATE TABLE IF NOT EXISTS daily_summaries (
    date TEXT NOT NULL,                        -- YYYY-MM-DD
    agent_name TEXT NOT NULL,
    project_name TEXT NOT NULL,
    model_name TEXT NOT NULL,
    total_sessions INTEGER DEFAULT 0,
    total_input_tokens INTEGER DEFAULT 0,
    total_output_tokens INTEGER DEFAULT 0,
    total_cache_read_tokens INTEGER DEFAULT 0,
    total_cache_creation_tokens INTEGER DEFAULT 0,
    total_cost_usd REAL DEFAULT 0,
    total_duration_seconds REAL DEFAULT 0,
    PRIMARY KEY (date, agent_name, project_name, model_name)
);

-- 6. Pricing Overrides (User-defined custom model rates)
CREATE TABLE IF NOT EXISTS pricing_overrides (
    model_pattern TEXT PRIMARY KEY,            -- Exact name or regex/prefix pattern
    input_cost_per_m REAL NOT NULL,            -- USD per 1M input tokens
    output_cost_per_m REAL NOT NULL,           -- USD per 1M output tokens
    cache_read_cost_per_m REAL DEFAULT 0,      -- USD per 1M cache read tokens
    cache_write_cost_per_m REAL DEFAULT 0,     -- USD per 1M cache write tokens
    source TEXT DEFAULT 'user_override',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 7. Scanner Checkpoints (Fast incremental scan resume)
CREATE TABLE IF NOT EXISTS scanner_checkpoints (
    file_path TEXT PRIMARY KEY,
    last_modified TIMESTAMP NOT NULL,
    file_size INTEGER NOT NULL,
    byte_offset INTEGER NOT NULL,
    line_number INTEGER NOT NULL,
    file_hash TEXT NOT NULL
);
