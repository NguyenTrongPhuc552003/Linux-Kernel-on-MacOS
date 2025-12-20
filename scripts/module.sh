#!/bin/bash
# scripts/module.sh
# Management script for out-of-tree kernel modules.
# Handles building, cleaning, and queuing for QEMU integration.

# Path to persist module queue state
MOD_CONFIG="${SCRIPT_DIR}/module.cfg"

# Load existing state if it exists
if [ -f "$MOD_CONFIG" ]; then
	source "$MOD_CONFIG"
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
		echo "MODULE_INS=($(printf "\"%s\" " "${MODULE_INS[@]}"))"
		echo "MODULE_REM=($(printf "\"%s\" " "${MODULE_REM[@]}"))"
	} >"$MOD_CONFIG"
}

# ─────────────────────────────────────────────────────────────
# 1. Build Logic (-b / default)
# ─────────────────────────────────────────────────────────────
_build_module_item() {
	local mod_name="$1"
	local mod_path="${MODULES_DIR}/${mod_name}"

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
		CROSS_COMPILE="$CROSS_COMPILE" \
		HOSTCFLAGS="$HOSTCFLAGS" \
		modules
}

# ─────────────────────────────────────────────────────────────
# 2. Information Logic (-f)
# ─────────────────────────────────────────────────────────────
# Design: Scans the source code for MODULE_ macros to provide host-side modinfo.
_module_info() {
	local mod_name="$1"
	local src_file="${MODULES_DIR}/${mod_name}/${mod_name}.c"

	[ ! -f "$src_file" ] && {
		echo -e "  [${RED}ERR${NC}] Source file not found: $mod_name.c"
		return 1
	}

	echo -e "  [${GREEN}INFO${NC}] Metadata for module: ${YELLOW}${mod_name}${NC}"
	echo "  --------------------------------------------------"
	grep -E "MODULE_LICENSE|MODULE_AUTHOR|MODULE_DESCRIPTION" "$src_file" |
		sed 's/MODULE_//g' | sed 's/("//g' | sed 's/");//g' | sed 's/)/: /' |
		awk -F'(' '{printf "  %-12s %s\n", $1, $2}'
}

# ─────────────────────────────────────────────────────────────
# 3. Status Logic (-s)
# ─────────────────────────────────────────────────────────────
# Design: Provides a dashboard of what is built and what is queued for QEMU.
_module_status() {
	echo -e "  [${GREEN}STATUS${NC}] Kernel Module Dashboard"
	echo "  NAME             BUILT    QUEUE:INS    QUEUE:REM"
	echo "  --------------------------------------------------"

	for d in "${MODULES_DIR}"/*/; do
		[ ! -d "$d" ] && continue
		local name
		name=$(basename "$d")

		# Check if .ko exists
		local built="[ ]"
		[ -f "$d/${name}.ko" ] && built="[${GREEN}X${NC}]"

		# Check if in INS queue
		local q_ins=" "
		[[ " ${MODULE_INS[*]} " =~ " ${name} " ]] || [[ " ${MODULE_INS[*]} " =~ " * " ]] && q_ins="${GREEN}insmod${NC}"

		# Check if in REM queue
		local q_rem=" "
		[[ " ${MODULE_REM[*]} " =~ " ${name} " ]] || [[ " ${MODULE_REM[*]} " =~ " * " ]] && q_rem="${RED}rmmod${NC}"

		printf "  %-15s %-8s %-12s %-12s\n" "$name" "$built" "$q_ins" "$q_rem"
	done
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
		-h | --help) action="help" ;;
		*)
			echo "Unknown option: $1"
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
EOF
		;;
	reset)
		MODULE_INS=()
		MODULE_REM=()
		_save_mod_state
		echo "  [${GREEN}OK${NC}] Queues cleared."
		;;
	status)
		_module_status
		;;
	info)
		[ -z "$target_mod" ] && {
			echo "Specify a module name."
			return 1
		}
		_module_info "$target_mod"
		;;
	insmod)
		local item="${target_mod:-*}"
		MODULE_INS+=("$item")
		_save_mod_state
		echo "  [${GREEN}+${NC}] Queued for insmod: $item"
		;;
	rmmod)
		local item="${target_mod:-*}"
		MODULE_REM+=("$item")
		_save_mod_state
		echo "  [${RED}-${NC}] Queued for rmmod: $item"
		;;
	clean)
		if [ -n "$target_mod" ]; then
			make -C "$KERNEL_DIR" M="${MODULES_DIR}/${target_mod}" ARCH="$TARGET_ARCH" clean
		else
			for d in "${MODULES_DIR}"/*/; do
				make -C "$KERNEL_DIR" M="$d" ARCH="$TARGET_ARCH" clean
			done
		fi
		;;
	build)
		if [ -n "$target_mod" ]; then
			_build_module_item "$target_mod"
		else
			for d in "${MODULES_DIR}"/*/; do
				[ -d "$d" ] && _build_module_item "$(basename "$d")"
			done
		fi
		;;
	esac
}
