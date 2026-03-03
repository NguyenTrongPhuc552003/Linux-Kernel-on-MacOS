# STRATEGY.md — ELMOS v1.0 Release Strategy

> **Purpose**: Complete roadmap from current state to production-quality v1.0 release.
> **Scope**: Security hardening, testing, CI/CD, Phase 3–5 completion, architecture improvements, new features.
> **Date**: March 2026

---

## 1. Executive Summary

ELMOS is an Embedded Linux SDK with a well-structured architecture across 79 Go files (~2,582 lines), six builtin plugins, a DAG-based build orchestrator, and multi-platform support (macOS, Linux/WSL2, Windows). However, the path to v1.0 requires closing critical gaps: **seven security vulnerabilities** must be remediated, a **complete testing infrastructure** must be built from zero (0 test files today), **CI/CD must be established**, the **DAG pipeline executor must be wired** to actually execute build tasks (currently a skeleton), and the **platform abstraction layer** must be completed for image assembly. This document provides the sequenced, dependency-aware roadmap.

---

## 2. Current State Assessment

### Subsystem Maturity

| Subsystem           | Maturity | Key Gap                                                                    |
| ------------------- | -------- | -------------------------------------------------------------------------- |
| Config & Loading    | 85%      | No JSON schema validation at load time                                     |
| Plugin Registry     | 75%      | `Plugin.Init()` receives nil context; user plugins stubbed                 |
| Builtin Plugins (6) | 60%      | Hooks are observers only — no task delegation; manual `Set*()` injection   |
| Kernel Builder      | 80%      | No GCC fallback; LLVM-only                                                 |
| Bootloader Builder  | 70%      | Executor must be injected manually                                         |
| BSP Registry        | 80%      | Silent cache write failures                                                |
| Fingerprinter       | 90%      | Complete but unused — cached artifacts never restored                      |
| DAG Orchestrator    | 40%      | `runTask()` fires hooks but doesn't execute; hook names misaligned         |
| Rootfs Creator      | 65%      | Works standalone; not wired to pipeline                                    |
| Image Assembler     | 45%      | Uses raw `dd`; `writeKernelToRootfs()` returns nil; no GPT                 |
| Platform Layer      | 75%      | Linux/macOS functional; IsMounted + hdiutil fixed; 9 TODO(platform) remain |
| QEMU Runner         | 75%      | Config values interpolated without sanitization                            |
| Patch Manager       | 70%      | Path traversal via absolute paths                                          |
| Doctor/Fixer        | 80%      | Functional across platforms                                                |
| CLI Commands (17)   | 85%      | Auto-registration pattern; well-structured                                 |
| Testing             | **0%**   | Zero `*_test.go` files in entire codebase                                  |
| CI/CD               | **0%**   | No `.github/workflows/`; no automation                                     |

### Architecture Health

- **Import discipline**: Clean — zero circular imports detected
- **Code quality**: gofmt clean, CC ≤ 10, MI ≥ 30 enforced
- **Dependency injection**: Constructor-based with clean interfaces (`executor.Executor`, `filesystem.FileSystem`, `platform.Platform`)
- **Dependencies**: 7 direct (cobra, viper, bubbletea, lipgloss, yaml), ~30 transitive

---

## 3. Release Milestones

### v0.5.0 — Stability Foundation

| Deliverable          | Description                                               |
| -------------------- | --------------------------------------------------------- |
| SEC-1 through SEC-7  | All seven security vulnerabilities remediated             |
| Test framework       | `go test ./...` runs, testify added, coverage reporting   |
| Unit test coverage   | ≥ 40% line coverage across `core/`                        |
| CI pipeline          | GitHub Actions: lint, test, build, complexity on every PR |
| golangci-lint config | `.golangci.yml` with project-specific rules               |
| Go version alignment | Resolve `go.mod` (1.23) vs CLAUDE.md (1.26) discrepancy   |

### v0.8.0 — Feature Complete

