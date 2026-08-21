package pricing_test

import (
	"context"
	"math"
	"testing"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
	"github.com/robin-paul/tokentelemetry-go/internal/pricing"
)

func almostEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}

func TestDatasetLoading(t *testing.T) {
	ds, err := pricing.LoadEmbeddedDataset()
	if err != nil {
		t.Fatalf("failed to load embedded dataset: %v", err)
	}

	rate, resolved := pricing.NewResolver(ds).Resolve("claude-3-7-sonnet", "", nil)
	if resolved != "claude-3-7-sonnet" || rate.InputCostPerM != 3.00 || rate.OutputCostPerM != 15.00 {
		t.Errorf("unexpected rate for claude-3-7-sonnet: %+v, resolved=%s", rate, resolved)
	}
}

func TestModelResolution(t *testing.T) {
	engine, err := pricing.NewEngine()
	if err != nil {
		t.Fatalf("failed to initialize pricing engine: %v", err)
	}

	tests := []struct {
		name          string
		rawModel      string
		provider      string
		overrides     []models.PricingOverride
		wantResolved  string
		wantInputCost float64
	}{
		{
			name:          "Exact Curated Match",
			rawModel:      "claude-3-7-sonnet",
			wantResolved:  "claude-3-7-sonnet",
			wantInputCost: 3.00,
		},
		{
			name:          "Provider Prefix Stripping (Anthropic)",
			rawModel:      "anthropic/claude-3-7-sonnet-20250219",
			wantResolved:  "claude-3-7-sonnet",
			wantInputCost: 3.00,
		},
		{
			name:          "Bedrock US Prefix Stripping",
			rawModel:      "us.anthropic.claude-3-5-haiku-20241022-v1:0",
			wantResolved:  "claude-3-5-haiku",
			wantInputCost: 0.80,
		},
		{
			name:          "OpenAI Prefix Stripping",
			rawModel:      "openai/gpt-4o-2024-08-06",
			wantResolved:  "gpt-4o",
			wantInputCost: 2.50,
		},
		{
			name:          "Gemini Match",
			rawModel:      "gemini-2.5-pro",
			wantResolved:  "gemini-2.5-pro",
			wantInputCost: 1.25,
		},
		{
			name:          "User Override Priority",
			rawModel:      "custom-llama-3",
			overrides: []models.PricingOverride{
				{
					ModelPattern:  "custom-llama-3",
					InputCostPerM: 8.50,
				},
			},
			wantResolved:  "custom-llama-3",
			wantInputCost: 8.50,
		},
		{
			name:          "Unknown Model Fallback",
			rawModel:      "nonexistent-alien-model-x99",
			wantResolved:  "_default",
			wantInputCost: 2.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate, resolved := engine.ResolveModel(tt.rawModel, tt.provider, tt.overrides)
			if resolved != tt.wantResolved {
				t.Errorf("ResolveModel(%q) resolved=%q; want %q", tt.rawModel, resolved, tt.wantResolved)
			}
			if rate.InputCostPerM != tt.wantInputCost {
				t.Errorf("ResolveModel(%q) inputCost=%.2f; want %.2f", tt.rawModel, rate.InputCostPerM, tt.wantInputCost)
			}
		})
	}
}

func TestCalculateCost(t *testing.T) {
	rate := models.ModelRate{
		ModelPattern:      "claude-3-7-sonnet",
		InputCostPerM:     3.00,
		OutputCostPerM:    15.00,
		CacheReadCostPerM: 0.30,
	}

	usage := models.TokenUsage{
		InputTokens:         1_000_000,
		OutputTokens:        1_000_000,
		CacheReadTokens:     2_000_000,
		CacheCreationTokens: 500_000,
	}

	// Gross: (1M + 2M + 0.5M) * $3 + 1M * $15 = $10.5 + $15 = $25.50
	// Net: 1M * $3 + 2M * $0.30 + 0.5M * $3.75 (125%) + 1M * $15 = $3 + $0.60 + $1.875 + $15 = $20.475
	gross, net := pricing.CalculateCost(usage, rate)

	if !almostEqual(gross, 25.50, 1e-4) {
		t.Errorf("expected gross cost 25.50, got %.4f", gross)
	}
	if !almostEqual(net, 20.475, 1e-4) {
		t.Errorf("expected net cost 20.475, got %.4f", net)
	}
}

