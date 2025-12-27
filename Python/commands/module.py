import os
from .base import BaseCommand
from ..utils import ServiceRunner
from ..config import Config
from ..managers.module_state import ModuleState
from ..strategies.riscv import RiscVStrategy
from ..strategies.arm import ArmStrategy
from ..strategies.arm64 import Arm64Strategy
from ..ui import UI
from rich.table import Table


class ModuleCommand(BaseCommand):
    def __init__(self):
        self.state = ModuleState()
        self.strategies = {
            "riscv": RiscVStrategy(),
            "arm": ArmStrategy(),
            "arm64": Arm64Strategy(),
        }

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
        cfg = Config()
        env_vars = os.environ.copy()

        # 1. Handle Queue Operations (Install/Remove/Reset/Status)
        if args.reset:
            self.state.clear()
            UI.success("Module queue cleared.")
            return

        if args.status:
            self.print_status()
            return

        if args.install or args.remove:
            if not args.name:
                UI.error("Please specify a module name to queue.")
                return

            if args.install:
                # Construct path: modules/<name>/<name>.ko
                ko_path = os.path.join(
                    cfg.project_root, "modules", args.name, f"{args.name}.ko"
                )

                if not os.path.exists(ko_path):
                    UI.error(f"Cannot queue '{args.name}' for install.")
                    UI.warn(f"Binary not found: {ko_path}")
                    UI.log(f"Action: Run 'km module {args.name}' to build it first.")
                    return

                self.state.add_install(args.name)
                UI.success(f"Queued for install: {args.name}")

            if args.remove:
                self.state.add_remove(args.name)
                UI.success(f"Queued for removal: {args.name}")
            return

        # 2. Handle Build/Clean Operations (Delegated to Bash)
        if args.headers:
            ServiceRunner.run("ModuleService.sh", ["headers"], env=env_vars)
            return

        service_args = []
        if args.clean:
            service_args = ["clean"]
            if args.name:
                service_args.append(args.name)
        else:
            # Default: Build
            service_args = ["build"]
            if args.name:
                service_args.append(args.name)

        ServiceRunner.run("ModuleService.sh", service_args, env=env_vars)

    def print_status(self):
        data = self.state.get_status()

        # Create a Rich Table
        table = Table(
            title="Kernel Module Queue", show_header=True, header_style="bold magenta"
        )
        table.add_column("Action", style="dim", width=12)
        table.add_column("Modules", style="bold white")

        # Format Install list
        install_str = (
            "\n".join(data["install"]) if data["install"] else "[dim](empty)[/]"
        )
        table.add_row("[green]INSTALL[/]", install_str)

        # Format Remove list
        remove_str = "\n".join(data["remove"]) if data["remove"] else "[dim](empty)[/]"
        table.add_row("[red]REMOVE[/]", remove_str)

        UI.console.print(table)
