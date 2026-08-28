package pricing

import (
	"strings"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

// NormalizeModelID strips common aggregator and provider prefixes to find the base model name.
func NormalizeModelID(model string) string {
	m := strings.ToLower(strings.TrimSpace(model))

	prefixes := []string{
		"fireworks/",
		"together/",
		"openrouter/",
		"anthropic/",
		"openai/",
		"google/",
		"bedrock/",
		"us.anthropic.",
		"us.openai.",
		"us.google.",
		"us.",
		"eu.",
		"global.",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(m, prefix) {
			m = m[len(prefix):]
			break
		}
	}
	return m
}

// Resolver resolves model rates using the two-tier resolution strategy.
type Resolver struct {
	Dataset *PricingDataset
}

// NewResolver constructs a model rate resolver with the provided pricing dataset.
func NewResolver(dataset *PricingDataset) *Resolver {
	return &Resolver{Dataset: dataset}
}

// Resolve identifies the authoritative model rate across database overrides and dataset tables.
func (r *Resolver) Resolve(modelName, provider string, overrides []models.PricingOverride) (models.ModelRate, string) {
	mNorm := NormalizeModelID(modelName)
	if mNorm == "" {
		mNorm = "_default"
	}

	// 1. Tier 2: User-defined custom overrides (highest priority)
	for _, o := range overrides {
		pat := strings.ToLower(strings.TrimSpace(o.ModelPattern))
		if pat == mNorm || pat == strings.ToLower(strings.TrimSpace(modelName)) || strings.Contains(mNorm, pat) {
			return models.ModelRate{
				ModelPattern:       o.ModelPattern,
				InputCostPerM:      o.InputCostPerM,
				OutputCostPerM:     o.OutputCostPerM,
				CacheReadCostPerM:  o.CacheReadCostPerM,
				CacheWriteCostPerM: o.CacheWriteCostPerM,
				Source:             "user_override",
			}, pat
		}
	}

	if r.Dataset == nil {
		fallback := curatedModels["_default"]
		return fallback, "_default"
	}

	r.Dataset.mu.RLock()
	defer r.Dataset.mu.RUnlock()

	// 2. Provider-specific exact lookup (e.g. from agent billing_provider)
	if provider != "" {
		provKey := strings.ToLower(strings.TrimSpace(provider)) + "\x00" + mNorm
		if rate, found := r.Dataset.byProvider[provKey]; found {
			return rate, mNorm
		}
	}

	// 3. Curated Exact match
	if rate, found := r.Dataset.curated[mNorm]; found {
		return rate, mNorm
	}

	// 4. Curated Fuzzy Longest-Key match (e.g., claude-3-7-sonnet-20250219 -> claude-3-7-sonnet)
	for _, k := range r.Dataset.sortedCuratedKeys {
		if FuzzyKeyMatches(k, mNorm) {
			if rate, found := r.Dataset.curated[k]; found {
				return rate, k
			}
		}
	}

	// 5. Bundled Exact match
	if rate, found := r.Dataset.bundled[mNorm]; found {
		return rate, mNorm
	}

	// 6. Bundled Fuzzy Longest-Key match
	for _, k := range r.Dataset.sortedBundledKeys {
		if FuzzyKeyMatches(k, mNorm) {
			if rate, found := r.Dataset.bundled[k]; found {
				return rate, k
			}
		}
	}

	// 7. Default Fallback
	if defRate, found := r.Dataset.curated["_default"]; found {
		return defRate, "_default"
	}

	return curatedModels["_default"], "_default"
}

// FuzzyKeyMatches performs substring matching while rejecting shorter dotted version prefixes.
// e.g. "grok-4" must not match "grok-4.6" (which would bill at grok-4 rates), but "grok-4" matches "grok-4-fast".
func FuzzyKeyMatches(key, model string) bool {
	idx := strings.Index(model, key)
	if idx < 0 {
		return false
	}
	after := model[idx+len(key):]
	if len(after) >= 2 && after[0] == '.' && (after[1] >= '0' && after[1] <= '9') {
		return false
	}
	return true
}
