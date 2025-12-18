#!/bin/bash
# scripts/doctor.sh
# Modular environment and dependency checker for native kernel builds + disk-based rootfs.

# ─────────────────────────────────────────────────────────────
# Helper: Colored status check
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
# 1. Check Homebrew packages
# ─────────────────────────────────────────────────────────────
check_homebrew_packages() {
	echo "Checking required Homebrew packages:"
	local REQUIRED=(
		llvm
		lld
		gnu-sed
		make
		libelf
		git
		qemu
		fakeroot
		e2fsprogs
		wget
	)

	# Get installed formulae (one per line)
	local INSTALLED
	INSTALLED=$(brew list --formulae 2>/dev/null | tr ' ' '\n')

	local missing=()

	for pkg in "${REQUIRED[@]}"; do
		if echo "$INSTALLED" | grep -Fxq "$pkg"; then
			check_item "Package: $pkg" true
		else
			check_item "Package: $pkg" false
			missing+=("$pkg")
			((issues_found++))
		fi
	done

	if [ ${#missing[@]} -gt 0 ]; then
		echo
		echo -e "${YELLOW}Fix:${NC} brew install ${missing[*]}"
		echo
	fi
}

# ─────────────────────────────────────────────────────────────
# 2. Check custom headers directory and shims
# ─────────────────────────────────────────────────────────────
check_custom_headers() {
	echo "Checking custom macOS kernel headers:"
	local headers_dir="${MACOS_HEADERS:-$HOME/Documents/kernel-dev/linux/libraries}"

	if [ ! -d "$headers_dir" ]; then
		echo -e "  [${RED}FAIL${NC}] Directory not found: $headers_dir"
		((issues_found++))
		return
	fi

	check_item "Directory exists" true

	check_item "byteswap.h" [ -f "$headers_dir/byteswap.h" ] || ((issues_found++))
	check_item "endian.h" [ -f "$headers_dir/endian.h" ] || ((issues_found++))
	check_item "elf.h" [ -f "$headers_dir/elf.h" ] || ((issues_found++))

	# asm-generic symlinks
	local asm_path="$headers_dir/asm"
	local uapi_path="${KERNEL_DIR:-/Volumes/kernel-dev/linux}/include/uapi/asm-generic"

	if [ -d "$asm_path" ] && [ -d "$uapi_path" ]; then
		for h in bitsperlong.h int-ll64.h posix_types.h types.h; do
			local link="$asm_path/$h"
			if [ -L "$link" ] && [ "$(readlink "$link")" = "$uapi_path/$h" ]; then
				check_item "asm/$h symlink" true
			else
				check_item "asm/$h symlink" false
				((issues_found++))
			fi
		done
	else
		echo -e "  [${YELLOW}SKIP${NC}] asm symlinks (kernel source not mounted yet)"
	fi
	echo
}

# ─────────────────────────────────────────────────────────────
# 3. Auto-fix missing elf.h
# ─────────────────────────────────────────────────────────────
fix_missing_elf_h() {
	local headers_dir="${MACOS_HEADERS:-$HOME/Documents/kernel-dev/linux/libraries}"

	if [ -f "$headers_dir/elf.h" ]; then
		return
	fi

	echo -e "${YELLOW}elf.h missing — download from glibc? (Y/n)${NC}"
	read -r choice
	if [[ "$choice" =~ ^[Nn]$ ]]; then
		return
	fi

	local glibc_ver="2.42"
	local url="https://raw.githubusercontent.com/bminor/glibc/glibc-${glibc_ver}/elf/elf.h"

	mkdir -p "$headers_dir"
	echo "Downloading elf.h..."
	if wget -q --show-progress "$url" -O "$headers_dir/elf.h"; then
		echo -e "  [${GREEN}FIXED${NC}] elf.h downloaded."
	else
		echo -e "  [${RED}ERROR${NC}] Download failed."
		((issues_found++))
	fi
	echo
}

# ─────────────────────────────────────────────────────────────
# 4. Check debootstrap submodule
# ─────────────────────────────────────────────────────────────
check_debootstrap() {
	echo "Checking debootstrap tool:"
	if [ -f "${DEBOOTSTRAP_PATH}/debootstrap" ]; then
		check_item "debootstrap script present" true
	else
		echo -e "  [${RED}MISSING${NC}] Not found at: ${DEBOOTSTRAP_PATH}/debootstrap"
		echo "  Fix: git submodule update --init --recursive tools/debootstrap"
		((issues_found++))
	fi
	echo
}

# ─────────────────────────────────────────────────────────────
# Main doctor function
# ─────────────────────────────────────────────────────────────
run_doctor() {
	echo -e "  [${YELLOW}INFO${NC}] Running doctor — environment check..."
	echo

	local issues_found=0

	check_homebrew_packages
	check_custom_headers
	check_debootstrap
	fix_missing_elf_h # Run after header check

	# ─────────────────────────────────────────────────────────────
	# Final verdict
	# ─────────────────────────────────────────────────────────────
	if [ "$issues_found" -eq 0 ]; then
		echo -e "  ${GREEN}All checks passed! Environment ready for build and rootfs creation.${NC}"
		return 0
	else
		echo -e "  ${RED}Found $issues_found issue(s). Please fix before proceeding.${NC}"
		return 1
	fi
}
