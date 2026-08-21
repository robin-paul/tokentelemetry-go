package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

var (
	ErrNotFound = errors.New("record not found")
)

// UpsertSession inserts or updates a session record.
func (d *DB) UpsertSession(ctx context.Context, s *models.Session) error {
	query := `
	INSERT INTO sessions (
		id, session_id, agent_name, project_name, file_path,
		created_at, updated_at, start_time, end_time, duration_seconds,
		model_raw, model_resolved, input_tokens, output_tokens,
		cache_read_tokens, cache_creation_tokens, gross_cost_usd, net_cost_usd,
		electricity_cost_usd, hardware_profile, status, git_branch,
		is_subagent, parent_session_id, subagent_type
	) VALUES (
		?, ?, ?, ?, ?,
		?, ?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?
	)
	ON CONFLICT(id) DO UPDATE SET
		session_id = excluded.session_id,
		agent_name = excluded.agent_name,
		project_name = excluded.project_name,
		file_path = excluded.file_path,
		updated_at = excluded.updated_at,
		start_time = excluded.start_time,
		end_time = excluded.end_time,
		duration_seconds = excluded.duration_seconds,
		model_raw = excluded.model_raw,
		model_resolved = excluded.model_resolved,
		input_tokens = excluded.input_tokens,
		output_tokens = excluded.output_tokens,
		cache_read_tokens = excluded.cache_read_tokens,
		cache_creation_tokens = excluded.cache_creation_tokens,
		gross_cost_usd = excluded.gross_cost_usd,
		net_cost_usd = excluded.net_cost_usd,
		electricity_cost_usd = excluded.electricity_cost_usd,
		hardware_profile = excluded.hardware_profile,
		status = excluded.status,
		git_branch = excluded.git_branch,
		is_subagent = excluded.is_subagent,
		parent_session_id = excluded.parent_session_id,
		subagent_type = excluded.subagent_type;
	`
	now := time.Now().UTC()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now

	return d.WithTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query,
			s.ID, s.SessionID, s.AgentName, s.ProjectName, s.FilePath,
			s.CreatedAt, s.UpdatedAt, s.StartTime, s.EndTime, s.DurationSeconds,
			s.ModelRaw, s.ModelResolved, s.InputTokens, s.OutputTokens,
			s.CacheReadTokens, s.CacheCreationTokens, s.GrossCostUSD, s.NetCostUSD,
			s.ElectricityCostUSD, s.HardwareProfile, s.Status, s.GitBranch,
			s.IsSubagent, s.ParentSessionID, s.SubagentType,
		)
		return err
	})
}

