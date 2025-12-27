#!/usr/bin/env python3
import sys
import argparse

# Import All Commands
from .commands.build import BuildCommand
from .commands.doctor import DoctorCommand
from .commands.config import ConfigCommand
from .commands.qemu import QemuCommand
from .commands.rootfs import RootFSCommand
from .commands.module import ModuleCommand
from .commands.patch import PatchCommand
from .commands.repo import RepoCommand
from .commands.image import MountCommand, UnmountCommand


def main():
    parser = argparse.ArgumentParser(
        description="Linux Kernel on macOS Manager (km)", prog="km"
    )
    subparsers = parser.add_subparsers(dest="command", help="Available commands")

    # Register all commands
    commands = [
        BuildCommand(),
        DoctorCommand(),
        ConfigCommand(),
        QemuCommand(),
        RootFSCommand(),
        ModuleCommand(),
        PatchCommand(),
        RepoCommand(),
        MountCommand(),
        UnmountCommand(),
    ]

    cmd_map = {}
    for cmd in commands:
        cmd_map[cmd.name] = cmd
        sp = subparsers.add_parser(cmd.name, help=cmd.help)
        cmd.register_args(sp)

    # Handle empty args
    if len(sys.argv) < 2:
        parser.print_help()
        sys.exit(1)

    args = parser.parse_args()

    # Execute
    if args.command in cmd_map:
        cmd_map[args.command].run(args)
    else:
        # Should be unreachable due to argparse, but good practice
        parser.print_help()
        sys.exit(1)


if __name__ == "__main__":
    main()
