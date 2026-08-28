-- Migration 0006: Backfill canonical project paths in sessions and daily_summaries
UPDATE sessions
SET project_name = CASE
    WHEN (project_name LIKE '_:%' OR project_name LIKE '\\%') THEN
        CASE
            WHEN rtrim(replace(project_name, '\', '/'), '/') = '' THEN replace(project_name, '\', '/')
            ELSE rtrim(replace(project_name, '\', '/'), '/')
        END
    ELSE
        CASE
            WHEN rtrim(project_name, '/') = '' THEN project_name
            ELSE rtrim(project_name, '/')
        END
END
WHERE project_name IS NOT NULL AND project_name != '';

DELETE FROM daily_summaries;
INSERT OR REPLACE INTO daily_summaries (
    date, agent_name, project_name, model_name,
    total_sessions, total_input_tokens, total_output_tokens,
    total_cache_read_tokens, total_cache_creation_tokens,
    total_cost_usd, total_duration_seconds
)
SELECT
    substr(start_time, 1, 10) AS date,
    agent_name,
    project_name,
    model_resolved AS model_name,
    COUNT(*) AS total_sessions,
    COALESCE(SUM(input_tokens), 0) AS total_input_tokens,
    COALESCE(SUM(output_tokens), 0) AS total_output_tokens,
    COALESCE(SUM(cache_read_tokens), 0) AS total_cache_read_tokens,
    COALESCE(SUM(cache_creation_tokens), 0) AS total_cache_creation_tokens,
    COALESCE(SUM(net_cost_usd), 0) AS total_cost_usd,
    COALESCE(SUM(duration_seconds), 0) AS total_duration_seconds
FROM sessions
WHERE start_time IS NOT NULL AND start_time != '' AND substr(start_time, 1, 4) != '0001'
GROUP BY substr(start_time, 1, 10), agent_name, project_name, model_resolved;
