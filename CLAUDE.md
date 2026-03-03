# CLAUDE.md — ELMOS Codebase Codex

This file governs how AI assistants (and human contributors) should approach this codebase.
Read it before writing a single line of code.

---

## Project Identity

| Field      | Value                                                         |
| ---------- | ------------------------------------------------------------- |
| Module     | `github.com/NguyenTrongPhuc552003/elmos`                      |
| Go version | `1.26` (see `go.mod`)                                         |
| Binary     | `build/elmos` (produced by `task build`)                      |
| Purpose    | Embedded Linux SDK — smarter, faster alternative to Buildroot |

---

## Quick-Start Commands

```bash
task build              # compile → build/elmos
task dev:check          # fmt + lint/vet (must produce zero output)
task dev:complexity     # cyclomatic CC > 10 flagged, MI < 30 flagged
task clean              # remove build/
task release:darwin     # darwin amd64 + arm64 release binaries
task release:linux      # linux amd64 + arm64 release binaries (Phase 4)
task release:windows    # windows amd64 .exe (Phase 4)
```

After **every** code change, run `task dev:check && task build` before committing.

---

## Architecture Layers

```
cmd/elmos/main.go            ← binary entry point only; no logic here
core/app/                    ← CLI wiring, cobra commands, App struct
core/config/                 ← config structs + loader; no business logic
core/context/                ← shared runtime state passed through layers
core/domain/                 ← all business logic lives here
  bsp/                       ← BSP registry client + firmware blob manager
  builder/                   ← kernel build orchestration (v1.0)
  doctor/                    ← environment health checks
  emulator/                  ← QEMU integration
  orchestrator/               ← DAG pipeline + fingerprinter
  patch/                     ← patch application
  plugin/builtin/            ← compiled-in plugins
  rootfs/                    ← rootfs creation
  toolchain/                 ← cross-compilation toolchain management
core/infra/                  ← I/O adapters (no domain logic)
  executor/                  ← shell command execution interface
  filesystem/                ← file I/O abstractions
  homebrew/                  ← Homebrew resolver (macOS only; use via platform layer)
  platform/                  ← OS abstraction (Phase 4; darwin/linux/windows impls)
core/plugin/                 ← plugin registry + hook executor + factory map
core/ui/                     ← printer, TUI components
```

### Import Rules (strictly enforced)

- `domain/` packages **must not** import `app/` or `app/commands/`
- `infra/` packages **must not** import `domain/`
- `plugin/builtin/` **must not** import `plugin/` directly — only `core/plugin` interfaces
- `app/commands/` may import any layer below it
- Circular imports are always wrong; use interface injection to break cycles
- The `core/plugin` ↔ `core/domain/plugin/builtin` cycle is broken via:
  - `core/plugin` exposes `RegisterBuiltinFactory(name, factory)`
  - `core/domain/plugin/builtin/init.go` calls `RegisterBuiltinFactory` in `init()`
  - `core/app/app.go` blank-imports `_ "github.com/.../core/domain/plugin/builtin"` to trigger `init()`

---

## Adding a New Builtin Plugin (3-Step Recipe)

### Step 1 — Create the plugin file

`core/domain/plugin/builtin/my_plugin.go`:

```go
package builtin

import (
    elcontext "github.com/NguyenTrongPhuc552003/elmos/core/context"
    "github.com/NguyenTrongPhuc552003/elmos/core/plugin"
)

type MyPlugin struct{ config *MyConfig }
type MyConfig struct{ /* fields */ }

func NewMyPlugin() (*MyPlugin, error) { return &MyPlugin{config: &MyConfig{}}, nil }

func (p *MyPlugin) Name() string        { return "my-plugin" }
func (p *MyPlugin) Version() string     { return "1.0.0" }
func (p *MyPlugin) Description() string { return "one-line description" }
func (p *MyPlugin) Cleanup() error      { return nil }
func (p *MyPlugin) Validate() error     { return nil }

func (p *MyPlugin) Init(ctx *elcontext.Context, cfg map[string]interface{}) error {
    // store ctx; parse cfg if needed; never fail on nil ctx
    return nil
}

func (p *MyPlugin) Hooks() []plugin.HookRegistration {
    return []plugin.HookRegistration{
        {Event: plugin.PreKernelBuild, Handler: p.onPreBuild, Priority: 5},
    }
}
```

### Step 2 — Register the factory

In `core/domain/plugin/builtin/init.go`, add:

```go
func init() {
    // existing registrations…
    plugin.RegisterBuiltinFactory("my-plugin", func() (plugin.Plugin, error) {
        return NewMyPlugin()
    })
}
```

### Step 3 — Enable in loader

In `core/plugin/loader.go`, add `"my-plugin"` to the `builtinPlugins` slice inside `LoadBuiltins`.

### Verify

```bash
task build && ./build/elmos plugins list   # must show my-plugin
```

---

## Adding a New CLI Command (2-Step Recipe)

### Step 1 — Create the command file

`core/app/commands/mycommand.go`:

```go
package commands

import "github.com/spf13/cobra"

func init() { registerCommand(newMyCommand) }

func newMyCommand(ctx *Context) *cobra.Command {
    return &cobra.Command{
        Use:   "mycommand",
        Short: "one-line description",
        RunE: func(cmd *cobra.Command, args []string) error {
            // use ctx.Config, ctx.Printer, etc.
            return nil
        },
    }
}
```

### Step 2 — Wire in registry

`core/app/commands/registry.go` auto-discovers commands registered via `registerCommand(...)` in `init()` — no manual addition needed if you used `init()`.

---

## Platform Path Rules

**Never** hardcode OS-specific paths. Every path that differs between operating systems must go through the platform layer (`core/infra/platform/`) once it exists (Phase 4). Until then, use the existing helpers:

| Don't do this                       | Do this instead                                                 |
| ----------------------------------- | --------------------------------------------------------------- |
| `/Volumes/<name>`                   | `platform.Current().Paths().WorkspaceRoot(name)` (Phase 4)      |
| `/opt/homebrew/bin/gcc`             | `platform.Current().Packages().GetBinPath("gcc")` (Phase 4)     |
| `exec.Command("brew", "install")`   | `platform.Current().Packages().Install(pkg)` (Phase 4)          |
| `exec.Command("hdiutil", "create")` | `platform.Current().DiskImage().Create(path, sizeGB)` (Phase 4) |
| `/usr/local/` hardcoded             | `platform.Current().Paths().ToolchainDir()` (Phase 4)           |

During Phase 1–3 (before platform layer exists), add a `// TODO(platform)` comment next to any macOS-specific call so they are findable:

```bash
grep -r "TODO(platform)" core/   # find all pending platform abstractions
```

---

## Cache Directory Convention

All cache/state stored under `~/.elmos/`. Never invent a new top-level path.

```go
// Correct — always derive from UserHomeDir
homeDir, err := os.UserHomeDir()
cacheDir := filepath.Join(homeDir, ".elmos", "bsp-cache")

// Wrong
cacheDir := "/tmp/elmos-cache"
cacheDir := os.Getenv("HOME") + "/.cache/elmos"
```

---

## Error Wrapping

Always wrap errors with context. Use `%w` not `%v` so callers can use `errors.Is` / `errors.As`.

```go
// Correct
return fmt.Errorf("bsp: failed to fetch machine %q: %w", name, err)

// Wrong — loses error type information
return fmt.Errorf("bsp: failed to fetch machine %q: %v", name, err)
return errors.New("fetch failed")
```

Sentinel errors for public APIs: define as `var ErrFoo = errors.New("...")` in the package that owns the concept.

---

## Complexity Limits

- **Cyclomatic Complexity (CC)**: max **10** per function. Functions flagged by `task dev:complexity` (`--cycloover 10`) must be fixed before merge.
- **Maintainability Index (MI)**: must stay **≥ 30**. Functions flagged by `--maintunder 30` must be refactored.

To reduce CC, extract decision points into named helpers:

```go
// Instead of:
func Build(ctx) error {
    if config.PatchDir != "" {
        if err := applyPatches(ctx); err != nil { ... }
    }
}

// Do:
func Build(ctx) error {
    if err := applyPatches(ctx); err != nil { ... }  // guard moved inside helper
}
func applyPatches(ctx) error {
    if config.PatchDir == "" { return nil }           // early-return guard
    ...
}
```

---

## Global State Rules

- **No global mutable state** except the `builtinFactories` map in `core/plugin/loader.go`
  (which is written only during `init()` before `main()` runs — effectively immutable at runtime).
- **No `init()` side effects** except registering factories via `plugin.RegisterBuiltinFactory`.
- All configuration reaches components via **constructor injection** — never `os.Getenv` mid-function
  (read env vars at startup/config-load time only).
- **No package-level `var` that holds runtime state** (open files, goroutines, connections).

---

## Shell Command Execution

All shell commands **must** go through `executor.Executor`. Never call `exec.Command` directly in domain or plugin code.

