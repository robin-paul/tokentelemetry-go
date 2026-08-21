package pricing

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/robin-paul/tokentelemetry-go/internal/models"
)

//go:embed pricing_data.json
var embeddedPricingJSON []byte

// PricingDataset stores in-memory pricing tables parsed from models.dev and curated entries.
type PricingDataset struct {
	mu                sync.RWMutex
	curated           map[string]models.ModelRate
	sortedCuratedKeys []string
	byProvider        map[string]models.ModelRate // key: provider + "\x00" + model
	bundled           map[string]models.ModelRate // key: normalized model name from bundled JSON
	sortedBundledKeys []string
}

type rawJSONEntry struct {
	In          *float64 `json:"in"`
	Out         *float64 `json:"out"`
	CachedRead  *float64 `json:"cached_read"`
	CachedWrite *float64 `json:"cached_write"`
}

type rawPricingFile struct {
	Pricing    map[string]rawJSONEntry `json:"pricing"`
	ByProvider map[string]rawJSONEntry `json:"by_provider"`
	Updated    string                  `json:"updated"`
	Source     string                  `json:"source"`
}

// Curated baseline models to ensure immediate high-accuracy lookups
var curatedModels = map[string]models.ModelRate{
	// Anthropic
	"claude-opus-4-7":   {ModelPattern: "claude-opus-4-7", InputCostPerM: 5.00, OutputCostPerM: 25.00, CacheReadCostPerM: 0.50, Source: "curated"},
	"claude-opus-4-6":   {ModelPattern: "claude-opus-4-6", InputCostPerM: 5.00, OutputCostPerM: 25.00, CacheReadCostPerM: 0.50, Source: "curated"},
	"claude-opus-4-5":   {ModelPattern: "claude-opus-4-5", InputCostPerM: 5.00, OutputCostPerM: 25.00, CacheReadCostPerM: 0.50, Source: "curated"},
	"claude-opus-4-1":   {ModelPattern: "claude-opus-4-1", InputCostPerM: 15.00, OutputCostPerM: 75.00, CacheReadCostPerM: 1.50, Source: "curated"},
	"claude-opus-4":     {ModelPattern: "claude-opus-4", InputCostPerM: 15.00, OutputCostPerM: 75.00, CacheReadCostPerM: 1.50, Source: "curated"},
	"claude-sonnet-4-6": {ModelPattern: "claude-sonnet-4-6", InputCostPerM: 3.00, OutputCostPerM: 15.00, CacheReadCostPerM: 0.30, Source: "curated"},
	"claude-sonnet-4-5": {ModelPattern: "claude-sonnet-4-5", InputCostPerM: 3.00, OutputCostPerM: 15.00, CacheReadCostPerM: 0.30, Source: "curated"},
	"claude-sonnet-4":   {ModelPattern: "claude-sonnet-4", InputCostPerM: 3.00, OutputCostPerM: 15.00, CacheReadCostPerM: 0.30, Source: "curated"},
	"claude-3-7-sonnet": {ModelPattern: "claude-3-7-sonnet", InputCostPerM: 3.00, OutputCostPerM: 15.00, CacheReadCostPerM: 0.30, Source: "curated"},
	"claude-3.7-sonnet": {ModelPattern: "claude-3.7-sonnet", InputCostPerM: 3.00, OutputCostPerM: 15.00, CacheReadCostPerM: 0.30, Source: "curated"},
	"claude-3-5-sonnet": {ModelPattern: "claude-3-5-sonnet", InputCostPerM: 3.00, OutputCostPerM: 15.00, CacheReadCostPerM: 0.30, Source: "curated"},
	"claude-3.5-sonnet": {ModelPattern: "claude-3.5-sonnet", InputCostPerM: 3.00, OutputCostPerM: 15.00, CacheReadCostPerM: 0.30, Source: "curated"},
	"claude-haiku-4-5":  {ModelPattern: "claude-haiku-4-5", InputCostPerM: 1.00, OutputCostPerM: 5.00, CacheReadCostPerM: 0.10, Source: "curated"},
	"claude-haiku-4.5":  {ModelPattern: "claude-haiku-4.5", InputCostPerM: 1.00, OutputCostPerM: 5.00, CacheReadCostPerM: 0.10, Source: "curated"},
	"claude-3-5-haiku":  {ModelPattern: "claude-3-5-haiku", InputCostPerM: 0.80, OutputCostPerM: 4.00, CacheReadCostPerM: 0.08, Source: "curated"},
	"claude-3.5-haiku":  {ModelPattern: "claude-3.5-haiku", InputCostPerM: 0.80, OutputCostPerM: 4.00, CacheReadCostPerM: 0.08, Source: "curated"},

	// OpenAI
	"gpt-5.5":           {ModelPattern: "gpt-5.5", InputCostPerM: 5.00, OutputCostPerM: 30.00, CacheReadCostPerM: 0.50, Source: "curated"},
	"gpt-5.4":           {ModelPattern: "gpt-5.4", InputCostPerM: 2.50, OutputCostPerM: 15.00, CacheReadCostPerM: 0.25, Source: "curated"},
	"gpt-5.4-mini":      {ModelPattern: "gpt-5.4-mini", InputCostPerM: 0.75, OutputCostPerM: 4.50, CacheReadCostPerM: 0.075, Source: "curated"},
	"gpt-5-mini":        {ModelPattern: "gpt-5-mini", InputCostPerM: 0.15, OutputCostPerM: 0.60, CacheReadCostPerM: 0.015, Source: "curated"},
	"gpt-5":             {ModelPattern: "gpt-5", InputCostPerM: 1.25, OutputCostPerM: 10.00, CacheReadCostPerM: 0.125, Source: "curated"},
	"gpt-4o":            {ModelPattern: "gpt-4o", InputCostPerM: 2.50, OutputCostPerM: 10.00, CacheReadCostPerM: 1.25, Source: "curated"},
	"gpt-4o-mini":       {ModelPattern: "gpt-4o-mini", InputCostPerM: 0.15, OutputCostPerM: 0.60, CacheReadCostPerM: 0.075, Source: "curated"},

	// Google Gemini
	"gemini-3.1-pro":    {ModelPattern: "gemini-3.1-pro", InputCostPerM: 2.00, OutputCostPerM: 12.00, CacheReadCostPerM: 0.20, Source: "curated"},
	"gemini-3.1-flash":  {ModelPattern: "gemini-3.1-flash", InputCostPerM: 0.25, OutputCostPerM: 1.50, CacheReadCostPerM: 0.025, Source: "curated"},
	"gemini-3-pro":      {ModelPattern: "gemini-3-pro", InputCostPerM: 2.00, OutputCostPerM: 12.00, CacheReadCostPerM: 0.20, Source: "curated"},
	"gemini-3-flash":    {ModelPattern: "gemini-3-flash", InputCostPerM: 0.25, OutputCostPerM: 1.50, CacheReadCostPerM: 0.025, Source: "curated"},
	"gemini-2.5-pro":    {ModelPattern: "gemini-2.5-pro", InputCostPerM: 1.25, OutputCostPerM: 10.00, CacheReadCostPerM: 0.125, Source: "curated"},
	"gemini-2.5-flash":  {ModelPattern: "gemini-2.5-flash", InputCostPerM: 0.30, OutputCostPerM: 2.50, CacheReadCostPerM: 0.03, Source: "curated"},
	"gemini-2.0-flash":  {ModelPattern: "gemini-2.0-flash", InputCostPerM: 0.075, OutputCostPerM: 0.30, CacheReadCostPerM: 0.0075, Source: "curated"},

	// DeepSeek
	"deepseek-chat":     {ModelPattern: "deepseek-chat", InputCostPerM: 0.14, OutputCostPerM: 0.28, CacheReadCostPerM: 0.0028, Source: "curated"},
	"deepseek-reasoner": {ModelPattern: "deepseek-reasoner", InputCostPerM: 0.14, OutputCostPerM: 0.28, CacheReadCostPerM: 0.0028, Source: "curated"},
	"deepseek-v4-pro":   {ModelPattern: "deepseek-v4-pro", InputCostPerM: 1.74, OutputCostPerM: 3.48, CacheReadCostPerM: 0.0145, Source: "curated"},

	// Grok
	"grok-4.3":          {ModelPattern: "grok-4.3", InputCostPerM: 1.25, OutputCostPerM: 2.50, Source: "curated"},
	"grok-build":        {ModelPattern: "grok-build", InputCostPerM: 0.20, OutputCostPerM: 1.50, Source: "curated"},

	// Fallback
	"_default":          {ModelPattern: "_default", InputCostPerM: 2.00, OutputCostPerM: 10.00, CacheReadCostPerM: 0.50, Source: "default"},
}

