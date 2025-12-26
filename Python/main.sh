#!/bin/bash
# Python/main.sh

# Resolve Project Root
THIS_DIR="$(cd "$(dirname "$0")" && pwd)"
export PROJECT_ROOT="$(cd "${THIS_DIR}/.." && pwd)"

# Source Environment
source "${PROJECT_ROOT}/Python/Services/EnvironmentService.sh"

# 1. Add Project Root to PYTHONPATH
export PYTHONPATH="${PROJECT_ROOT}:${PYTHONPATH}"

# 2. Python Interpreter Detection
# Priority:
#   1. Local .venv (if it exists)
#   2. System python3
if [ -f "${PROJECT_ROOT}/.venv/bin/python3" ]; then
	PYTHON_BIN="${PROJECT_ROOT}/.venv/bin/python3"
else
	PYTHON_BIN="python3"
fi

# 3. Run as a Module
# We use exec to replace the shell process with Python
exec "$PYTHON_BIN" -m Python.run "$@"
