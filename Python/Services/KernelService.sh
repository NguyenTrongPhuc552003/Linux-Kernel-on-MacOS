#!/bin/bash
# Python/Services/KernelService.sh
# Handles kernel configuration and compilation.

# 1. Source Environment (Fixes 'ensure_mounted not found')
# ------------------------------------------------------
source "$(dirname "$0")/EnvironmentService.sh"

# 2. Config & Build Functions
# ---------------------------

run_config() {
	local target="$1"
	[ -z "$target" ] && target="defconfig"

	ensure_mounted
	cd "$KERNEL_DIR" || exit 1

	echo -e "  [KERNEL] Running config: $target (ARCH=${TARGET_ARCH})"

	# Run make with environment variables set
	# We use 'eval' or direct execution. Direct is safer.
	make ARCH="$TARGET_ARCH" CROSS_COMPILE="$CROSS_COMPILE" LLVM=1 "$target"
}

run_build() {
	local jobs="$1"
	shift
	local targets="$*"
	[ -z "$targets" ] && targets="Image dtbs modules"
	[ -z "$jobs" ] && jobs=$(sysctl -n hw.ncpu) # Auto-detect cores

	ensure_mounted
	cd "$KERNEL_DIR" || exit 1

	# Pre-flight check
	if [ ! -f .config ]; then
		echo "  [ERROR] .config file not found."
		echo "  Run 'km config' first to generate a configuration."
		exit 1
	fi

	echo -e "  [BUILD] Starting build..."
	echo "  -> ARCH: ${TARGET_ARCH} | Jobs: ${jobs} | Targets: ${targets}"

	make -j"$jobs" ARCH="$TARGET_ARCH" CROSS_COMPILE="$CROSS_COMPILE" LLVM=1 $targets
}

run_clean() {
	ensure_mounted
	cd "$KERNEL_DIR" || exit 1
	echo "  [BUILD] Cleaning kernel tree..."
	make ARCH="$TARGET_ARCH" distclean
}

# 3. Dispatcher
# -------------
# The first argument determines the mode (build, config, clean)
MODE="$1"
shift

case "$MODE" in
config)
	run_config "$@"
	;;
build)
	run_build "$@"
	;;
clean)
	run_clean
	;;
*)
	echo "Usage: $0 {build|config|clean} [args...]"
	exit 1
	;;
esac
