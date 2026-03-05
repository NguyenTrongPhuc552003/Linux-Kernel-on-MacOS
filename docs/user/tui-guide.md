# TUI Guide

ELMOS features an interactive Terminal User Interface (TUI) for streamlined workflows.

## Launching TUI

```bash
./build/bin/elmos tui
```

## Interface

The TUI provides menus for:

- **Kernel**: Clone, config, build
- **Toolchains**: Install, build, manage
- **QEMU**: Run, debug
- **Modules/Apps**: Create, build
- **Doctor**: Environment checks
- **Status**: Workspace overview

Navigate with arrow keys, Enter to select, Esc to back.

## Real-time Output

Commands run with live output in the TUI. Errors highlighted.

## Shortcuts

- `Ctrl+C`: Cancel current command
- `q`: Quit

## Benefits

- No memorizing commands
- Visual progress for long builds
- Integrated help

## Fallback

If TUI fails, use CLI commands directly.