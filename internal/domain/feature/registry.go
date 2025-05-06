package feature

import (
	"fmt"
	"regexp" // Import regexp
	"sync"
)

// ExecutorRegistry defines the interface for managing FeatureExecutor instances.
type ExecutorRegistry interface {
	RegisterExecutor(name string, executor FeatureExecutor) error
	GetExecutor(name string) (FeatureExecutor, error)
}

// InMemoryExecutorRegistry provides a simple in-memory implementation of ExecutorRegistry.
type InMemoryExecutorRegistry struct {
	executors map[string]FeatureExecutor
	mu        sync.RWMutex
}

// NewInMemoryExecutorRegistry creates a new InMemoryExecutorRegistry.
func NewInMemoryExecutorRegistry() *InMemoryExecutorRegistry {
	return &InMemoryExecutorRegistry{
		executors: make(map[string]FeatureExecutor),
	}
}

// RegisterExecutor adds a FeatureExecutor to the registry.
// It returns an error if an executor with the same name is already registered.
func (r *InMemoryExecutorRegistry) RegisterExecutor(name string, executor FeatureExecutor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.executors[name]; exists {
		return fmt.Errorf("executor with name '%s' already registered", name)
	}
	if !IsValidFeatureName(name) { // Reuse validation logic
		return fmt.Errorf("invalid feature name format: '%s'", name)
	}
	r.executors[name] = executor
	return nil
}

// GetExecutor retrieves a FeatureExecutor by its name.
// It returns an error if no executor with the given name is found.
func (r *InMemoryExecutorRegistry) GetExecutor(name string) (FeatureExecutor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	executor, exists := r.executors[name]
	if !exists {
		return nil, fmt.Errorf("no executor found for feature '%s'", name)
	}
	return executor, nil
}

// Compile-time check to ensure InMemoryExecutorRegistry implements ExecutorRegistry.
var _ ExecutorRegistry = (*InMemoryExecutorRegistry)(nil)

// --- Helper function (consider moving to a shared validation package) ---

var validFeatureNameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// IsValidFeatureName checks if a feature name is valid (e.g., snake_case).
func IsValidFeatureName(name string) bool {
	if name == "" {
		return false
	}
	return validFeatureNameRegex.MatchString(name)
}
