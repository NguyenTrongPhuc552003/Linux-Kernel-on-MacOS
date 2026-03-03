// Package orchestrator provides build pipeline orchestration for ELMOS.
// This file implements the parallel DAG execution engine.
package orchestrator

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/NguyenTrongPhuc552003/elmos/core/plugin"
)

// TaskResult holds the outcome of a single task execution.
type TaskResult struct {
	TaskID    string
	StartTime time.Time
	EndTime   time.Time
	FromCache bool
	Err       error
	Warnings  []error
}

// Duration returns the wall-clock time spent on this task.
func (r *TaskResult) Duration() time.Duration {
	return r.EndTime.Sub(r.StartTime)
}

// BuildReport summarises the outcomes of a full pipeline execution.
type BuildReport struct {
	StartTime   time.Time
	EndTime     time.Time
	TaskResults map[string]*TaskResult
	Artifacts   map[string][]string // task ID → artifact file paths
}

// Duration returns the total wall-clock time for the pipeline.
func (r *BuildReport) Duration() time.Duration {
	return r.EndTime.Sub(r.StartTime)
}

// CacheHits returns the number of tasks that were served from cache.
func (r *BuildReport) CacheHits() int {
	n := 0
	for _, tr := range r.TaskResults {
		if tr.FromCache {
			n++
		}
	}
	return n
}

// Success returns true when no task produced an error.
func (r *BuildReport) Success() bool {
	for _, tr := range r.TaskResults {
		if tr.Err != nil {
			return false
		}
	}
	return true
}

// PipelineExecutor schedules and runs a Pipeline, respecting task dependencies.
// Tasks whose dependencies are all satisfied run in parallel up to a bounded
// worker pool. Cache hits skip execution entirely.
type PipelineExecutor struct {
	pipeline    *Pipeline
	fingerprint *BuildFingerprinter // may be nil (disables caching)
	hooks       *plugin.HookExecutor
	workers     int
	runners     map[string]plugin.TaskRunner // task type → runner
}

// NewPipelineExecutor creates a PipelineExecutor.
// workers ≤ 0 defaults to runtime.GOMAXPROCS(0).
func NewPipelineExecutor(
	pipeline *Pipeline,
	fingerprint *BuildFingerprinter,
	hooks *plugin.HookExecutor,
	workers int,
) *PipelineExecutor {
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	return &PipelineExecutor{
		pipeline:    pipeline,
		fingerprint: fingerprint,
		hooks:       hooks,
		workers:     workers,
		runners:     make(map[string]plugin.TaskRunner),
	}
}

// RegisterRunner registers a TaskRunner for a given task type.
// Plugins that implement executable build work call this to wire into the pipeline.
func (pe *PipelineExecutor) RegisterRunner(taskType string, runner plugin.TaskRunner) {
	pe.runners[taskType] = runner
}

// Execute runs all tasks in the pipeline respecting their dependency order.
// Tasks with no unsatisfied dependencies are run concurrently (up to pe.workers).
// Returns a BuildReport and the first error encountered (if any).
func (pe *PipelineExecutor) Execute(ctx context.Context) (*BuildReport, error) {
	sorted, err := pe.pipeline.Sort()
	if err != nil {
		return nil, fmt.Errorf("pipeline: dependency resolution failed: %w", err)
	}

	report := &BuildReport{
		StartTime:   time.Now(),
		TaskResults: make(map[string]*TaskResult, len(sorted)),
		Artifacts:   make(map[string][]string),
	}

	done := pe.initDoneChannels(sorted)
	results, execErr := pe.runAllTasks(ctx, sorted, done)

	report.EndTime = time.Now()
	for id, r := range results {
		report.TaskResults[id] = r
	}
	return report, execErr
}

// initDoneChannels creates a broadcast channel per task, closed when the task finishes.
func (pe *PipelineExecutor) initDoneChannels(ids []string) map[string]chan struct{} {
	done := make(map[string]chan struct{}, len(ids))
	for _, id := range ids {
		done[id] = make(chan struct{})
	}
	return done
}

