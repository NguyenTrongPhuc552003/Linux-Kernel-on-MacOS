# ELMOS Diagrams — Architecture Documentation

This directory contains comprehensive PlantUML diagrams documenting ELMOS architecture at multiple abstraction levels. These diagrams are automatically rendered by MkDocs during documentation builds.

## Diagram Overview

### 1. **architecture.puml** — Layered Architecture
**Purpose:** Complete system architecture showing all layers and components

**Abstraction Level:** High-level component and interface view

**Audience:** Architects, senior engineers, system designers

**Key Concepts Shown:**
- **UI Layer:** CLI, TUI, Printers
- **Application Layer:** Command registry, Error handling
- **Configuration Layer:** Loaders, Machine definitions, Workspaces
- **Context Layer:** Build state management
- **Domain Layer:** Business logic (5 subdomains)
  - Builders (Kernel, Module, App)
  - Orchestration (Pipeline, Fingerprinting)
  - BSP (Registry, Firmware)
  - System (Rootfs, Patcher, Doctor, QEMU)
  - Toolchain Management
  - Plugin Subsystem
- **Infrastructure Layer:** Execution, Filesystem, Platform, Package Management
- **Cross-Cutting Concerns:** Results, Logging, Versioning

**Relationships:**
- Shows unidirectional dependencies (top → bottom)
- Dotted lines for cross-cutting concerns
- Pointer injection pattern throughout

**Use This For:**
- Onboarding new team members
- High-level architecture reviews
- Understanding communication between layers
- Identifying coupling points

---

### 2. **system-context.puml** — System Context Diagram
**Purpose:** ELMOS as a black box within the larger ecosystem

**Abstraction Level:** System boundary view (outside-in perspective)

**Audience:** Product owners, stakeholders, integration teams

**Key Elements:**
- **Actors:** Developers, CI/CD pipelines, package managers
- **External Systems:** GCC toolchain, Kernel repo, BSP registry, Package repos
- **Artifacts:** zImage, Kernel modules, Rootfs, Boot image, QEMU disk
- **Deployment:** Embedded boards, QEMU emulator, Docker containers

**Message Flows:**
- Developer commands → ELMOS
- ELMOS queries → External dependencies
- ELMOS produces → Output artifacts
- Artifacts deploy → Target systems

**Use This For:**
- Presenting ELMOS role in larger systems
- Identifying external dependencies
- Planning integrations
- Understanding artifact flow

---

### 3. **dataflow.puml** — Data Flow Diagram
**Purpose:** How data transforms through build pipeline

**Abstraction Level:** Process-centric view with data stores

**Audience:** Build infrastructure engineers, DevOps, automation specialists

**Key Data Stores:**
- Workspace Config (YAML)
- Machine Definitions
- Kernel Sources
- Toolchain Cache
- Build Cache (fingerprints)
- Build Artifacts
- Final Images

**Processing Stages:**
1. Config Loading
2. Context Building
3. Fingerprinting
4. DAG Planning
5. Parallel Builds (Kernel, Modules, Rootfs)
6. Integration
7. Assembly
8. QEMU Testing

**Data Flows:**
- Input → Processing → Intermediate outputs → Final outputs
- Caching enables incremental builds
- Fingerprints validate cache correctness

**Use This For:**
- Understanding build process flow
- Cache mechanism design
- Optimization opportunities
- Debug process bottlenecks

---

### 4. **plugin-architecture.puml** — Plugin Subsystem
**Purpose:** Extensibility mechanism and hook-based architecture

**Abstraction Level:** Subsystem design pattern view

**Audience:** Plugin developers, extension architects

**Key Components:**
- **Plugin Interface:** Abstract contract all plugins implement
- **Lifecycle Management:** Loader, Registry, Factory map, Config resolver
- **Hook System:** Executor, Priority scheduler, Error recovery
- **Integration Points:** 13 hook events at critical build phases
- **Builtin Plugins:** 5 example implementations

**Hook Events:**
- Pre/Post Kernel Build
- Pre/Post Bootloader
- Config Loaded
- Pre/Post Rootfs
- Pre/Post Image Assembly
- Pre/Post QEMU Boot
- Build Error
- Cleanup

**Execution Model:**
- Priority-based scheduling (0-10 range)
- Higher priority runs first
- Non-fatal errors don't stop build
- Plugin state managed separately

**Use This For:**
- Implementing custom plugins
- Understanding plugin lifecycle
- Hook event timing and ordering
- Plugin interaction patterns

---

### 5. **deployment-infrastructure.puml** — Infrastructure & Deployment
**Purpose:** How ELMOS runs across platforms and deploys artifacts

