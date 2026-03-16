# Assuming that we're anywhere in any operating system with an installed elmos package. So, this is the common workflow for all users regardless of their operating system. The only difference is the workspace management and the location of the workspace image and mount point based on the operating system, but the commands are the same for all users.

## 1. Check required dependencies

```bash
# Checks for required tools (cmake, ninja, qemu, etc.) and reports any missing dependencies.
# For Windows users, this also checks for WSL installation and configuration.
elmos doctor
```

## 2. Workspace Management

```bash
# List all available workspaces
elmos workspace list

# So if no workspace exists, create a new workspace with the specified name (e.g., "my_workspace") and auto-pick it as the active workspace with first workspace as default. The workspace will be created in the default location based on the operating system:
# - MacOS: /Volumes/<workspace_name>/
# - Linux: /mnt/<workspace_name>/
# - Windows (WSL): /mnt/<workspace_name>/
elmos workspace init <workspace_name> [other_workspace]

# If the workspace already exists or the above command is run again, this command will simply confirm that it is available and ready to use without overwriting any existing data.
# Select active workspace (all subsequent commands target this workspace)
elmos workspace <workspace_name>

# Note: the workspace image is stored in the default location: $HOME/.elmos/<workspace_name>.<format> (e.g., my_workspace.sparseimage or my_workspace.img, etc. depending on the operating system). The workspace is mounted to the default mount point based on the operating system:
# - MacOS: /Volumes/<workspace_name>/
# - Linux: /mnt/<workspace_name>/
# - Windows (WSL): /mnt/<workspace_name>/

# Clean up the specified workspace by unmounting it and removing the workspace image file. This will permanently delete all data in the workspace, so use with caution.
elmos workspace clean <workspace_name>||all

# Show the detailed information of the current active workspace or other workspace if specified, including the workspace name, location of the workspace image file, mount point, and current status (mounted or unmounted). This will help you confirm that the workspace is properly set up and ready to use for your kernel development.
elmos workspace show [workspace_name] [other_workspace]

# Exit the current active, multiple, or all workspaces by unmounting them. This will keep the workspace image files but unmount them from the system, so you can pick them again later without losing any data.
elmos workspace exit [workspace_name] [other_workspace]||all
```

## 3. Set Architecture

```bash
# Set the target architecture for the kernel build and toolchain configuration. This will determine which kernel source branch to use and which toolchain target to build. The default architecture is arm64, but you can change it to arm or riscv based on your needs.
elmos arch list      # List available architectures
elmos arch <arch>    # Set architecture (arm64, arm, riscv)
elmos arch show      # Show current architecture to confirm
```

## 4. Toolchain Installation

```bash
# Install crosstool-ng toolchain builder (if not already installed)
elmos toolchain clone

# Check if toolchain repository is installed or not
# This will be integrated with "env" command in the future
elmos toolchain status

# List available toolchain targets based on the selected architecture
elmos toolchain list

# Pick (replace "select" command) a toolchain target (the default is based on the toolchain pre-configured by the target architecture) -> Optionally, you can specify a custom target if you want to build a different toolchain configuration. This will update the toolchain configuration file with the selected target and prepare it for building.
elmos toolchain <target> # Optional

# Run menuconfig to customize the toolchain configuration (optional, for advanced users)
elmos toolchain menuconfig

# Build the selected toolchain (this may take 30-60 minutes)
elmos toolchain build # Build based on the selected target architecture and default toolchain configuration, or the custom configuration if you have modified it using menuconfig.

# Show the picked (or not) toolchain's information
elmos toolchain show

# Clean toolchain build artifacts (optional)
elmos toolchain clean
```

## 5. Build Kernel

```bash
# Clone the Linux kernel source code (if not already cloned)
elmos kernel clone

# Check the current branch and version of the kernel source and the latest commit information
elmos kernel status

# Switch to a specific kernel version or branch (optional)
# Automatically detects the specified branch or version and checks it out in the kernel source directory
elmos kernel switch <branch_or_version>

# Update the specified kernel branch or version to the latest commit (optional)
elmos kernel update

# Configure the kernel using a default config for the selected architecture
# Default: defconfig - a standard config based on the selected architecture
elmos kernel config [config_type]

# Build the kernel with the default targets (Image, dtbs, module) or specify custom targets (e.g., Image, vmlinux, module, etc.). Default parallelism is based on the number of CPU cores, but you can specify a custom number of jobs for faster builds.
elmos kernel build [targets] [-j <num_jobs>]

# Show the result of the kernel build, including the location of the built kernel image, dtbs, and module based on the targets specified in the build command. This will also show any build errors or warnings if the build failed.
elmos kernel show

# Clean the kernel build artifacts (optional)
elmos kernel clean
```