```go
// Correct — testable, mockable, dry-run capable
func (p *Plugin) Build(ctx context.Context, dir string) error {
    return p.exec.Run(ctx, "make", "-C", dir, "-j8")
}

// Wrong — untestable, bypasses dry-run mode
func (p *Plugin) Build(ctx context.Context, dir string) error {
    return exec.Command("make", "-C", dir, "-j8").Run()
}
```

The `executor.Executor` interface is in `core/infra/executor/executor.go`. Inject it via the struct constructor, not via `init()` or globals.

---

## Plugin Hook Events

Defined in `core/plugin/interface.go`. Current supported events:

| Constant                     | When fired                            |
| ---------------------------- | ------------------------------------- |
| `plugin.PreKernelBuild`      | Before kernel compilation starts      |
| `plugin.PostKernelBuild`     | After kernel compilation completes    |
| `plugin.PreBootloaderBuild`  | Before bootloader build starts        |
| `plugin.PostBootloaderBuild` | After bootloader build completes      |
| `plugin.AfterConfigLoad`     | After workspace config YAML is parsed |
| `plugin.PreRootfsCreate`     | Before rootfs population              |
| `plugin.PostRootfsCreate`    | After rootfs population               |
| `plugin.PreImageAssemble`    | Before final image assembly           |
| `plugin.PostImageAssemble`   | After final image assembly            |

Hook priority: **higher number runs first** (range 0–10). Use 8–10 for validation hooks that must run before builders.

---

## Config YAML Structure

### Workspace root (`default.yml`)
```
version: 2.0
machine: "rock5b_plus"
kernel: { repo, version, patches }
bootloader: { type, repo, version }
rootfs: { distribution, release, packages }
build: { parallel_jobs, verbose, cache_strategy }
plugins: { enabled: [...], disabled: [...] }
```

### Machine definition (`machine/<name>.yml`)
```
machine:
  name: "rock5b_plus"
  manufacturer: "Radxa"
  kernel: { arch: "arm64", defconfig: "...", device_trees: [...] }
  bootloader: { type: "u-boot", defconfig: "...", binary: "u-boot.itb", firmware: [...] }
  qemu: { system: "qemu-system-aarch64", machine: "virt", cpu: "cortex-a72" }
  rootfs: { extra_packages: [...], kernel_modules: [...] }
```

The top-level `machine:` envelope is **required**. The BSP manager uses
`yaml.Unmarshal` into `struct { Machine *config.MachineDefinition \`yaml:"machine"\` }`.

---

## DAG Build Pipeline (Phase 3)

Task IDs follow `<component>.<action>` naming. Standard pipeline for a full board build:

```
kernel.clone → kernel.patch → kernel.config → kernel.build ──┐
uboot.clone  → uboot.patch  → uboot.blobs  → uboot.build  ──→ image.assemble
rootfs.create → rootfs.customize ─────────────────────────────┘
```

Independent branches run in parallel (bounded by `min(GOMAXPROCS, ready_tasks)`).
Cache hits (fingerprint match) skip the task entirely and restore artifacts.

---

## Code Review Checklist

Before submitting any PR, verify:

- [ ] `task dev:check` produces zero output
- [ ] `task dev:complexity` produces zero function-level warnings
- [ ] `task build` succeeds and `./build/elmos plugins list` shows all expected plugins
- [ ] No new `exec.Command` calls in `core/domain/` or `core/plugin/`
- [ ] No hardcoded `/Volumes/`, `/opt/homebrew`, `/usr/local/`; add `// TODO(platform)` if unavoidable
- [ ] All errors wrapped with `%w` and context prefix
- [ ] No new `init()` side effects except factory registration
- [ ] CC ≤ 10 for every new function (count manually if unsure)
- [ ] New builtin plugin registered in `builtin/init.go` AND enabled in `loader.go`
- [ ] New CLI command uses `init()` self-registration pattern (no manual registry edit)

---

## Phases Roadmap

| Phase   | Status      | Scope                                                          |
| ------- | ----------- | -------------------------------------------------------------- |
| Phase 1 | Complete    | Config, plugin registry, kernel builder, domain scaffold       |
| Phase 2 | Complete    | BSP manager, firmware blobs, bootloader builder, fingerprinter |
| Phase 3 | In progress | DAG orchestrator, caching backend, rootfs/image plugins        |
| Phase 4 | Planned     | Platform abstraction (Linux + Windows support)                 |
| Phase 5 | Future      | Go `.so` user plugin loading, TUI live progress                |
