# Toolchain feature:

```shell
Manage cross-compiler toolchain

USAGE
  toolchain

COMMANDS
  status          Show installed toolchains
  clone           Install crosstool-ng
  build           Build selected toolchain
  list            List available toolchain targets
  menuconfig      Interactive toolchain configuration
  show            Show current toolchain information
  clean           Clean toolchain build

FLAGS
  --help-all  Show all help
  target  Toolchain target to select

Use "toolchain [command] --help" for more information.
```

# Issues:
- `build` command got some problems when building toolchain although the final building process is successful, please check and resolve the issues to ensure a smooth user experience. Because of the time-consuming nature of building toolchain, u should find and resolve me this error to prevent users from facing unexpected interruptions during the build process. Please investigate the root cause of this error and implement a solution to ensure that the toolchain can be built successfully without any issues. I'll rebuild this later.
```shell
# Building toolchain output's error:
[INFO ]  =================================================================
[INFO ]  Installing final gcc compiler
[EXTRA]    Configuring final gcc compiler
[EXTRA]    Building final gcc compiler
[ERROR]    clang++: error: unsupported option '-print-multi-os-directory'
[ERROR]    clang++: error: no input files
```
