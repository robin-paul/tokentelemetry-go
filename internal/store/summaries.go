package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

// UpsertDailySummary inserts or updates a pre-aggregated daily summary entry.
func (d *DB) UpsertDailySummary(ctx context.Context, s *models.DailySummary) error {
	query := `
	INSERT INTO daily_summaries (
		date, agent_name, project_name, model_name,
		total_sessions, total_input_tokens, total_output_tokens,
		total_cache_read_tokens, total_cache_creation_tokens,
		total_cost_usd, total_duration_seconds
	) VALUES (
		?, ?, ?, ?,
		?, ?, ?,
		?, ?,
		?, ?
	)
	ON CONFLICT(date, agent_name, project_name, model_name) DO UPDATE SET
		total_sessions = total_sessions + excluded.total_sessions,
		total_input_tokens = total_input_tokens + excluded.total_input_tokens,
		total_output_tokens = total_output_tokens + excluded.total_output_tokens,
		total_cache_read_tokens = total_cache_read_tokens + excluded.total_cache_read_tokens,
		total_cache_creation_tokens = total_cache_creation_tokens + excluded.total_cache_creation_tokens,
		total_cost_usd = total_cost_usd + excluded.total_cost_usd,
		total_duration_seconds = total_duration_seconds + excluded.total_duration_seconds;
	`
	return d.WithTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query,
			s.Date, s.AgentName, s.ProjectName, s.ModelName,
			s.TotalSessions, s.TotalInputTokens, s.TotalOutputTokens,
			s.TotalCacheReadTokens, s.TotalCacheCreationTokens,
			s.TotalCostUSD, s.TotalDurationSeconds,
		)
		return err
	})
}

