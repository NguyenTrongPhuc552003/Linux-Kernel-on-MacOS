# Architecture Diagrams

This page provides comprehensive visual documentation of ELMOS architecture at multiple abstraction levels. All diagrams are created with PlantUML and automatically rendered during documentation builds.

## Diagram Overview

### 1. Layered Architecture Diagram

```plantuml
--8<-- "docs/diagrams/architecture.puml"
```

**Purpose:** Shows the complete 7-layer architecture of ELMOS with all major components and their relationships.

**Key Elements:**
- **UI Layer:** CLI interface, TUI application, and output formatting
- **Application Layer:** Command handlers, dependency injection container, CLI11 wiring
- **Configuration Layer:** YAML parsing, machine definitions, workspace management
- **Context Layer:** Build state, environment management, output/error handling
- **Domain Layer:** Business logic (builder, orchestrator, plugin system, etc.)
- **Infrastructure Layer:** System abstractions (executor, filesystem, platform)
- **Cross-Cutting:** Logging, error handling, versioning

**Audience:** Architects, senior engineers, anyone understanding system structure

**When to reference:** During design reviews, when adding new subsystems, understanding integration points

---

### 2. System Context Diagram

```plantuml
--8<-- "docs/diagrams/system-context.puml"
```

**Purpose:** Shows ELMOS within its broader ecosystem and external system interactions.

**Key Elements:**
- **ELMOS System:** Core SDK with all services
- **Actors:** Developers, CI/CD pipelines, end users
- **External Systems:** 
  - Toolchain registries (GCC, LLVM sources)
  - Board support package registries
  - Linux kernel repositories
  - Package managers (Homebrew, apt, etc.)
- **Outputs:** Compiled kernels, rootfs images, executables

**Audience:** Stakeholders, deployment teams, users understanding system scope

**When to reference:** During deployment planning, integration discussions, understanding external dependencies

---

### 3. Build Pipeline State Machine

```plantuml
--8<-- "docs/diagrams/build-pipeline.puml"
```

**Purpose:** Models the complete build workflow as states and transitions, from initialization through final assembly.

**Key States:**
- **Initialization:** Validation, configuration loading, context building
- **DAG Planning:** Dependency analysis, fingerprinting, cache optimization
- **Compilation:** Parallel kernel/module/app builds
- **Assembly:** Rootfs population, image creation
- **Verification:** Testing and validation

**Transitions:** Include success paths, error recovery, and cache shortcuts

**Audience:** Build engineers, CI/CD specialists, anyone implementing build logic

**When to reference:** Implementing new build stages, understanding parallelization strategy

---

### 4. Data Flow Diagram

```plantuml
--8<-- "docs/diagrams/dataflow.puml"
```

**Purpose:** Illustrates how data flows through the system during a typical build operation.

**Key Data Entities:**
- **Configuration Sources:** Workspace config (elmos.yaml), machine definitions
- **Source Materials:** Linux kernel sources, module code, app code
- **Processing Pipeline:** Cross-compilation, patching, linking
- **Outputs:** Binary artifacts, filesystem images, boot images

**Data Transformations:** Shows which components transform which data

**Audience:** Integration engineers, developers adding new components

**When to reference:** Understanding data dependencies, adding new data flows

---

### 5. Plugin Architecture Diagram

```plantuml
--8<-- "docs/diagrams/plugin-architecture.puml"
```

**Purpose:** Details the plugin hook system and how third-party extensions integrate.

**Key Concepts:**
- **Plugin Interface:** Standard interface that all plugins implement
- **Hook System:** 13 lifecycle events that plugins can subscribe to
- **Priority Scheduling:** Hooks execute in priority order (0-10 range)
- **Factory Registration:** Builtin plugins register through factory pattern

**Lifecycle Hooks:**
- Pre/Post kernel build, bootloader, rootfs creation
- Pre/Post image assembly, QEMU boot
- On error, on cleanup

**Audience:** Plugin developers, extension authors

**When to reference:** Creating custom plugins, understanding extensibility points

---

### 6. Domain Classes Diagram

```plantuml
--8<-- "docs/diagrams/domain-classes-simple.puml"
```

**Purpose:** Shows the class hierarchy and relationships for major domain objects.

**Key Classes:**
- **Config:** Machine configuration, architecture definitions, workspace settings
- **Context:** Build state, environment, artifact tracking
- **Builders:** KernelBuilder, ModuleBuilder, AppBuilder (orchestration)
- **Orchestrator:** Pipeline DAG, fingerprinting, caching
- **Infrastructure:** Executor, FileSystem, Platform adapters

**Relationships:** Dependency injection patterns, composition hierarchies

**Audience:** C++ developers, implementation engineers

**When to reference:** Implementing new domain services, understanding class contracts

---

