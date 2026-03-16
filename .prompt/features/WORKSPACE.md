# Workspace feature:

```shell
Manage workspaces

USAGE
  workspace

COMMANDS
  list            List all available workspaces
  init            Initialize workspace(s) — create disk image and mount
  clean           Unmount and remove workspace image(s)
  exit            Unmount workspace(s) — keeps image files
  show            Show workspace details

FLAGS
  --help-all  Show all help
  workspace  Workspace to activate

Use "workspace [command] --help" for more information.

```

# Issues:
- `init` command should be added one more flag `-p|--pick` to allow user to pick an existing workspace image file instead of creating a new one. This is useful for users who want to reuse their existing workspaces or share them across different machines.
- `show` command should be displayed more friendly and human-readable output, such as showing the workspace name, status (mounted/unmounted), size, and mount point (if mounted). This will help users quickly understand the state of their workspaces without needing to parse raw data. And a big issue for this command is that when running on OrbStack, it creates/loads a new workspace from MacOS platform instead of the current OS. This may be happen also on the pure Linux host with WSL2, so it should be fixed by detecting the current platform and loading the correct workspace image accordingly.
- After running the above `init` command (create a new one), we should get a new subfolder at $HOME/.elmos/workspaces/<workspace_name>/ with the workspace image file and mount point (if mounted). This will help users easily find and manage their workspace files. Note: u should analyze and think carefully about the workspace directory structure and file naming convention to ensure it is intuitive and scalable for multiple workspaces. In addition, which one should be generated at this $HOME/.elmos/workspaces/<workspace_name>/, and which one should be generated at /Volumes/<workspace_name>/ (for macOS) or /mnt/<workspace_name>/ (for Linux)? This is a critical design decision that will affect the user experience and system performance, so it should be carefully considered and tested.


# WRONG OUTPUT:

```shell
❯ elmos workspace init hello
ℹ Creating 'hello' (40G)...
created: /Users/trongphucnguyen/.elmos/hello.sparseimage
ℹ Mounting 'hello'...
✓ Workspace 'hello' ready at /Volumes/hello
ℹ   Extracted 3 files
ℹ Active workspace: hello
❯ tree ~/.elmos
/Users/trongphucnguyen/.elmos
├── active
├── hello.sparseimage # Wrong path, this should be at ~/.elmos/workspaces/hello/hello.sparseimage
├── sysroot
│   ├── byteswap.h
│   ├── elf.h
│   └── endian.h
└── workspaces
    └── hello
        └── config.yaml

4 directories, 6 files

╭─ /tmp ·······························································································
╰─❯ 
```