| Deliverable            | Description                                                   |
| ---------------------- | ------------------------------------------------------------- |
| PipelineExecutor wired | `runTask()` delegates to plugin Build/Create methods          |
| Hook event alignment   | Orchestrator uses standard `plugin.*` event constants         |
| Cache artifact restore | `CachingBackendPlugin.RestoreArtifacts()` called on cache hit |
| Task timeout           | `BuildTask.Timeout` enforced via `context.WithTimeout`        |
| Dry-run mode           | Pipeline prints execution plan without running                |
| Plugin context fix     | `Plugin.Init()` receives real `*elcontext.Context`            |
| Plugin DI container    | Automatic executor/filesystem injection for all plugins       |
| Integration tests      | End-to-end tests for `init`, `doctor`, `plugins list`         |
| Coverage               | ≥ 65% line coverage                                           |

### v1.0.0 — Production Ready

| Deliverable                          | Description                                   |
| ------------------------------------ | --------------------------------------------- |
| Image assembler via DiskImageManager | Replace `dd` calls with platform layer        |
| `writeKernelToRootfs`                | Mount rootfs, copy kernel to `/boot`, unmount |
| GPT partition support                | Proper partition table creation               |
| Structured logging                   | `slog`-based logging replacing `fmt.Printf`   |
| Config validation                    | JSON schema validation at load time           |
| Release automation                   | Cross-platform builds via GitHub Actions      |
| Security scanning                    | gosec + govulncheck in CI                     |
| Coverage                             | ≥ 80% line coverage                           |
| Documentation                        | CLI reference, architecture guide             |

---

## 4. Security Hardening (Priority 0)

All seven must be fixed before feature work. Each fix includes a unit test.

### SEC-1: Path Traversal in Patch Manager

**File**: `core/domain/patch/manager.go` (lines 32-51)
**Issue**: `Apply()` accepts absolute paths without confinement to workspace.
**Fix**: Add `validatePathConfinement(basePath, targetPath)` — reject paths containing `..` segments or outside `PatchesDir`/`ProjectRoot` after `filepath.Clean()`.

### SEC-2: Privilege Escalation in Rootfs Creator

**File**: `core/domain/rootfs/creator.go` (lines 67, 88-89, 147-153)
**Issue**: Paths used in `sudo` commands without validation; error silently discarded on line 153.
**Fix**: Validate paths against expected prefixes (`RootfsDir`, `DiskImage`); log errors instead of discarding.

### SEC-3: Command Injection in QEMU Runner

**File**: `core/domain/emulator/qemu.go` (lines 118-175)
**Issue**: Config values interpolated into QEMU args without validation.
**Fix**: Add `validateQEMUConfig()` — regex-validate Memory (`^\d+[MG]$`), port numbers (1-65535), reject null bytes in paths.

### SEC-4: Environment Variable Duplication

**File**: `core/infra/executor/shell.go` (lines 56-59, 72-75, 86-89)
**Issue**: `append(os.Environ(), env...)` creates duplicate keys; behavior undefined.
**Fix**: Create `mergeEnv(base, override []string) []string` helper in `core/infra/executor/env.go`.

### SEC-5: Silent Post-Hook Errors

**File**: `core/domain/orchestrator/executor.go` (lines 222, 231)
**Issue**: `_ = pe.firePostHook()` discards errors silently.
**Fix**: Log post-hook errors; add `Warnings []error` field to `TaskResult`.

### SEC-6: Cache Write Opacity

**File**: `core/domain/bsp/registry.go` (lines 76-78, 106)
**Issue**: Cache write failures silently discarded.
**Fix**: Log at WARNING level via printer/logger dependency.

### SEC-7: Context Timeout Ignored

**Files**: `platform/linux.go`, `platform/darwin.go`, `platform/windows.go`, `rootfs/creator.go`
**Issue**: `context.Background()` used instead of caller's context.
**Fix**: Thread `context.Context` through all affected methods where possible.

