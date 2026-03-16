# Default state returns the elmos helper. Look at the following output for the list of available commands and features.

```shell

 ███████╗██╗     ███╗   ███╗ ██████╗ ███████╗
 ██╔════╝██║     ████╗ ████║██╔═══██╗██╔════╝
 █████╗  ██║     ██╔████╔██║██║   ██║███████╗
 ██╔══╝  ██║     ██║╚██╔╝██║██║   ██║╚════██║
 ███████╗███████╗██║ ╚═╝ ██║╚██████╔╝███████║
 ╚══════╝╚══════╝╚═╝     ╚═╝ ╚═════╝ ╚══════╝

elmos

USAGE
  Embedded Linux SDK - Native kernel build tools

COMMANDS
  ─── Core ───
  doctor          Check environment health
  arch            Set or show target architecture
  version         Show version information
  tui             Launch interactive TUI
  ─── Build ───
  kernel          Kernel configuration commands
  module          Manage kernel modules
  app             Manage userspace apps
  rootfs          Manage root filesystem
  bootloader      Build and configure bootloader (U-Boot)
  patch           Apply patches to kernel source
  ─── Runtime ───
  qemu            Run kernel in emulator
  ─── Config ───
  bsp             Board support package management
  ─── Other ───
  workspace       Manage workspaces
  toolchain       Manage cross-compiler toolchain
  plugin          Manage plugins

FLAGS
  --help-all  Show all help
  --verbose  Enable verbose output
  --config  Config file path

Use "Embedded Linux SDK - Native kernel build tools [command] --help" for more information.

```

# After this `elmos` command, we should get a local .elmos/ folder and its subdirectories at $HOME path for any OSes, such as the following on MacOS:

```shell
/Users/trongphucnguyen/.elmos
├── sysroot
└── workspaces

3 directories, 0 files
```

# Issues:
- When user run this command for the first time, it should ask user to set up the above .elmos/ directory and its subdirectories, and automatically create them if user agrees. This is to ensure the correct setup of the environment for all OSes.
