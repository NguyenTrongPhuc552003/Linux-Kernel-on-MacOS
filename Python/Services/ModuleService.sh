#!/bin/bash
# Python/Services/ModuleService.sh
# Handles compilation of out-of-tree modules.

source "$(dirname "$0")/EnvironmentService.sh"

# 1. Prepare Headers
prepare_headers() {
	ensure_mounted
	cd "$KERNEL_DIR" || exit 1

	echo "  [MODULE] Preparing kernel headers for $TARGET_ARCH..."

	# ADDED: LLVM=1 (to use Clang) and HOSTCFLAGS (for macOS headers)
	if make ARCH="$TARGET_ARCH" \
		LLVM=1 \
		CROSS_COMPILE="$CROSS_COMPILE" \
		HOSTCFLAGS="$HOSTCFLAGS" \
		modules_prepare; then

		echo "  [SUCCESS] Headers ready. You can now build external modules."
	else
		echo "  [FAIL] Kernel headers preparation failed."
		exit 1
	fi
}

# 2. Build Function
build_module() {
	local mod_name="$1"

	ensure_mounted

	# Auto-check for headers
	if [ ! -f "$KERNEL_DIR/Module.symvers" ]; then
		echo "  [WARN] Module.symvers not found. Triggering auto-prepare..."
		prepare_headers
	fi

	if [ -z "$mod_name" ]; then
		echo "  [MODULE] Building ALL modules in $MODULES_DIR..."
		for d in "$MODULES_DIR"/*/; do
			[ -d "$d" ] || continue
			dirname=$(basename "$d")
			build_single_module "$dirname"
		done
	else
		build_single_module "$mod_name"
	fi
}

build_single_module() {
	local name="$1"
	local path="$MODULES_DIR/$name"

	if [ ! -d "$path" ]; then
		echo "  [ERROR] Module '$name' not found in $MODULES_DIR"
		exit 1
	fi

	# Check TARGET_ARCH (Debug safety)
	if [ -z "$TARGET_ARCH" ]; then
		echo "  [ERROR] TARGET_ARCH is empty. Python environment injection failed."
		exit 1
	fi

	echo "  [MODULE] Building: $name (ARCH=$TARGET_ARCH)..."
	cd "$path" || exit 1

	# ADDED: LLVM=1 and HOSTCFLAGS
	make -C "$KERNEL_DIR" \
		M="$path" \
		ARCH="$TARGET_ARCH" \
		LLVM=1 \
		CROSS_COMPILE="$CROSS_COMPILE" \
		HOSTCFLAGS="$HOSTCFLAGS" \
		modules
}

# 3. Clean Function
clean_module() {
	local mod_name="$1"
	ensure_mounted

	if [ -z "$mod_name" ]; then
		echo "  [MODULE] Cleaning ALL modules..."
		for d in "$MODULES_DIR"/*/; do
			[ -d "$d" ] || continue
			make -C "$KERNEL_DIR" M="$d" ARCH="$TARGET_ARCH" clean
		done
	else
		echo "  [MODULE] Cleaning: $mod_name"
		make -C "$KERNEL_DIR" M="$MODULES_DIR/$mod_name" ARCH="$TARGET_ARCH" clean
	fi
}

# 4. Dispatcher
MODE="$1"
shift

case "$MODE" in
headers)
	prepare_headers
	;;
build)
	build_module "$1"
	;;
clean)
	clean_module "$1"
	;;
*)
	echo "Usage: $0 {headers|build|clean} [module_name]"
	exit 1
	;;
esac
