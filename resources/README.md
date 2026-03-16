# resources/ — Compile-Time Embedded Resources

Files in this directory are embedded into the `elmos` binary at build time via
`cmake/EmbedResources.cmake`. They are accessible at runtime through the
`elmos::resources::get(path)` API without any filesystem dependency.

## Structure

```
resources/
├── templates/                 # inja-syntax code generation templates
│   ├── app/                   # Userspace application scaffolding
│   │   ├── main.c.tmpl
│   │   └── Makefile.tmpl
│   ├── module/                # Kernel module scaffolding
│   │   ├── module.c.tmpl
│   │   └── Makefile.tmpl
│   ├── configs/               # Default workspace configuration
│   │   └── elmos.yaml.tmpl
│   └── init/                  # QEMU guest init scripts
│       ├── init.sh.tmpl
│       └── guesync.sh.tmpl
├── schemas/                   # JSON Schema validation files
│   ├── machine.schema.json
│   ├── workspace.schema.json
│   └── plugins.schema.json
└── toolchains/configs/        # crosstool-ng seed configurations
    ├── aarch64-unknown-linux-gnu.config
    ├── arm-cortex_a15-linux-gnueabihf.config
    └── riscv64-unknown-linux-gnu.config
```

## Template Syntax

Templates use [inja](https://github.com/pantor/inja) syntax (Jinja2-compatible):

```
{{ name }}          — variable substitution
{{ c_name }}        — C-safe identifier (hyphens replaced with underscores)
{{ description }}   — free-text description
{{ version }}       — elmos version string
```

## C++ API

```cpp
#include <embedded_resources.hpp>

// Look up a resource by path
auto tmpl = elmos::resources::get("templates/app/main.c.tmpl");
if (tmpl) {
    // *tmpl is a std::string_view of the file content
}

// List all embedded resources
auto paths = elmos::resources::list();
```

## Adding Resources

1. Place the file under `resources/` in the appropriate subdirectory
2. Rebuild — CMake automatically picks up new files via `file(GLOB_RECURSE ...)`
3. Access via `elmos::resources::get("relative/path/from/resources")`