---

## 5. Testing Strategy

### 5.1 Framework Setup

- Add `github.com/stretchr/testify` to `go.mod`
- Add `dev:test` and `dev:coverage` tasks to `Taskfile.yml`
- Update `dev:check` to include `dev:test`
- Create `MockFileSystem` following existing `MockExecutor` pattern
- Create `MockPlatform` with sub-interfaces

### 5.2 Unit Test Priority

| Priority | Package                      | Key Cases                                                |
| -------- | ---------------------------- | -------------------------------------------------------- |
| P0       | `core/domain/patch`          | Path traversal rejection                                 |
| P0       | `core/infra/executor`        | Env dedup, all Run* variants                             |
| P0       | `core/domain/rootfs`         | Path validation before sudo                              |
| P0       | `core/domain/emulator`       | QEMU arg construction with malformed config              |
| P1       | `core/domain/orchestrator`   | DAG sort (cycles, diamond deps), fingerprint determinism |
| P1       | `core/plugin`                | Hook priority ordering, required vs non-required         |
| P1       | `core/config`                | Loader missing file, defaults, machine validation        |
| P2       | `core/domain/bsp`            | Registry cache hit/miss, TTL                             |
| P2       | `core/domain/plugin/builtin` | Each plugin Init/Validate/Hooks                          |
| P3       | `core/context`               | GetMakeEnv, mount checks                                 |
| P3       | `core/app/commands`          | Command smoke tests                                      |

### 5.3 Integration Tests

- Build tag: `//go:build integration`
- `elmos init` end-to-end with temp directory
- `elmos doctor` with mocked environment
- `elmos plugins list` verifying all 6 plugins
- Pipeline sort determinism
- Fingerprint reproducibility

### 5.4 Benchmarks

- `fingerprinter.go`: `BenchmarkHashFile`, `BenchmarkHashDirectory`
- `dag.go`: `BenchmarkPipelineSort` with 10/50/100 tasks
- `executor.go` (hooks): `BenchmarkHookExecute` with varying counts

---

## 6. CI/CD Pipeline

### `vercel.json` (CI/CD Configuration)

```json
{
  "buildCommand": "task build",
  "devCommand": "task dev:check",
  "installCommand": "go mod download",
  "framework": null,
  "builds": [
    {
      "src": "cmd/elmos",
      "use": "@vercel/static-build"
    }
  ],
  "github": {
    "enabled": true,
    "silent": false
  }
}
```

**Note:** Testing infrastructure will be maintained in a separate `test` branch.
Vercel will run the following checks on every push:
- `task dev:check` - Format checking, linting, and tests
- `task dev:complexity` - Cyclomatic complexity analysis
- `task build` - Multi-platform binary builds

### Release Automation (Vercel Deploy Hooks)

- Triggered on: git tag push (v*)
- Builds: darwin (amd64, arm64), linux (amd64, arm64), windows (amd64)
- Artifacts: Binaries + SHA256 checksums + Release notes
- Distribution: GitHub Releases + Homebrew formula update

### `.golangci.yml`

Linters: `errcheck`, `govet`, `staticcheck`, `gosec`, `cyclop` (CC ≤ 10), `gofmt`, `misspell`

---

## 7. Phase 3 Completion — Wire the DAG

### 7.1 TaskRunner Interface

**File**: `core/domain/orchestrator/runner.go` (new)

```go
type TaskRunner interface {
    RunTask(ctx context.Context, taskID string, config map[string]any) error
}
```

Each builtin plugin implements this. PipelineExecutor gets `runners map[string]TaskRunner`.

### 7.2 Wire `runTask()`

**File**: `core/domain/orchestrator/executor.go` (lines 201-233)

Between cache-hit check and post-hook, delegate to runner:

```go
if runner, ok := pe.runners[task.Type]; ok {
    result.Err = runner.RunTask(taskCtx, task.ID, task.Config)
}
```