**Abstraction Level:** Infrastructure and operations view

**Audience:** DevOps engineers, systems administrators, platform teams

**Deployment Environments:**
- **macOS Host:** Intel/Apple Silicon, Homebrew resolving, QEMU.app
- **Linux Host:** x86_64/ARM64, Package managers, KVM/QEMU
- **Windows Host:** WSL2, MSYS2, QEMU/hyperV

**Toolchain Infrastructure:**
- GCC Toolchain Cache (ARM, ARM64, RISC-V)
- Kernel sources with patch sets
- BSP Registry (definitions, firmware blobs, device trees)

**Build Execution:**
- Parallel stages: Kernel, Module, Rootfs builds
- Serial stages: Image assembly, QEMU testing

**Deployment Targets:**
- Embedded boards (ARM, ARM64, RISC-V)
- QEMU emulation with virtual network
- Docker containerization
- Kubernetes orchestration

**Caching Strategy:**
- Fingerprints validate cache hits
- Incremental builds skip unchanged work

**Use This For:**
- Setting up build infrastructure
- Understanding cross-platform support
- Planning deployment strategies
- Infrastructure cost optimization

---

### 6. **build-pipeline.puml** — Build Pipeline State Machine
**Purpose:** Complete build workflow with state transitions

**Abstraction Level:** Execution flow and state management

**Audience:** Build engineers, developers, CI/CD specialists

**Pipeline Phases:**

1. **Initialization:** Validation → Config load → Context setup → Cache check
2. **DAG Planning:** Dependency analysis → Task planning → Parallelization strategy
3. **Parallel Build:** 
   - Kernel build (patch → configure → compile → headers)
   - Module build (collect → configure → build)
   - Rootfs creation (bootstrap → packages → modules → init scripts)
4. **Assembly:** BSP integration → Firmware → Device trees → Final image
5. **Validation:** Image verification → QEMU test → Archiving

**State Transitions:**
- Success path: Init → DAG → Build → Assembly → Test → Success
- Error path: Any phase → Error recovery → Failed
- Optimization: Valid cache → Assembly (skip builds)

**Caching Optimization:**
- If cache valid: Skip builds entirely
- If cache invalid: Execute full pipeline

**Error Recovery:**
- Rollback partial artifacts
- Execute plugin cleanup hooks
- Return descriptive error

**Cancellation:**
- Stop token can interrupt any phase
- Graceful cleanup on cancellation

**Use This For:**
- Understanding full build lifecycle
- Identifying optimization points
- Planning incremental build strategies
- Understanding error handling flow

---

### 7. **domain-classes.puml** — Domain Class Model
**Purpose:** Core domain classes and their relationships

**Abstraction Level:** Class and interface design view

**Audience:** Developers, architects, code reviewers

**Core Packages:**
- **Config:** Architecture, Machine, Configuration
- **Context:** Build context (immutable)
- **Domain/Builder:** KernelBuilder, ModuleBuilder, AppBuilder
- **Domain/Orchestrator:** Pipeline, Fingerprinter
- **Domain/System:** RootfsBuilder, Patcher, QEMUEmulator
- **Domain/Toolchain:** ToolchainManager
- **Domain/Plugin:** Plugin interface, PluginLoader
- **Infrastructure:** Executor, FileSystem, Platform

**Design Patterns:**
- **Dependency Injection:** Pointer-based constructor injection
- **Strategy Pattern:** Executor, FileSystem, Platform interfaces
- **Factory Pattern:** Plugin factory map
- **Template Method:** Builder methods with private implementation

**Key Relationships:**
- Configuration owns Machine and Architecture
- Context references Architecture and Machine
- Builders inject Context, Executor, FileSystem, Toolchain
- Pipeline uses Fingerprinter and Executor
- Plugin interface enables extensibility

**Immutability:**
- Context is immutable after construction
- Passed as const pointer to components
- Ensures thread safety

**Use This For:**
- Understanding class responsibilities
- Adding new domain classes
- Refactoring and redesign
- Understanding design patterns used

---

### 8. **sequence-build.puml** — Build Execution Sequence
**Purpose:** Detailed interaction sequence during actual build

**Abstraction Level:** Message sequence / interaction diagram

**Audience:** Developers, QA, debugging teams

**Participants:**
- Developer (initiates command)
- CLI Handler (routes to app)
- App Container (orchestrates)
- ConfigLoader (parses YAML)
- ContextBuilder (builds context)
- Pipeline (orchestrates build)
- KernelBuilder, RootfsBuilder (build components)
- Fingerprinter (handles caching)
- Executor, FileSystem (execute operations)