## 6. Create RootFS

```bash
# Build a Debian-based root filesystem using debootstrap. This will create a minimal rootfs with essential packages and dependencies for running the kernel. The rootfs will be created in the workspace and can be customized with additional packages if needed.
# Clone the rootfs repository (if not already cloned) to /Volumes/<workspace_name>/rootfs/ (or /mnt/<workspace_name>/rootfs/ based on the operating system) and prepare for building the rootfs image.
elmos rootfs clone

# Build the rootfs using debootstrap based on the selected architecture (this may take 10-20 minutes)
elmos rootfs build

# Show the result of the rootfs build, including the location of the built rootfs image and any errors or warnings if the build failed. This is different from the "status" command which shows the current status of the rootfs build and whether it is ready to be used for running the kernel, while this "show" command shows the detailed information of the built rootfs image and any build errors or warnings.
elmos rootfs show

# Clean the rootfs build artifacts (optional)
elmos rootfs clean
```

## 7. Bootloader Configuration

```bash
# Clone the bootloader repository (if not already cloned) to /Volumes/<workspace_name>/bootloader/ (or /mnt/<workspace_name>/bootloader/ based on the operating system) and prepare for building the bootloader. This will be used for configuring and building the bootloader (e.g., U-Boot) for the selected architecture and kernel.
elmos bootloader clone

# Configure the bootloader (e.g., U-Boot) for the selected architecture and kernel. This will generate the necessary bootloader configuration files and scripts based on the kernel image and rootfs created in the previous steps. Default config is based on the selected target architecture, but you can specify a custom config type if needed.
elmos bootloader config [config_type]

# Build the bootloader (this may take 10-20 minutes)
elmos bootloader build

# Show the result of the bootloader build, including the location of the built bootloader image and any errors or warnings if the build failed.
elmos bootloader show

# Clean the bootloader build artifacts (optional)
elmos bootloader clean
```

## 8. Run in QEMU

```bash
# Show the current status of the QEMU components, including the kernel image, rootfs image, and bootloader image. This will help you confirm that all the necessary components are built and ready for running in QEMU before executing the run command.
elmos qemu show

# Check if any QEMU processes are running and show their status (optional)
elmos qemu status

# Clean up any existing QEMU processes and temporary files (optional)
elmos qemu clean <process_id>||all

# Run the built kernel in QEMU with the created rootfs and configured bootloader. This will start a QEMU virtual machine with the specified kernel, rootfs, and bootloader configuration. You should see the Linux boot process in the QEMU window. Login with "root" (no password) to access the shell.
# You can pass additional flag like -d|--debug to enable QEMU debugging output for troubleshooting. This command will automatically create a split terminal setup with the QEMU window and a separate terminal for debugging output if the debug flag is enabled using tmux.
# In addition, you can pass also graphical options (e.g., -g|--graphical) to enable or disable the QEMU graphical output based on your preference. By default, the graphical output is enabled, but you can disable it for faster performance or if you prefer to use a serial console for interaction if the built kernel and bootloader support it after configuring with kvm_guest.config or similar.
elmos qemu run [-d|--debug] [-g|--graphical]
```

# After these above steps, you should have a fully functional Linux kernel running in a QEMU virtual machine with a Debian-based root filesystem. You can now start exploring the kernel, testing changes, and developing new features based on this setup. For debugging and development, you can use the QEMU debug output and connect with GDB for more advanced debugging capabilities. Happy hacking! But, you want to extend or customize the workflow? Let's continue to the next section for more advanced usage and customization options.

## 9. Kernel Module Development (Optional)

