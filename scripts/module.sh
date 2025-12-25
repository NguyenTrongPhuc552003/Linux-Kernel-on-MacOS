#!/bin/bash
# scripts/module.sh
# Management script for out-of-tree kernel modules.
# Handles building, cleaning, and queuing for QEMU integration.

# Path to persist module queue state
MOD_CONFIG="${MODULES_DIR}/module.cfg"

# Load existing state if it exists
if [ -f "$MOD_CONFIG" ]; then
	source "$MOD_CONFIG"
	# Convert the space-separated strings from the file into Bash arrays
	MODULE_INS=($MODULE_INS)
	MODULE_REM=($MODULE_REM)
else
	# Initialize empty arrays if no config exists
	MODULE_INS=()
	MODULE_REM=()
fi

# ─────────────────────────────────────────────────────────────
# Internal: Persist Array State
# ─────────────────────────────────────────────────────────────
_save_mod_state() {
	{
		echo "# Auto-generated module state"
		# Convert arrays to space-separated strings
		echo "MODULE_INS=\"${MODULE_INS[*]}\""
		echo "MODULE_REM=\"${MODULE_REM[*]}\""
	} >"$MOD_CONFIG"
}

# ─────────────────────────────────────────────────────────────
# Internal: Check if queue contains element
# ─────────────────────────────────────────────────────────────
_queue_contains() {
	local e match="$1"
	shift
	for e; do [[ "$e" == "$match" ]] && return 0; done
	return 1
}