// SaveSessionWithTurnsAndSubagents atomically saves a session and replaces its message turns and subagent runs.
func (d *DB) SaveSessionWithTurnsAndSubagents(ctx context.Context, s *models.Session) error {
	return d.WithTx(ctx, func(tx *sql.Tx) error {
		// 1. Upsert session
		upsertSessionQuery := `
		INSERT INTO sessions (
			id, session_id, agent_name, project_name, file_path,
			created_at, updated_at, start_time, end_time, duration_seconds,
			model_raw, model_resolved, input_tokens, output_tokens,
			cache_read_tokens, cache_creation_tokens, gross_cost_usd, net_cost_usd,
			electricity_cost_usd, hardware_profile, status, git_branch,
			is_subagent, parent_session_id, subagent_type
		) VALUES (
			?, ?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?
		)
		ON CONFLICT(id) DO UPDATE SET
			session_id = excluded.session_id,
			agent_name = excluded.agent_name,
			project_name = excluded.project_name,
			file_path = excluded.file_path,
			updated_at = excluded.updated_at,
			start_time = excluded.start_time,
			end_time = excluded.end_time,
			duration_seconds = excluded.duration_seconds,
			model_raw = excluded.model_raw,
			model_resolved = excluded.model_resolved,
			input_tokens = excluded.input_tokens,
			output_tokens = excluded.output_tokens,
			cache_read_tokens = excluded.cache_read_tokens,
			cache_creation_tokens = excluded.cache_creation_tokens,
			gross_cost_usd = excluded.gross_cost_usd,
			net_cost_usd = excluded.net_cost_usd,
			electricity_cost_usd = excluded.electricity_cost_usd,
			hardware_profile = excluded.hardware_profile,
			status = excluded.status,
			git_branch = excluded.git_branch,
			is_subagent = excluded.is_subagent,
			parent_session_id = excluded.parent_session_id,
			subagent_type = excluded.subagent_type;
		`
		now := time.Now().UTC()
		if s.CreatedAt.IsZero() {
			s.CreatedAt = now
		}
		s.UpdatedAt = now

		_, err := tx.ExecContext(ctx, upsertSessionQuery,
			s.ID, s.SessionID, s.AgentName, s.ProjectName, s.FilePath,
			s.CreatedAt, s.UpdatedAt, s.StartTime, s.EndTime, s.DurationSeconds,
			s.ModelRaw, s.ModelResolved, s.InputTokens, s.OutputTokens,
			s.CacheReadTokens, s.CacheCreationTokens, s.GrossCostUSD, s.NetCostUSD,
			s.ElectricityCostUSD, s.HardwareProfile, s.Status, s.GitBranch,
			s.IsSubagent, s.ParentSessionID, s.SubagentType,
		)
		if err != nil {
			return fmt.Errorf("failed to upsert session %s: %w", s.ID, err)
		}

		// 2. Replace message turns
		if len(s.Turns) > 0 {
			if _, err := tx.ExecContext(ctx, `DELETE FROM message_turns WHERE session_id = ?;`, s.ID); err != nil {
				return fmt.Errorf("failed to clean existing message turns: %w", err)
			}
			turnStmt, err := tx.PrepareContext(ctx, `
				INSERT INTO message_turns (
					id, session_id, turn_index, timestamp, role,
					model_name, input_tokens, output_tokens, cache_read_tokens,
					cache_creation_tokens, cost_usd, tools_invoked_json
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
			`)
			if err != nil {
				return fmt.Errorf("failed to prepare message turn insert: %w", err)
			}
			defer turnStmt.Close()

			for _, t := range s.Turns {
				toolsJSON := t.ToolsInvokedJSON
				if toolsJSON == "" {
					if len(t.ToolsInvoked) > 0 {
						b, _ := json.Marshal(t.ToolsInvoked)
						toolsJSON = string(b)
					} else {
						toolsJSON = "[]"
					}
				}
				if _, err := turnStmt.ExecContext(ctx,
					t.ID, s.ID, t.TurnIndex, t.Timestamp, t.Role,
					t.ModelName, t.InputTokens, t.OutputTokens, t.CacheReadTokens,
					t.CacheCreationTokens, t.CostUSD, toolsJSON,
				); err != nil {
					return fmt.Errorf("failed to insert message turn: %w", err)
				}
			}
		}

		// 3. Insert Subagent runs if any
		if len(s.SubagentRuns) > 0 {
			subStmt, err := tx.PrepareContext(ctx, `
				INSERT INTO subagent_runs (
					id, parent_session_id, child_session_id, agent_type, tokens, cost_usd, created_at
				) VALUES (?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(child_session_id) DO UPDATE SET
					tokens = excluded.tokens,
					cost_usd = excluded.cost_usd;
			`)
			if err != nil {
				return fmt.Errorf("failed to prepare subagent insert: %w", err)
			}
			defer subStmt.Close()

			for _, sub := range s.SubagentRuns {
				if sub.CreatedAt.IsZero() {
					sub.CreatedAt = now
				}
				if _, err := subStmt.ExecContext(ctx,
					sub.ID, s.ID, sub.ChildSessionID, sub.AgentType, sub.Tokens, sub.CostUSD, sub.CreatedAt,
				); err != nil {
					return fmt.Errorf("failed to insert subagent run: %w", err)
				}
			}
		}

		return nil
	})
}