```bash
# List available kernel module in the specified workspace (or all workspaces if "all" is specified). This will show the existing kernel module that you can use as references or starting points for your own module development. The module are located in the workspace under the "module" directory (e.g., /path/to/workspace/module/).
elmos module list

# Create a new kernel module scaffold with the specified name (e.g., "my_module") in the workspace. This will generate a basic kernel module template with the necessary files and structure for development. This step is optional because we already provide you 3 example module in the workspace (hello_world, kvm_guest, and custom) that you can use as references or starting points for your own module development. You can also create additional module based on these examples or from scratch using this command.
elmos module create <module_name>

# Open the kernel module source code in your preferred code editor for development. You can use the provided examples as references for how to structure your module and interact with the kernel APIs. The module source code will be located in the workspace under the "module" directory (e.g., /path/to/workspace/module/<module_name>/).
elmos module edit <module_name> # Ask user to use nano, vim, code, etc. based on their preference

# Build the kernel module using the same toolchain and kernel configuration as the main kernel build. This will compile the module source code and generate the corresponding .ko file that can be loaded into the kernel. The built module will be located in the workspace under the "build/module" directory (e.g., /path/to/workspace/build/module/<module_name>.ko). Default parallelism is based on the number of CPU cores, but you can specify a custom number of jobs for faster builds. This command will build all module in the workspace if no module name is specified, or build a specific module if the module name is provided.
elmos module build [module_name] [-j|--jobs <num_jobs>]

# Show the result of the kernel module build, including the location of the built .ko file and any errors or warnings if the build failed. This will also show the status of the module build and whether it is ready to be loaded into the kernel.
elmos module show [module_name]

# Mark the specified kernel module as ready for loading. This will perform any necessary post-build steps to prepare the module for loading into the kernel, such as signing the module if required by the kernel configuration. This command is optional because you can load the built module directly without marking it as ready, but marking it as ready can help ensure that the module is properly prepared and compatible with the running kernel.
# So, after booting QEMU, you will see the specified module is loaded and running at the startup based on the kernel configuration (e.g., kvm_guest.config will automatically load the kvm_guest module at startup). You can also load the module manually after booting into the kernel using the "insmod" command in the kernel shell if the module is not set to load at startup.
elmos module load [module_name]

# Clean the kernel module build artifacts (optional)
elmos module clean [module_name]||all
```

# 10. User Space Application Development (Optional)

```bash
# List available user space applications in the specified workspace (or all workspaces if "all" is specified). This will show the existing user space applications that you can use as references or starting points for your own application development. The applications are located in the workspace under the "apps" directory (e.g., /path/to/workspace/apps/).
elmos app list

# Create a new user space application scaffold with the specified name (e.g., "my_app") in the workspace. This will generate a basic user space application template with the necessary files and structure for development. This step is optional because we already provide you 1 example applications in the workspace (hello_world, kvm_guest, and custom) that you can use as references or starting points for your own application development. You can also create additional applications based on these examples or from scratch using this command.
elmos app create <app_name>

# Open the user space application source code in your preferred code editor for development. You can use the provided example as a reference for how to structure your application and interact with the kernel and system APIs. The application source code will be located in the workspace under the "apps" directory (e.g., /path/to/workspace/apps/<app_name>/).
elmos app edit <app_name> # Ask user to use nano, vim, code, etc. based on their preference

# Build the user space application using the same toolchain and rootfs configuration as the main kernel build. This will compile the application source code and generate the corresponding executable file that can be run in the kernel. The built application will be located in the workspace under the "build/apps" directory (e.g., /path/to/workspace/build/apps/<app_name>). Default parallelism is based on the number of CPU cores, but you can specify a custom number of jobs for faster builds. This command will build all applications in the workspace if no application name is specified, or build a specific application if the application name is provided.
elmos app build [app_name] [-j|--jobs <num_jobs>]

# Show the result of the user space application build, including the location of the built executable file and any errors or warnings if the build failed. This will also show the status of the application build and whether it is ready to be run in the kernel.
elmos app show [app_name]

# Load the built user space application into the rootfs. This will copy the built executable file into the rootfs image so that it can be accessed and run in the kernel. This command is optional because you can also manually copy the built application into the rootfs after building it, but using this command can help automate the process and ensure that the application is properly integrated into the rootfs.
# Copy the built application into /usr/local/bin/ in the rootfs so that it can be run from anywhere in the kernel shell. You can also specify a custom location in the rootfs if needed, but /usr/local/bin/ is a common location for user-installed applications. After booting QEMU, you will see the specified application is available in the rootfs and can be run from the kernel shell using its name (e.g., "my_app") if it is copied to /usr/local/bin/. You can also specify a custom location in the rootfs if needed, but make sure to update the command accordingly to reflect the correct path in the rootfs.
elmos app load [app_name]

# Clean the user space application build artifacts (optional)
elmos app clean [app_name]||all
```

# 11. BSP Development (Optional)

