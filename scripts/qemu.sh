#!/bin/bash
# scripts/qemu.sh
# Handles QEMU execution for a disk-based Linux kernel (ext4 rootfs on disk.img).
# Requires: KERNEL_DIR, TARGET_ARCH, DISK_IMAGE (from common.env)

# ─────────────────────────────────────────────────────────────
# Configurable constants
# ─────────────────────────────────────────────────────────────
QEMU_GDB_PORT=1234 # Port for GDB stub

# ─────────────────────────────────────────────────────────────
# 1. Architecture-specific QEMU configuration
# ─────────────────────────────────────────────────────────────
_configure_qemu_for_arch() {
	# Default settings
	QEMU_MEMORY="2G"
	QEMU_SMP="$(sysctl -n hw.logicalcpu)"

	case "$TARGET_ARCH" in
	riscv)
		QEMU_BIN="qemu-system-riscv64"
		QEMU_MACHINE="virt"
		QEMU_CPU="rv64"
		QEMU_CONSOLE="ttyS0"
		QEMU_BIOS="-bios default" # Uses built-in OpenSBI
		;;
	arm64)
		QEMU_BIN="qemu-system-aarch64"
		QEMU_MACHINE="virt"
		QEMU_CPU="cortex-a72" # Reliable and widely supported
		QEMU_CONSOLE="ttyAMA0"
		QEMU_BIOS="" # EFI not needed for direct kernel boot on virt
		;;
	*)
		echo -e "  [${RED}ERROR${NC}] Unsupported TARGET_ARCH for QEMU: ${TARGET_ARCH}" >&2
		exit 1
		;;
	esac
}

# ─────────────────────────────────────────────────────────────
# 2. Core execution logic
# ─────────────────────────────────────────────────────────────
_execute_qemu() {
	local GDB_FLAGS="$1"    # e.g., "-s -S" for gdb
	local VERBOSE_MODE="$2" # if "verbose", skip -nographic

	_configure_qemu_for_arch

	local KERNEL_IMAGE="${KERNEL_DIR}/arch/${TARGET_ARCH}/boot/Image"

	# Validation
	if ! command -v "$QEMU_BIN" >/dev/null 2>&1; then
		echo -e "  [${RED}ERROR${NC}] $QEMU_BIN not found. Install QEMU via Homebrew: brew install qemu" >&2
		exit 1
	fi

	if [ ! -f "$KERNEL_IMAGE" ]; then
		echo -e "  [${RED}ERROR${NC}] Kernel Image not found: $KERNEL_IMAGE"
		echo "  Run './run.sh build' first."
		exit 1
	fi

	if [ ! -f "$DISK_IMAGE" ]; then
		echo -e "  [${RED}ERROR${NC}] Disk image not found: $DISK_IMAGE"
		echo "  Run './run.sh rootfs' first to create the root filesystem."
		exit 1
	fi

	echo -e "  [${YELLOW}QEMU${NC}] Starting ${GREEN}${TARGET_ARCH}${NC} emulation with disk-based rootfs..."
	[ -n "$GDB_FLAGS" ] && echo -e "  [${YELLOW}DEBUG${NC}] GDB stub enabled (-s -S). Connect with: gdb -ex \"target remote localhost:${QEMU_GDB_PORT}\""

	# Base command
	local QEMU_CMD=(
		"$QEMU_BIN"
		-m "$QEMU_MEMORY"
		-smp "$QEMU_SMP"
		-kernel "$KERNEL_IMAGE"
		$QEMU_BIOS
		-machine "$QEMU_MACHINE"
		${QEMU_CPU:+-cpu "$QEMU_CPU"}

		# Disk: virtio-blk with our ext4 image
		-drive file="${DISK_IMAGE}",format=raw,if=virtio

		# Basic GPU (virtio-gpu-pci)
		-device virtio-gpu-pci

		# Networking: user-mode + SSH forwarding (host:2222 -> guest:22)
		-device virtio-net-device,netdev=net0
		-netdev user,id=net0,hostfwd=tcp::2222-:22

		# Serial console
		-serial mon:stdio
	)

	# Graphics mode
	if [ "$VERBOSE_MODE" != "verbose" ]; then
		QEMU_CMD+=(-nographic)
	fi

	# GDB support
	[ -n "$GDB_FLAGS" ] && QEMU_CMD+=($GDB_FLAGS)

	# Kernel command line: critical for disk boot
	local KERNEL_CMDLINE="console=${QEMU_CONSOLE} root=/dev/vda rw init=/init earlycon"
	QEMU_CMD+=(-append "$KERNEL_CMDLINE")

	# Show and execute
	echo -e "  [${YELLOW}CMD${NC}] ${QEMU_CMD[*]}"
	"${QEMU_CMD[@]}"
}

# ─────────────────────────────────────────────────────────────
# 3. Unified QEMU runner – handles debug and verbose modes
# ─────────────────────────────────────────────────────────────
run_qemu() {
	local debug_mode=""
	local verbose_mode=""

	while [ $# -gt 0 ]; do
		case "$1" in
		-d | --debug)
			debug_mode="yes"
			shift
			;;
		-v | --verbose)
			verbose_mode="yes"
			shift
			;;
		-h | --help)
			cat <<EOF
${GREEN}QEMU Launcher Help${NC}

Usage: ./run.sh qemu [options]

Options:
  -d      	 Start QEMU in debug mode
			 > Opens GDB stub on tcp::${QEMU_GDB_PORT}
			 > Pauses CPU at startup (-S flag)
			 > Connect with: riscv64-unknown-linux-gnu-gdb vmlinux
					 (gdb) target remote localhost:${QEMU_GDB_PORT}
					 (gdb) continue

  -v, --verbose  Disable -nographic → shows QEMU graphical window
            	 (useful for virtio-gpu testing or future framebuffer console)

  -h, --help     Show this help message

Examples:
  ./run.sh qemu              Normal console boot
  ./run.sh qemu -d           Debug mode (paused, waiting for GDB)
  ./run.sh qemu -verbose     Boot with graphical window
  ./run.sh qemu -d -verbose  Debug mode + graphical window
  ./run.sh qemu -h           Show this help

EOF
			return 0
			;;
		*)
			echo -e "  [${RED}ERROR${NC}] Unknown argument: $1"
			echo "  Usage: ./run.sh qemu [-h] [-d] [-verbose]"
			return 1
			;;
		esac
	done

	if [ -n "$debug_mode" ]; then
		echo -e " [${YELLOW}QEMU${NC}] Starting in DEBUG mode (GDB stub on port $QEMU_GDB_PORT, CPU paused)"
		_execute_qemu "-s -S" "$verbose_mode"
	else
		_execute_qemu "" "$verbose_mode"
	fi
}
