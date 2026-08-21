package api

import (
	"net/http"
)

// Notification represents a system alert.
type Notification struct {
	ID        int64  `json:"id"`
	Kind      string `json:"kind"`
	DedupKey  string `json:"dedup_key"`
	Severity  string `json:"severity"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Href      string `json:"href"`
	CreatedAt string `json:"created_at"`
	Toasted   bool   `json:"toasted"`
	Read      bool   `json:"read"`
	Cleared   bool   `json:"cleared"`
}

// GetNotifications handles GET /notifications.
func (s *Server) GetNotifications(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"notifications": []Notification{},
		"unread_count":  0,
		"to_toast":      []interface{}{},
	})
}

// MarkNotificationsToasted handles POST /notifications/toasted.
func (s *Server) MarkNotificationsToasted(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok":      true,
		"updated": 0,
	})
}

// MarkNotificationsRead handles POST /notifications/read.
func (s *Server) MarkNotificationsRead(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok":      true,
		"updated": 0,
	})
}

// MarkNotificationsCleared handles POST /notifications/clear.
func (s *Server) MarkNotificationsCleared(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok":      true,
		"updated": 0,
	})
}
