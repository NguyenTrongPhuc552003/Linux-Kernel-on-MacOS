#!/bin/bash
# scripts/patch.sh
# Modular patch application using 'git am' (preferred over git apply).
# Supports standard .patch files with commit metadata.

# ─────────────────────────────────────────────────────────────
# Helper: Validate patch file existence and resolve path
# ─────────────────────────────────────────────────────────────
resolve_patch_path() {
	local input_path="$1"

	if [[ -z "$input_path" ]]; then
		echo -e "  [${RED}ERROR${NC}] No patch file specified."
		echo "  Usage: ./run.sh patch <path/to/patchfile.patch>"
		return 1
	fi

	local full_path
	if [[ "$input_path" == /* ]]; then
		# Absolute path
		full_path="$input_path"
	else
		# Relative to project root (SCRIPT_DIR)
		full_path="${SCRIPT_DIR}/${input_path}"
	fi

	if [ ! -f "$full_path" ]; then
		echo -e "  [${RED}FAIL${NC}] Patch file not found: $full_path"
		return 1
	fi

	echo "$full_path"
}

# ─────────────────────────────────────────────────────────────
# Helper: Check if patch can be applied cleanly
# ─────────────────────────────────────────────────────────────
check_patch_applicability() {
	local patch_file="$1"

	echo -e "  [${YELLOW}CHECK${NC}] Testing patch applicability..."

	# Clean any aborted am session
	git am --abort 2>/dev/null || true

	# First try: git am dry-run (preferred, preserves commit)
	if git am --3way --dry-run "$patch_file" >/dev/null 2>&1; then
		echo -e "  [${GREEN}OK${NC}] Patch passes 'git am' dry-run."
		return 0
	fi

	echo -e "  [${YELLOW}NOTE${NC}] 'git am --dry-run' failed, trying fallback check..."

	# Fallback: git apply --check (more tolerant for context)
	if git apply --3way --check "$patch_file" >/dev/null 2>&1; then
		echo -e "  [${GREEN}OK${NC}] Patch passes 'git apply --check' (will use git am anyway)."
		return 0
	fi

	echo -e "  [${RED}FAIL${NC}] Patch cannot be applied cleanly with either method."
	return 1
}

# ─────────────────────────────────────────────────────────────
# Core: Apply patch using git am
# ─────────────────────────────────────────────────────────────
check_patch_git() {
	local patch_file="$1"
	local patch_name
	patch_name=$(basename "$patch_file")

	echo -e "  [${YELLOW}APPLY${NC}] Applying patch: $patch_name"

	# Final application
	if git am --3way --signoff "$patch_file"; then
		echo -e "  [${GREEN}SUCCESS${NC}] Patch '$patch_name' applied successfully with commit."
	else
		echo -e "  [${RED}CONFLICT${NC}] Patch application failed due to conflicts."
		echo "  → Run 'git am --abort' to cancel, or 'git am --continue' after resolving."
		echo "  → Conflicted files are marked with conflict markers."
		return 1
	fi
}

# ─────────────────────────────────────────────────────────────
# Public function called by run.sh patch
# ─────────────────────────────────────────────────────────────
apply_patch() {
	local patch_path="$1"
	local full_patch

	# 1. Resolve and validate patch path
	full_patch=$(resolve_patch_path "$patch_path") || return 1

	# 2. Ensure kernel directory is ready
	ensure_mounted
	cd "$KERNEL_DIR" || {
		echo -e "  [${RED}FAIL${NC}] Cannot access kernel source directory: $KERNEL_DIR"
		return 1
	}

	# 3. Dry-run check
	if ! check_patch_applicability "$full_patch"; then
		echo
		echo -e "  [${YELLOW}HINT${NC}] Possible reasons:"
		echo "    • Patch already applied"
		echo "    • Kernel version mismatch"
		echo "    • Patch format incompatible with 'git am'"
		echo
		echo "  Try: git apply --check '$full_patch' for raw apply test"
		return 1
	fi

	echo

	# 4. Apply the patch
	if check_patch_git "$full_patch"; then
		echo
		echo -e "  [${GREEN}DONE${NC}] Patch applied and committed."
		echo "  Use 'git log -1' to view the new commit."
		return 0
	else
		return 1
	fi
}