```bash
# List available BSPs in the specified workspace (or all workspaces if "all" is specified). This will show the existing BSPs that you can use as references or starting points for your own BSP development. The BSPs are located in the workspace under the "bsp" directory (e.g., /path/to/workspace/bsp/).
elmos bsp list

# Create a new BSP scaffold with the specified name (e.g., "my_bsp") in the workspace. This will generate a basic BSP template with the necessary files and structure for development. This step is optional because we already provide you 1 example BSP in the workspace (qemu_virt, etc.) that you can use as a reference or starting point for your own BSP development. You can also create additional BSPs based on this example or from scratch using this command.
elmos bsp create <bsp_name>

# Open the BSP source code in your preferred code editor for development. You can use the provided example as a reference for how to structure your BSP and interact with the kernel and hardware. The BSP source code will be located in the workspace under the "bsp" directory (e.g., /path/to/workspace/bsp/<bsp_name>/).
elmos bsp edit <bsp_name> # Ask user to use nano, vim, code, etc. based on their preference

# Build the BSP using the same toolchain and kernel configuration as the main kernel build. This will compile the BSP source code and generate the corresponding files that can be used for running the kernel on specific hardware or in QEMU with specific machine configuration. The built BSP will be located in the workspace under the "build/bsp" directory (e.g., /path/to/workspace/build/bsp/<bsp_name>/). Default parallelism is based on the number of CPU cores, but you can specify a custom number of jobs for faster builds. This command will build all BSPs in the workspace if no BSP name is specified, or build a specific BSP if the BSP name is provided.
elmos bsp build [bsp_name] [-j|--jobs <num_jobs>]

# Show the result of the BSP build, including the location of the built files and any errors or warnings if the build failed. This will also show the status of the BSP build and whether it is ready to be used for running the kernel.
elmos bsp show [bsp_name]

# Clean the BSP build artifacts (optional)
elmos bsp clean [bsp_name]||all
```

# 12. Plugin Development (Optional)

```bash
# List available plugins in the specified workspace (or all workspaces if "all" is specified). This will show the existing plugins that you can use as references or starting points for your own plugin development. The plugins are located in the workspace under the "plugins" directory (e.g., /path/to/workspace/plugins/).
elmos plugin list

# Create a new plugin scaffold with the specified name (e.g., "my_plugin") in the workspace. This will generate a basic plugin template with the necessary files and structure for development. This step is optional because we already provide you 1 example plugin in the workspace (hello_world, etc.) that you can use as a reference or starting point for your own plugin development. You can also create additional plugins based on this example or from scratch using this command.
elmos plugin create <plugin_name>

# Open the plugin source code in your preferred code editor for development. You can use the provided example as a reference for how to structure your plugin and interact with the elmos API. The plugin source code will be located in the workspace under the "plugins" directory (e.g., /path/to/workspace/plugins/<plugin_name>/).
elmos plugin edit <plugin_name> # Ask user to use nano, vim, code, etc. based on their preference

# Build the plugin using the same toolchain and configuration as the main elmos build. This will compile the plugin source code and generate the corresponding files that can be used for extending the functionality of elmos. The built plugin will be located in the workspace under the "build/plugins" directory (e.g., /path/to/workspace/build/plugins/<plugin_name>/). Default parallelism is based on the number of CPU cores, but you can specify a custom number of jobs for faster builds. This command will build all plugins in the workspace if no plugin name is specified, or build a specific plugin if the plugin name is provided.
elmos plugin build [plugin_name] [-j|--jobs <num_jobs>]

# Show the result of the plugin build, including the location of the built files and any errors or warnings if the build failed. This will also show the status of the plugin build and whether it is ready to be used for extending elmos.
elmos plugin show [plugin_name]

# Clean the plugin build artifacts (optional)
elmos plugin clean [plugin_name]||all
```

# 13. Patch Applying (Optional)

```bash
# Apply patches to the kernel source code or other components. This will allow you to easily apply patches from the Linux kernel mailing list or other sources to your kernel source code for testing and development.
# List available patches that can be applied to the kernel source code. This will show the patches that are available in the workspace under the "patches" directory (e.g., /path/to/workspace/patches/) that you can apply to the kernel source code. The patches are organized based on the kernel version and component they target (e.g., v6.18/generic/fix-copy-range).
elmos patch list

# Apply specific patches from loaded workspace to the kernel source code. This will apply the specified patches to the kernel source code in the workspace, allowing you to test and develop with the changes introduced by the patches. You can specify the patches using their paths in the workspace (e.g., v6.18/generic/fix-copy-range) to apply them to the kernel source code.
elmos patch apply <specified_patch> [other_patches]

# Show the applied patches and their status. This will show the patches that have been applied to the kernel source code, including their paths, target kernel version and component, and whether they were applied successfully or if there were any errors or warnings during the application process. This will help you keep track of the patches that have been applied and their impact on the kernel source code.
elmos patch status

# Show the specified patch information, including the patch content, target kernel version and component, and any errors or warnings if the patch cannot be applied successfully. This will help you understand the changes introduced by the patch and whether it is compatible with your kernel source code.
elmos patch show <specified_patch>

# Remove the applied patch(es) from the kernel source code (optional). This will allow you to easily revert the changes introduced by the patch if you want to test without it or if it causes issues with your kernel source code.
elmos patch remove <specified_patch>||all
```

# After these steps, you can start developing your own kernel modules, user space applications, BSPs, and plugins based on the provided examples and templates. You can also customize the build process and configuration to fit your specific needs and preferences. For more advanced usage and customization options, please refer to the documentation and resources provided in the elmos repository. Happy hacking!
