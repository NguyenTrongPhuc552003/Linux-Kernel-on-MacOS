#!/bin/bash
# scripts/qemu.sh
# Handles QEMU execution and GDB debugging for a pre-built, initramfs-based Linux kernel.

# Requires: INITRAMFS_ARCHIVE, KERNEL_DIR, TARGET_ARCH are set.

# ─────────────────────────────────────────────────────────────
# 1. Architecture Configuration Strategy
# ─────────────────────────────────────────────────────────────
_configure_qemu_for_arch() {
	# Default settings
	QEMU_MEMORY="2G"
	QEMU_SMP="$(sysctl -n hw.ncpu)"
	QEMU_CONSOLE="ttyS0" # Default, overridden below if needed

	case "$TARGET_ARCH" in
	riscv)
		QEMU_BIN="qemu-system-riscv64"
		QEMU_ARCH_FLAGS=(
			-machine virt
			-bios default
			-cpu rv64
		)
		QEMU_CONSOLE="ttyS0"
		;;
	arm64)
		QEMU_BIN="qemu-system-aarch64"
		QEMU_ARCH_FLAGS=(
			-machine virt
			-cpu max
		)
		QEMU_CONSOLE="ttyAMA0" # Standard console for ARM64 virt machine
		;;
	*)
		echo -e "  [${RED}ERROR${NC}] Unsupported ARCH for QEMU: ${TARGET_ARCH}" >&2
		exit 1
		;;
	esac
}

# ─────────────────────────────────────────────────────────────
# 2. Main Execution Logic
# ─────────────────────────────────────────────────────────────
_execute_qemu() {
	local GDB_FLAGS="$1"
	local DEBUG_MODE="$2"

	# 1. Configuration & Validation
	local KERNEL_IMAGE="${KERNEL_DIR}/arch/${TARGET_ARCH}/boot/Image"

	_configure_qemu_for_arch

	if ! command -v "$QEMU_BIN" &>/dev/null; then
		echo -e "  [${RED}ERROR${NC}] Binary '$QEMU_BIN' not found." >&2
		exit 1
	fi
	if [ ! -f "$KERNEL_IMAGE" ]; then
		echo -e "  [${RED}ERROR${NC}] Kernel image '$KERNEL_IMAGE' not found. Run './run.sh build' first."
		exit 1
	fi
	if [ ! -f "$INITRAMFS_ARCHIVE" ]; then
		echo -e "  [${RED}ERROR${NC}] Initramfs archive '$INITRAMFS_ARCHIVE' not found. Run './run.sh rootfs' first."
		exit 1
	fi

	echo -e "  [${YELLOW}QEMU${NC}] Starting emulation for ${GREEN}${TARGET_ARCH}${NC}..."
	[ -n "$GDB_FLAGS" ] && echo -e "  [${YELLOW}DEBUG${NC}] GDB Stub enabled."

	# 3. Construct Command Array (SAFELY)
	local QEMU_CMD=(
		"$QEMU_BIN"
		-m "$QEMU_MEMORY"
		-smp "$QEMU_SMP"
		-kernel "$KERNEL_IMAGE"
		-initrd "$INITRAMFS_ARCHIVE"

		# Serial Console (Routes Linux output to terminal)
		-serial mon:stdio

		# Networking Device (VirtIO Net + User NAT)
		-device virtio-net-device,netdev=net0
		-netdev user,id=net0,hostfwd=tcp::2222-:22 # Host port 2222 -> Guest port 22

		# Architecture Specific Flags
		"${QEMU_ARCH_FLAGS[@]}"

		# Debug Flags (e.g., -s -S)
		$GDB_FLAGS
	)

	# Debug/Verbose handling for -nographic flag
	if [ "$DEBUG_MODE" != "verbose" ]; then
		QEMU_CMD+=(-nographic)
	fi

	# 4. Append Kernel Command Line (Boots directly from the prepared disk image)
	# root=/dev/vda is used because the entire disk was formatted (mkfs.ext4 "${DISK_IMAGE}").
	local KERNEL_CMDLINE="console=$QEMU_CONSOLE root=/dev/vda rw earlycon"
	QEMU_CMD+=(-append "$KERNEL_CMDLINE")

	# 5. Execute Safely
	echo -e "  [${YELLOW}CMD${NC}] Executing: ${QEMU_CMD[*]}"

	if "${QEMU_CMD[@]}"; then # Use quotes and @ to pass array elements safely
		echo -e "  [${GREEN}QEMU${NC}] Emulation session finished."
	else
		echo -e "  [${RED}FAIL${NC}] QEMU execution failed." >&2
		exit 1
	fi
}

# ─────────────────────────────────────────────────────────────
# 3. Public Functions (Called by run.sh)
# ─────────────────────────────────────────────────────────────
run_qemu() {
	_execute_qemu "" "$1"
}

run_qemu_gdb() {
	_execute_qemu "-s -S" "$1"
}