### 7.3 Hook Event Alignment

**File**: `core/domain/orchestrator/executor.go`

Replace string concatenation (`"pre_"+task.Type+"_execute"`) with mapping:

```go
var taskTypeToPreEvent = map[string]string{
    "kernel-builder":     plugin.PreKernelBuild,
    "bootloader-builder": plugin.PreBootloaderBuild,
    "rootfs-builder":     plugin.PreRootfsCreate,
    "image-assembler":    plugin.PreImageAssemble,
}
```

### 7.4 Cache Artifact Restoration

On cache hit, call `CachingBackendPlugin.RestoreArtifacts()` before returning. Fall through to execution if restore fails.

### 7.5 Task Timeout Enforcement

Wrap context: `context.WithTimeout(ctx, task.Timeout)` when `task.Timeout > 0`.

### 7.6 Dry-Run Mode

Add `DryRun bool` field; when true, log task plan without executing. Add `--dry-run` CLI flag.

### 7.7 Error Recovery

Add `OnError` strategy: `StopOnFirst` (default) | `ContinueIndependent` (cancel only dependent tasks).

---

## 8. Phase 4 Completion — Platform Abstraction

### 8.1 Image Assembler → DiskImageManager

**File**: `core/domain/plugin/builtin/image_assembler.go`

- `createImageFile()`: Replace `dd` with `platform.DiskImage().Create()`
- `writeRootfs()`: Mount → rsync → Unmount via platform layer
- `writeKernelToRootfs()`: Mount → `mkdir /boot` → `cp kernel` → Unmount

### 8.2 GPT Partition Support

Add `CreateWithPartitions(ctx, path, []PartitionSpec) error` to `DiskImageManager`.
Linux: `sgdisk`; macOS: `diskutil`; Windows: WSL2 `sgdisk`.

### 8.3 Windows WSL2 Fix

**File**: `core/infra/platform/windows.go`

Replace shell string interpolation in `Mount()` with separate exec calls.

---

## 9. Phase 5 — User Plugin System

### Architecture

```
User code → go build -buildmode=plugin → myplugin.so
Runtime  → plugin.Open() → Lookup("ElmosPlugin") → Register
```

### Plugin Manifest (`~/.elmos/plugins/<name>/plugin.yml`)

```yaml
name: my-plugin
entrypoint: myplugin.so
needs: [executor, filesystem]
hooks:
  - event: pre_kernel_build
    priority: 7
```

### Security

- Capability-based injection (only declared `needs`)
- Timeout enforcement on all hook calls
- Panic recovery wrapping
- Optional `.sig` file for signature verification

---

## 10. Architecture Improvements

### 10.1 Plugin DI Container

**File**: `core/plugin/injector.go` (new)

```go
type ExecutorAware interface { SetExecutor(executor.Executor) }
type FilesystemAware interface { SetFilesystem(filesystem.FileSystem) }
type ContextAware interface { SetContext(*elcontext.Context) }

func InjectDependencies(p Plugin, deps Dependencies) { ... }
```

Auto-inject via interface type assertions. Eliminates manual `Set*()` calls in `app.go`.

### 10.2 Context Propagation

**File**: `core/plugin/loader.go` (line 141)

Fix `plugin.Init(nil, config)` → pass real `*elcontext.Context`. Add `SetContext()` to Registry.

### 10.3 Sentinel Errors

Per-package sentinel errors with `%w` wrapping:

```go
var ErrPathTraversal = errors.New("patch: path escapes workspace")
```

### 10.4 Structured Logging

Replace `fmt.Printf` in domain code with `log/slog`. Keep `ui.Printer` for TUI-only output.

### 10.5 JSON Schema Validation

Embed existing `assets/schemas/*.json` via `//go:embed`. Validate config YAML against schema in `config.Load()`.

---

## 11. New Features (Post-v1.0)