// LoadEmbeddedDataset parses the embedded pricing JSON and initializes curated tables.
func LoadEmbeddedDataset() (*PricingDataset, error) {
	ds := &PricingDataset{
		curated:    make(map[string]models.ModelRate),
		byProvider: make(map[string]models.ModelRate),
		bundled:    make(map[string]models.ModelRate),
	}

	// 1. Curated models
	for k, v := range curatedModels {
		ds.curated[k] = v
	}
	curatedKeys := make([]string, 0, len(ds.curated))
	for k := range ds.curated {
		if k != "_default" {
			curatedKeys = append(curatedKeys, k)
		}
	}
	sort.Slice(curatedKeys, func(i, j int) bool {
		return len(curatedKeys[i]) > len(curatedKeys[j])
	})
	ds.sortedCuratedKeys = curatedKeys

	// 2. Parse embedded JSON
	if len(embeddedPricingJSON) > 0 {
		var raw rawPricingFile
		if err := json.Unmarshal(embeddedPricingJSON, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse embedded pricing json: %w", err)
		}

		// Bundled flat pricing
		for model, entry := range raw.Pricing {
			modelLower := strings.ToLower(strings.TrimSpace(model))
			ds.bundled[modelLower] = convertRawEntry(modelLower, entry, "bundled_json")
		}

		// Bundled by_provider
		for key, entry := range raw.ByProvider {
			keyLower := strings.ToLower(strings.TrimSpace(key))
			ds.byProvider[keyLower] = convertRawEntry(keyLower, entry, "bundled_json_provider")

			parts := strings.Split(keyLower, "\x00")
			if len(parts) == 2 {
				m := parts[1]
				if _, exists := ds.bundled[m]; !exists {
					ds.bundled[m] = convertRawEntry(m, entry, "bundled_json")
				}
			}
		}
	}

	// 3. Build sorted bundled keys
	bundledKeys := make([]string, 0, len(ds.bundled))
	for k := range ds.bundled {
		if k != "_default" {
			bundledKeys = append(bundledKeys, k)
		}
	}
	sort.Slice(bundledKeys, func(i, j int) bool {
		return len(bundledKeys[i]) > len(bundledKeys[j])
	})
	ds.sortedBundledKeys = bundledKeys

	return ds, nil
}

func convertRawEntry(pattern string, e rawJSONEntry, source string) models.ModelRate {
	rate := models.ModelRate{
		ModelPattern: pattern,
		Source:       source,
	}
	if e.In != nil {
		rate.InputCostPerM = *e.In
	}
	if e.Out != nil {
		rate.OutputCostPerM = *e.Out
	}
	if e.CachedRead != nil {
		rate.CacheReadCostPerM = *e.CachedRead
	}
	if e.CachedWrite != nil {
		rate.CacheWriteCostPerM = *e.CachedWrite
	}
	return rate
}

// GetAllRates returns a copy of all known model rates combining bundled JSON and curated overrides.
func (ds *PricingDataset) GetAllRates() map[string]models.ModelRate {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	all := make(map[string]models.ModelRate, len(ds.bundled)+len(ds.curated))
	for k, v := range ds.bundled {
		all[k] = v
	}
	for k, v := range ds.curated {
		all[k] = v
	}
	return all
}

