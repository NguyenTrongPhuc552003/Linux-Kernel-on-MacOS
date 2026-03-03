// Package orchestrator provides build pipeline orchestration for ELMOS.
// This file defines the build task graph (DAG) and topological sorting.
package orchestrator

import (
	"fmt"
	"time"
)

// BuildTask is a single unit of work in the build pipeline.
// Tasks declare their dependencies so the executor can schedule them correctly.
type BuildTask struct {
	// ID is the unique identifier for this task, e.g. "kernel.build".
	ID string

	// Type maps to a plugin name, e.g. "kernel-builder", "bootloader-builder".
	Type string

	// Config holds task-specific configuration passed to the plugin.
	Config map[string]any

	// Dependencies lists IDs of tasks that must complete before this one starts.
	Dependencies []string

	// Cacheable indicates whether artifact caching should be attempted.
	Cacheable bool

	// Timeout is the maximum allowed wall-clock duration for this task.
	// Zero means no limit.
	Timeout time.Duration
}

// Pipeline is a collection of BuildTasks forming a directed acyclic graph.
// Use Add to register tasks and Sort to obtain an execution order.
type Pipeline struct {
	tasks      map[string]*BuildTask
	dependents map[string][]string // reverse map: id → tasks that depend on id
}

// NewPipeline creates an empty Pipeline.
func NewPipeline() *Pipeline {
	return &Pipeline{
		tasks:      make(map[string]*BuildTask),
		dependents: make(map[string][]string),
	}
}

// Add registers a task with the pipeline.
// Returns an error if the ID is empty or already registered.
func (p *Pipeline) Add(task *BuildTask) error {
	if task.ID == "" {
		return fmt.Errorf("orchestrator: task ID must not be empty")
	}
	if _, dup := p.tasks[task.ID]; dup {
		return fmt.Errorf("orchestrator: task %q already registered", task.ID)
	}
	p.tasks[task.ID] = task
	for _, dep := range task.Dependencies {
		p.dependents[dep] = append(p.dependents[dep], task.ID)
	}
	return nil
}

// Get returns the task with the given ID, or nil if not registered.
func (p *Pipeline) Get(id string) *BuildTask {
	return p.tasks[id]
}

// Len returns the number of registered tasks.
func (p *Pipeline) Len() int {
	return len(p.tasks)
}

// Dependents returns the IDs of tasks that directly depend on id.
func (p *Pipeline) Dependents(id string) []string {
	return p.dependents[id]
}

// Sort performs Kahn's topological sort and returns the sorted task IDs.
// Returns an error if any dependency references an unknown task ID,
// or if the graph contains a cycle.
func (p *Pipeline) Sort() ([]string, error) {
	if len(p.tasks) == 0 {
		return nil, nil
	}

	inDegree, err := p.buildInDegree()
	if err != nil {
		return nil, err
	}

	// Seed the queue with tasks that have no dependencies.
	queue := make([]string, 0, len(p.tasks))
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	sorted := make([]string, 0, len(p.tasks))
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)

		for _, dependent := range p.dependents[current] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(sorted) != len(p.tasks) {
		return nil, fmt.Errorf("orchestrator: pipeline contains a dependency cycle (%d/%d tasks sorted)",
			len(sorted), len(p.tasks))
	}
	return sorted, nil
}

// buildInDegree validates dependency IDs and returns the in-degree map
// (the number of unsatisfied prerequisites per task).
func (p *Pipeline) buildInDegree() (map[string]int, error) {
	inDegree := make(map[string]int, len(p.tasks))
	for id := range p.tasks {
		inDegree[id] = 0
	}
	for _, task := range p.tasks {
		for _, dep := range task.Dependencies {
			if _, ok := p.tasks[dep]; !ok {
				return nil, fmt.Errorf("orchestrator: task %q depends on unknown task %q", task.ID, dep)
			}
			inDegree[task.ID]++
		}
	}
	return inDegree, nil
}

// StandardPipeline constructs the default ELMOS build pipeline.
// Callers selectively enable tasks by omitting unused ones.
//
//	kernel.clone → kernel.patch → kernel.config → kernel.build ──┐
//	uboot.clone  → uboot.patch  → uboot.blobs  → uboot.build  ──→ image.assemble
//	rootfs.create → rootfs.customize ────────────────────────────┘
func StandardPipeline() *Pipeline {
	p := NewPipeline()

	tasks := []*BuildTask{
		{ID: "kernel.clone", Type: "kernel-builder", Cacheable: false},
		{ID: "kernel.patch", Type: "kernel-builder", Dependencies: []string{"kernel.clone"}, Cacheable: false},
		{ID: "kernel.config", Type: "kernel-builder", Dependencies: []string{"kernel.patch"}, Cacheable: false},
		{ID: "kernel.build", Type: "kernel-builder", Dependencies: []string{"kernel.config"}, Cacheable: true, Timeout: 90 * time.Minute},

		{ID: "uboot.clone", Type: "bootloader-builder", Cacheable: false},
		{ID: "uboot.patch", Type: "bootloader-builder", Dependencies: []string{"uboot.clone"}, Cacheable: false},
		{ID: "uboot.blobs", Type: "bootloader-builder", Dependencies: []string{"uboot.clone"}, Cacheable: true},
		{ID: "uboot.build", Type: "bootloader-builder", Dependencies: []string{"uboot.patch", "uboot.blobs"}, Cacheable: true, Timeout: 45 * time.Minute},

		{ID: "rootfs.create", Type: "rootfs-builder", Cacheable: true, Timeout: 30 * time.Minute},
		{ID: "rootfs.customize", Type: "rootfs-builder", Dependencies: []string{"rootfs.create"}, Cacheable: true},

		{
			ID:           "image.assemble",
			Type:         "image-assembler",
			Dependencies: []string{"kernel.build", "uboot.build", "rootfs.customize"},
			Cacheable:    false,
			Timeout:      10 * time.Minute,
		},
	}

	for _, t := range tasks {
		// Errors only arise from duplicate IDs, which cannot happen here.
		_ = p.Add(t)
	}
	return p
}
