## 4. Toolchain Installation -> NOTE!!! Build errors ignored (PRIORITY 3 - LOWEST)
```shell
# 4th issue: Build toolchain
→ Building toolchain 'riscv64-unknown-linux-gnu' for arch 'riscv'...
[INFO ]  Performing some trivial sanity checks
[INFO ]  Build started 20260308.231914
[INFO ]  Building environment variables
[EXTRA]  Preparing working directories
[EXTRA]  Installing user-supplied crosstool-NG configuration
[EXTRA]  =================================================================
[EXTRA]  Dumping internal crosstool-NG configuration
[EXTRA]    Building a toolchain for:
[EXTRA]      build  = aarch64-apple-darwin25.2.0
[EXTRA]      host   = aarch64-apple-darwin25.2.0
[EXTRA]      target = riscv64-unknown-linux-gnu
[EXTRA]  Dumping internal crosstool-NG configuration: done in 0.04s (at 00:01)
[INFO ]  =================================================================
[INFO ]  Retrieving needed toolchain components' tarballs
[INFO ]  Retrieving needed toolchain components' tarballs: done in 0.48s (at 00:02)
[INFO ]  =================================================================
[INFO ]  Extracting and patching toolchain components
[EXTRA]    Extracting linux-6.16
[EXTRA]    Patching linux-6.16
[EXTRA]    Extracting zlib-1.3.1
[EXTRA]    Patching zlib-1.3.1
[EXTRA]    Extracting zstd-1.5.7
[EXTRA]    Patching zstd-1.5.7
[EXTRA]    Extracting gmp-6.3.0
[EXTRA]    Patching gmp-6.3.0
[EXTRA]    Extracting mpfr-4.2.2
[EXTRA]    Patching mpfr-4.2.2
[EXTRA]    Extracting isl-0.27
[EXTRA]    Patching isl-0.27
[EXTRA]    Extracting mpc-1.3.1
[EXTRA]    Patching mpc-1.3.1
[EXTRA]    Extracting expat-2.7.1
[EXTRA]    Patching expat-2.7.1
[EXTRA]    Extracting ncurses-6.5
[EXTRA]    Patching ncurses-6.5
[EXTRA]    Extracting libiconv-1.18
[EXTRA]    Patching libiconv-1.18
[EXTRA]    Extracting gettext-0.26
[EXTRA]    Patching gettext-0.26
[EXTRA]    Extracting binutils-2.45
[EXTRA]    Patching binutils-2.45
[EXTRA]    Extracting gcc-15.2.0
[EXTRA]    Patching gcc-15.2.0
[EXTRA]    Extracting glibc-2.42
[EXTRA]    Patching glibc-2.42
[EXTRA]    Extracting gdb-16.3
[EXTRA]    Patching gdb-16.3
[INFO ]  Extracting and patching toolchain components: done in 423.65s (at 07:05)
[INFO ]  =================================================================
[INFO ]  Installing ncurses for build
[EXTRA]    Configuring ncurses
[EXTRA]    Building ncurses
[EXTRA]    Installing ncurses
[INFO ]  Installing ncurses for build: done in 33.35s (at 07:39)
[INFO ]  =================================================================
[INFO ]  Installing zlib for host
[EXTRA]    Configuring zlib
[EXTRA]    Building zlib
[EXTRA]    Installing zlib
[INFO ]  Installing zlib for host: done in 1.95s (at 07:41)
[INFO ]  =================================================================
[INFO ]  Installing zstd for host
[EXTRA]    Building zstd
[EXTRA]    Installing zstd
[INFO ]  Installing zstd for host: done in 2.17s (at 07:43)
[INFO ]  =================================================================
[INFO ]  Installing GMP for host
[EXTRA]    Configuring GMP
[EXTRA]    Building GMP
[EXTRA]    Installing GMP
[INFO ]  Installing GMP for host: done in 43.07s (at 08:26)
[INFO ]  =================================================================
[INFO ]  Installing MPFR for host
[EXTRA]    Configuring MPFR
[EXTRA]    Building MPFR
[EXTRA]    Installing MPFR
[INFO ]  Installing MPFR for host: done in 26.33s (at 08:52)
[INFO ]  =================================================================
[INFO ]  Installing ISL for host
[EXTRA]    Configuring ISL
[EXTRA]    Building ISL
[EXTRA]    Installing ISL
[INFO ]  Installing ISL for host: done in 13.38s (at 09:06)
[INFO ]  =================================================================
[INFO ]  Installing MPC for host
[EXTRA]    Configuring MPC
[EXTRA]    Building MPC
[EXTRA]    Installing MPC
[INFO ]  Installing MPC for host: done in 8.61s (at 09:14)
[INFO ]  =================================================================
[INFO ]  Installing expat for host
[EXTRA]    Configuring expat
[EXTRA]    Building expat
[EXTRA]    Installing expat
[INFO ]  Installing expat for host: done in 10.25s (at 09:25)
[INFO ]  =================================================================
[INFO ]  Installing ncurses for host
[EXTRA]    Configuring ncurses
[EXTRA]    Building ncurses
[EXTRA]    Installing ncurses
[INFO ]  Installing ncurses for host: done in 37.79s (at 10:02)
[INFO ]  =================================================================
[INFO ]  Installing libiconv for host
[EXTRA]    Configuring libiconv
[EXTRA]    Building libiconv
[EXTRA]    Installing libiconv
[INFO ]  Installing libiconv for host: done in 28.07s (at 10:31)
[INFO ]  =================================================================
[INFO ]  Installing gettext for host
[EXTRA]    Configuring gettext
[EXTRA]    Building gettext
[EXTRA]    Installing gettext
[INFO ]  Installing gettext for host: done in 450.59s (at 18:01)
[INFO ]  =================================================================
[INFO ]  Installing binutils for host
[EXTRA]    Configuring binutils
[EXTRA]    Building binutils
[EXTRA]    Installing binutils
[INFO ]  Installing binutils for host: done in 90.43s (at 19:32)
[INFO ]  =================================================================
[INFO ]  Installing kernel headers
[EXTRA]    Installing kernel headers
[INFO ]  Installing kernel headers: done in 22.52s (at 19:54)
[INFO ]  =================================================================
[INFO ]  Installing core C gcc compiler
[EXTRA]    Configuring core C gcc compiler
[EXTRA]    Building gcc
[EXTRA]    Installing gcc
[EXTRA]    Housekeeping for core gcc compiler
[EXTRA]       '' --> lib (gcc)   lib (os)
[INFO ]  Installing core C gcc compiler: done in 208.62s (at 23:23)
[INFO ]  =================================================================
[INFO ]  Installing C library
[INFO ]    =================================================================
[INFO ]    Building for multilib 1/1: ''
[EXTRA]      Configuring C library
[EXTRA]      Building C library
[EXTRA]      Installing C library
[INFO ]    Building for multilib 1/1: '': done in 148.14s (at 25:51)
[INFO ]  Installing C library: done in 148.23s (at 25:51)
[INFO ]  =================================================================
[INFO ]  Installing final gcc compiler
[EXTRA]    Configuring final gcc compiler
[EXTRA]    Building final gcc compiler
[ERROR]    clang++: error: unsupported option '-print-multi-os-directory'
[ERROR]    clang++: error: no input files
[EXTRA]    Installing final gcc compiler
[EXTRA]    Housekeeping for final gcc compiler
[EXTRA]       '' --> lib (gcc)   lib (os)
[INFO ]  Installing final gcc compiler: done in 271.45s (at 30:22)
[INFO ]  =================================================================
[INFO ]  Installing cross-gdb
[EXTRA]    Configuring cross gdb
[EXTRA]    Building cross gdb
[EXTRA]    Installing cross gdb
[EXTRA]    Installing '.gdbinit' template
[INFO ]  Installing cross-gdb: done in 154.53s (at 32:57)
[INFO ]  =================================================================
[INFO ]  Finalizing the toolchain's directory
[INFO ]    Stripping all toolchain executables
[EXTRA]    Installing the populate helper
[EXTRA]    Installing a cross-ldd helper
[EXTRA]    Creating toolchain aliases
[EXTRA]    Removing installed documentation
[EXTRA]    Collect license information from: /Volumes/elmos/toolchains/build/riscv64-unknown-linux-gnu/.
build/riscv64-unknown-linux-gnu/src                                                                     [EXTRA]    Put the license information to: /Volumes/elmos/toolchains/x-tools/riscv64-unknown-linux-gnu/s
hare/licenses                                                                                           [INFO ]  Finalizing the toolchain's directory: done in 4.45s (at 33:01)
[INFO ]  Build completed at 20260308.235215
[INFO ]  (elapsed: 33:01.47)
[INFO ]  Finishing installation (may take a few seconds)...
[33:01] / ✓ Toolchain 'riscv64-unknown-linux-gnu' built!
```

