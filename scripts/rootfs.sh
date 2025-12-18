#!/bin/bash
# scripts/rootfs.sh
# Handles creation of a Debian root filesystem on an ext4 disk image.
# Requires: ROOTFS_DIR, DISK_IMAGE, DEBOOTSTRAP_PATH, DEBIAN_MIRROR, TARGET_ARCH
# Uses mke2fs (from e2fsprogs) to populate the disk image directly.

# ─────────────────────────────────────────────────────────────
# Internal Helpers
# ─────────────────────────────────────────────────────────────
_is_stage1_ready() {
	[ -d "${ROOTFS_DIR}/debootstrap" ] && [ -f "${ROOTFS_DIR}/debootstrap/debootstrap" ]
}

_confirm_rebuild() {
	echo -e " [${YELLOW}INFO${NC}] $1 already exists."
	echo -n " [?] Do you want to rebuild it (y/N)? "
	read -r response
	[[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]
}

# ─────────────────────────────────────────────────────────────
# 1. Perform debootstrap stage 1 (foreign) and create smart /init
# ─────────────────────────────────────────────────────────────
prepare_rootfs_directory() {
	ensure_mounted

	# Map "riscv" to "riscv64" for debootstrap
	local arch="${TARGET_ARCH}"
	if [ "$TARGET_ARCH" = "riscv" ]; then
		arch="riscv64"
	fi

	if _is_stage1_ready && ! _confirm_rebuild "Rootfs directory (stage 1)"; then
		echo -e " [${YELLOW}SKIP${NC}] Using existing stage 1 rootfs."
		return 0
	fi

	echo -e " [${YELLOW}ROOTFS${NC}] Preparing Debian root filesystem (stage 1) for arch=${arch}..."

	# Clean old rootfs directory
	sudo rm -rf "${ROOTFS_DIR}"
	mkdir -p "${ROOTFS_DIR}"

	# Run debootstrap stage 1 (--foreign)
	echo -e " [${YELLOW}DEBOOTSTRAP${NC}] Running stage 1 (foreign)..."
	sudo DEBOOTSTRAP_DIR="${DEBOOTSTRAP_PATH}" fakeroot "${DEBOOTSTRAP_PATH}/debootstrap" \
		--foreign \
		--arch="${arch}" \
		--no-check-gpg \
		stable \
		"${ROOTFS_DIR}" \
		"${DEBIAN_MIRROR}"

	if [ $? -ne 0 ]; then
		echo -e " [${RED}FAIL${NC}] Debootstrap stage 1 failed."
		exit 1
	fi

	# Create smart /init script – runs stage 2 only once
	cat <<'EOF' >"${ROOTFS_DIR}/init"
#!/bin/sh

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

# Mount essential virtual filesystems
mount -t proc  proc  /proc
mount -t sysfs sys   /sys
mount -t devtmpfs dev /dev 2>/dev/null || mount -t tmpfs dev /dev
[ -d /dev/pts ] || mkdir /dev/pts
mount -t devpts devpts /dev/pts

echo "System ready."

# Drop to interactive shell
exec /bin/sh
EOF

	chmod +x "${ROOTFS_DIR}/init"
	echo -e " [${GREEN}SUCCESS${NC}] Stage 1 rootfs prepared with smart /init."
}

# ─────────────────────────────────────────────────────────────
# 2. Create and populate the ext4 disk image
# ─────────────────────────────────────────────────────────────
create_disk_image() {
	prepare_rootfs_directory

	if [ -f "${DISK_IMAGE}" ] && ! _confirm_rebuild "Disk image (${DISK_IMAGE})"; then
		echo -e " [${YELLOW}SKIP${NC}] Using existing disk image."
		return 0
	fi

	echo -e " [${YELLOW}DISK${NC}] Creating ext4 disk image (${DISK_SIZE})..."

	# Remove old image if rebuilding
	[ -f "${DISK_IMAGE}" ] && rm -f "${DISK_IMAGE}"

	# Use mke2fs to create and directly populate the image from ROOTFS_DIR
	mke2fs -t ext4 -E lazy_itable_init=0,lazy_journal_init=0 -d "${ROOTFS_DIR}" "${DISK_IMAGE}" "${DISK_SIZE}"

	if [ $? -eq 0 ]; then
		echo -e " [${GREEN}SUCCESS${NC}] Disk image created and populated: ${DISK_IMAGE}"
	else
		echo -e " [${RED}FAIL${NC}] Failed to create disk image. Ensure 'e2fsprogs' is installed via Homebrew."
		echo "   Run: brew install e2fsprogs"
		exit 1
	fi
}
