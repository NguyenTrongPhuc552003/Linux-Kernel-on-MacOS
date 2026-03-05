# QEMU Integration

Run and debug kernels using ELMOS's QEMU integration.

---

## CLI Reference

```bash
elmos qemu [subcommand]

Subcommands:
  run         Run mode (boot kernel)
  debug       Debug mode (GDB attached)
  list        List available machines
```

---

## Run Modes

### Basic Run

```bash
elmos qemu run
```

Boots kernel with generated rootfs. Output:

```
→ Starting QEMU...
[    0.000000] Booting Linux on physical CPU 0x0000000000
[    0.000000] Linux version 6.18.0 ...
...
System ready.
#
```

### Debug Mode

```bash
elmos qemu debug
```

Launches QEMU with GDB stub enabled:

```bash
# In another terminal
gdb-multiarch vmlinux -ex "target remote :1234"
(gdb) break start_kernel
(gdb) continue
```

### With Graphical Display

```bash
elmos qemu run --graphical
```

Opens QEMU with GUI window (requires virtio-gpu kernel config).

---

## Machine Selection

### List Machines

```bash
elmos qemu list
```

Shows available QEMU machines for current architecture:

```
ℹ Available QEMU Machines for arm64:
  * virt - QEMU ARM Virtual Machine (default)
    raspi3b - Raspberry Pi 3B
    raspi4b - Raspberry Pi 4B
```

---

## RunOptions (Developer Reference)

```cpp
// src/domain/emulator/qemu.hpp
struct RunOptions {
    bool gdb = false;                        // Enable GDB stub
    bool graphic = false;                    // GUI display
    std::string initrd;                      // Custom initrd
    std::string append;                      // Kernel cmdline
    std::vector<std::string> extra_args;     // Extra QEMU args
};
```

---

## Architecture Defaults

| Arch  | QEMU Binary           | Default Machine    | Console   |
| ----- | --------------------- | ------------------ | --------- |
| arm64 | `qemu-system-aarch64` | `virt`             | `ttyAMA0` |
| arm   | `qemu-system-arm`     | `virt,highmem=off` | `ttyAMA0` |
| riscv | `qemu-system-riscv64` | `virt`             | `ttyS0`   |

---

## Networking

QEMU runs with user-mode networking:

- Guest can access internet
- Host accessible at `10.0.2.2`
- SSH forwarded: host `:2222` → guest `:22`

```bash
# From host
ssh -p 2222 root@localhost
```

---

## Troubleshooting

| Issue              | Solution                                   |
| ------------------ | ------------------------------------------ |
| "Kernel not found" | Run `elmos kernel build` first             |
| "No rootfs"        | Run `elmos rootfs build`                   |
| Boot hangs         | Check kernel config for `CONFIG_SERIAL_*`  |
| Invalid machine    | Run `elmos qemu list` to see valid options |
| GDB fails          | Install `gdb-multiarch`                    |