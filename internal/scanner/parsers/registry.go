package parsers

import (
	"sync"
)

// Registry manages and detects available AgentParser implementations.
type Registry struct {
	mu      sync.RWMutex
	parsers []AgentParser
	byName  map[string]AgentParser
}

// NewDefaultRegistry instantiates a registry containing all 18+ agent parsers.
func NewDefaultRegistry() *Registry {
	r := &Registry{
		parsers: make([]AgentParser, 0),
		byName:  make(map[string]AgentParser),
	}

	// Register all agent parsers
	r.Register(NewClaudeParser())
	r.Register(NewAntigravityParser())
	r.Register(NewGeminiParser())
	r.Register(NewCodexParser())
	r.Register(NewCursorParser())
	r.Register(NewCopilotParser())
	r.Register(NewOpenCodeParser())
	r.Register(NewGrokParser())
	r.Register(NewPiParser())
	r.Register(NewDSHParser())
	r.Register(NewMetaMuseParser())
	r.Register(NewPrimeParser())
	r.Register(NewQwenParser())
	r.Register(NewClineParser())
	r.Register(NewSmallCodeParser())
	r.Register(NewVibeParser())
	r.Register(NewWindsurfParser())
	r.Register(NewOllamaParser())

	return r
}

// Register registers a new AgentParser instance.
func (r *Registry) Register(p AgentParser) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.parsers = append(r.parsers, p)
	r.byName[p.AgentName()] = p
}

// Detect matches a file path to the appropriate AgentParser.
func (r *Registry) Detect(filePath string) AgentParser {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.parsers {
		if p.Detect(filePath) {
			return p
		}
	}
	return nil
}

// Get retrieves a parser by canonical agent name.
func (r *Registry) Get(name string) AgentParser {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byName[name]
}

// All returns a slice of all registered parsers.
func (r *Registry) All() []AgentParser {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]AgentParser, len(r.parsers))
	copy(result, r.parsers)
	return result
}