| Feature           | Description                                     | Depends On          |
| ----------------- | ----------------------------------------------- | ------------------- |
| `elmos workspace` | Multi-workspace management (create/list/switch) | Context propagation |
| `elmos flash`     | Direct board flashing (dd/rkdevtool/fastboot)   | Image assembler     |
| BSP Registry v1   | Formalized HTTP API with versioning + auth      | BSP registry        |
| Artifact signing  | Ed25519 signing + `elmos verify`                | Release automation  |
| Remote builds     | `--remote <host>` via SSH + rsync               | Pipeline executor   |
| TUI DAG progress  | Live pipeline visualization in bubbletea        | DAG wiring          |

---

## 12. Implementation Order

```
Seq   Work Item                           Depends On      Milestone
---   ----------------------------------  --------------  ---------
S-01  Go version alignment                (none)          v0.5.0
S-02  Add testify to go.mod               (none)          v0.5.0
S-03  Create MockFileSystem/MockPlatform  S-02            v0.5.0
S-04  SEC-4: Env dedup (executor)         (none)          v0.5.0
S-05  SEC-1: Path traversal (patch)       S-02,S-03       v0.5.0
S-06  SEC-2: Rootfs path validation       S-02,S-03       v0.5.0
S-07  SEC-3: QEMU config validation       S-02            v0.5.0
S-08  SEC-5: Post-hook error handling     S-02            v0.5.0
S-09  SEC-6: Cache write logging          S-02            v0.5.0
S-10  SEC-7: Context propagation          (none)          v0.5.0
S-11  golangci-lint config                (none)          v0.5.0
S-12  CI pipeline (lint+test+build)       S-02,S-11       v0.5.0
S-13  Unit tests: orchestrator DAG        S-02            v0.5.0
S-14  Unit tests: plugin hook executor    S-02            v0.5.0
S-15  Unit tests: config loader           S-02,S-03       v0.5.0
S-16  Unit tests: BSP registry            S-02            v0.5.0
S-17  Unit tests: platform providers      S-02            v0.5.0
S-18  Coverage gate in CI (40%)           S-12..S-17      v0.5.0

S-19  Plugin DI container (injector.go)   S-10            v0.8.0
S-20  Context fix (nil → real ctx)        S-19            v0.8.0
S-21  Hook event name alignment           S-08            v0.8.0
S-22  TaskRunner interface + wiring       S-19,S-21       v0.8.0
S-23  Cache artifact restoration          S-22            v0.8.0
S-24  Task timeout enforcement            S-22            v0.8.0
S-25  Dry-run mode                        S-22            v0.8.0
S-26  Error recovery (ContinueIndep.)     S-22            v0.8.0
S-27  Sentinel errors per package         S-05..S-09      v0.8.0
S-28  Integration tests                   S-18,S-20       v0.8.0
S-29  Unit tests: pipeline execution      S-22            v0.8.0
S-30  Coverage gate in CI (65%)           S-28,S-29       v0.8.0

S-31  Image assembler → DiskImageMgr      S-22            v1.0.0
S-32  writeKernelToRootfs implementation  S-31            v1.0.0
S-33  GPT partition support               S-31            v1.0.0
S-34  Windows WSL2 cmd injection fix      S-07            v1.0.0
S-35  Structured logging (slog)           S-27            v1.0.0
S-36  JSON schema validation              S-15            v1.0.0
S-37  Security scanning in CI             S-12            v1.0.0
S-38  Release automation                  S-12            v1.0.0
S-39  Coverage gate (80%)                 S-30+tests      v1.0.0
S-40  Documentation                       (none)          v1.0.0

S-41  elmos workspace commands            S-20            post-1.0
S-42  elmos flash                         S-32            post-1.0
S-43  User plugin .so loading             S-19            post-1.0
S-44  BSP registry protocol v1            S-16            post-1.0
S-45  Build artifact signing              S-38            post-1.0
S-46  Remote build support                S-22            post-1.0
S-47  TUI DAG progress                    S-22            post-1.0
```

