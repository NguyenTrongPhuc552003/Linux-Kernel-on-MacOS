#!/bin/bash
# Python/Services/PatchService.sh
# Applies patches to the kernel source tree.

source "$(dirname "$0")/EnvironmentService.sh"

apply_patch() {
	local target="$1"

	ensure_mounted
	cd "$KERNEL_DIR" || exit 1

	# Handle "auto" mode
	if [ "$target" == "auto" ]; then
		# 1. Get Kernel Version (from Makefile)
		local kver=$(make kernelversion 2>/dev/null)
		# Extract Major.Minor (e.g., 6.18)
		local short_ver=$(echo "$kver" | cut -d. -f1-2)
		local patch_subdir="${PATCH_DIR}/v${short_ver}"

		echo "  [PATCH] Auto-detecting version: $kver"

		if [ -d "$patch_subdir" ]; then
			echo "  [PATCH] Found patch directory: patches/v${short_ver}"
			# Apply all patches in that dir
			for p in "$patch_subdir"/*.patch; do
				[ -e "$p" ] || continue
				_apply_single_file "$p"
			done
			return
		else
			echo "  [WARN] No specific patches found for v${short_ver} in patches/"
			return
		fi
	fi

	# Handle single file
	# If path is relative to repo root, fix it
	if [ ! -f "$target" ] && [ -f "${PROJECT_ROOT}/patches/$target" ]; then
		target="${PROJECT_ROOT}/patches/$target"
	fi

	if [ -f "$target" ]; then
		_apply_single_file "$target"
	else
		echo "  [ERROR] Patch file not found: $target"
		exit 1
	fi
}

_apply_single_file() {
	local file="$1"
	local filename=$(basename "$file")

	echo "  [PATCH] Applying: $filename..."

	# Try git apply first (it handles renames/binary better and checks indexes)
	if git apply --check "$file" &>/dev/null; then
		git apply "$file"
		echo "  [SUCCESS] Applied via git."
	elif patch -p1 --dry-run <"$file" &>/dev/null; then
		# Fallback to standard patch
		patch -p1 <"$file"
		echo "  [SUCCESS] Applied via patch -p1."
	else
		echo "  [SKIP] Patch seems already applied or conflicts exist."
	fi
}

# Dispatcher
MODE="$1"
shift

case "$MODE" in
apply)
	apply_patch "$1"
	;;
*)
	echo "Usage: $0 apply <file|auto>"
	exit 1
	;;
esac