// GetSession retrieves a single session by primary ID.
func (d *DB) GetSession(ctx context.Context, id string) (*models.Session, error) {
	query := `
	SELECT
		id, session_id, agent_name, project_name, file_path,
		created_at, updated_at, start_time, end_time, duration_seconds,
		model_raw, model_resolved, input_tokens, output_tokens,
		cache_read_tokens, cache_creation_tokens, gross_cost_usd, net_cost_usd,
		electricity_cost_usd, hardware_profile, status, git_branch,
		is_subagent, parent_session_id, subagent_type
	FROM sessions
	WHERE id = ? OR session_id = ?
	LIMIT 1;
	`
	row := d.readerDB.QueryRowContext(ctx, query, id, id)

	var s models.Session
	var isSubagentInt int
	err := row.Scan(
		&s.ID, &s.SessionID, &s.AgentName, &s.ProjectName, &s.FilePath,
		&s.CreatedAt, &s.UpdatedAt, &s.StartTime, &s.EndTime, &s.DurationSeconds,
		&s.ModelRaw, &s.ModelResolved, &s.InputTokens, &s.OutputTokens,
		&s.CacheReadTokens, &s.CacheCreationTokens, &s.GrossCostUSD, &s.NetCostUSD,
		&s.ElectricityCostUSD, &s.HardwareProfile, &s.Status, &s.GitBranch,
		&isSubagentInt, &s.ParentSessionID, &s.SubagentType,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	s.IsSubagent = isSubagentInt == 1
	return &s, nil
}

// GetSessionDetail fetches a full session along with all message turns and subagent runs.
func (d *DB) GetSessionDetail(ctx context.Context, id string) (*models.Session, error) {
	s, err := d.GetSession(ctx, id)
	if err != nil {
		return nil, err
	}

	// 1. Fetch turns
	turns, err := d.GetMessageTurns(ctx, s.ID)
	if err != nil {
		return nil, err
	}
	s.Turns = turns

	// 2. Fetch subagent runs
	subagents, err := d.GetSubagentRuns(ctx, s.ID)
	if err != nil {
		return nil, err
	}
	s.SubagentRuns = subagents

	return s, nil
}

// GetMessageTurns fetches all message turns for a session in sequential order.
func (d *DB) GetMessageTurns(ctx context.Context, sessionID string) ([]models.MessageTurn, error) {
	query := `
	SELECT
		id, session_id, turn_index, timestamp, role,
		model_name, input_tokens, output_tokens, cache_read_tokens,
		cache_creation_tokens, cost_usd, tools_invoked_json
	FROM message_turns
	WHERE session_id = ?
	ORDER BY turn_index ASC;
	`
	rows, err := d.readerDB.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query message turns: %w", err)
	}
	defer rows.Close()

	var turns []models.MessageTurn
	for rows.Next() {
		var t models.MessageTurn
		if err := rows.Scan(
			&t.ID, &t.SessionID, &t.TurnIndex, &t.Timestamp, &t.Role,
			&t.ModelName, &t.InputTokens, &t.OutputTokens, &t.CacheReadTokens,
			&t.CacheCreationTokens, &t.CostUSD, &t.ToolsInvokedJSON,
		); err != nil {
			return nil, fmt.Errorf("failed to scan message turn: %w", err)
		}
		if t.ToolsInvokedJSON != "" {
			_ = json.Unmarshal([]byte(t.ToolsInvokedJSON), &t.ToolsInvoked)
		}
		turns = append(turns, t)
	}
	return turns, rows.Err()
}

