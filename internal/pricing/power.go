package pricing

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

var (
	localProviders = map[string]bool{
		"ollama":    true,
		"lmstudio":  true,
		"llama.cpp": true,
		"vllm":      true,
		"localai":   true,
		"jan":       true,
		"gpt4all":   true,
		"koboldcpp": true,
		"local":     true,
	}

	paramRegex = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*b\b`)
)

// DefaultPowerProfile provides sensible fallback wattage and electricity tariffs (Apple Silicon Mac / typical workstation).
var DefaultPowerProfile = models.PowerConfig{
	ProfileName:   "default",
	IdleWatts:     15.0,
	LoadWatts:     80.0,
	CostPerKWhUSD: 0.15,
}

// IsLocalSession determines if an inference session was executed on local hardware.
func IsLocalSession(billingMode, endpoint, provider string) bool {
	if strings.ToLower(billingMode) == "local" {
		return true
	}

	provLower := strings.ToLower(strings.TrimSpace(provider))
	if localProviders[provLower] {
		return true
	}

	epLower := strings.ToLower(strings.TrimSpace(endpoint))
	if epLower != "" {
		for _, loopback := range []string{"localhost", "127.0.0.1", "0.0.0.0", "::1"} {
			if strings.Contains(epLower, loopback) {
				return true
			}
		}
	}

	return false
}

// EstimateThroughput estimates tokens per second based on model parameter count in the name.
func EstimateThroughput(modelName string) float64 {
	matches := paramRegex.FindStringSubmatch(modelName)
	if len(matches) > 1 {
		if params, err := strconv.ParseFloat(matches[1], 64); err == nil {
			switch {
			case params <= 1.0:
				return 150.0
			case params <= 4.0:
				return 90.0
			case params <= 8.0:
				return 70.0
			case params <= 14.0:
				return 50.0
			case params <= 34.0:
				return 30.0
			case params <= 70.0:
				return 18.0
			default:
				return 10.0
			}
		}
	}
	return 30.0
}

// CalculateElectricityCost computes hardware electricity cost in USD.
func CalculateElectricityCost(outputTokens int64, durationSeconds float64, profile models.PowerConfig, modelName string) float64 {
	if outputTokens <= 0 && durationSeconds <= 0 {
		return 0.0
	}

	loadWatts := profile.LoadWatts
	if loadWatts <= 0 {
		loadWatts = DefaultPowerProfile.LoadWatts
	}

	costPerKWh := profile.CostPerKWhUSD
	if costPerKWh <= 0 {
		costPerKWh = DefaultPowerProfile.CostPerKWhUSD
	}

	genSeconds := durationSeconds
	if genSeconds <= 0 {
		genSeconds = float64(outputTokens) / EstimateThroughput(modelName)
	}

	kwh := (loadWatts * genSeconds) / 3600000.0
	return kwh * costPerKWh
}
