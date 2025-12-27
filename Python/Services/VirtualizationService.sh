#!/bin/bash
# Python/Services/VirtualizationService.sh
# Handles QEMU execution (Server) and GDB connection (Client).

source "$(dirname "$0")/EnvironmentService.sh"

# ─────────────────────────────────────────────────────────────
# 1. Run QEMU (Server)
# ─────────────────────────────────────────────────────────────
run_qemu() {
	ensure_mounted

	# Defaults
	: "${QEMU_BIN:=qemu-system-riscv64}"
	: "${QEMU_FLAGS:=-M virt -m 2G}"

	# Locate Kernel
	local kernel_img=""
	if [ -f "$KERNEL_DIR/arch/$TARGET_ARCH/boot/Image" ]; then
		kernel_img="$KERNEL_DIR/arch/$TARGET_ARCH/boot/Image"
	elif [ -f "$KERNEL_DIR/arch/$TARGET_ARCH/boot/zImage" ]; then
		kernel_img="$KERNEL_DIR/arch/$TARGET_ARCH/boot/zImage"
	elif [ -f "$KERNEL_DIR/arch/$TARGET_ARCH/boot/Image.gz" ]; then
		kernel_img="$KERNEL_DIR/arch/$TARGET_ARCH/boot/Image.gz"
	else
		echo "  [ERROR] Kernel image not found in arch/$TARGET_ARCH/boot/"
		exit 1
	fi

	# Parse Arguments
	local use_gui="no"
	local debug_flags=""

	for arg in "$@"; do
		case $arg in
		--gui) use_gui="yes" ;;
		--nographic) use_gui="no" ;;
		--debug)
			echo "  [QEMU] Debug Stub Active: tcp::1234 (Waiting for connection...)"
			debug_flags="-S -s"
			;;
		esac
	done

	# Console Setup
	local display_args=""
	local console_args=""

	if [ "$use_gui" == "yes" ]; then
		display_args="-display cocoa -device virtio-gpu-pci -device virtio-keyboard-pci -device virtio-mouse-pci"
		console_args="console=tty0"
		echo "  [QEMU] Mode: Graphical (Cocoa)"
	else
		display_args="-nographic"
		case "$TARGET_ARCH" in
		riscv) console_args="console=ttyS0" ;;
		arm64) console_args="console=ttyAMA0" ;;
		arm) console_args="console=ttyAMA0" ;;
		*) console_args="console=ttyS0" ;;
		esac
		echo "  [QEMU] Mode: Console/Nographic"
	fi

	# Kernel Boot Args (init=/init ensures our custom script runs)
	local append_cmd="root=/dev/vda rw init=/init earlycon $console_args"

	echo "  [QEMU] Network: localhost:2222 -> Guest:22"
	echo "  [QEMU] Modules: /mnt/modules"

	# Execute
	$QEMU_BIN $QEMU_FLAGS \
		-kernel "$kernel_img" \
		-append "$append_cmd" \
		-drive file="$DISK_IMAGE",format=raw,id=hd0,if=none \
		-device virtio-blk-device,drive=hd0 \
		-device virtio-net-device,netdev=net0 \
		-netdev user,id=net0,hostfwd=tcp::2222-:22 \
		-fsdev local,id=moddev,path="$MODULES_DIR",security_model=none \
		-device virtio-9p-pci,fsdev=moddev,mount_tag=modules_mount \
		$display_args \
		$debug_flags
}

# ─────────────────────────────────────────────────────────────
# 2. Run GDB (Client)
# ─────────────────────────────────────────────────────────────
run_client() {
	ensure_mounted

	local gdb_bin=""
	# Map Architecture to GDB Binary (Matches Legacy)
	case "$TARGET_ARCH" in
	riscv) gdb_bin="riscv64-elf-gdb" ;;
	arm64) gdb_bin="aarch64-elf-gdb" ;;
	arm) gdb_bin="arm-none-eabi-gdb" ;;
	*)
		echo "  [ERROR] Unsupported architecture for GDB: $TARGET_ARCH"
		exit 1
		;;
	esac

	if ! command -v "$gdb_bin" >/dev/null; then
		echo "  [ERROR] GDB binary '$gdb_bin' not found."
		echo "  Try: brew install $gdb_bin"
		exit 1
	fi

	local vmlinux="$KERNEL_DIR/vmlinux"
	if [ ! -f "$vmlinux" ]; then
		echo "  [ERROR] vmlinux symbol file not found."
		echo "  Run 'km build' first."
		exit 1
	fi

	echo "  [GDB] Connecting to localhost:1234 using $gdb_bin..."

	# Execute GDB with auto-connect commands
	exec "$gdb_bin" "$vmlinux" \
		-ex "target remote localhost:1234" \
		-ex "layout src" \
		-ex "break start_kernel"
}

# Dispatcher
case "$1" in
client)
	run_client
	;;
run)
	shift
	run_qemu "$@"
	;;
*)
	# Default behavior if called without 'run' (legacy support)
	run_qemu "$@"
	;;
esac
