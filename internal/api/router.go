package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Router returns an http.Handler with all TokenTelemetry routes and middleware configured.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	// 1. Core Middleware
	r.Use(middleware.Recoverer)
	if s.cfg.Logger != nil {
		r.Use(RequestLogger(s.cfg.Logger))
	}
	r.Use(CORSMiddleware(s.cfg.AllowedOrigins))
	r.Use(RemoteAuthMiddleware(s.cfg.AuthToken))

	// 2. Health & System
	r.Get("/healthz", s.Healthz)
	r.Get("/version", s.Version)
	r.Get("/agents", s.Agents)
	r.Get("/remote-access", s.RemoteAccess)

	if s.cfg.WebHandler != nil {
		r.Get("/", s.cfg.WebHandler.ServeHTTP)
		r.Get("/api", s.Root)
	} else {
		r.Get("/", s.Root)
	}

	// 3. Real-Time SSE Stream
	if s.broker != nil {
		r.Get("/events", s.broker.ServeHTTP)
		r.Get("/api/events", s.broker.ServeHTTP)
	}

	// 4. REST API routes (mounted under /api and root aliases)
	registerAPIRoutes := func(router chi.Router) {
		// Sessions
		router.Get("/sessions", s.ListSessions)
		router.Get("/sessions/{id}", s.GetSession)
		router.Delete("/sessions/{id}", s.DeleteSession)
		router.Get("/recent", s.GetRecentSessions)
		router.Get("/sessions/{id}/subagents/{subagent_id}/trace", s.GetSubagentTrace)
		router.Get("/sessions/{id}/delegation", s.GetDelegation)
		router.Get("/sessions/{id}/grok-forensics", s.GetGrokForensics)
		router.Get("/sessions/{id}/hermes-overlay", s.GetHermesOverlay)

		// Stats & Analytics
		router.Get("/stats", s.GetStats)
		router.Get("/stats/daily", s.GetDailyStats)
		router.Get("/leaderboard", s.GetLeaderboard)
		router.Get("/analytics", s.GetAnalytics)

		// Projects
		router.Get("/projects", s.GetProjects)
		router.Get("/projects/*", s.GetProjectDetail)

		// Pricing
		router.Get("/pricing", s.GetPricing)
		router.Post("/pricing/override", s.UpsertPricingOverride)
		router.Put("/pricing/override", s.UpsertPricingOverride)
		router.Delete("/pricing/override/{pattern}", s.DeletePricingOverride)

		// Hermes
		router.Get("/hermes/kanban", s.GetHermesKanban)
	}

	// Mount /api subtree
	r.Route("/api", registerAPIRoutes)

	// Root-level aliases for headless API testing or specific sub-resources
	if s.cfg.WebHandler == nil {
		registerAPIRoutes(r)
	} else {
		r.Get("/sessions/{id}/subagents/{subagent_id}/trace", s.GetSubagentTrace)
		r.Get("/sessions/{id}/delegation", s.GetDelegation)
		r.Get("/sessions/{id}/grok-forensics", s.GetGrokForensics)
		r.Get("/sessions/{id}/hermes-overlay", s.GetHermesOverlay)
	}

	// Additional Root Endpoints
	r.Get("/artifacts", s.GetArtifact)
	r.Head("/artifacts", s.GetArtifact)

	// Budgets
	r.Get("/budgets", s.GetBudgets)
	r.Put("/budgets", s.SetBudgets)

	// Notifications
	r.Get("/notifications", s.GetNotifications)
	r.Post("/notifications/toasted", s.MarkNotificationsToasted)
	r.Post("/notifications/read", s.MarkNotificationsRead)
	r.Post("/notifications/clear", s.MarkNotificationsCleared)

	// Configuration & Hardware
	r.Get("/config", s.GetConfig)
	r.Get("/config/hidden", s.GetHiddenProjects)
	r.Post("/config/hide", s.HideProject)
	r.Post("/config/unhide", s.UnhideProject)
	r.Get("/config/aliases", s.GetAliases)
	r.Post("/config/aliases", s.SetAliases)
	r.Get("/config/update-check", s.GetUpdateCheck)
	r.Post("/config/update-check", s.SetUpdateCheck)
	r.Get("/config/telemetry", s.GetTelemetryConfig)
	r.Post("/config/telemetry", s.SetTelemetryConfig)
	r.Post("/config/telemetry/ack", s.AckTelemetry)
	r.Get("/config/telemetry/preview", s.GetTelemetryPreview)
	r.Post("/telemetry/event", s.PostTelemetryEvent)
	r.Get("/config/retention", s.GetRetentionConfig)
	r.Post("/config/retention", s.SetRetentionConfig)
	r.Delete("/history/transcripts", s.DeleteTranscripts)
	r.Get("/config/power", s.GetPowerConfig)
	r.Put("/config/power", s.SetPowerConfig)
	r.Get("/config/power/meter", s.GetPowerMeter)
	r.Post("/config/power/calibrate", s.CalibratePower)
	r.Get("/config/billing", s.GetBillingConfig)
	r.Put("/config/billing", s.SetBillingConfig)
	r.Get("/config/billing-route", s.GetBillingRouteConfig)
	r.Put("/config/billing-route", s.SetBillingRouteConfig)
	r.Get("/config/agent-features", s.GetAgentFeatures)

	// Summarizer
	r.Get("/summarizer/available", s.GetSummarizerAvailable)
	r.Get("/config/summarizer", s.GetSummarizerConfig)
	r.Put("/config/summarizer", s.SetSummarizerConfig)
	r.Get("/summarizer/ollama/models", s.GetOllamaModels)
	r.Get("/summarizer/codex/models", s.GetCodexModels)
	r.Post("/summarizer/openai-compat/test", s.TestOpenAICompat)
	r.Get("/sessions/{id}/summary", s.GetSessionSummary)
	r.Post("/sessions/{id}/summary", s.GenerateSessionSummary)
	r.Post("/summaries/recent", s.SummarizeRecent)

	// Hermes Subsystem
	r.Get("/hermes/overview", s.GetHermesOverview)
	r.Get("/hermes/telemetry", s.GetHermesTelemetry)
	r.Get("/hermes/sessions", s.GetHermesSessions)
	r.Get("/hermes/skills", s.GetHermesSkills)
	r.Get("/hermes/memory", s.GetHermesMemory)
	r.Get("/hermes/soul", s.GetHermesSoul)
	r.Get("/hermes/profiles", s.GetHermesProfiles)
	r.Get("/hermes/tools", s.GetHermesTools)

	// DSH & Cache
	r.Get("/dsh/lifecycle", s.GetDSHLifecycle)
	r.Get("/cache/status", s.GetCacheStatus)
	r.Post("/cache/invalidate", s.InvalidateCache)

	// 5. Embedded Static Web Handler Fallback
	if s.cfg.WebHandler != nil {
		r.NotFound(s.cfg.WebHandler.ServeHTTP)
	}

	return r
}
