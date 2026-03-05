# Modules and Apps

Develop kernel modules and userspace applications with ELMOS.

## Kernel Modules

### Create Module

```bash
./build/bin/elmos module create <name>
```

Generates template in `examples/modules/<name>/` with `Makefile` and `<name>.c`.

### Build Module

```bash
./build/bin/elmos module build <name>
```

Or build all modules:

```bash
./build/bin/elmos module build
```

Outputs `.ko` file.

### Example Template

```c
#include <linux/module.h>
#include <linux/kernel.h>

static int __init hello_init(void) {
    pr_info("Hello, ELMOS!\n");
    return 0;
}

static void __exit hello_exit(void) {
    pr_info("Goodbye, ELMOS!\n");
}

module_init(hello_init);
module_exit(hello_exit);
MODULE_LICENSE("GPL");
```

### Load in QEMU

After booting kernel:

```bash
insmod /path/to/module.ko
dmesg | tail
```

## Userspace Apps

### Create App

```bash
./build/bin/elmos app create <name>
```

Generates template in `examples/apps/<name>/` with `Makefile` and `<name>.c`.

### Build App

```bash
./build/bin/elmos app build <name>
```

Or build all apps:

```bash
./build/bin/elmos app build
```

Outputs cross-compiled executable.

### Example Template

```c
#include <stdio.h>

int main() {
    printf("Hello from ELMOS app!\n");
    return 0;
}
```

### Run in QEMU

Boot with QEMU and execute in the guest:

```bash
./<name>
```

## Cross-Compilation

- Auto-detects toolchain based on `./build/bin/elmos arch`
- Sets `CROSS_COMPILE` and `PATH` automatically

## Templates

Customize templates in `assets/templates/`.

## Examples

See `examples/` for sample modules (e.g., `char-test`, `hello-world`) and apps.