# ─────────────────────────────────────────────────────────────
# Internal: Manage the state of module queues
# ─────────────────────────────────────────────────────────────
_update_queue() {
	local mode="$1" target="$2" targets=()

	# 1. Expand targets: if empty or "*", find all directories with a Makefile
	if [ -z "$target" ] || [ "$target" = "*" ]; then
		for d in "${MODULES_DIR}"/*/; do
			[ -f "${d}Makefile" ] && targets+=("$(basename "$d")")
		done
	else
		targets=("$target")
	fi

	for item in "${targets[@]}"; do
		local mod_path="${MODULES_DIR}/${item}"
		local ko_file="${mod_path}/${item}.ko"

		if [ "$mode" = "insmod" ]; then
			# Critical check: .ko must exist to queue for insmod
			if [ ! -f "$ko_file" ]; then
				echo -e "  [${RED}SKIP${NC}] Cannot queue '$item' for insmod: .ko file not found"
				echo "  → Run './run.sh module $item' to build it first."
				continue
			fi

			if _queue_contains "$item" "${MODULE_INS[@]}"; then
				echo -e "  [${YELLOW}INFO${NC}] Already queued for insmod: $item"
				continue
			fi

			MODULE_INS+=("$item")
			# Remove from REM queue if present
			local tmp=()
			for m in "${MODULE_REM[@]}"; do [[ "$m" != "$item" ]] && tmp+=("$m"); done
			MODULE_REM=("${tmp[@]}")
			echo -e "  [${GREEN}+${NC}] Queued for insmod: $item"

		else # rmmod mode
			if _queue_contains "$item" "${MODULE_REM[@]}"; then
				echo -e "  [${YELLOW}INFO${NC}] Already queued for rmmod: $item"
				continue
			fi

			MODULE_REM+=("$item")
			# Remove from INS queue if present
			local tmp=()
			for m in "${MODULE_INS[@]}"; do [[ "$m" != "$item" ]] && tmp+=("$m"); done
			MODULE_INS=("${tmp[@]}")
			echo -e "  [${RED}-${NC}] Queued for rmmod: $item"
		fi
	done
	_save_mod_state
}

# ─────────────────────────────────────────────────────────────
# 1. Build Logic (-b / default)
# ─────────────────────────────────────────────────────────────
_module_item_action() {
	local mod_name="$1"
	local mod_path="${MODULES_DIR}/${mod_name}"
	local mod_target="${2:-modules}" # Default target is 'modules'

	[ ! -d "$mod_path" ] && {
		echo -e "  [${RED}ERR${NC}] Module directory not found: $mod_name"
		return 1
	}

	echo -e "  [${YELLOW}BUILD${NC}] Compiling: ${GREEN}${mod_name}${NC}"

	# We use the Kernel Kbuild system.
	# M= points to the module source, -C points to the kernel source.
	make -C "$KERNEL_DIR" \
		M="$mod_path" \
		ARCH="$TARGET_ARCH" \
		LLVM=1 \
		CROSS_COMPILE="$CROSS_COMPILE" \
		HOSTCFLAGS="$HOSTCFLAGS" \
		"$mod_target"
}

# ─────────────────────────────────────────────────────────────
# 2. Information Logic (-f)
# ─────────────────────────────────────────────────────────────
# Design: Scans the source code for MODULE_ macros to provide host-side modinfo.
_module_info() {
	local target_mod="$1"

	if [ -n "$target_mod" ]; then
		# Individual module view
		local src_file="${MODULES_DIR}/${target_mod}/${target_mod}.c"
		[ ! -f "$src_file" ] && {
			echo -e "  [${RED}ERR${NC}] Source file not found: $target_mod.c"
			return 1
		}

		echo -e "  [${GREEN}INFO${NC}] Metadata for module: ${YELLOW}${target_mod}${NC}"
		echo "  --------------------------------------------------"
		grep -E "MODULE_LICENSE|MODULE_AUTHOR|MODULE_DESCRIPTION" "$src_file" |
			sed -E 's/MODULE_([^ ]*)\("(.*)"\);/\1:\2/' |
			awk -F':' '{printf "  %-15s %s\n", $1, $2}'
	else
		# Global table view
		echo -e "  [${GREEN}INFO${NC}] Metadata for all available modules"
		echo "  --------------------------------------------------------------------------------"
		printf "  %-15s %-10s %-15s %-30s\n" "MODULE" "LICENSE" "AUTHOR" "DESCRIPTION"
		echo "  --------------------------------------------------------------------------------"

		for d in "${MODULES_DIR}"/*/; do
			[ ! -d "$d" ] && continue
			local name=$(basename "$d")
			local src="${d}${name}.c"
			[ ! -f "$src" ] && continue

			# Optimization: Read only the last 20 lines where macros are constrained
			# Use awk to format and wrap the description column
			tail -n 20 "$src" | grep -E "MODULE_LICENSE|MODULE_AUTHOR|MODULE_DESCRIPTION" |
				sed -E 's/MODULE_([^ ]*)\("(.*)"\);/\1:\2/' |
				awk -v mod="$name" -F':' '
					{ data[$1] = $2 }
					END {
						# Wrap description text at 35 characters for alignment
						desc = data["DESCRIPTION"]
						printf "  %-15s %-10s %-15s ", mod, data["LICENSE"], data["AUTHOR"]
						
						len = length(desc)
						if (len <= 35) {
							printf "%s\n", desc
						} else {
							printf "%s\n", substr(desc, 1, 35)
							# Print remaining chunks aligned to the description column
							start = 36
							while (start <= len) {
								printf "  %-15s %-10s %-15s %s\n", "", "", "", substr(desc, start, 35)
								start += 35
							}
						}
					}'
			echo "  --------------------------------------------------------------------------------"
		done
	fi
}

