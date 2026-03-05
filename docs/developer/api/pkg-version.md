# Version API

Version information injected at compile time via CMake.

---

## Compile-Time Macros

Defined in `cmake/Version.cmake` and injected via `add_compile_definitions()`:

| Macro              | Example Value          | Description         |
| ------------------ | ---------------------- | ------------------- |
| `ELMOS_VERSION`    | `v3.2.0-6-gd781c77`    | Git describe tag    |
| `ELMOS_COMMIT`     | `d781c77`              | Short commit SHA    |
| `ELMOS_BUILD_DATE` | `2025-01-15T10:30:00Z` | ISO 8601 build time |

---

## Usage in Source Code

```cpp
// src/app/commands/version.cpp
auto register_version(App& app, CLI::App& cli) {
    auto* cmd = cli.add_subcommand("version", "Show version info");
    cmd->callback([&app] {
        app.printer().info("{} ({})", ELMOS_VERSION, ELMOS_COMMIT);
    });
}
```

---

## CMake Implementation

```cmake
# cmake/Version.cmake
execute_process(
    COMMAND git describe --tags --always --dirty
    OUTPUT_VARIABLE ELMOS_GIT_VERSION
    OUTPUT_STRIP_TRAILING_WHITESPACE
)
execute_process(
    COMMAND git rev-parse --short HEAD
    OUTPUT_VARIABLE ELMOS_GIT_COMMIT
    OUTPUT_STRIP_TRAILING_WHITESPACE
)
string(TIMESTAMP ELMOS_BUILD_DATE "%Y-%m-%dT%H:%M:%SZ" UTC)

add_compile_definitions(
    ELMOS_VERSION="${ELMOS_GIT_VERSION}"
    ELMOS_COMMIT="${ELMOS_GIT_COMMIT}"
    ELMOS_BUILD_DATE="${ELMOS_BUILD_DATE}"
)
```

---

## CLI Output

```bash
$ ./build/bin/elmos version
v3.2.0-6-gd781c77-dirty (d781c77)
```