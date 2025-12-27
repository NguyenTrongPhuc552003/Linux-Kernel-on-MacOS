#!/bin/bash
# Python/Services/RootFSService.sh
# Handles creation of a Debian root filesystem on an ext4 disk image.

source "$(dirname "$0")/EnvironmentService.sh"
DEBIAN_MIRROR="http://deb.debian.org/debian"

_confirm_rebuild() {
	[ "$FORCE_REBUILD" == "true" ] && return 0
	echo -e "  [INFO] $1 already exists."
	echo -n "  [?] Do you want to rebuild it (y/N)? "
	read -r response
	[[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]
}

_is_stage1_ready() {
	[ -d "${ROOTFS_DIR}/debootstrap" ] && [ -f "${ROOTFS_DIR}/debootstrap/debootstrap" ]
}

create_rootfs() {
	[ "$1" == "--force" ] && FORCE_REBUILD="true" || FORCE_REBUILD="false"

	ensure_mounted
	cd "$MOUNT_POINT" || exit 1

	local deb_arch="${TARGET_ARCH}"
	[ "$TARGET_ARCH" = "riscv" ] && deb_arch="riscv64"
	[ "$TARGET_ARCH" = "arm" ] && deb_arch="armhf"

	# ─────────────────────────────────────────────────────────────
	# Stage 1: Debootstrap (Legacy Logic)
	# ─────────────────────────────────────────────────────────────
	local rebuild_stage1="yes"
	if _is_stage1_ready; then
		if ! _confirm_rebuild "Rootfs directory (stage 1)"; then
			echo "  [SKIP] Using existing stage 1 rootfs."
			rebuild_stage1="no"
		fi
	fi

	if [ "$rebuild_stage1" == "yes" ]; then
		echo "  [ROOTFS] Preparing Debian root filesystem (stage 1) for arch=${deb_arch}..."
		[ -d "$ROOTFS_DIR" ] && sudo rm -rf "$ROOTFS_DIR"
		mkdir -p "$ROOTFS_DIR"

		echo "  [DEBOOTSTRAP] Running stage 1 (foreign)..."
		sudo DEBOOTSTRAP_DIR="$TOOLS_DIR/debootstrap" fakeroot "$TOOLS_DIR/debootstrap/debootstrap" \
			--foreign \
			--arch="${deb_arch}" \
			--no-check-gpg \
			stable \
			"$ROOTFS_DIR" \
			"$DEBIAN_MIRROR"

		if [ $? -ne 0 ]; then
			echo "  [FAIL] Debootstrap stage 1 failed."
			exit 1
		fi

		_install_init_script
	fi

	# ─────────────────────────────────────────────────────────────
	# Stage 2: Disk Image
	# ─────────────────────────────────────────────────────────────
	local rebuild_image="yes"
	if [ -f "$DISK_IMAGE" ]; then
		if ! _confirm_rebuild "Disk image (${DISK_IMAGE})"; then
			echo "  [SKIP] Using existing disk image."
			rebuild_image="no"
		fi
	fi

	if [ "$rebuild_image" == "yes" ]; then
		echo "  [DISK] Creating ext4 disk image (${DISK_SIZE})..."
		rm -f "$DISK_IMAGE"

		if ! command -v mke2fs &>/dev/null; then
			export PATH="/opt/homebrew/opt/e2fsprogs/sbin:$PATH"
		fi

		mke2fs -t ext4 \
			-E lazy_itable_init=0,lazy_journal_init=0 \
			-d "$ROOTFS_DIR" \
			"$DISK_IMAGE" \
			"$DISK_SIZE" >/dev/null

		[ $? -eq 0 ] && echo "  [SUCCESS] Disk image created: $DISK_IMAGE" || exit 1
	fi
}

_install_init_script() {
	echo "  [ROOTFS] Installing custom /init script..."
	cat <<'EOF' >"${ROOTFS_DIR}/init"
#!/bin/sh
export PATH=/sbin:/usr/sbin:/bin:/usr/bin
MARKER="/.rootfs-setup-complete"

echo "Booting Debian root filesystem..."

if [ ! -f "$MARKER" ]; then
    echo "First boot detected – running debootstrap second stage..."
    /debootstrap/debootstrap --second-stage
    if [ $? -eq 0 ]; then
        touch "$MARKER"
        echo "Second stage completed successfully."
    else
        echo "Second stage failed – dropping to emergency shell."
        exec /bin/sh
    fi
else
    echo "Root filesystem already set up."
fi

# Mounts
mount -t proc  proc  /proc
mount -t sysfs sys   /sys
mount -t devtmpfs dev /dev 2>/dev/null || mount -t tmpfs dev /dev
[ -d /dev/pts ] || mkdir /dev/pts
mount -t devpts devpts /dev/pts

# Network
echo "Configuring network..."
ip link set lo up
ip link set eth0 up
ip addr add 10.0.2.15/24 dev eth0
ip route add default via 10.0.2.2
echo "nameserver 8.8.8.8" > /etc/resolv.conf

# Modules
mkdir -p /mnt/modules
mount -t 9p -o trans=virtio,version=9p2000.L modules_mount /mnt/modules

if [ -f /mnt/modules/guesync.sh ]; then
    echo "Running guest module synchronization script..."
    /mnt/modules/guesync.sh
fi

echo "System ready."
exec /bin/sh
EOF
	chmod +x "${ROOTFS_DIR}/init"
}

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