# ─────────────────────────────────────────────────────────────
# 3. Status Logic (-s)
# ─────────────────────────────────────────────────────────────
# Design: Provides a dashboard of what is built and what is queued for QEMU.
_module_status() {
	local target_mod="$1"

	echo -e "  [${GREEN}STATUS${NC}] Kernel Module Dashboard"
	# Updated header with fixed widths to accommodate color padding
	printf "  %-20s %-12s %-12s %-12s\n" "NAME" "BUILT" "QUEUE:INS" "QUEUE:REM"
	echo "  ------------------------------------------------------------"

	for d in "${MODULES_DIR}"/*/; do
		[ ! -d "$d" ] && continue
		local name=$(basename "$d")

		# 1. Filter: If a target_mod is specified, skip others
		if [ -n "$target_mod" ] && [ "$target_mod" != "$name" ]; then
			continue
		fi

		# 2. Check BUILT status
		local built_status="[   ]"
		local built_length=12

		if [ -f "${d}${name}.ko" ]; then
			built_status="[${GREEN} X ${NC}]"
			built_length=29 # Corrected length to account for ANSI codes
		fi

		# 3. Check QUEUE status
		local q_ins=" "
		local q_rem=" "

		if _queue_contains "$name" "${MODULE_INS[@]}"; then
			q_ins="${GREEN}insmod${NC}"
		fi

		if _queue_contains "$name" "${MODULE_REM[@]}"; then
			q_rem="${RED}rmmod${NC}"
		fi

		# 4. Use echo -e with manual spacing for the formatted line
		# Note: printf %-s counts ANSI codes as characters, so we print
		# the name with printf and the status icons with echo -e for alignment.
		printf "  %-20s " "$name"

		# We pad the strings manually to ensure columns align regardless of color codes
		local p_built=$(printf "%-${built_length}s" "$built_status")
		local p_ins=$(printf "%-${ins_len}s" "$q_ins")
		local p_rem=$(printf "%-${rem_len}s" "$q_rem")

		echo -e "${p_built} ${p_ins} ${p_rem}"
	done
}

# ─────────────────────────────────────────────────────────────
# 4. Kernel Module Header (-p)
# ─────────────────────────────────────────────────────────────
# Design: Prepare kernel headers building from linux kernel with modules_prepare target
_prepare_kernel_headers() {
	local build_jobs=4

	# Take run_build() function's logic for building headers only
	ensure_mounted
	cd "$KERNEL_DIR" || {
		echo -e "  [${RED}FAIL${NC}] Could not enter KERNEL_DIR."
		exit 1
	}

	echo -e "  [${YELLOW}KHEADERS${NC}] Preparing kernel headers for module building..."
	run_build "$build_jobs" "modules_prepare" || {
		echo -e "  [${RED}FAIL${NC}] Kernel headers preparation failed."
		exit 1
	}
	echo -e "  [${GREEN}SUCCESS${NC}] Kernel headers are ready."
}

# ─────────────────────────────────────────────────────────────
# Main Dispatcher
# ─────────────────────────────────────────────────────────────
run_module() {
	local target_mod=""
	local action="build"

	# If first arg doesn't start with '-', it's a module name
	if [[ -n "$1" && "$1" != -* ]]; then
		target_mod="$1"
		shift
	fi

	while [ $# -gt 0 ]; do
		case "$1" in
		-i | --insmod) action="insmod" ;;
		-r | --rmmod) action="rmmod" ;;
		-c | --clean) action="clean" ;;
		-n | --reset) action="reset" ;;
		-s | --status) action="status" ;;
		-f | --info) action="info" ;;
		-e | --headers) action="headers" ;;
		-h | --help) action="help" ;;
		*)
			echo -e "  [${RED}ERROR${NC}] Unknown argument: $1"
			echo "  Use './run.sh module -h' for help."
			return 1
			;;
		esac
		shift
	done

	case "$action" in
	help)
		cat <<EOF
Module Manager Usage: ./run.sh module [km-name] [options]

Options:
  [no flag]      Build all modules (or specific km-name)
  -i, --insmod   Queue module(s) for loading in QEMU (use '*' for all)
  -r, --rmmod    Queue module(s) for removal in QEMU
  -c, --clean    Clean build artifacts
  -n, --reset    Clear all INS/REM queues
  -s, --status   Show module build and queue dashboard
  -f, --info     Display module metadata from source macros
  -e, --headers  Prepare kernel headers for module building
  -h, --help     Show this help message

Examples:
  ./run.sh module               Build all modules
  ./run.sh module mymodule -i   Queue 'mymodule' for insmod
  ./run.sh module -s            Show status of all modules
EOF
		;;

	reset)
		MODULE_INS=()
		MODULE_REM=()
		_save_mod_state
		echo -e "  [${GREEN}OK${NC}] Queues cleared."
		;;

	status)
		_module_status "$target_mod"
		;;

	info)
		_module_info "$target_mod"
		;;

	headers)
		_prepare_kernel_headers
		;;

	insmod)
		_update_queue "insmod" "$target_mod"
		;;

	rmmod)
		_update_queue "rmmod" "$target_mod"
		;;

	clean)
		if [ -n "$target_mod" ]; then
			_module_item_action "$target_mod" clean
		else
			for d in "${MODULES_DIR}"/*/; do
				[ -d "$d" ] && _module_item_action "$(basename "$d")" clean
			done
		fi
		;;

	build)
		if [ -n "$target_mod" ]; then
			_module_item_action "$target_mod" # Default target is 'modules'
		else
			for d in "${MODULES_DIR}"/*/; do
				[ -d "$d" ] && _module_item_action "$(basename "$d")"
			done
		fi
		;;

	esac
}
