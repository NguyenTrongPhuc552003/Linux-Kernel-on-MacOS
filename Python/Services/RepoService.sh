#!/bin/bash
# Python/Services/RepoService.sh
# Manages git operations: status, updates, branching, and resetting.

source "$(dirname "$0")/EnvironmentService.sh"

# ─────────────────────────────────────────────────────────────
# 1. Repository Status & Health
# ─────────────────────────────────────────────────────────────
repo_status() {
	ensure_mounted
	cd "$KERNEL_DIR" || return 1

	# Handle detached HEAD state gracefully
	local branch_name=$(git branch --show-current)
	[ -z "$branch_name" ] && branch_name="(detached HEAD at $(git rev-parse --short HEAD))"

	echo "  [GIT] Branch: $branch_name"
	echo "  [GIT] Last Commit: $(git log -1 --format='%h - %s (%cr)')"
	echo "  [GIT] Status:"
	git status -s | sed 's/^/    /'
}

# ─────────────────────────────────────────────────────────────
# 2. Updates & Resets
# ─────────────────────────────────────────────────────────────
repo_update() {
	ensure_mounted
	cd "$KERNEL_DIR" || exit 1

	local branch_name=$(git branch --show-current)
	if [ -z "$branch_name" ]; then
		echo "  [GIT] Detached HEAD. Fetching all..."
		git fetch --all
		return
	fi

	# Handle tag-based branches
	if [[ "$branch_name" == tag-* ]]; then
		echo "  [GIT] Current branch '$branch_name' is based on a tag."
		echo "  [GIT] Tags are static. Fetching remote updates only..."
		git fetch --tags origin
	else
		echo "  [GIT] Pulling latest changes for $branch_name..."
		git pull origin "$branch_name"
	fi
}

repo_reset() {
	ensure_mounted
	cd "$KERNEL_DIR" || exit 1
	echo "  [GIT] Hard resetting to HEAD..."
	git reset --hard HEAD
	git clean -fd
}

# ─────────────────────────────────────────────────────────────
# 3. Branch Management
# ─────────────────────────────────────────────────────────────
git_branch() {
	local target="$1"
	[ -z "$target" ] && echo "Usage: branch <name>" && return 1

	ensure_mounted
	cd "$KERNEL_DIR" || exit 1

	# Case A: It is an existing local branch
	if git show-ref --verify --quiet "refs/heads/$target"; then
		echo "  [GIT] Switching to local branch: $target"
		git checkout "$target"
		return
	fi

	# Case B: It is a Remote Branch (but not local yet)
	if git show-ref --verify --quiet "refs/remotes/origin/$target"; then
		echo "  [GIT] Tracking remote branch: $target"
		git checkout -b "$target" "origin/$target"
		return
	fi

	# Case C: It is a Tag (e.g., v6.12)
	# We explicitly check tags and create a branch named "tag-<version>"
	if git show-ref --tags --quiet "refs/tags/$target"; then
		local new_branch="tag-$target"

		# If the branch "tag-v6.12" already exists, switch to it
		if git show-ref --verify --quiet "refs/heads/$new_branch"; then
			echo "  [GIT] Switching to existing tag-branch: $new_branch"
			git checkout "$new_branch"
		else
			echo "  [GIT] Creating new branch from tag: $target -> $new_branch"
			git checkout -b "$new_branch" "refs/tags/$target"
		fi
		return
	fi

	# Case D: It doesn't exist anywhere -> Create new branch from current HEAD
	echo "  [GIT] Creating new branch: $target"
	git checkout -b "$target"
}

delete_branch() {
	local branch="$1"
	ensure_mounted
	cd "$KERNEL_DIR" || exit 1

	if [ "$branch" == "master" ]; then
		echo "  [ERROR] Cannot delete master."
		return 1
	fi
	git branch -D "$branch"
}

# Dispatcher
case "$1" in
status) repo_status ;;
update) repo_update ;;
reset) repo_reset ;;
branch)
	shift
	git_branch "$@"
	;;
delete)
	shift
	delete_branch "$@"
	;;
esac
