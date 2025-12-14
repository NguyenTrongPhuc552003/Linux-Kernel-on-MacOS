#!/bin/bash
# scripts/image.sh
# Handles the lifecycle of the APFS sparse image (Create, Mount, Unmount).

# ─────────────────────────────────────────────────────────────
# Internal Helper: Check mount status
# ─────────────────────────────────────────────────────────────
_is_mounted() {
	mount | grep -q "${MOUNT_POINT}"
}

# ─────────────────────────────────────────────────────────────
# 1. Mount Image (Creates if missing)
# ─────────────────────────────────────────────────────────────
mount_image() {
	# Check if already mounted
	if _is_mounted; then
		echo -e "  [${YELLOW}INFO${NC}] Volume already mounted at: ${MOUNT_POINT}"
		return 0
	fi

	# Check if image file exists, create if not
	if [ ! -f "${IMAGE_FILE}" ]; then
		echo -e "  [${YELLOW}INIT${NC}] Creating 20GB Case-Sensitive APFS Sparse Image..."
		# Linux kernel requires Case-Sensitive FS.
		# -type SPARSE ensures it only takes up space as used.
		hdiutil create -size 20g \
			-fs 'Case-sensitive APFS' \
			-type SPARSE \
			-volname "${VOLUME_NAME}" \
			"${IMAGE_FILE}" || {
			echo -e "  [${RED}FAIL${NC}] Could not create image."
			exit 1
		}
	fi

	# Attach the image
	echo -e "  [${YELLOW}DISK${NC}] Mounting ${VOLUME_NAME}..."
	if hdiutil attach "${IMAGE_FILE}" -quiet; then
		echo -e "  [${GREEN}SUCCESS${NC}] Mounted at ${MOUNT_POINT}"
	else
		echo -e "  [${RED}FAIL${NC}] hdiutil attach failed."
		exit 1
	fi
}

# ─────────────────────────────────────────────────────────────
# 2. Unmount Image
# ─────────────────────────────────────────────────────────────
unmount_image() {
	if ! _is_mounted; then
		echo -e "  [${YELLOW}INFO${NC}] Volume is not mounted."
		return 0
	fi

	echo -e "  [${YELLOW}DISK${NC}] Unmounting ${MOUNT_POINT}..."
	# -force is used to ensure we don't get stuck if a shell is open inside
	if hdiutil detach "${MOUNT_POINT}" -force; then
		echo -e "  [${GREEN}SUCCESS${NC}] Unmounted successfully."
	else
		echo -e "  [${RED}FAIL${NC}] Could not unmount. Is a process holding it?"
		hdiutil info | grep "${VOLUME_NAME}"
		exit 1
	fi
}

# ─────────────────────────────────────────────────────────────
# 3. Guard: Ensure Mounted
# Use this at the start of other scripts (repo.sh, build.sh)
# ─────────────────────────────────────────────────────────────
ensure_mounted() {
	if ! _is_mounted; then
		echo -e "  [${RED}ERROR${NC}] Kernel volume not mounted."
		echo "  Run './run.sh' (no args) or './run.sh mount' first."
		exit 1
	fi
}
