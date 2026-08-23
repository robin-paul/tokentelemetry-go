package models

import "time"

// ClientMetadata provides machine and environment provenance for an ingestion batch.
type ClientMetadata struct {
	MachineID     string    `json:"machine_id"`
	Hostname      string    `json:"hostname"`
	ClientVersion string    `json:"client_version"`
	User          string    `json:"user,omitempty"`
	OS            string    `json:"os,omitempty"`
	SentAt        time.Time `json:"sent_at"`
	BatchID       string    `json:"batch_id"`
}

// IngestionBatch is the top-level payload transmitted from Collector to Hub over POST /api/v1/ingest.
type IngestionBatch struct {
	Metadata ClientMetadata `json:"metadata"`
	Sessions []Session      `json:"sessions"`
}

// IngestionResponse is returned by the Hub upon processing an IngestionBatch.
type IngestionResponse struct {
	Status           string    `json:"status"`
	BatchID          string    `json:"batch_id"`
	AcceptedSessions int       `json:"accepted_sessions"`
	AcceptedTurns    int       `json:"accepted_turns"`
	RejectedSessions int       `json:"rejected_sessions"`
	Errors           []string  `json:"errors,omitempty"`
	ServerTime       time.Time `json:"server_time"`
}
