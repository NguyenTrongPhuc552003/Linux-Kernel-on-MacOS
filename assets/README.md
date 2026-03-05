# assets/ — Embedded Resources and Assets

This directory is reserved for static resources that will be embedded into the ELMOS binary at compile-time.

## Purpose

The `assets/` directory provides a centralized location for non-code files that should be distributed with the binary, including:

- **Templates** — Code generation templates (Makefile, main.c, etc.)
- **Schemas** — JSON schemas for configuration validation
- **Headers** — Standard library headers for embedded targets
- **Configuration files** — Toolchain configs, board definitions
- **Documentation** — Embedded help text, man pages
- **Scripts** — Shell scripts distributed with binary

## Current Structure

Currently empty. Ready for population with the following structure:

```
assets/
├── templates/              # Code generation templates
│   ├── app/               # Application templates
│   │   ├── main.c.tmpl
│   │   └── Makefile.tmpl
│   ├── kernel/            # Kernel build templates
│   │   └── Kconfig.tmpl
│   ├── module/            # Kernel module templates
│   │   ├── module.c.tmpl
│   │   └── Makefile.tmpl
│   └── rootfs/            # Rootfs templates
│       └── init.sh.tmpl
├── schemas/               # JSON schema validation
│   ├── machine.schema.json
│   ├── workspace.schema.json
│   └── plugins.schema.json
├── headers/               # Standard headers for targets
│   ├── asm/
│   ├── sys/
│   └── linux/
├── toolchains/            # Toolchain configurations
│   ├── arm-cortex_a15-linux-gnueabihf.config
│   ├── aarch64-unknown-linux-gnu.config
│   └── riscv64-unknown-linux-gnu.config
├── help/                  # Help text and documentation
│   ├── init.md            # Documentation for init command
│   ├── kernel.md          # Documentation for kernel command
│   └── qemu.md            # Documentation for qemu command
├── scripts/               # Embedded scripts
│   ├── init.sh            # Init script for root filesystem
│   ├── setup.sh           # Setup script
│   └── cleanup.sh         # Cleanup script
└── patches/               # Optional: kernel patches
    ├── v6.0/
    ├── v6.18/
    └── v6.19/
```

## Asset Embedding Strategy

### C++ Implementation Pattern

Assets will be embedded using the approach documented in [embedded-assets.md](../docs/developer/embedded-assets.md):

1. **At Build Time (CMake):**
   - `cmake/EmbedResources.cmake` generates C++ source
   - Files converted to `constexpr` byte arrays
   - Lookup function provides runtime access

2. **At Runtime (Code):**
   - Access via `embed::resources::get("path/to/file")`
   - Returns `std::string_view` (no copy overhead)
   - Zero-copy access to embedded data

3. **Code Generation:**
   - Files written to disk on demand
   - Proper permissions set (executable for .sh)
   - Temporary files cleaned up

### Example Usage

```cpp
// Write embedded template to disk
auto template_data = embed::resources::get("templates/app/main.c.tmpl");
std::ofstream out("main.c");
out << template_data;

// Validate config against schema
auto schema = embed::resources::get("schemas/machine.schema.json");
// ... validation logic
```

## CMakeLists.txt Configuration

When populating resources, update `src/CMakeLists.txt`:

```cmake
# Embed resources into binary
include(cmake/EmbedResources)

embed_resources(
    TARGET elmos_embedded_resources
    NAMESPACE embed::resources
    FILES
        templates/app/main.c.tmpl
        templates/module/module.c.tmpl
        schemas/machine.schema.json
        toolchains/arm-cortex_a15-linux-gnueabihf.config
        # ... more files
)

target_link_libraries(elmos_app PRIVATE elmos_embedded_resources)
```

## Guidelines for Adding Resources

### 1. Template Files (Jinja2 Format)

**Location:** `assets/templates/<category>/`

**Format:** Jinja2 template syntax for code generation

**Example:** `assets/templates/app/main.c.tmpl`
```c
#include <stdio.h>
#include <unistd.h>

// Generated from template for {{ app_name }}
int main(int argc, char** argv) {
    printf("Hello from {{ app_name }}\n");
    return 0;
}
```

