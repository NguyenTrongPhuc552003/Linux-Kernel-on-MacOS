# Kernel feature:

```shell
Kernel configuration commands

USAGE
  kernel

COMMANDS
  config          Configure the kernel
  clone           Clone the Linux kernel source
  build           Build the Linux kernel
  clean           Clean kernel build artifacts
  status          Show kernel source status
  pull            Update kernel source (pull latest)
  update          Update kernel source (alias for pull)
  switch          Switch branch/tag
  reset           Reset kernel source (reclone)
  show            Show kernel build result
  install         Install kernel artifacts

FLAGS
  --help-all  Show all help

Use "kernel [command] --help" for more information.
```

# Issues:
- `build` command has not still been contained the necessary flags to $HOME/.elmos/workspaces/<workspace_name>/sysroot/asm/ folder, this may be similiar to the global $HOME/.elmos/sysroot/ folder
