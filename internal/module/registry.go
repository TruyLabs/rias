package module

import (
	"fmt"
	"sync"
)

// Constructor builds a Module from raw config values parsed from config.yaml.
type Constructor func(cfg map[string]interface{}) (Module, error)

// Registry maps module names to their constructors and descriptions.
type Registry struct {
	mu           sync.RWMutex
	constructors map[string]Constructor
	descriptions map[string]string
}

var defaultRegistry = &Registry{
	constructors: map[string]Constructor{},
	descriptions: map[string]string{},
}

func init() {
	defaultRegistry.Register("github_prs", "Read GitHub pull requests into the brain", NewGitHubPRsModule)
	defaultRegistry.Register("google_sheets", "Read a Google Sheet into the brain", NewGoogleSheetsModule)
}

// Default returns the shared registry with all built-in modules.
func Default() *Registry {
	return defaultRegistry
}

// Register adds a module constructor under the given name.
func (r *Registry) Register(name, description string, fn Constructor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.constructors[name] = fn
	r.descriptions[name] = description
}

// Build instantiates a module by name with the provided config map.
func (r *Registry) Build(name string, cfg map[string]interface{}) (Module, error) {
	r.mu.RLock()
	fn, ok := r.constructors[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown module %q — run 'kai module list' to see available modules", name)
	}
	return fn(cfg)
}

// Available returns the names of all registered modules.
func (r *Registry) Available() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.constructors))
	for name := range r.constructors {
		names = append(names, name)
	}
	return names
}

// Description returns the description for a named module.
func (r *Registry) Description(name string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.descriptions[name]
}