**Usage in Code:**
```cpp
auto rendered = inja.render_template(
    embed::resources::get("templates/app/main.c.tmpl"),
    {{"app_name", "my_app"}}
);
```

### 2. Schema Files (JSON Schema)

**Location:** `assets/schemas/`

**Format:** Valid JSON Schema (draft 2020-12)

**Purpose:** Runtime validation of configuration

**Example:** Validate machine definition before loading

### 3. Configuration Files

**Location:** `assets/toolchains/`

**Format:** Kernel .config files or proprietary format

**Purpose:** Pre-configured settings for common targets

### 4. Shell Scripts

**Location:** `assets/scripts/`

**Format:** POSIX shell script (#!/bin/sh)

**Permissions:** Must be marked executable (chmod +x)

**Example:** Init script for rootfs startup

### 5. Help Text

**Location:** `assets/help/`

**Format:** Markdown (.md) or plain text (.txt)

**Purpose:** Embedded help for CLI commands

**Usage:**
```cpp
auto help_text = embed::resources::get("help/kernel.md");
printer.info("{}", help_text);
```

## Adding Files Step-by-Step

### 1. Create the resource file
```bash
# Example: Add template for kernel module
mkdir -p assets/templates/kernel
cat > assets/templates/kernel/Makefile.tmpl << 'EOF'
# Makefile for {{ module_name }}
obj-m := {{ module_name }}.o

all:
	make -C /lib/modules/$(shell uname -r)/build M=$(PWD) modules
EOF
```

### 2. Update CMakeLists.txt
Add to the `embed_resources()` call in `src/CMakeLists.txt`:
```cmake
templates/kernel/Makefile.tmpl
```

### 3. Rebuild with `task build`
```bash
cmake --build build --parallel
```

### 4. Access in code
```cpp
auto makefile = embed::resources::get("templates/kernel/Makefile.tmpl");
```

### 5. Commit changes
```bash
git add assets/templates/kernel/Makefile.tmpl
git add src/CMakeLists.txt  # if CMakeLists.txt changed
git commit -m "Add kernel module Makefile template"
```

## Size Considerations

**Binary Size Impact:**
- Each resource adds raw file size to binary (compressed)
- Typical overhead: +20-30% for well-designed resources
- Benefit: Single standalone executable, no runtime file I/O

**Optimization Tips:**
- Keep templates minimal (comments removed)
- Use schema references instead of duplicating definitions
- Remove unused/deprecated resources promptly
- Consider gzip compression for large files

## Validation and Testing

### Schema Validation Tests

```cpp
TEST_CASE("Schema validation - machine definition", "[resources]") {
    auto schema = embed::resources::get("schemas/machine.schema.json");
    auto validator = json::schema::from_string(schema);
    
    json::object test_machine = {
        {"name", "test_board"},
        {"architecture", "aarch64"}
    };
    
    REQUIRE(validator.validate(test_machine));
}
```

### Template Rendering Tests

```cpp
TEST_CASE("Template rendering - app main.c", "[resources]") {
    auto template_text = embed::resources::get("templates/app/main.c.tmpl");
    auto rendered = inja.render_template(template_text, 
        {{"app_name", "hello"}});
    
    REQUIRE_THAT(rendered, ContainsSubstring("hello"));
}
```

## Documentation

When adding new resource types:

1. **Update this README** with description and usage
2. **Add examples** in code comments
3. **Create tests** for validation and rendering
4. **Document parameters** in template headers

## Future Enhancements

Planned additions to resources:

- [ ] Board-specific device tree overlays
- [ ] Pre-compiled binutil utilities (minimal versions)
- [ ] License texts and attributions
- [ ] Architecture-specific optimization guides
- [ ] Example project templates
- [ ] Troubleshooting guide (embedded)

## References

- [Embedded Assets Design Document](../docs/developer/embedded-assets.md)
- [CMake EmbedResources Module](../../cmake/EmbedResources.cmake)
- [Jinja2 Template Documentation](https://jinja.palletsprojects.com/)
- [JSON Schema Specification](https://json-schema.org/specification.html)

---

**Status:** Ready for population  
**Last Updated:** 2024-03-05  
**Maintainer:** ELMOS Build System Team  
