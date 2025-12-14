#!/bin/bash
# scripts/branch.sh
# Handles Git branch and tag management (checkout, creation, deletion).

# ─────────────────────────────────────────────────────────────
# 1. Branch/Tag Checkout or Creation
# ─────────────────────────────────────────────────────────────
git_branch() {
	local branch_name="$1"

	if [[ -z "$branch_name" ]]; then
		echo -e "  [${RED}ERROR${NC}] Usage: branch <name|tag>"
		exit 1
	fi

	ensure_mounted
	cd "$KERNEL_DIR" || {
		echo -e "  [${RED}FAIL${NC}] Could not enter KERNEL_DIR."
		exit 1
	}

	echo -e "  [${YELLOW}GIT${NC}] Attempting to check out: ${branch_name}"

	# Check 1: Is it an official tag (e.g., v6.18)? (Matches old logic)
	if [[ "$branch_name" =~ ^v[0-9]+\.[0-9]+ ]]; then
		if git show-ref --tags --quiet "refs/tags/$branch_name"; then
			echo -e "  [${GREEN}FOUND${NC}] Official tag $branch_name found. Checking out..."
			git checkout "$branch_name" || {
				echo -e "  [${RED}FAIL${NC}] Checkout failed."
				exit 1
			}
			echo -e "  [${GREEN}SUCCESS${NC}] HEAD is now at tag: $branch_name"
			return 0
		else
			echo -e "  [${YELLOW}INFO${NC}] Tag $branch_name not found locally or remotely."
		fi
	fi

	# Check 2: Does the local branch exist? (Matches old logic)
	if git show-ref --quiet "refs/heads/$branch_name"; then
		echo -e "  [${GREEN}FOUND${NC}] Local branch $branch_name exists. Switching..."
		git checkout "$branch_name" || {
			echo -e "  [${RED}FAIL${NC}] Switching branch failed."
			exit 1
		}
		echo -e "  [${GREEN}SUCCESS${NC}] Switched to existing branch: $branch_name"
		return 0
	fi

	# Check 3: Final attempt, create new branch tracking origin/BASE_BRANCH (Matches old logic)
	echo -e "  [${YELLOW}INIT${NC}] Creating new branch '$branch_name' tracking origin/${BASE_BRANCH}..."
	if git checkout -b "$branch_name" --track "origin/$BASE_BRANCH"; then
		echo -e "  [${GREEN}SUCCESS${NC}] Created and switched to new branch: $branch_name"
	else
		echo -e "  [${RED}FAIL${NC}] Failed to create or switch to branch: $branch_name"
		exit 1
	fi
}

# ─────────────────────────────────────────────────────────────
# 2. Safe Local Branch Deletion
# ─────────────────────────────────────────────────────────────
delete_branch() {
	local branch_to_delete="$1"

	if [[ -z "$branch_to_delete" ]]; then
		echo -e "  [${RED}ERROR${NC}] Usage: delete <branch_name>"
		exit 1
	fi

	ensure_mounted
	cd "$KERNEL_DIR" || {
		echo -e "  [${RED}FAIL${NC}] Could not enter KERNEL_DIR."
		exit 1
	}

	echo -e "  [${YELLOW}WARNING${NC}] Attempting to delete local branch: $branch_to_delete"

	# Safety Check A: Refuse to delete core branches (Matches old logic)
	if [[ "$branch_to_delete" == "master" || "$branch_to_delete" == "main" || "$branch_to_delete" == "$BASE_BRANCH" ]]; then
		echo -e "  [${RED}FAIL${NC}] Cowardly refusing to delete core branch: '$branch_to_delete'"
		exit 1
	fi

	# Safety Check B: Must not be on the branch you're deleting (Matches old logic)
	CURRENT_BRANCH=$(git branch --show-current)
	if [[ "$CURRENT_BRANCH" == "$branch_to_delete" ]]; then
		echo -e "  [${RED}FAIL${NC}] You are currently on '$branch_to_delete'. Switch away first."
		exit 1
	fi

	# Check C: Does the local branch exist?
	if git show-ref --quiet "refs/heads/$branch_to_delete"; then
		# Use -D (force delete) to ensure it works even if unmerged changes exist.
		echo -e "  [${YELLOW}ACTION${NC}] Deleting local branch: $branch_to_delete (using -D to force)..."
		if git branch -D "$branch_to_delete"; then
			echo -e "  [${GREEN}SUCCESS${NC}] Deleted local branch: $branch_to_delete"
		else
			echo -e "  [${RED}FAIL${NC}] Failed to delete branch: $branch_to_delete"
			exit 1
		fi
	else
		echo -e "  [${YELLOW}INFO${NC}] Branch '$branch_to_delete' does not exist locally. Nothing to delete."
	fi
}
