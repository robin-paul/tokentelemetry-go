package api

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/robin-paul/tokentelemetry-go/internal/events"
	"github.com/robin-paul/tokentelemetry-go/internal/pricing"
	"github.com/robin-paul/tokentelemetry-go/internal/scanner"
	"github.com/robin-paul/tokentelemetry-go/internal/store"
)

// Config encapsulates server configuration parameters.
type Config struct {
	AuthToken      string
	AllowedOrigins []string
	Version        string
	Commit         string
	WebHandler     http.Handler
	Logger         func(string, ...interface{})
}

// Server holds application state, database handle, pricing engine, SSE broker, and scanner.
type Server struct {
	db            *store.DB
	pricingEngine *pricing.Engine
	scannerEngine *scanner.Engine
	broker        *events.Broker
	cfg           Config
	mu            sync.RWMutex

	// In-memory runtime state for settings/preferences/budgets
	hiddenProjects map[string]bool
	aliases        map[string]string
	budgetsJSON    []byte
	telemetryCfg   map[string]interface{}
	powerCfg       map[string]interface{}
	billingCfg     map[string]interface{}
	billingRoute   map[string]interface{}
	summarizerCfg  map[string]interface{}
	updateCheckCfg map[string]interface{}
}

// NewServer initializes a new Server.
func NewServer(
	db *store.DB,
	pe *pricing.Engine,
	sc *scanner.Engine,
	broker *events.Broker,
	cfg Config,
) *Server {
	if cfg.Version == "" {
		cfg.Version = "1.0.0"
	}
	if cfg.Commit == "" {
		cfg.Commit = "unknown"
	}

	return &Server{
		db:             db,
		pricingEngine:  pe,
		scannerEngine:  sc,
		broker:         broker,
		cfg:            cfg,
		hiddenProjects: make(map[string]bool),
		aliases:        make(map[string]string),
		telemetryCfg: map[string]interface{}{
			"enabled":        true,
			"env_forced_off": false,
			"is_ci":          false,
			"effective":      true,
			"notice_ack":     true,
		},
		powerCfg: map[string]interface{}{
			"loadWatts":             80.0,
			"costPerKwh":            0.15,
			"gridCarbonIntensity":   400.0,
			"subscriptionEndpoints": []string{},
			"subscriptionModels":    []string{},
			"localEndpoints":        []string{},
			"referenceCloudModel":   "claude-sonnet-4-6",
			"configured":            true,
			"deviceDefault": map[string]interface{}{
				"watts":  80.0,
				"source": "apple-silicon-default",
			},
		},
		billingCfg: map[string]interface{}{
			"agents": map[string]interface{}{
				"claude": map[string]interface{}{
					"mode":          "subscription",
					"source":        "default",
					"detected":      "subscription",
					"default":       "subscription",
					"detect_source": "ANTHROPIC_API_KEY env var",
				},
			},
			"modes": []string{"subscription", "api", "local", "unknown"},
		},
		billingRoute: map[string]interface{}{
			"agents": map[string]interface{}{
				"claude": map[string]interface{}{
					"buckets": []map[string]interface{}{
						{
							"id":            "sdk_credit",
							"label":         "Agent SDK credit",
							"charges":       "included",
							"task_types":    []string{"programmatic"},
							"pool_usd":      20.0,
							"pool_requests": nil,
							"pool_period":   "month",
							"no_spillover":  true,
							"note":          "Default plan credit",
						},
					},
					"routes": map[string]interface{}{
						"interactive":  map[string]interface{}{"bucket": "subscription", "charges": "included", "warn_at": nil},
						"programmatic": map[string]interface{}{"bucket": "sdk_credit", "charges": "included", "warn_at": 20.0},
					},
					"plan": "pro",
				},
			},
			"task_types": []string{"interactive", "programmatic"},
			"charges":    []string{"included", "api_rate", "electricity"},
			"as_of":      "2026-08-21",
		},
		summarizerCfg: map[string]interface{}{
			"enabled": true,
			"backend": "ollama",
			"model":   "llama3.2:latest",
			"openai_compat": map[string]interface{}{
				"endpoint":    "http://localhost:8080/v1",
				"api_key":     nil,
				"temperature": 0.2,
			},
		},
		updateCheckCfg: map[string]interface{}{
			"enabled":        true,
			"env_forced_off": false,
			"effective":      true,
		},
	}
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
