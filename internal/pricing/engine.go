package pricing

import (
	"context"
	"math"
	"strings"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

// Engine orchestrates model rate resolution, token cost calculations, and power estimation.
type Engine struct {
	Dataset      *PricingDataset
	Resolver     *Resolver
	PowerProfile models.PowerConfig
}

// NewEngine initializes the pricing engine with embedded dataset tables.
func NewEngine() (*Engine, error) {
	ds, err := LoadEmbeddedDataset()
	if err != nil {
		return nil, err
	}
	resolver := NewResolver(ds)
	return &Engine{
		Dataset:      ds,
		Resolver:     resolver,
		PowerProfile: DefaultPowerProfile,
	}, nil
}

// CalculateCost computes gross and net USD cost from token usage and rates.
func CalculateCost(usage models.TokenUsage, rate models.ModelRate) (grossUSD, netUSD float64) {
	// 1. Gross: All prompt tokens priced at standard input rate
	totalPrompt := float64(usage.InputTokens + usage.CacheReadTokens + usage.CacheCreationTokens)
	grossUSD = (totalPrompt / 1_000_000.0) * rate.InputCostPerM +
		(float64(usage.OutputTokens) / 1_000_000.0) * rate.OutputCostPerM

	// 2. Net: Discounted cache reads and cache write markup
	readRate := rate.CacheReadCostPerM
	if readRate == 0 && rate.InputCostPerM > 0 {
		readRate = rate.InputCostPerM * 0.10 // 90% discount fallback
	}

	writeRate := rate.CacheWriteCostPerM
	if writeRate == 0 && rate.InputCostPerM > 0 {
		writeRate = rate.InputCostPerM * 1.25 // 25% markup fallback
	}

	netUSD = (float64(usage.InputTokens) / 1_000_000.0) * rate.InputCostPerM +
		(float64(usage.CacheReadTokens) / 1_000_000.0) * readRate +
		(float64(usage.CacheCreationTokens) / 1_000_000.0) * writeRate +
		(float64(usage.OutputTokens) / 1_000_000.0) * rate.OutputCostPerM

	// Round to 6 decimal places to prevent floating point drift
	grossUSD = roundToDecimals(grossUSD, 6)
	netUSD = roundToDecimals(netUSD, 6)

	return grossUSD, netUSD
}

// ResolveModel resolves an arbitrary model identifier into canonical ModelRate.
func (e *Engine) ResolveModel(modelName, provider string, overrides []models.PricingOverride) (models.ModelRate, string) {
	return e.Resolver.Resolve(modelName, provider, overrides)
}

// XAILongPromptThreshold is the xAI long-context cliff (200k tokens).
// Requests whose prompt reaches 200k tokens are billed at 2x all rates for that turn.
const XAILongPromptThreshold = 200_000

// CalculateXAITurnCost prices one xAI request, applying the 200k-prompt 2x cliff.
// promptTokens is the full prompt (cached + uncached). Uncached input is prompt - cached.
func CalculateXAITurnCost(rate models.ModelRate, promptTokens, completionTokens, cachedTokens int64) float64 {
	prompt := promptTokens
	if prompt < 0 {
		prompt = 0
	}
	cached := cachedTokens
	if cached < 0 {
		cached = 0
	}
	if cached > prompt {
		cached = prompt
	}
	uncached := prompt - cached
	completion := completionTokens
	if completion < 0 {
		completion = 0
	}

	usage := models.TokenUsage{
		InputTokens:     uncached,
		OutputTokens:    completion,
		CacheReadTokens: cached,
	}
	_, net := CalculateCost(usage, rate)
	if prompt >= XAILongPromptThreshold {
		net = roundToDecimals(net*2.0, 6)
	}
	return net
}

// CostSession calculates monetary and electricity costs across a session and its turns.
func (e *Engine) CostSession(ctx context.Context, s *models.Session, endpoint, provider string, overrides []models.PricingOverride) {
	isLocal := IsLocalSession("", endpoint, provider)

	rate, resolvedName := e.ResolveModel(s.ModelRaw, provider, overrides)
	s.ModelResolved = resolvedName

	if isLocal {
		elecCost := CalculateElectricityCost(s.OutputTokens, s.DurationSeconds, e.PowerProfile, s.ModelRaw)
		s.ElectricityCostUSD = roundToDecimals(elecCost, 6)
		s.GrossCostUSD = 0.0
		s.NetCostUSD = s.ElectricityCostUSD
	} else {
		usage := models.TokenUsage{
			InputTokens:         s.InputTokens,
			OutputTokens:        s.OutputTokens,
			CacheReadTokens:     s.CacheReadTokens,
			CacheCreationTokens: s.CacheCreationTokens,
		}
		gross, net := CalculateCost(usage, rate)
		s.GrossCostUSD = gross
		s.NetCostUSD = net
		s.ElectricityCostUSD = 0.0
	}

	// Cost individual turns if present
	var sumTurnCosts float64
	hasTurnPricing := false

	for i := range s.Turns {
		turnModel := s.Turns[i].ModelName
		if turnModel == "" {
			turnModel = s.ModelRaw
		}
		turnRate, _ := e.ResolveModel(turnModel, provider, overrides)
		prompt := s.Turns[i].InputTokens + s.Turns[i].CacheReadTokens
		if s.AgentName == "grok" || strings.HasPrefix(strings.ToLower(turnModel), "grok") {
			s.Turns[i].CostUSD = CalculateXAITurnCost(turnRate, prompt, s.Turns[i].OutputTokens, s.Turns[i].CacheReadTokens)
			sumTurnCosts += s.Turns[i].CostUSD
			hasTurnPricing = true
		} else {
			turnUsage := models.TokenUsage{
				InputTokens:         s.Turns[i].InputTokens,
				OutputTokens:        s.Turns[i].OutputTokens,
				CacheReadTokens:     s.Turns[i].CacheReadTokens,
				CacheCreationTokens: s.Turns[i].CacheCreationTokens,
			}
			_, turnNet := CalculateCost(turnUsage, turnRate)
			s.Turns[i].CostUSD = turnNet
		}
	}

	if !isLocal && hasTurnPricing && len(s.Turns) > 0 && (s.OutputTokens > 0 || s.CacheReadTokens > 0) {
		s.NetCostUSD = roundToDecimals(sumTurnCosts, 6)
		s.GrossCostUSD = s.NetCostUSD
	}
}

func roundToDecimals(val float64, decimals int) float64 {
	factor := math.Pow(10, float64(decimals))
	return math.Round(val*factor) / factor
}
