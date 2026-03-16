# ============================================================================
# Colors.cmake — ANSI color escape sequences for CMake message() output
#
# Include this module wherever colored output is needed:
#   include(Colors)
#
# Variables defined (guard-protected against double inclusion):
#   C_RST  — reset
#   C_GRN  — bold green
#   C_RED  — bold red
#   C_YLW  — bold yellow
#   C_BLU  — bold blue
#   C_MAG  — bold magenta
#   C_CYN  — bold cyan
#   C_DIM  — dim/faint
# ============================================================================

if(DEFINED C_RST)
    return()
endif()

string(ASCII 27 ESC)
set(C_RST  "${ESC}[0m")
set(C_GRN  "${ESC}[1;32m")
set(C_RED  "${ESC}[1;31m")
set(C_YLW  "${ESC}[1;33m")
set(C_BLU  "${ESC}[1;34m")
set(C_MAG  "${ESC}[1;35m")
set(C_CYN  "${ESC}[1;36m")
set(C_DIM  "${ESC}[2m")