<!-- NEXT ISSUES -->

## 5. Build Kernel -> NOTE!!! Build errors (PRIORITY 1)
```shell
❯ ll include/sysroot/asm
total 0
lrwxr-xr-x@ 1 trongphucnguyen  staff    59B Mar 10 21:08 bitsperlong.h -> /Volumes/elmos/linux/include/uapi/asm-generic/bitsperlong.h
lrwxr-xr-x@ 1 trongphucnguyen  staff    56B Mar 10 21:08 int-ll64.h -> /Volumes/elmos/linux/include/uapi/asm-generic/int-ll64.h
lrwxr-xr-x@ 1 trongphucnguyen  staff    59B Mar 10 21:08 posix_types.h -> /Volumes/elmos/linux/include/uapi/asm-generic/posix_types.h
lrwxr-xr-x@ 1 trongphucnguyen  staff    53B Mar 10 21:08 types.h -> /Volumes/elmos/linux/include/uapi/asm-generic/types.h

╭─ ~/Documents/kernel-dev/linux on refactor !92 ?13 ···················································
╰─❯ tree include/sysroot
├── asm # Auto-creating these symbolic links after cloning the linux kernel to the platform-specific workspace on localhost, e.g. /Volumes/elmos/sysroot/ from this elmos repository's include/sysroot/asm folder
│   ├── bitsperlong.h -> /Volumes/elmos/linux/include/uapi/asm-generic/bitsperlong.h
│   ├── int-ll64.h -> /Volumes/elmos/linux/include/uapi/asm-generic/int-ll64.h
│   ├── posix_types.h -> /Volumes/elmos/linux/include/uapi/asm-generic/posix_types.h
│   └── types.h -> /Volumes/elmos/linux/include/uapi/asm-generic/types.h
├── byteswap.h # Cloning from this elmos repository's include/sysroot/byteswap.h
├── elf.h # Cloning from this: https://elixir.bootlin.com/glibc/glibc-2.43/source/elf/elf.h (latest version of elf.h in glibc or older version of elf.h in glibc should work, automatically download the latest version of elf.h in glibc if not exist. Upgrade or downgrade the version of elf.h in glibc if the latest version of elf.h in glibc doesn't work)
└── endian.h # Cloning from this elmos repository's include/sysroot/endian.h

2 directories, 7 files
```

