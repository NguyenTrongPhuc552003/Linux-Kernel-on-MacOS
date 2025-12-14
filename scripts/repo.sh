#!/bin/bash
# scripts/repo.sh
# Handles Git repository operations (clone, status, update, reset, reinit).

# Requires: SCRIPT_DIR, MOUNT_POINT, KERNEL_DIR, BASE_BRANCH from common.env
# Requires: ensure_mounted() from image.sh

# ─────────────────────────────────────────────────────────────
# 1. Clone/Check Repository (Initial Setup)
# ─────────────────────────────────────────────────────────────
check_repo() {
	# Ensure the volume is available before checking paths inside it
	ensure_mounted

	if [ ! -d "${KERNEL_DIR}/.git" ]; then
		echo -e "  [${YELLOW}INFO${NC}] Kernel repository not found."
		echo -e "  [${YELLOW}GIT${NC}] Cloning mainline Linux kernel into ${KERNEL_DIR}..."
		# Use shallow clone (depth 1) for speed if we always reset to master,
		# but full clone is safer for history/tags/branches. Sticking to full clone.
		local REPO_URL="https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git"

		git clone "$REPO_URL" "${KERNEL_DIR}" || {
			echo -e "  [${RED}FAIL${NC}] Git clone failed. Check connectivity or disk space."
			exit 1
		}
		echo -e "  [${GREEN}SUCCESS${NC}] Repository cloned."
	else
		echo -e "  [${YELLOW}INFO${NC}] Kernel repository already exists at ${KERNEL_DIR}."
	fi
}

# ─────────────────────────────────────────────────────────────
# 2. Status
# ─────────────────────────────────────────────────────────────
repo_status() {
	ensure_mounted

	echo -e "  [${YELLOW}GIT${NC}] Showing status in ${KERNEL_DIR}"
	cd "$KERNEL_DIR"
	git status
}

# ─────────────────────────────────────────────────────────────
# 3. Reinitialize (Delete and Re-clone)
# ─────────────────────────────────────────────────────────────
repo_reinit() {
	ensure_mounted

	echo -e "  [${YELLOW}WARNING${NC}] This will DELETE the entire kernel tree and RE-CLONE."
	read -r -p "Continue? (Y/n): " ans
	[[ "$ans" == "n" || "$ans" == "N" ]] && echo -e "  [ABORT] Aborted." && exit 0

	echo -e "  [${YELLOW}FS${NC}] Removing directory: ${KERNEL_DIR}"
	rm -rf "$KERNEL_DIR"

	# Re-clone the repository using the same logic
	check_repo
}

# ─────────────────────────────────────────────────────────────
# 4. Update (Fetch and Hard Reset to Base Branch)
# ─────────────────────────────────────────────────────────────
repo_update() {
	ensure_mounted

	echo -e "  [${YELLOW}GIT${NC}] Fetching latest changes and updating to origin/${BASE_BRANCH}..."
	cd "$KERNEL_DIR"

	# Fetch updates from all remotes (usually 'origin')
	git fetch origin || {
		echo -e "  [${RED}FAIL${NC}] git fetch failed."
		exit 1
	}

	# Hard reset to the base branch (default: master)
	git reset --hard "origin/$BASE_BRANCH"

	# Clean untracked files and directories
	git clean -fd
	echo -e "  [${GREEN}SUCCESS${NC}] Repository is clean and synchronized with origin/${BASE_BRANCH}."
}

# ─────────────────────────────────────────────────────────────
# 5. Reset Local Changes (without fetching)
# ─────────────────────────────────────────────────────────────
repo_reset() {
	ensure_mounted

	echo -e "  [${YELLOW}WARNING${NC}] This will discard all local changes and reset to origin/${BASE_BRANCH}."
	read -r -p "Continue? (Y/n): " ans
	[[ "$ans" == "n" || "$ans" == "N" ]] && echo -e "  [ABORT] Aborted." && exit 0

	cd "$KERNEL_DIR"

	# Discard changes and reset HEAD
	git reset --hard "origin/$BASE_BRANCH" || {
		echo -e "  [${RED}FAIL${NC}] Git reset failed."
		exit 1
	}

	# Clean untracked files and directories
	git clean -fd
	echo -e "  [${GREEN}SUCCESS${NC}] Local changes discarded."
}
