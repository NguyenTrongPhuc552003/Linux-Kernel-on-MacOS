#!/bin/bash
# Python/Services/KernelService.sh
# Handles kernel configuration and compilation.

# 1. Source Environment (Fixes 'ensure_mounted not found')
# ------------------------------------------------------
source "$(dirname "$0")/EnvironmentService.sh"

# ─────────────────────────────────────────────────────────────
# 2. Config & Build Functions
# ─────────────────────────────────────────────────────────────

run_config() {
	local target="$1"
	[ -z "$target" ] && target="defconfig"

	ensure_mounted
	cd "$KERNEL_DIR" || exit 1

	echo -e "  [KERNEL] Running config: $target (ARCH=${TARGET_ARCH})"

	# 1. Run the standard make config
	make ARCH="$TARGET_ARCH" CROSS_COMPILE="$CROSS_COMPILE" LLVM=1 "$target"

	# 2. LEGACY LOGIC: Post-process kvm_guest.config
	# If using kvm_guest.config, we MUST explicitly enable DRM/FB
	# because Kconfig might drop them if dependencies aren't met during the merge.
	if [[ "$target" == *"kvm_guest.config"* ]]; then
		echo "  [CONFIG] applying macOS/QEMU graphics fixups..."

		# Ensure scripts/config is executable
		chmod +x scripts/config

		./scripts/config --file .config \
			--enable CONFIG_DRM \
			--enable CONFIG_DRM_VIRTIO_GPU \
			--enable CONFIG_FB \
			--enable CONFIG_FRAMEBUFFER_CONSOLE

		# Refresh the config to ensure dependency tree is valid
		# (Equivalent to 'make olddefconfig')
		make ARCH="$TARGET_ARCH" CROSS_COMPILE="$CROSS_COMPILE" LLVM=1 olddefconfig
	fi
}

run_build() {
	local jobs="$1"
	shift
	local targets="$*"
	[ -z "$targets" ] && targets="Image dtbs modules"
	[ -z "$jobs" ] && jobs=$(sysctl -n hw.ncpu)

	ensure_mounted
	cd "$KERNEL_DIR" || exit 1

	# Pre-flight check
	if [ ! -f .config ]; then
		echo "  [ERROR] .config file not found."
		echo "  Run 'km config' first."
		exit 1
	fi

	echo -e "  [BUILD] Starting build..."
	echo "  -> ARCH: ${TARGET_ARCH} | Jobs: ${jobs} | Targets: ${targets}"

	# Check for Rust Tools
	RUST_FLAGS=""
	if command -v rustc >/dev/null && command -v bindgen >/dev/null; then
		export RUSTC=$(command -v rustc)
		export BINDGEN=$(command -v bindgen)
		export RUSTFMT=$(command -v rustfmt)
		export LIBCLANG_PATH="$(brew --prefix llvm)/lib"
		RUST_FLAGS="RUSTC=$RUSTC BINDGEN=$BINDGEN RUSTFMT=$RUSTFMT"
		echo "  [INFO] Rust support enabled."
	fi

	make -j"$jobs" \
		ARCH="$TARGET_ARCH" \
		LLVM=1 \
		CROSS_COMPILE="$CROSS_COMPILE" \
		HOSTCFLAGS="$HOSTCFLAGS" \
		$RUST_FLAGS \
		$targets
}

run_clean() {
	ensure_mounted
	cd "$KERNEL_DIR" || exit 1
	echo "  [CLEAN] Running distclean..."
	make ARCH="$TARGET_ARCH" distclean
}

# 3. Dispatcher
# -------------
# The first argument determines the mode (build, config, clean)
case "$1" in
config)
	shift
	run_config "$@"
	;;
build)
	shift
	run_build "$@"
	;;
clean)
	shift
	run_clean "$@"
	;;
esac
