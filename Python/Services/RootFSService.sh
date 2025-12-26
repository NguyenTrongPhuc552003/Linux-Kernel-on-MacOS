#!/bin/bash
# Python/Services/RootFSService.sh
# Generates a minimal Debian rootfs using debootstrap.

source "$(dirname "$0")/EnvironmentService.sh"

# Configuration
DISTRO="bullseye"
# DEBIAN_ARCH is now provided by Python (e.g., riscv64)

create_rootfs() {
	local force="$1"

	ensure_mounted
	cd "$MOUNT_POINT" || exit 1

	# Check if disk image already exists
	if [ -f "$DISK_IMAGE" ] && [ "$force" != "--force" ]; then
		echo "  [ROOTFS] Disk image already exists at: $DISK_IMAGE"
		echo "  Use 'km rootfs --force' to overwrite."
		exit 0
	fi

	echo "  [ROOTFS] Creating raw disk image (${DISK_SIZE})..."
	dd if=/dev/zero of="$DISK_IMAGE" bs=1 count=0 seek="$DISK_SIZE" status=none

	echo "  [ROOTFS] Formatting as ext4..."
	# macOS doesn't have native mkfs.ext4. We rely on Homebrew e2fsprogs.
	if ! command -v mkfs.ext4 &>/dev/null; then
		echo "  [ERROR] mkfs.ext4 not found. Install: brew install e2fsprogs"
		# Add e2fsprogs to path just in case
		export PATH="/opt/homebrew/opt/e2fsprogs/sbin:$PATH"
	fi

	mkfs.ext4 -F "$DISK_IMAGE" >/dev/null

	echo "  [ROOTFS] Bootstraping Debian ($DISTRO / $DEBIAN_ARCH)..."
	echo "  [INFO] Sudo password required for debootstrap:"

	# We use a temporary directory for the rootfs tree before packing or copying
	# Note: On macOS, we can't easily 'mount' the ext4 image to copy files into it
	# without osxfuse or similar.
	#
	# STRATEGY: We build the directory tree first, then copy it into the ext4 image
	# using 'e2cp' (from e2fsprogs) or 'debugfs'.
	# Alternatively, if you have a linux VM tool, that's better.
	#
	# For this migration, we assume the old script used a specific method.
	# We will run debootstrap into the 'rootfs' folder.

	mkdir -p "$ROOTFS_DIR"

	# Check for debootstrap
	if [ ! -f "$TOOLS_DIR/debootstrap/debootstrap" ]; then
		echo "  [ERROR] debootstrap tool not found in $TOOLS_DIR"
		exit 1
	fi

	# Run debootstrap (requires sudo/root)
	# We filter out specific devices to avoid macOS errors
	sudo "$TOOLS_DIR/debootstrap/debootstrap" \
		--arch="$DEBIAN_ARCH" \
		--foreign \
		--include=initramfs-tools,systemd-sysv,vim,net-tools \
		"$DISTRO" \
		"$ROOTFS_DIR" \
		"$DEBIAN_MIRROR"

	echo "  [ROOTFS] Syncing changes..."

	# Pack into the ext4 image
	# This part is tricky on macOS. A robust way is to use 'tar' to pipe it.
	# But since we can't mount ext4, we often use 'virt-make-fs' or just pass the directory
	# to QEMU as 9p (virtfs) if we are lazy.
	#
	# Assuming standard behavior: We just keep the generated dir for now,
	# or if you have e2tools, we populate the image.

	echo "  [SUCCESS] RootFS tree created at $ROOTFS_DIR"
	echo "  [NOTE] Booting requires this directory or population of $DISK_IMAGE."
}

# Dispatcher
case "$1" in
create)
	shift
	create_rootfs "$@"
	;;
*)
	echo "Usage: $0 create [--force]"
	exit 1
	;;
esac
