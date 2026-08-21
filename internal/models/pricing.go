package models

import "time"

// ModelRate represents the pricing parameters for a specific LLM.
type ModelRate struct {
	ModelPattern        string  `json:"model_pattern"`
	InputCostPerM       float64 `json:"input_cost_per_m"`
	OutputCostPerM      float64 `json:"output_cost_per_m"`
	CacheReadCostPerM   float64 `json:"cache_read_cost_per_m"`
	CacheWriteCostPerM  float64 `json:"cache_write_cost_per_m"`
	Source              string  `json:"source"`
}

// PricingOverride represents a user-defined custom price override in the database.
type PricingOverride struct {
	ModelPattern       string    `json:"model_pattern"`
	InputCostPerM      float64   `json:"input_cost_per_m"`
	OutputCostPerM     float64   `json:"output_cost_per_m"`
	CacheReadCostPerM  float64   `json:"cache_read_cost_per_m"`
	CacheWriteCostPerM float64   `json:"cache_write_cost_per_m"`
	Source             string    `json:"source"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// PowerConfig defines power estimation parameters for local hardware profiles.
type PowerConfig struct {
	ProfileName     string  `json:"profile_name"`
	IdleWatts       float64 `json:"idle_watts"`
	LoadWatts       float64 `json:"load_watts"`
	CostPerKWhUSD   float64 `json:"cost_per_kwh_usd"`
}
