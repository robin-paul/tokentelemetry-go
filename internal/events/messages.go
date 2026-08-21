package events

import (
	"encoding/json"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

// EventType defines valid real-time event names.
type EventType string

const (
	EventSessionCreated EventType = "session.created"
	EventSessionUpdated EventType = "session.updated"
	EventScanProgress   EventType = "scan.progress"
	EventStatsUpdated   EventType = "stats.updated"
)

// EventPayload represents a generic SSE payload.
type EventPayload struct {
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// SessionEventData is the payload for session events.
type SessionEventData struct {
	Session models.Session `json:"session"`
}

// ScanProgressData is the payload for filesystem scan progress.
type ScanProgressData struct {
	TotalFiles     int `json:"total_files"`
	ProcessedFiles int `json:"processed_files"`
	CurrentFile    string `json:"current_file"`
	IsComplete     bool `json:"is_complete"`
}

// ToSSE converts the payload into SSE format (event: <type>\ndata: <json>\n\n).
func (e EventPayload) ToSSE() ([]byte, error) {
	dataBytes, err := json.Marshal(e.Data)
	if err != nil {
		return nil, err
	}
	res := append([]byte("event: "+string(e.Type)+"\ndata: "), dataBytes...)
	res = append(res, []byte("\n\n")...)
	return res, nil
}