### 7. Sequence Diagram — Build Workflow

```plantuml
--8<-- "docs/diagrams/sequence-build.puml"
```

**Purpose:** Shows the interaction sequence between components during a kernel build operation.

**Key Interactions:**
1. Developer invokes `elmos kernel build`
2. CLI handler parses arguments and delegates to App container
3. App loads configuration and builds context
4. KernelBuilder orchestrates make invocations
5. Executor runs shell commands on host or target
6. FileSystem handles artifact storage and caching
7. Results propagate back through layers to CLI output

**Message Flow:** Synchronous calls with error handling and state updates

**Audience:** Developers debugging build issues, new team members

**When to reference:** Understanding call chains, tracing invocation flow

---

### 8. Deployment & Infrastructure Diagram

```plantuml
--8<-- "docs/diagrams/deployment-infrastructure-simple.puml"
```

**Purpose:** Shows how ELMOS components are deployed across development and build infrastructure.

**Deployment Contexts:**
- **Developer Machines:** Linux, macOS, Windows/WSL2 with ELMOS binary
- **Toolchain Infrastructure:** Downloaded/compiled cross-compilers, cached artifacts
- **Build Execution:** Local parallelization, optional remote builders
- **Emulation Layer:** QEMU VMs, disk images, networking

**Platform Abstractions:** Linux, Darwin (macOS), Windows (WSL2) implementations

**Audience:** DevOps engineers, infrastructure architects, deployment teams

**When to reference:** Setting up build infrastructure, understanding platform support

---

## Diagram Maintenance

### Adding New Diagrams

1. Create a new `.puml` file in `docs/diagrams/` following PlantUML 3.x syntax
2. Add the diagram to the appropriate subsection on this page
3. Include a "Purpose", "Key Elements", and "Audience" explanation
4. Run `task docs:serve` to verify rendering

### Updating Existing Diagrams

1. Edit the `.puml` file in `docs/diagrams/`
2. Check syntax with `task docs:serve`
3. If 2+ diagrams fail to render, reduce complexity or split into separate diagrams
4. Update explanatory text if behavior changes

### PlantUML Syntax Reference

**Common diagram types:**
- `@startuml` — Start diagram block
- `skinparam linetype ortho` — Orthogonal layout
- `package "name" {}` — Logical grouping
- `component [name]` — Component notation
- `state "name" {}` — State machine
- `actor User` — External actor
- Connectivity: `-->`, `<--`, `<-->` with labels

**Rendering:**
- Server-rendered via PlantUML.com web service
- SVG output for web/print
- Light and dark theme variants generated

### Common Issues

**Issue: Diagram fails to render (Status 400)**
- **Cause:** PlantUML server rejects syntax or diagram too complex
- **Solution:** 
  - Validate syntax locally with PlantUML CLI
  - Break into multiple simpler diagrams
  - Reduce nesting depth or component count
  - Check for invalid characters in labels

**Issue: MkDocs can't find diagram file**
- **Cause:** File path incorrect in `--8<-- "path"` directive
- **Solution:** Use relative path from `docs/` directory

**Issue: Diagram not in navigation**
- **Cause:** File exists but not referenced in `mkdocs.yml`
- **Solution:** Add entry to `nav` section in mkdocs.yml

---

## Recommended Reading Order

**For new team members:**
1. System Context (understand scope and actors)
2. Layered Architecture (understand structure)
3. Build Pipeline (understand workflow)
4. Sequence Diagram (understand interactions)

**For contributors:**
1. Architecture (overall design)
2. Domain Classes (implementation detail)
3. Plugin Architecture (extension points)
4. Data Flow (dependency tracking)

**For DevOps/Infrastructure:**
1. Deployment Infrastructure (platform support)
2. System Context (external dependencies)
3. Build Pipeline (build process)

**For plugin developers:**
1. Plugin Architecture (hooks and lifecycle)
2. Domain Classes (available services)
3. Sequence Diagram (invocation flow)

---

## Technical Details

### PlantUML Configuration

All diagrams use:
- **Version:** PlantUML 3.x (rendered server-side via plantuml.com)
- **Theme:** `!theme plain` for light backgrounds
- **Rendering:** Automatic SVG generation with light/dark variants
- **Update Rate:** On save, documentation rebuild triggers re-rendering

### Performance Notes

- Diagram rendering ~2-3 seconds per complete diagram during builds
- Parallel rendering of multiple diagrams
- Server caching reduces repeated render times
- Complex diagrams (>150 lines) may timeout and require simplification

### Cross-Reference

All diagrams are embedded in developer documentation:
- [Architecture](architecture.md) — Main architecture page
- [Code Patterns](code-patterns.md) — References domain classes diagram
- [Build System](build-system.md) — References pipeline and dataflow
- [Testing](testing.md) — References system context and plugin architecture
