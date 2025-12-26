import sys
import os
from .base import BaseCommand
from ..utils import ServiceRunner
from ..config import Config
from ..managers.module_state import ModuleState
from ..strategies.riscv import RiscVStrategy
from ..strategies.arm64 import Arm64Strategy


class ModuleCommand(BaseCommand):
    def __init__(self):
        self.state = ModuleState()
        self.strategies = {"riscv": RiscVStrategy(), "arm64": Arm64Strategy()}

    @property
    def name(self):
        return "module"

    @property
    def help(self):
        return "Manages kernel modules (build, install queue, status)."

    def register_args(self, parser):
        parser.add_argument("name", nargs="?", help="Name of the module (folder name)")
        # Actions
        parser.add_argument(
            "-i", "--install", action="store_true", help="Queue module for installation"
        )
        parser.add_argument(
            "-r", "--remove", action="store_true", help="Queue module for removal"
        )
        parser.add_argument(
            "-c", "--clean", action="store_true", help="Clean module build artifacts"
        )
        parser.add_argument(
            "-s", "--status", action="store_true", help="Show current module queue"
        )
        parser.add_argument("--reset", action="store_true", help="Clear the queue")
        # New Flag
        parser.add_argument(
            "-e",
            "--headers",
            action="store_true",
            help="Prepare kernel headers (make modules_prepare)",
        )

        # Arch Override
        parser.add_argument("-a", "--arch", help="Override target architecture")

    def run(self, args):
        # 1. Handle Status & Reset
        if args.status:
            self.print_status()
            return

        if args.reset:
            self.state.clear_queue()
            print("  [MODULE] Queue cleared.")
            return

        # 2. Handle Queue Updates
        if args.name:
            if args.install:
                self.state.add_install(args.name)
                print(f"  [MODULE] Queued for install: {args.name}")
            elif args.remove:
                self.state.add_remove(args.name)
                print(f"  [MODULE] Queued for removal: {args.name}")

        # 3. Prepare Environment (Fixing the ARCH bug)
        cfg = Config()
        target_arch = args.arch if args.arch else cfg.get("TARGET_ARCH", "riscv")

        if target_arch not in self.strategies:
            print(f"  [ERROR] Unsupported architecture: {target_arch}")
            sys.exit(1)

        strategy = self.strategies[target_arch]

        # Inject Environment
        env_vars = os.environ.copy()
        env_vars.update(strategy.get_env())
        # IMPORTANT: Explicitly set TARGET_ARCH so ModuleService.sh sees it
        env_vars["TARGET_ARCH"] = strategy.name

        # 4. Determine Action
        service_args = []

        if args.headers:
            # New Action: Prepare Headers
            service_args = ["headers"]
        elif args.clean:
            service_args = ["clean"]
            if args.name:
                service_args.append(args.name)
        else:
            # Default Action: Build
            # Only build if we aren't just queuing things (unless name provided without -i/-r)
            if args.name and not (args.install or args.remove):
                service_args = ["build", args.name]
            elif not args.name and not (args.install or args.remove):
                # No args = Build all
                service_args = ["build"]
            else:
                # Just updating queue, don't call service
                return

        ServiceRunner.run("ModuleService.sh", service_args, env=env_vars)

    def print_status(self):
        data = self.state.get_status()
        print("\n  [MODULE QUEUE STATUS]")
        print("  ─────────────────────")
        print("  [INSTALL]: ", end="")
        print(", ".join(data["install"]) if data["install"] else "(empty)")
        print("  [REMOVE] : ", end="")
        print(", ".join(data["remove"]) if data["remove"] else "(empty)")
        print()
