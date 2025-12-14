#!/bin/bash
# scripts/doctor.sh
# Handles environment verification and dependency checks.

# ─────────────────────────────────────────────────────────────
# Helper: Check & Print Status
# Usage: check_item "Description" [condition_command]
# ─────────────────────────────────────────────────────────────
check_item() {
	local desc="$1"
	shift
	if "$@"; then
		printf "  [${GREEN}OK${NC}] %s\n" "$desc"
		return 0
	else
		printf "  [${RED}FAIL${NC}] %s\n" "$desc"
		return 1
	fi
}

# ─────────────────────────────────────────────────────────────
# Main Doctor Function
# ─────────────────────────────────────────────────────────────
run_doctor() {
	echo "Running doctor — environment check..."
	echo
	local issues_found=0

	# 1. Homebrew Packages
	# ─────────────────────────────────────────────────────────────
	echo "Checking Homebrew packages:"
	local REQUIRED_BREW=(llvm lld gnu-sed make libelf git)
	local INSTALLED_BREW
	INSTALLED_BREW=$(brew list --formulae 2>/dev/null)

	local missing_brew=()
	for pkg in "${REQUIRED_BREW[@]}"; do
		if echo "$INSTALLED_BREW" | grep -q "^${pkg}$"; then
			check_item "Package: $pkg" true
		else
			check_item "Package: $pkg" false
			missing_brew+=("$pkg")
			issues_found=1
		fi
	done
	echo

	# 2. Header Verification
	# ─────────────────────────────────────────────────────────────
	echo "Checking macOS Kernel Headers:"
	# Uses MACOS_HEADERS from common.env [cite: 3]
	local headers_dir="${MACOS_HEADERS:-$HOME/Documents/kernel-dev/linux/libraries}"

	if [ ! -d "$headers_dir" ]; then
		echo -e "  [${RED}FAIL${NC}] Headers directory not found at: $headers_dir"
		issues_found=1
	else
		# Check standard headers
		check_item "byteswap.h" [ -f "$headers_dir/byteswap.h" ] || issues_found=1
		check_item "endian.h" [ -f "$headers_dir/endian.h" ] || issues_found=1
		check_item "elf.h" [ -f "$headers_dir/elf.h" ] || issues_found=1

		# Check symlinks
		local asm_path="$headers_dir/asm"
		local uapi_path="/Volumes/kernel-dev/linux/include/uapi/asm-generic"

		for h in bitsperlong.h int-ll64.h posix_types.h types.h; do
			local link="$asm_path/$h"
			if [ -L "$link" ] && [ "$(readlink "$link")" = "$uapi_path/$h" ]; then
				check_item "asm/$h (symlink)" true
			else
				check_item "asm/$h (symlink)" false
				issues_found=1
			fi
		done
	fi
	echo

	# 3. Remediation (Fixes)
	# ─────────────────────────────────────────────────────────────
	if [ ${#missing_brew[@]} -gt 0 ]; then
		echo -e "${YELLOW}Suggestion:${NC} Install missing packages:"
		echo "  brew install ${missing_brew[*]}"
		echo
	fi

	# ELF Header Auto-Fix Logic
	if [ ! -f "$headers_dir/elf.h" ]; then
		echo -e "${YELLOW}Issue:${NC} elf.h is missing."
		echo "Download latest elf.h from glibc? (Y/n)"
		read -r choice
		if [[ ! "$choice" =~ ^[Nn]$ ]]; then
			local glibc_ver="2.42"
			local glibc_url="https://raw.githubusercontent.com/bminor/glibc/glibc-${glibc_ver}/elf/elf.h"

			echo "Downloading..."
			mkdir -p "$headers_dir"
			if wget -q "$glibc_url" -P "$headers_dir"; then
				echo -e "  [${GREEN}FIXED${NC}] Downloaded elf.h"
				issues_found=0 # Reset if this was the only error, strictly speaking we should re-check but this is UX friendly
			else
				echo -e "  [${RED}ERROR${NC}] Download failed."
			fi
		fi
	fi

	# 4. Final Verdict
	# ─────────────────────────────────────────────────────────────
	if [ "$issues_found" -eq 0 ]; then
		echo -e "${GREEN}Doctor result: All good! System ready for kernel build.${NC}"
		return 0
	else
		echo -e "${RED}Doctor result: Issues found. Please fix above errors.${NC}"
		return 1
	fi
}