// runAllTasks launches one goroutine per task and waits for all to finish.
// Each goroutine waits for its dependency channels before acquiring the semaphore.
func (pe *PipelineExecutor) runAllTasks(
	ctx context.Context,
	sorted []string,
	done map[string]chan struct{},
) (map[string]*TaskResult, error) {
	results := make(map[string]*TaskResult, len(sorted))
	var mu sync.Mutex
	var firstErr error

	sem := make(chan struct{}, pe.workers)
	var wg sync.WaitGroup

	for _, id := range sorted {
		wg.Add(1)
		go func(taskID string) {
			defer wg.Done()
			pe.runTaskWithDeps(ctx, taskID, done, sem, &mu, results, &firstErr)
		}(id)
	}

	wg.Wait()
	return results, firstErr
}

// runTaskWithDeps waits for dependency channels, acquires a semaphore slot,
// executes the task, and signals its own done channel.
func (pe *PipelineExecutor) runTaskWithDeps(
	ctx context.Context,
	taskID string,
	done map[string]chan struct{},
	sem chan struct{},
	mu *sync.Mutex,
	results map[string]*TaskResult,
	firstErr *error,
) {
	task := pe.pipeline.Get(taskID)
	defer close(done[taskID])

	if !pe.waitForDeps(ctx, task, done) {
		return // context cancelled
	}

	select {
	case sem <- struct{}{}:
		defer func() { <-sem }()
	case <-ctx.Done():
		return
	}

	result := pe.runTask(ctx, task)

	mu.Lock()
	results[taskID] = result
	if result.Err != nil && *firstErr == nil {
		*firstErr = fmt.Errorf("task %q failed: %w", taskID, result.Err)
	}
	mu.Unlock()
}

// waitForDeps blocks until all dependencies have closed their done channels,
// or until ctx is cancelled. Returns false on cancellation.
func (pe *PipelineExecutor) waitForDeps(ctx context.Context, task *BuildTask, done map[string]chan struct{}) bool {
	for _, dep := range task.Dependencies {
		select {
		case <-done[dep]:
		case <-ctx.Done():
			return false
		}
	}
	return true
}

// runTask executes a single task, firing pre/post hooks and checking the cache.
func (pe *PipelineExecutor) runTask(ctx context.Context, task *BuildTask) *TaskResult {
	result := &TaskResult{
		TaskID:    task.ID,
		StartTime: time.Now(),
	}

	// Fire pre-build hook
	preEvent := pe.buildEvent("pre_"+task.Type+"_execute", task)
	if pe.hooks != nil {
		if err := pe.hooks.Execute(ctx, preEvent); err != nil {
			result.Err = fmt.Errorf("pre-hook: %w", err)
			result.EndTime = time.Now()
			return result
		}
	}

	// Check and apply cache if fingerprinter is available and task is cacheable.
	if task.Cacheable && pe.fingerprint != nil {
		if hit := pe.checkCacheHit(task); hit {
			result.FromCache = true
			result.EndTime = time.Now()
			if err := pe.firePostHook(ctx, task, result); err != nil {
				result.Warnings = append(result.Warnings, fmt.Errorf("post-hook warning: %w", err))
			}
			return result
		}
	}

	// No cache hit: delegate to the registered runner for this task type.
	if runner, ok := pe.runners[task.Type]; ok {
		result.Err = runner.RunTask(ctx, task.ID, task.Config)
	}
	result.EndTime = time.Now()
	if err := pe.firePostHook(ctx, task, result); err != nil {
		result.Warnings = append(result.Warnings, fmt.Errorf("post-hook warning: %w", err))
	}

	return result
}

// checkCacheHit returns true if a successful prior run exists in the cache index.
func (pe *PipelineExecutor) checkCacheHit(task *BuildTask) bool {
	entry := pe.fingerprint.LookupEntry(task.ID, task.Type)
	return entry != nil && entry.Success
}

// firePostHook fires the post-execution hook for a task.
func (pe *PipelineExecutor) firePostHook(ctx context.Context, task *BuildTask, result *TaskResult) error {
	if pe.hooks == nil {
		return nil
	}
	postEvent := pe.buildEvent("post_"+task.Type+"_execute", task)
	postEvent.Metadata[plugin.MetadataDuration] = int64(result.Duration().Seconds())
	postEvent.Metadata["task.from_cache"] = result.FromCache
	return pe.hooks.Execute(ctx, postEvent)
}

// buildEvent constructs a hook Event for pipeline task execution.
func (pe *PipelineExecutor) buildEvent(name string, task *BuildTask) *plugin.Event {
	return &plugin.Event{
		Name: name,
		Metadata: map[string]interface{}{
			"task.id":        task.ID,
			"task.type":      task.Type,
			"task.cacheable": task.Cacheable,
		},
	}
}
