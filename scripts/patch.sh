#!/bin/bash
# scripts/patch.sh
# Handles the application of a patch file using git's 3-way merge.

# ─────────────────────────────────────────────────────────────
# 1. Apply a Patch File
# ─────────────────────────────────────────────────────────────
apply_patch() {
	local patch_path="$1"

	if [[ -z "$patch_path" ]]; then
		echo -e "  [${RED}ERROR${NC}] Usage: patch <path/to/patchfile>"
		echo "  Example: patch patches/v6.18/0001-my-change.patch"
		exit 1
	fi

	# 1. Resolve path
	local full_patch_path
	# Check if the path is absolute or relative to the project root
	if [[ "$patch_path" == /* ]]; then
		# Absolute path provided
		full_patch_path="$patch_path"
	else
		# Assume path is relative to SCRIPT_DIR (project root)
		full_patch_path="${SCRIPT_DIR}/${patch_path}"
	fi

	# 2. Check existence
	if [ ! -f "$full_patch_path" ]; then
		echo -e "  [${RED}FAIL${NC}] Patch file not found at: $full_patch_path"
		exit 1
	fi

	ensure_mounted
	cd "$KERNEL_DIR" || {
		echo -e "  [${RED}FAIL${NC}] Could not enter KERNEL_DIR."
		exit 1
	}

	echo -e "  [${YELLOW}PATCH${NC}] Applying patch: $(basename "$patch_path")"

	# Use git apply with --3way for conflict handling.
	# --check: Verify the patch can be applied without actually applying it.
	if git apply --check --3way "$full_patch_path"; then
		echo -e "  [${YELLOW}INFO${NC}] Patch check passed. Applying..."

		# Apply the patch
		if git apply --3way "$full_patch_path"; then
			echo -e "  [${GREEN}SUCCESS${NC}] Patch applied successfully."

			# Check for rejected files (.rej) that indicate merge conflicts
			# This is critical for 3-way to work; conflicts still leave .rej files
			if find . -name "*.rej" -print -quit 2>/dev/null; then
				echo -e "  [${RED}WARNING${NC}] Conflicts found! Review and manually resolve *.rej files."
			fi

		else
			echo -e "  [${RED}FAIL${NC}] Patch application failed. Check above output."
			exit 1
		fi
	else
		echo -e "  [${RED}FAIL${NC}] Patch check failed. Patch may be outdated or already applied."
		echo "  Try applying manually or updating the kernel source."
		exit 1
	fi
}