**Sequence Phases:**

1. **Initialization:**
   - User runs `elmos kernel build`
   - CLI creates app container
   - Config loads from YAML
   - Context built

2. **Cache Check:**
   - Fingerprinter validates cache
   - If valid: skip builds
   - If invalid: proceed to build

3. **Parallel Execution:**
   - Pipeline creates DAG
   - Kernel and Rootfs build in parallel
   - Each has multiple steps

4. **Artifact Management:**
   - Fingerprints computed
   - Results stored
   - Cache updated

5. **Completion:**
   - Summary printed
   - Artifacts ready

**Timing Notes:**
- Kernel compilation: 30-60 seconds
- Full build: 2-5 minutes (first time)
- Incremental build: <10 seconds (with cache)

**Use This For:**
- Understanding build execution flow
- Debugging build issues
- Performance optimization
- Tracing user workflow

---

## Diagram Relationships

```
system-context.puml
  ↓ (details internals)
architecture.puml
  ├→ domain-classes.puml (class design)
  ├→ plugin-architecture.puml (extension mechanism)
  ├→ dataflow.puml (data transformation)
  ├→ build-pipeline.puml (execution flow)
  ├→ sequence-build.puml (interaction detail)
  └→ deployment-infrastructure.puml (operations)
```

**Recommended Reading Order:**

1. **System Context** — Understand ELMOS in ecosystem
2. **Layered Architecture** — See all components
3. **Build Pipeline** — Understand execution flow
4. **Dataflow** — Trace data transformations
5. **Domain Classes** — Deep dive into code structure
6. **Sequence Build** — See detailed interactions
7. **Plugin Architecture** — Understand extensibility
8. **Deployment** — Infrastructure considerations

## Diagram Maintenance

### When to Update Diagrams

- New domain component added
- Layer/interface changes
- Build pipeline modifications
- Plugin hook additions
- Major refactoring

### How to Modify Diagrams

1. **Identify which diagram(s)** to update
2. **Edit .puml file** in this directory
3. **Test rendering:** Run `task docs:serve` locally
4. **Verify accuracy** in browser at http://127.0.0.1:8000/developer/architecture/
5. **Commit changes** with diagram updates

### PlantUML Syntax

Key syntax elements used:

```puml
' Components
component [Name] as ID
interface "<<interface>> Name"
class ClassName { }

' Relationships
ComponentA --> ComponentB : label
ComponentA -.-> ComponentB : dotted (cross-cutting)
ComponentA <|-- ComponentB : inheritance
ComponentA *-- ComponentB : composition
ComponentA "0..*" -- "1" ComponentB : multiplicity

' Grouping
package "Name" { ... }
cloud "Name" { ... }
node "Name" { ... }

' Styling
note right of Component
  Explanatory text
end note
```

### Rendering

Diagrams are automatically rendered by:
- **MkDocs Material Plugin:** `mkdocs_puml`
- **Build Command:** `task docs:serve`
- **Output:** Embedded SVGs in HTML documentation

Cached diagrams in `~/.cache/mkdocs_puml/` for fast rebuilds.

## Diagram Validation Checklist

Before committing diagram changes:

- [ ] All components correctly labeled
- [ ] Relationships show correct direction
- [ ] No circular dependencies (layered architecture)
- [ ] Consistent notation throughout
- [ ] Notes explain design decisions
- [ ] Diagram fits on typical screen
- [ ] Renders without PlantUML errors
- [ ] SVG output is legible
- [ ] Consistent with architecture in code

## Tips for Using Diagrams

### For Code Reviews
- Reference appropriate diagram level
- Ensure code matches architecture
- Check for layer violations

### For Debugging
- Trace the sequence diagram
- Verify dataflow assumptions
- Check if error is in expected component

### For Onboarding
- Start with system context
- Move to architecture layers
- Deep dive with domain classes

### For Design Discussions
- Use architecture diagram
- Consider decomposition opportunities
- Identify cross-cutting concerns

## Future Diagram Expansions

Potential additions:

1. **Error Handling Flow** — How errors propagate through layers
2. **Memory/Resource Management** — Object lifetimes, ownership
3. **Concurrency Model** — Stop tokens, thread safety
4. **Configuration Validation** — Rules and constraints
5. **Caching Strategy Detail** — Fingerprint computation algorithm
6. **CLI Command Routing** — How commands map to handlers
7. **Performance Bottlenecks** — Known slow paths
8. **Testing Strategy** — Test pyramid, mocking approach

---

**Last Updated:** 2024-03-05  
**Maintainer:** ELMOS Architecture Team  
**License:** Same as ELMOS project (see LICENSE file)