### Dependency DAG (visual)

```
        S-01  S-02  S-11
         |   / |  \   |
         |  S-03  S-04 |
         | / | \   |   |
        S-05 S-06 S-07 |
         |    |    |   |
        S-08 S-09 S-10 |
         |    |    |   |
        S-12--+----+---+
         |
  S-13..S-17
         |
        S-18 ═══ v0.5.0 gate
         |
   S-19──S-20
    |     |
   S-21  S-28
    |
   S-22
  / | \  \
S-23 S-24 S-25 S-26
         |
        S-29
         |
        S-30 ═══ v0.8.0 gate
         |
 S-31──S-32──S-33
   |
  S-35  S-36  S-37  S-38
         |
        S-39 ═══ v1.0.0 gate
```

---

## 13. Critical Files

| File                                            | Changes Needed                                                     |
| ----------------------------------------------- | ------------------------------------------------------------------ |
| `core/domain/orchestrator/executor.go`          | Wire runTask(), align hook events, restore cache, enforce timeouts |
| `core/plugin/loader.go`                         | Fix nil context (line 141), DI integration, user plugin loading    |
| `core/domain/plugin/builtin/image_assembler.go` | Replace dd → DiskImageMgr, implement writeKernelToRootfs           |
| `core/infra/executor/shell.go`                  | Env deduplication via mergeEnv helper                              |
| `core/domain/patch/manager.go`                  | Path confinement for Apply/Reverse                                 |
| `core/domain/rootfs/creator.go`                 | Path validation, error logging                                     |
| `core/domain/emulator/qemu.go`                  | Config validation before arg construction                          |
| `core/domain/bsp/registry.go`                   | Cache write error logging                                          |
| `core/domain/orchestrator/runner.go`            | New: TaskRunner interface                                          |
| `core/plugin/injector.go`                       | New: Auto-injection for plugin dependencies                        |
| `.github/workflows/ci.yml`                      | New: CI pipeline                                                   |
| `.golangci.yml`                                 | New: Lint configuration                                            |

---

## 14. Verification Checklist

### v0.5.0

- [ ] `task dev:check` produces zero output (includes `go test ./...`)
- [ ] `task dev:complexity` produces zero warnings
- [ ] `task build` succeeds on Linux, macOS, and Windows cross-compile
- [ ] All 7 SEC-* issues fixed with corresponding `*_test.go` files
- [ ] `go test -cover ./...` reports ≥ 40%
- [ ] GitHub Actions CI green on PR to main
- [ ] `.golangci.yml` present and passing
- [ ] No `exec.Command` in `core/domain/` or `core/plugin/`

### v0.8.0

- [ ] All v0.5.0 criteria passing
- [ ] `PipelineExecutor.Execute()` with mock runners completes all 11 tasks
- [ ] Cache hit restores artifacts (verified by test)
- [ ] Task with 1s timeout fails with `context.DeadlineExceeded`
- [ ] `--dry-run` prints plan without side effects
- [ ] `Plugin.Init()` receives non-nil context
- [ ] No manual `Set*()` calls — all via `InjectDependencies`
- [ ] Hook events match `plugin.*` constants
- [ ] `go test -cover ./...` ≥ 65%

### v1.0.0

- [ ] All v0.8.0 criteria passing
- [ ] `ImageAssemblerPlugin` uses DiskImageManager
- [ ] `writeKernelToRootfs()` copies kernel to `/boot`
- [ ] GPT partition creation in test
- [ ] `gosec` + `govulncheck` report zero high-severity findings
- [ ] `go test -cover ./...` ≥ 80%
- [ ] Release binaries: darwin/{amd64,arm64}, linux/{amd64,arm64}, windows/amd64
- [ ] `slog` in domain code, `Printer` only in UI
- [ ] Config validated against JSON schema
- [ ] All `// TODO(platform)` resolved or tracked as post-1.0

---

