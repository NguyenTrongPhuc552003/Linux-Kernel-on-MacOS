#!/bin/bash
# Python/main.sh

# Resolve True Path
REAL_SCRIPT_PATH=$(python3 -c "import os, sys; print(os.path.realpath(sys.argv[1]))" "$0")
THIS_DIR="$(dirname "$REAL_SCRIPT_PATH")"
export PROJECT_ROOT="$(cd "${THIS_DIR}/.." && pwd)"

# Source Environment Services
source "${PROJECT_ROOT}/Python/Services/EnvironmentService.sh"

export PYTHONPATH="${PROJECT_ROOT}:${PYTHONPATH}"
VENV_DIR="${PROJECT_ROOT}/.venv"
REQUIREMENTS="${PROJECT_ROOT}/requirements.txt"

# ─────────────────────────────────────────────────────────────
# VENV BOOTSTRAP LOGIC
# ─────────────────────────────────────────────────────────────

# Check if we need to create the venv
if [ ! -d "$VENV_DIR" ]; then
	echo "  [INIT] Virtual environment not found. Creating .venv..."
	python3 -m venv "$VENV_DIR" || {
		echo "Error creating venv"
		exit 1
	}

	# Force install dependencies immediately
	echo "  [INIT] Installing dependencies from requirements.txt..."
	"$VENV_DIR/bin/pip" install -q -r "$REQUIREMENTS"
fi

# Check if requirements changed (simple check: if rich is missing)
# This handles the case where venv exists but user pulled new requirements
if ! "$VENV_DIR/bin/python3" -c "import rich" 2>/dev/null; then
	echo "  [UPDATE] Missing dependencies detected. Updating..."
	"$VENV_DIR/bin/pip" install -q -r "$REQUIREMENTS"
fi

# ─────────────────────────────────────────────────────────────
# EXECUTION
# ─────────────────────────────────────────────────────────────

# Always use the VENV python
PYTHON_BIN="${VENV_DIR}/bin/python3"

# Run
exec "$PYTHON_BIN" -m Python.run "$@"