func TestPowerAndElectricity(t *testing.T) {
	// 1. Local session detection
	if !pricing.IsLocalSession("local", "", "") {
		t.Errorf("expected local billingMode to be detected")
	}
	if !pricing.IsLocalSession("", "http://localhost:11434", "") {
		t.Errorf("expected localhost endpoint to be detected")
	}
	if !pricing.IsLocalSession("", "", "ollama") {
		t.Errorf("expected ollama provider to be detected")
	}
	if pricing.IsLocalSession("", "https://api.anthropic.com", "anthropic") {
		t.Errorf("expected cloud session not to be local")
	}

	// 2. Throughput estimation
	tp8b := pricing.EstimateThroughput("llama-3.1-8b-instruct")
	if tp8b != 70.0 {
		t.Errorf("expected 8b throughput 70 tok/s, got %.1f", tp8b)
	}

	tp70b := pricing.EstimateThroughput("deepseek-r1-70b")
	if tp70b != 18.0 {
		t.Errorf("expected 70b throughput 18 tok/s, got %.1f", tp70b)
	}

	// 3. Electricity cost
	profile := models.PowerConfig{
		LoadWatts:     100.0,
		CostPerKWhUSD: 0.20,
	}
	// 3600 seconds at 100W = 0.1 kWh * $0.20 = $0.02
	cost := pricing.CalculateElectricityCost(0, 3600, profile, "custom-model")
	if !almostEqual(cost, 0.02, 1e-4) {
		t.Errorf("expected electricity cost 0.02, got %.6f", cost)
	}
}

func TestCostSession(t *testing.T) {
	engine, err := pricing.NewEngine()
	if err != nil {
		t.Fatalf("failed to initialize pricing engine: %v", err)
	}

	ctx := context.Background()

	// Cloud session
	cloudSess := &models.Session{
		ID:                  "sess-cloud",
		ModelRaw:            "claude-3-7-sonnet-20250219",
		InputTokens:         10_000,
		OutputTokens:        2_000,
		CacheReadTokens:     5_000,
		CacheCreationTokens: 1_000,
		Turns: []models.MessageTurn{
			{
				ID:           "turn-1",
				ModelName:    "claude-3-7-sonnet",
				InputTokens:  10_000,
				OutputTokens: 2_000,
			},
		},
	}

	engine.CostSession(ctx, cloudSess, "", "anthropic", nil)

	if cloudSess.ModelResolved != "claude-3-7-sonnet" {
		t.Errorf("expected resolved model claude-3-7-sonnet, got %q", cloudSess.ModelResolved)
	}
	if cloudSess.NetCostUSD <= 0 || cloudSess.GrossCostUSD <= 0 {
		t.Errorf("expected non-zero cloud costs: gross=%.4f, net=%.4f", cloudSess.GrossCostUSD, cloudSess.NetCostUSD)
	}
	if cloudSess.Turns[0].CostUSD <= 0 {
		t.Errorf("expected non-zero turn cost: %.4f", cloudSess.Turns[0].CostUSD)
	}

	// Local session
	localSess := &models.Session{
		ID:              "sess-local",
		ModelRaw:        "llama-3.2-3b",
		OutputTokens:    1_000,
		DurationSeconds: 10,
	}

	engine.CostSession(ctx, localSess, "http://localhost:11434", "ollama", nil)

	if localSess.ElectricityCostUSD <= 0 {
		t.Errorf("expected positive electricity cost, got %.6f", localSess.ElectricityCostUSD)
	}
	if localSess.GrossCostUSD != 0 {
		t.Errorf("expected 0 gross cost for local session, got %.4f", localSess.GrossCostUSD)
	}
	if localSess.NetCostUSD != localSess.ElectricityCostUSD {
		t.Errorf("expected net cost to match electricity cost for local session")
	}
}