## 15. Consolidated Improvement Roadmap (March 2026 Update)

> Based on deep codebase audit. Groups are **de-duplicated** against existing S-XX items.
> New items have N-XX IDs and slot into the existing dependency DAG.

### Group A — Critical Correctness (blocks pipeline execution)

| ID   | Task                                                          | Files                                    | Status      | Ref            |
| ---- | ------------------------------------------------------------- | ---------------------------------------- | ----------- | -------------- |
| S-21 | Fix DAG hook event name mismatch                              | `orchestrator/executor.go`               | not-started | §7.3           |
| N-01 | Fix cache lookup parameter mismatch (`task.ID` → fingerprint) | `orchestrator/executor.go`               | not-started | §7.4           |
| S-22 | Wire `TaskRunner` implementations in builtin plugins          | 6 builtin plugin files                   | not-started | §7.2           |
| N-02 | Remove `exec.Command` from `toolchain/manager.go`             | `toolchain/manager.go`                   | not-started | CLAUDE.md rule |
| N-03 | Remove `os.Getenv` from domain layer                          | `toolchain/builder.go`, `builder/env.go` | not-started | CLAUDE.md rule |

### Group B — Error Handling & Diagnostics

| ID   | Task                                                  | Files                                   | Status      | Ref                         |
| ---- | ----------------------------------------------------- | --------------------------------------- | ----------- | --------------------------- |
| N-04 | ~~Surface `exec.ExitError.Stderr` in platform layer~~ | `darwin.go`                             | **DONE**    | Fixed via `wrapExecError()` |
| S-24 | Add timeout enforcement to pipeline tasks             | `orchestrator/executor.go`              | not-started | §7.5                        |
| N-05 | Add panic recovery to hook execution                  | `plugin/executor.go`                    | not-started | —                           |
| N-06 | Remove dead `errorHandler` code in HookExecutor       | `plugin/executor.go`                    | not-started | —                           |
| N-07 | Replace `fmt.Printf` in domain with injected printer  | `emulator/qemu.go`, `rootfs/creator.go` | not-started | Prep for slog               |

### Group C — Performance Optimization

| ID   | Task                                                      | Files                           | Rationale                    |
| ---- | --------------------------------------------------------- | ------------------------------- | ---------------------------- |
| N-08 | Cache fingerprint index in memory (11+ disk reads → 1)    | `orchestrator/fingerprinter.go` | O(n) file reads per pipeline |
| N-09 | Sort hooks lazily (dirty flag, not on every `Register()`) | `plugin/executor.go`            | O(n log n) per registration  |
| N-10 | Use `strings.Builder` in TUI viewport refresh             | `ui/tui/view.go`                | O(n) join per log line       |
| N-11 | Replace `maxInt` with Go 1.26 `max()` builtin             | `ui/tui/update.go`              | Dead helper                  |

### Group D — Deduplication & Code Health

| ID   | Task                                                | Files                               | Issue                                                    |
| ---- | --------------------------------------------------- | ----------------------------------- | -------------------------------------------------------- |
| N-12 | Merge `handleQuit()`/`popMenuStack()` in TUI        | `ui/tui/update.go`                  | Identical logic duplicated                               |
| N-13 | Remove `GetPlugin()` (duplicate of `Get()`)         | `plugin/loader.go`                  | Two methods, same behavior                               |
| N-14 | Unify `commandFormatters`/`actionArgsDispatch` maps | `ui/tui/update.go`                  | Parallel dispatch tables                                 |
| N-15 | Sync CLAUDE.md with actual command registry pattern | `CLAUDE.md`, `commands/registry.go` | Docs say `init()` pattern, code uses manual `Register()` |

### Group E — Platform Abstraction (Phase 4, existing S-31..S-34)