// QueryDailySummaries retrieves time-series summaries filtered by date range and dimensions.
func (d *DB) QueryDailySummaries(ctx context.Context, from, to, agent, project, model string) ([]models.DailySummary, error) {
	var whereClauses []string
	var args []interface{}

	if from != "" {
		whereClauses = append(whereClauses, "date >= ?")
		args = append(args, from)
	}
	if to != "" {
		whereClauses = append(whereClauses, "date <= ?")
		args = append(args, to)
	}
	if agent != "" {
		whereClauses = append(whereClauses, "agent_name = ?")
		args = append(args, agent)
	}
	if project != "" {
		whereClauses = append(whereClauses, "project_name = ?")
		args = append(args, project)
	}
	if model != "" {
		whereClauses = append(whereClauses, "model_name = ?")
		args = append(args, model)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	query := fmt.Sprintf(`
	SELECT
		date, agent_name, project_name, model_name,
		total_sessions, total_input_tokens, total_output_tokens,
		total_cache_read_tokens, total_cache_creation_tokens,
		total_cost_usd, total_duration_seconds
	FROM daily_summaries
	%s
	ORDER BY date ASC;
	`, whereSQL)

	rows, err := d.readerDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily summaries: %w", err)
	}
	defer rows.Close()

	var results []models.DailySummary
	for rows.Next() {
		var s models.DailySummary
		if err := rows.Scan(
			&s.Date, &s.AgentName, &s.ProjectName, &s.ModelName,
			&s.TotalSessions, &s.TotalInputTokens, &s.TotalOutputTokens,
			&s.TotalCacheReadTokens, &s.TotalCacheCreationTokens,
			&s.TotalCostUSD, &s.TotalDurationSeconds,
		); err != nil {
			return nil, fmt.Errorf("failed to scan daily summary row: %w", err)
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

// GetStatsOverview calculates system-wide overview statistics.
func (d *DB) GetStatsOverview(ctx context.Context, from, to, agent, project string) (*models.StatsOverview, error) {
	var whereClauses []string
	var args []interface{}

	if from != "" {
		whereClauses = append(whereClauses, "start_time >= ?")
		args = append(args, from)
	}
	if to != "" {
		whereClauses = append(whereClauses, "start_time <= ?")
		args = append(args, to)
	}
	if agent != "" {
		whereClauses = append(whereClauses, "agent_name = ?")
		args = append(args, agent)
	}
	if project != "" {
		whereClauses = append(whereClauses, "project_name = ?")
		args = append(args, project)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	query := fmt.Sprintf(`
	SELECT
		COUNT(*) AS total_sessions,
		COALESCE(SUM(input_tokens), 0) AS total_input_tokens,
		COALESCE(SUM(output_tokens), 0) AS total_output_tokens,
		COALESCE(SUM(gross_cost_usd), 0) AS gross_cost_usd,
		COALESCE(SUM(net_cost_usd), 0) AS net_cost_usd,
		COUNT(DISTINCT agent_name) AS active_agents,
		COUNT(DISTINCT project_name) AS active_projects
	FROM sessions
	%s;
	`, whereSQL)

	var stats models.StatsOverview
	err := d.readerDB.QueryRowContext(ctx, query, args...).Scan(
		&stats.TotalSessions,
		&stats.TotalInputTokens,
		&stats.TotalOutputTokens,
		&stats.GrossCostUSD,
		&stats.NetCostUSD,
		&stats.ActiveAgents,
		&stats.ActiveProjects,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate stats overview: %w", err)
	}
	stats.TotalTokens = stats.TotalInputTokens + stats.TotalOutputTokens

	return &stats, nil
}

// GetLeaderboard retrieves top models and top agents ranked by total token consumption.
func (d *DB) GetLeaderboard(ctx context.Context, limit int, from, to string) ([]models.LeaderboardEntry, []models.LeaderboardEntry, error) {
	if limit <= 0 {
		limit = 10
	}

	var whereClauses []string
	var args []interface{}
	if from != "" {
		whereClauses = append(whereClauses, "start_time >= ?")
		args = append(args, from)
	}
	if to != "" {
		whereClauses = append(whereClauses, "start_time <= ?")
		args = append(args, to)
	}
	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 1. Top Models
	modelQuery := fmt.Sprintf(`
	SELECT
		model_resolved AS name,
		COALESCE(SUM(input_tokens + output_tokens + cache_read_tokens + cache_creation_tokens), 0) AS total_tokens,
		COALESCE(SUM(net_cost_usd), 0) AS total_cost_usd,
		COUNT(*) AS session_count
	FROM sessions
	%s
	GROUP BY model_resolved
	ORDER BY total_tokens DESC
	LIMIT ?;
	`, whereSQL)

	modelArgs := append(args, limit)
	rows, err := d.readerDB.QueryContext(ctx, modelQuery, modelArgs...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query model leaderboard: %w", err)
	}
	defer rows.Close()

	var modelsList []models.LeaderboardEntry
	for rows.Next() {
		var entry models.LeaderboardEntry
		if err := rows.Scan(&entry.Name, &entry.TotalTokens, &entry.TotalCostUSD, &entry.SessionCount); err != nil {
			return nil, nil, fmt.Errorf("failed to scan model entry: %w", err)
		}
		modelsList = append(modelsList, entry)
	}

	// 2. Top Agents
	agentQuery := fmt.Sprintf(`
	SELECT
		agent_name AS name,
		COALESCE(SUM(input_tokens + output_tokens + cache_read_tokens + cache_creation_tokens), 0) AS total_tokens,
		COALESCE(SUM(net_cost_usd), 0) AS total_cost_usd,
		COUNT(*) AS session_count
	FROM sessions
	%s
	GROUP BY agent_name
	ORDER BY total_tokens DESC
	LIMIT ?;
	`, whereSQL)

	agentArgs := append(args, limit)
	agentRows, err := d.readerDB.QueryContext(ctx, agentQuery, agentArgs...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query agent leaderboard: %w", err)
	}
	defer agentRows.Close()

	var agentsList []models.LeaderboardEntry
	for agentRows.Next() {
		var entry models.LeaderboardEntry
		if err := agentRows.Scan(&entry.Name, &entry.TotalTokens, &entry.TotalCostUSD, &entry.SessionCount); err != nil {
			return nil, nil, fmt.Errorf("failed to scan agent entry: %w", err)
		}
		agentsList = append(agentsList, entry)
	}

	return modelsList, agentsList, nil
}

// GetProjects lists all discovered projects with token and cost aggregations.
func (d *DB) GetProjects(ctx context.Context) ([]models.ProjectSummary, error) {
	query := `
	SELECT
		project_name,
		COUNT(*) AS session_count,
		COALESCE(SUM(input_tokens + output_tokens + cache_read_tokens + cache_creation_tokens), 0) AS total_tokens,
		COALESCE(SUM(net_cost_usd), 0) AS total_cost_usd,
		MAX(start_time) AS last_active
	FROM sessions
	GROUP BY project_name
	ORDER BY last_active DESC;
	`
	rows, err := d.readerDB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var projects []models.ProjectSummary
	for rows.Next() {
		var p models.ProjectSummary
		var lastActiveRaw interface{}
		if err := rows.Scan(&p.ProjectName, &p.SessionCount, &p.TotalTokens, &p.TotalCostUSD, &lastActiveRaw); err != nil {
			return nil, fmt.Errorf("failed to scan project summary: %w", err)
		}
		p.LastActive = parseTime(lastActiveRaw)
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// GetProjectDetail retrieves project aggregations and recent sessions for a specific project.
func (d *DB) GetProjectDetail(ctx context.Context, projectName string) (*models.ProjectSummary, []models.Session, error) {
	query := `
	SELECT
		project_name,
		COUNT(*) AS session_count,
		COALESCE(SUM(input_tokens + output_tokens + cache_read_tokens + cache_creation_tokens), 0) AS total_tokens,
		COALESCE(SUM(net_cost_usd), 0) AS total_cost_usd,
		MAX(start_time) AS last_active
	FROM sessions
	WHERE project_name = ?
	GROUP BY project_name;
	`
	var p models.ProjectSummary
	var lastActiveRaw interface{}
	err := d.readerDB.QueryRowContext(ctx, query, projectName).Scan(
		&p.ProjectName, &p.SessionCount, &p.TotalTokens, &p.TotalCostUSD, &lastActiveRaw,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get project summary: %w", err)
	}
	p.LastActive = parseTime(lastActiveRaw)

	sessions, _, err := d.ListSessions(ctx, models.FilterParams{
		Project: projectName,
		Limit:   50,
	})
	if err != nil {
		return nil, nil, err
	}

	return &p, sessions, nil
}

func parseTime(v interface{}) time.Time {
	switch val := v.(type) {
	case time.Time:
		return val
	case string:
		layouts := []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02 15:04:05.999999999-07:00",
			"2006-01-02 15:04:05.999999999",
			"2006-01-02 15:04:05-07:00",
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
		}
		for _, layout := range layouts {
			if t, err := time.Parse(layout, val); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}