```shell
# 5th issue: Build kernel
→ Building kernel for riscv...
make: Entering directory '/Volumes/elmos/linux'
  HOSTCC  scripts/sorttable
scripts/sorttable.c:27:10: fatal error: 'elf.h' file not found
   27 | #include <elf.h>
      |          ^~~~~~~
1 error generated.
make[2]: *** [scripts/Makefile.host:114: scripts/sorttable] Error 1
make[1]: *** [/Volumes/elmos/linux/Makefile:1265: scripts] Error 2
make[1]: *** Waiting for unfinished jobs....
make: *** [Makefile:248: __sub-make] Error 2
make: Leaving directory '/Volumes/elmos/linux'
✗ Build failed: make exited with code 2
```

<!-- NEXT ISSUES -->

## 6. Create RootFS -> NOTE!!! Build failed -> Solution: download the debootstrap repository (refer .gitmodules file to know the repository URL, not from this elmos repository) to localhost at the workspace's rootfs folder. Then, build and install debootstrap from source on localhost. Finally, run the `elmos rootfs build` command after cloning to create the root filesystem. In addition, u need to read this debootstrap repository's README and its all component to resolve this issue (PRIORITY 2)
```shell
# 6th issue: Create RootFS
ℹ Rootfs directory already exists at /Volumes/elmos/rootfs
→ Cloning debootstrap...
Cloning into '/Volumes/elmos/tools/debootstrap'...
remote: Enumerating objects: 70, done.
remote: Counting objects: 100% (70/70), done.
remote: Compressing objects: 100% (52/52), done.
remote: Total 70 (delta 19), reused 35 (delta 3), pack-reused 0 (from 0)
Receiving objects: 100% (70/70), 106.22 KiB | 65.00 KiB/s, done.
Resolving deltas: 100% (19/19), done.
→ Building debootstrap from source...
make: *** No rule to make target 'devices.tar.gz'.  Stop.
ℹ Skipped devices.tar.gz generation (non-fatal)
✓ Rootfs ready (debootstrap at /Volumes/elmos/tools/debootstrap/debootstrap)
```
