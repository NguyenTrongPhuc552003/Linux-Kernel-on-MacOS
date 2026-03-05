# Embedded Assets

ELMOS embeds templates and headers at compile time using CMake's resource embedding.

---

## Directory Structure

```
assets/
├── embed.go              # Legacy Go wrapper (unused in C++ build)
├── libraries/            # Compatibility headers
│   ├── elf.h             # ELF definitions
│   ├── byteswap.h        # Byte swapping macros
│   ├── endian.h          # Endianness macros
│   └── asm/
│       ├── bitsperlong.h
│       ├── int-ll64.h
│       ├── posix_types.h
│       └── types.h
├── schemas/              # JSON schemas for validation
│   ├── machine.schema.json
│   ├── plugins.schema.json
│   └── workspace.schema.json
├── templates/
│   ├── app/              # Userspace app templates
│   │   ├── main.c.tmpl
│   │   └── Makefile.tmpl
│   ├── configs/          # Configuration templates
│   │   └── elmos.yaml.tmpl
│   ├── init/             # Guest init scripts
│   │   ├── init.sh.tmpl
│   │   └── guesync.sh.tmpl
│   └── module/           # Kernel module templates
│       ├── module.c.tmpl
│       └── Makefile.tmpl
└── toolchains/
    └── configs/          # Crosstool-ng configurations
        ├── aarch64-unknown-linux-gnu.config
        ├── arm-cortex_a15-linux-gnueabihf.config
        └── riscv64-unknown-linux-gnu.config
```

---

## CMake Resource Embedding

Templates are embedded at compile time via `cmake/EmbedResources.cmake`. This replaces Go's `//go:embed` directive.

```cmake
# cmake/EmbedResources.cmake
elmos_embed_resources(target
    DIRECTORY assets/templates
    NAMESPACE elmos::assets
)
```

---

## Template Engine

Templates use [inja](https://github.com/pantor/inja) (v3.4.0) syntax, fetched via FetchContent:

```
{{ name }}           → Variable substitution
{% if condition %}   → Conditional
{% for item in list %} → Loop
```

---

## Template Variables

### Module Template

```c
// templates/module/module.c.tmpl
#include <linux/module.h>
#include <linux/kernel.h>

MODULE_LICENSE("GPL");
MODULE_AUTHOR("{{ author }}");
MODULE_DESCRIPTION("{{ description }}");

static int __init {{ name }}_init(void) {
    pr_info("{{ name }}: loaded\n");
    return 0;
}
module_init({{ name }}_init);
```

### App Template

```c
// templates/app/main.c.tmpl
#include <stdio.h>

int main(void) {
    printf("Hello from {{ name }}!\n");
    return 0;
}
```

---

## Compatibility Headers

The `assets/libraries/` directory contains ELF and architecture headers used for kernel module building on platforms that lack them natively.

| Header              | Purpose                |
| ------------------- | ---------------------- |
| `elf.h`             | ELF format definitions |
| `byteswap.h`        | Byte swapping macros   |
| `endian.h`          | Endianness detection   |
| `asm/types.h`       | Linux type definitions |
| `asm/bitsperlong.h` | Architecture bit width |

---

## JSON Schemas

Validation schemas in `assets/schemas/` define the structure of:
- `machine.schema.json` — Machine definition files
- `plugins.schema.json` — Plugin configuration
- `workspace.schema.json` — Workspace YAML

---

## Adding New Templates

1. Create template in `assets/templates/<category>/`
2. Use inja syntax (`{{ variable }}`) for substitution
3. Rebuild: `cmake --build build --parallel`