| ID   | Task                                                | TODO Ref         | Notes                                          |
| ---- | --------------------------------------------------- | ---------------- | ---------------------------------------------- |
| N-16 | Add `GetLibexecBin()` to `PackageManager` interface | context.go L24   | Eliminates direct homebrew.Resolver in Context |
| S-31 | Wire `DiskImageManager` into `image_assembler.go`   | 4 TODOs          | Replace raw `dd`/`hdiutil`                     |
| S-33 | GPT partition table support                         | linux.go TODO    | Real hardware images                           |
| N-17 | Add force-unmount to platform layer                 | init.go L274     | `hdiutil detach -force`                        |
| N-18 | Replace hardcoded brew path in toolchain builder    | builder.go TODOs | Use `Packages().GetBinPath()`                  |

### Group F — TUI Enhancement (Phase 5+)

| ID   | Task                                   | Priority | Notes                                   |
| ---- | -------------------------------------- | -------- | --------------------------------------- |
| S-47 | DAG pipeline visualization             | High     | Running/pending/complete/failed states  |
| N-19 | Build progress bars                    | High     | Replace spinner with real progress      |
| N-20 | Log search/filter in viewport          | Medium   | `grep`-style search                     |
| N-21 | Persistent log file (`~/.elmos/logs/`) | Medium   | Output survives TUI exit                |
| N-22 | Plugin management in TUI               | Low      | `plugins list/enable/disable` from menu |
| N-23 | Config editor in TUI                   | Low      | Edit `elmos.yaml` fields                |
| N-24 | Full navigation breadcrumbs            | Low      | `Kernel > Build > arm64` trail          |

### Group G — Testing & CI (0% → target 80%)

| ID         | Task                                                 | Priority | Ref                  |
| ---------- | ---------------------------------------------------- | -------- | -------------------- |
| S-02..S-03 | Test framework + mocks                               | Critical | §5.1                 |
| N-25       | Platform layer tests (mock-based)                    | High     | darwin/linux/windows |
| N-26       | Plugin system tests (hook ordering, builtin loading) | High     | §5.2 P1              |
| N-27       | DAG/orchestrator tests (topo sort, parallel exec)    | High     | §5.2 P1              |
| N-28       | TUI snapshot tests (golden output)                   | Medium   | —                    |
| N-29       | Integration test: full `init` → mount → exit flow    | Medium   | §5.3                 |
| S-12       | CI pipeline (GitHub Actions)                         | High     | §6                   |

### Recommended Execution Order

```
Phase A (Correctness):  S-21 → N-01 → S-22 → N-02 → N-03
Phase B (Diagnostics):  N-05 → N-06 → N-07 → S-24
Phase G1 (Test Setup):  S-02 → S-03 → N-25 → N-26 → N-27
Phase C (Performance):  N-08 → N-09 → N-10 → N-11
Phase D (Code Health):  N-12 → N-13 → N-14 → N-15
Phase G2 (Coverage):    N-28 → N-29 → S-12
Phase E (Platform):     N-16 → S-31 → S-33 → N-17 → N-18
Phase F (TUI):          S-47 → N-19 → N-20 → N-21
```

### Recently Fixed (March 3, 2026)

| Bug                              | Root Cause                                                                             | Fix                                                            |
| -------------------------------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------- |
| `hdiutil attach: exit status 1`  | Extension mismatch: config set `.sparseimage` but `Create()` used `-type SPARSEBUNDLE` | Changed to `-type SPARSE` producing single `.sparseimage` file |
| `volname` stripping no-op        | `TrimSuffix(base, ".sparsebundle")` on a `.sparseimage` path                           | Trim `.sparseimage` to get clean volume name                   |
| `IsMounted()` always false       | Exact `==` comparison on full `hdiutil info` line                                      | Changed to `HasSuffix` match on line endings                   |
| No diagnostic on mount failure   | `exec.ExitError.Stderr` not surfaced                                                   | Added `wrapExecError()` helper extracting stderr               |
| Pre-existing image check missing | `TODO(platform)` in `Create()`                                                         | Added `os.Stat` guard before `hdiutil create`                  |
