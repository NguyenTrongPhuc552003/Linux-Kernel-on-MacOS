// Package plugin provides hook execution management.
package plugin

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/NguyenTrongPhuc552003/elmos/core/ui"
)

// HookExecutor manages hook registration and execution.
// It maintains a registry of hooks by event and executes them in priority order.
type HookExecutor struct {
	mu           sync.RWMutex
	hooks        map[string][]*registeredHook
	errorHandler func(event string, err error) // Optional error callback
	printer      *ui.Printer
	verbose      bool
}

// registeredHook is an internal wrapper around a hook with metadata
type registeredHook struct {
	name     string // Plugin name that registered this hook
	handler  HookFunc
	priority int
	required bool
}

// NewHookExecutor creates a new HookExecutor.
func NewHookExecutor(printer *ui.Printer, verbose bool) *HookExecutor {
	return &HookExecutor{
		hooks:   make(map[string][]*registeredHook),
		printer: printer,
		verbose: verbose,
	}
}

// Register registers a hook handler for a specific event.
// If multiple handlers are registered for the same event, they will be executed
// in priority order (higher priority first).
func (he *HookExecutor) Register(pluginName, event string, registration HookRegistration) {
	if registration.Handler == nil {
		return
	}

	he.mu.Lock()
	defer he.mu.Unlock()

	rh := &registeredHook{
		name:     pluginName,
		handler:  registration.Handler,
		priority: registration.Priority,
		required: registration.Required,
	}

	he.hooks[event] = append(he.hooks[event], rh)

	// Sort hooks by priority (higher first) after each registration
	sort.Slice(he.hooks[event], func(i, j int) bool {
		return he.hooks[event][i].priority > he.hooks[event][j].priority
	})
}

// RegisterBatch registers multiple hooks at once.
// Useful for plugin initialization to register all its hooks.
func (he *HookExecutor) RegisterBatch(pluginName string, registrations []HookRegistration) {
	for _, reg := range registrations {
		he.Register(pluginName, reg.Event, reg)
	}
}

// Execute executes all registered hooks for a given event.
// Returns a combined error if any required hooks fail.
// Non-required hook failures are logged but do not stop execution.
func (he *HookExecutor) Execute(ctx context.Context, event *Event) error {
	he.mu.RLock()
	hooks, exists := he.hooks[event.Name]
	he.mu.RUnlock()

	if !exists || len(hooks) == 0 {
		return nil
	}

	if he.verbose {
		he.printer.Info("Executing %d hooks for event: %s", len(hooks), event.Name)
	}

	var errs []error

	for _, hook := range hooks {
		if he.verbose {
			he.printer.Step("  → [%s] %s (priority: %d)", hook.name, event.Name, hook.priority)
		}

		err := hook.handler(ctx, event)
		if err != nil {
			if hook.required {
				errs = append(errs, fmt.Errorf("[%s:%s] %w", hook.name, event.Name, err))
			} else {
				// Log non-required failures but continue
				he.printer.Warn("[%s:%s] Non-fatal error: %v", hook.name, event.Name, err)
			}
		}
	}

	if len(errs) > 0 {
		// Return first error (most critical)
		return errs[0]
	}

	return nil
}

// ExecuteAsync executes hooks asynchronously and returns a channel for results.
// Useful for long-running hooks. Waits for all hooks to complete.
// If any required hook fails, returns error.
func (he *HookExecutor) ExecuteAsync(ctx context.Context, event *Event) <-chan error {
	resultChan := make(chan error, 1)

	he.mu.RLock()
	hooks, exists := he.hooks[event.Name]
	he.mu.RUnlock()

	go func() {
		defer close(resultChan)

		if !exists || len(hooks) == 0 {
			return
		}

		var wg sync.WaitGroup
		errChan := make(chan error, len(hooks))

		for _, hook := range hooks {
			wg.Add(1)

			// Wrap hook to handle errors
			go func(h *registeredHook) {
				defer wg.Done()

				if he.verbose {
					he.printer.Step("  → [async] [%s] %s", h.name, event.Name)
				}

				err := h.handler(ctx, event)
				if err != nil {
					if h.required {
						errChan <- fmt.Errorf("[%s:%s] %w", h.name, event.Name, err)
					} else {
						he.printer.Warn("[%s:%s] Non-fatal error: %v", h.name, event.Name, err)
					}
				}
			}(hook)
		}

		// Wait for all hooks to finish
		wg.Wait()
		close(errChan)

		// Return first error if any
		for err := range errChan {
			if err != nil {
				resultChan <- err
				return
			}
		}
	}()

	return resultChan
}

// HasHooks returns true if any hooks are registered for the given event.
func (he *HookExecutor) HasHooks(event string) bool {
	he.mu.RLock()
	defer he.mu.RUnlock()

	hooks, exists := he.hooks[event]
	return exists && len(hooks) > 0
}

// HooksForEvent returns a copy of hooks registered for an event.
func (he *HookExecutor) HooksForEvent(event string) []HookRegistration {
	he.mu.RLock()
	defer he.mu.RUnlock()

	hooks, exists := he.hooks[event]
	if !exists {
		return nil
	}

	result := make([]HookRegistration, len(hooks))
	for i, h := range hooks {
		result[i] = HookRegistration{
			Event:    event,
			Handler:  h.handler,
			Priority: h.priority,
			Required: h.required,
		}
	}

	return result
}

// GetRegisteredEvents returns all events with registered hooks.
func (he *HookExecutor) GetRegisteredEvents() []string {
	he.mu.RLock()
	defer he.mu.RUnlock()

	events := make([]string, 0, len(he.hooks))
	for event := range he.hooks {
		events = append(events, event)
	}
	sort.Strings(events)

	return events
}

// Clear removes all registered hooks (useful for testing).
func (he *HookExecutor) Clear() {
	he.mu.Lock()
	defer he.mu.Unlock()

	he.hooks = make(map[string][]*registeredHook)
}

// SetErrorHandler sets an optional callback for hook errors.
// Useful for logging or metrics collection.
func (he *HookExecutor) SetErrorHandler(handler func(event string, err error)) {
	he.mu.Lock()
	defer he.mu.Unlock()

	he.errorHandler = handler
}
