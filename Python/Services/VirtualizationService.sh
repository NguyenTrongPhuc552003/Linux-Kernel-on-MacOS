#!/bin/bash
# Python/Services/VirtualizationService.sh
# Handles QEMU execution.

source "$(dirname "$0")/EnvironmentService.sh"

run_qemu() {
	ensure_mounted

	# Defaults provided by Python, but fallback if run standalone
	: "${QEMU_BIN:=qemu-system-riscv64}"
	: "${QEMU_FLAGS:=-M virt -m 2G}"

	# Kernel Image Path
	# Note: different arches have different image names (Image vs Image.gz)
	# Python strategies know this, but for now we look for 'Image' or 'Image.gz'
	local kernel_img=""
	if [ -f "$KERNEL_DIR/arch/$TARGET_ARCH/boot/Image" ]; then
		kernel_img="$KERNEL_DIR/arch/$TARGET_ARCH/boot/Image"
	elif [ -f "$KERNEL_DIR/arch/$TARGET_ARCH/boot/Image.gz" ]; then
		kernel_img="$KERNEL_DIR/arch/$TARGET_ARCH/boot/Image.gz"
	else
		echo "  [ERROR] Kernel image not found in arch/$TARGET_ARCH/boot/"
		exit 1
	fi

	# Parse Flags
	local extra_flags=""
	local append_cmd="root=/dev/vda rw console=ttyS0 earlycon"

	for arg in "$@"; do
		case $arg in
		--debug)
			echo "  [QEMU] Debug mode enabled. Waiting for GDB on tcp::1234..."
			extra_flags="$extra_flags -S -s"
			;;
		--nographic)
			extra_flags="$extra_flags -nographic"
			;;
		esac
	done

	echo "  [QEMU] Binary: $QEMU_BIN"
	echo "  [QEMU] Kernel: $kernel_img"

	# Execute
	# We use $QEMU_FLAGS (from Python) and $extra_flags (from args)
	$QEMU_BIN $QEMU_FLAGS \
		-kernel "$kernel_img" \
		-drive file="$DISK_IMAGE",format=raw,id=hd0,if=none \
		-device virtio-blk-device,drive=hd0 \
		-append "$append_cmd" \
		$extra_flags
}

# Dispatcher
case "$1" in
run)
	shift
	run_qemu "$@"
	;;
*)
	# Default behavior if called without 'run' (legacy support)
	run_qemu "$@"
	;;
esac
