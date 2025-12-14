#!/bin/bash
# scripts/build.sh
# Handles kernel configuration, compilation, and cleaning.

# ─────────────────────────────────────────────────────────────
# Internal State/Config Persistence
# ─────────────────────────────────────────────────────────────
# Path to store the currently configured architecture.
# We store this outside the kernel source tree so `make distclean` doesn't remove it.
CONFIG_FILE="${SCRIPT_DIR}/.build_config.env"

# Load TARGET_ARCH from config file if it exists
if [ -f "$CONFIG_FILE" ]; then
	source "$CONFIG_FILE"
fi

# Set default if not loaded
if [ -z "$TARGET_ARCH" ]; then
	# Default to arm64 if running on Apple Silicon, otherwise x86_64
	if [[ "$(uname -m)" == "arm64" ]]; then
		TARGET_ARCH="arm64"
	else
		TARGET_ARCH="x86_64"
	fi
fi

# ─────────────────────────────────────────────────────────────
# 1. Set Target Architecture (`arch <target>`)
# ─────────────────────────────────────────────────────────────
set_arch() {
	local new_arch="$1"

	if [[ -z "$new_arch" ]]; then
		echo -e "  [${YELLOW}INFO${NC}] Current ARCH is: ${GREEN}${TARGET_ARCH}${NC}"
		echo -e "  [${YELLOW}INFO${NC}] Usage: arch <arm64|x86_64|...>"
		return 0
	fi

	echo -e "  [${YELLOW}ARCH${NC}] Setting TARGET_ARCH to: ${GREEN}${new_arch}${NC}"
	export TARGET_ARCH="$new_arch"

	# Persist setting to file
	echo "export TARGET_ARCH=\"${TARGET_ARCH}\"" >"$CONFIG_FILE"
	echo -e "  [${GREEN}SAVED${NC}] Configuration saved to $CONFIG_FILE"
}

# ─────────────────────────────────────────────────────────────
# 2. Configure Kernel (`config [type]`)
# ─────────────────────────────────────────────────────────────
run_config() {
	local config_type="${1:-defconfig}" # Default to defconfig

	ensure_mounted
	cd "$KERNEL_DIR" || {
		echo -e "  [${RED}FAIL${NC}] Could not enter KERNEL_DIR."
		exit 1
	}

	echo -e "  [${YELLOW}CONFIG${NC}] Running 'make ${config_type}' for ARCH=${TARGET_ARCH}..."

	# Execute make command with required environment variables
	# We use LLVM=1 and the cross-compiler prefix
	if make ARCH="$TARGET_ARCH" LLVM=1 CROSS_COMPILE=llvm- "$config_type"; then
		echo -e "  [${GREEN}SUCCESS${NC}] Configuration complete."
	else
		echo -e "  [${RED}FAIL${NC}] Configuration failed. Check 'make ${config_type}' output."
		exit 1
	fi
}

# ─────────────────────────────────────────────────────────────
# 3. Build Kernel (`build [jobs] [targets]`)
# ─────────────────────────────────────────────────────────────
run_build() {
	# Default number of jobs is all available cores
	local jobs="${1:-$(sysctl -n hw.ncpu)}"
	# Default targets
	shift || true # Remove $1 (jobs)
	local targets="${*:-Image dtbs modules}"

	ensure_mounted
	cd "$KERNEL_DIR" || {
		echo -e "  [${RED}FAIL${NC}] Could not enter KERNEL_DIR."
		exit 1
	}

	# Pre-flight check for .config
	if [ ! -f .config ]; then
		echo -e "  [${RED}ERROR${NC}] .config file not found."
		echo "  Run './run.sh config' first to generate configuration."
		exit 1
	fi

	echo -e "  [${YELLOW}BUILD${NC}] Starting build process..."
	echo "  -> ARCH: ${TARGET_ARCH}, Jobs: ${jobs}, Targets: ${targets}"

	# Execute build command
	# Assuming 'make' in PATH is GNU make (via common.env)
	if make -j"$jobs" ARCH="$TARGET_ARCH" LLVM=1 CROSS_COMPILE=llvm- $targets; then
		echo -e "  [${GREEN}SUCCESS${NC}] Build complete!"
	else
		echo -e "  [${RED}FAIL${NC}] Build failed. Check logs."
		exit 1
	fi
}

# ─────────────────────────────────────────────────────────────
# 4. Clean Build Artifacts (`clean`)
# ─────────────────────────────────────────────────────────────
run_clean() {
	ensure_mounted
	cd "$KERNEL_DIR" || {
		echo -e "  [${RED}FAIL${NC}] Could not enter KERNEL_DIR."
		exit 1
	}

	echo -e "  [${YELLOW}CLEAN${NC}] Running 'make distclean' for ARCH=${TARGET_ARCH}..."

	# distclean removes generated files, configs, and prepares for a fresh build
	if make ARCH="$TARGET_ARCH" distclean; then
		echo -e "  [${GREEN}SUCCESS${NC}] Clean complete. Repository state is fresh."
	else
		echo -e "  [${RED}FAIL${NC}] Clean failed."
		exit 1
	fi
}