// GetSubagentRuns fetches all subagent runs linked to a parent session.
func (d *DB) GetSubagentRuns(ctx context.Context, parentSessionID string) ([]models.SubagentRun, error) {
	query := `
	SELECT
		id, parent_session_id, child_session_id, agent_type, tokens, cost_usd, created_at
	FROM subagent_runs
	WHERE parent_session_id = ?
	ORDER BY created_at ASC;
	`
	rows, err := d.readerDB.QueryContext(ctx, query, parentSessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query subagents: %w", err)
	}
	defer rows.Close()

	var runs []models.SubagentRun
	for rows.Next() {
		var r models.SubagentRun
		if err := rows.Scan(
			&r.ID, &r.ParentSessionID, &r.ChildSessionID, &r.AgentType, &r.Tokens, &r.CostUSD, &r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan subagent run: %w", err)
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

// ListSessions queries sessions with filtering and pagination.
func (d *DB) ListSessions(ctx context.Context, params models.FilterParams) ([]models.Session, int64, error) {
	var whereClauses []string
	var args []interface{}

	if params.Agent != "" {
		whereClauses = append(whereClauses, "agent_name = ?")
		args = append(args, params.Agent)
	}
	if params.Project != "" {
		whereClauses = append(whereClauses, "project_name = ?")
		args = append(args, params.Project)
	}
	if params.Model != "" {
		whereClauses = append(whereClauses, "(model_raw = ? OR model_resolved = ?)")
		args = append(args, params.Model, params.Model)
	}
	if !params.StartDate.IsZero() {
		whereClauses = append(whereClauses, "start_time >= ?")
		args = append(args, params.StartDate)
	}
	if !params.EndDate.IsZero() {
		whereClauses = append(whereClauses, "start_time <= ?")
		args = append(args, params.EndDate)
	}
	if params.Search != "" {
		whereClauses = append(whereClauses, "(session_id LIKE ? OR project_name LIKE ? OR model_raw LIKE ?)")
		pattern := "%" + params.Search + "%"
		args = append(args, pattern, pattern, pattern)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 1. Total Count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM sessions %s;", whereSQL)
	var total int64
	if err := d.readerDB.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count sessions: %w", err)
	}

	// 2. Paginated Query
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	page := params.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	query := fmt.Sprintf(`
	SELECT
		id, session_id, agent_name, project_name, file_path,
		created_at, updated_at, start_time, end_time, duration_seconds,
		model_raw, model_resolved, input_tokens, output_tokens,
		cache_read_tokens, cache_creation_tokens, gross_cost_usd, net_cost_usd,
		electricity_cost_usd, hardware_profile, status, git_branch,
		is_subagent, parent_session_id, subagent_type
	FROM sessions
	%s
	ORDER BY start_time DESC
	LIMIT ? OFFSET ?;
	`, whereSQL)

	queryArgs := append(args, limit, offset)
	rows, err := d.readerDB.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var s models.Session
		var isSubagentInt int
		if err := rows.Scan(
			&s.ID, &s.SessionID, &s.AgentName, &s.ProjectName, &s.FilePath,
			&s.CreatedAt, &s.UpdatedAt, &s.StartTime, &s.EndTime, &s.DurationSeconds,
			&s.ModelRaw, &s.ModelResolved, &s.InputTokens, &s.OutputTokens,
			&s.CacheReadTokens, &s.CacheCreationTokens, &s.GrossCostUSD, &s.NetCostUSD,
			&s.ElectricityCostUSD, &s.HardwareProfile, &s.Status, &s.GitBranch,
			&isSubagentInt, &s.ParentSessionID, &s.SubagentType,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan session row: %w", err)
		}
		s.IsSubagent = isSubagentInt == 1
		sessions = append(sessions, s)
	}

	return sessions, total, rows.Err()
}

// GetRecentSessions returns the most recent N sessions.
func (d *DB) GetRecentSessions(ctx context.Context, limit int) ([]models.Session, error) {
	if limit <= 0 {
		limit = 20
	}
	sessions, _, err := d.ListSessions(ctx, models.FilterParams{
		Page:  1,
		Limit: limit,
	})
	return sessions, err
}

// DeleteSession removes a session by ID (cascades to turns and subagent runs).
func (d *DB) DeleteSession(ctx context.Context, id string) error {
	return d.WithTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?;`, id)
		if err != nil {
			return fmt.Errorf("failed to delete session %s: %w", id, err)
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			return ErrNotFound
		}
		return nil
	})
}
