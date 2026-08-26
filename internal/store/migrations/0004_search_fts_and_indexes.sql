-- Composite B-Tree Indexes for Fast Sorting & Multi-Faceted Filtering
CREATE INDEX IF NOT EXISTS idx_sessions_cost_start ON sessions(net_cost_usd DESC, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_model_start ON sessions(model_resolved, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_agent_start ON sessions(agent_name, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_project_start ON sessions(project_name, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_total_tokens ON sessions((input_tokens + output_tokens) DESC, start_time DESC);

-- Pure-Go SQLite FTS5 External Content Table
CREATE VIRTUAL TABLE IF NOT EXISTS sessions_fts USING fts5(
    session_id,
    project_name,
    agent_name,
    model_resolved,
    git_branch,
    content='sessions',
    content_rowid='rowid',
    tokenize='unicode61 remove_diacritics 2'
);

-- Populate FTS5 table from existing sessions (if any)
INSERT INTO sessions_fts(rowid, session_id, project_name, agent_name, model_resolved, git_branch)
SELECT rowid, session_id, project_name, agent_name, model_resolved, git_branch FROM sessions;

-- Triggers for Zero-Duplication Synchronization
CREATE TRIGGER IF NOT EXISTS sessions_ai AFTER INSERT ON sessions BEGIN
    INSERT INTO sessions_fts(rowid, session_id, project_name, agent_name, model_resolved, git_branch)
    VALUES (new.rowid, new.session_id, new.project_name, new.agent_name, new.model_resolved, new.git_branch);
END;

CREATE TRIGGER IF NOT EXISTS sessions_ad AFTER DELETE ON sessions BEGIN
    INSERT INTO sessions_fts(sessions_fts, rowid, session_id, project_name, agent_name, model_resolved, git_branch)
    VALUES ('delete', old.rowid, old.session_id, old.project_name, old.agent_name, old.model_resolved, old.git_branch);
END;

CREATE TRIGGER IF NOT EXISTS sessions_au AFTER UPDATE ON sessions BEGIN
    INSERT INTO sessions_fts(sessions_fts, rowid, session_id, project_name, agent_name, model_resolved, git_branch)
    VALUES ('delete', old.rowid, old.session_id, old.project_name, old.agent_name, old.model_resolved, old.git_branch);
    INSERT INTO sessions_fts(rowid, session_id, project_name, agent_name, model_resolved, git_branch)
    VALUES (new.rowid, new.session_id, new.project_name, new.agent_name, new.model_resolved, new.git_branch);
END;
