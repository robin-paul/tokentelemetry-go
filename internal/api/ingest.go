package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/events"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

// IngestBatch processes incoming telemetry batches from remote or local collectors.
func (s *Server) IngestBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var batch models.IngestionBatch
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<20)) // 32MB payload limit
	if err := decoder.Decode(&batch); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON payload: "+err.Error())
		return
	}

	// Extract MachineID from metadata or header
	machineID := batch.Metadata.MachineID
	if machineID == "" {
		machineID = r.Header.Get("X-TT-Machine-ID")
	}
	if machineID == "" {
		respondError(w, http.StatusBadRequest, "Missing required field: metadata.machine_id")
		return
	}
	batch.Metadata.MachineID = machineID

	ctx := r.Context()
	acceptedSessions := 0
	acceptedTurns := 0
	rejectedSessions := 0
	var errorsList []string
	affectedDates := make(map[string]bool)

	// Check which sessions exist prior to upserting for SSE event distinction
	existingIDs, err := s.db.GetExistingSessionIDs(ctx, batch.Sessions)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database check failed: "+err.Error())
		return
	}

	for i := range batch.Sessions {
		sess := &batch.Sessions[i]
		if sess.ID == "" {
			if sess.FilePath != "" {
				sess.ID = sess.FilePath
			} else if sess.SessionID != "" {
				sess.ID = sess.SessionID
			} else {
				sess.ID = fmt.Sprintf("session_%d_%d", time.Now().UnixNano(), i)
			}
		}
		if sess.SessionID == "" {
			sess.SessionID = sess.ID
		}
		if sess.MachineID == "" {
			sess.MachineID = machineID
		}

		if err := s.db.SaveSessionWithTurnsAndSubagents(ctx, sess); err != nil {
			rejectedSessions++
			errorsList = append(errorsList, fmt.Sprintf("session %s: %v", sess.ID, err))
			continue
		}

		acceptedSessions++
		acceptedTurns += len(sess.Turns)

		dateStr := sess.StartTime.Format("2006-01-02")
		if dateStr == "" || dateStr == "0001-01-01" {
			dateStr = time.Now().Format("2006-01-02")
		}
		affectedDates[dateStr] = true

		// Broadcast SSE session event
		if s.broker != nil {
			eventType := events.EventSessionCreated
			if existingIDs[sess.ID] {
				eventType = events.EventSessionUpdated
			}
			_ = s.broker.Broadcast(events.EventPayload{
				Type:      eventType,
				Timestamp: time.Now().UTC(),
				Data:      events.SessionEventData{Session: *sess},
			})
		}
	}

	// Recalculate daily summaries for affected dates
	for date := range affectedDates {
		_ = s.db.RollupDailySummariesForDate(ctx, date)
	}

	// Broadcast SSE stats updated event
	if s.broker != nil && len(affectedDates) > 0 {
		_ = s.broker.Broadcast(events.EventPayload{
			Type:      events.EventStatsUpdated,
			Timestamp: time.Now().UTC(),
			Data:      map[string]interface{}{"affected_dates": affectedDates},
		})
	}

	respondJSON(w, http.StatusOK, models.IngestionResponse{
		Status:           "success",
		BatchID:          batch.Metadata.BatchID,
		AcceptedSessions: acceptedSessions,
		AcceptedTurns:    acceptedTurns,
		RejectedSessions: rejectedSessions,
		Errors:           errorsList,
		ServerTime:       time.Now().UTC(),
	})
}
