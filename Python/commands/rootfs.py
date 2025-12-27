import sys
import os
from .base import BaseCommand
from ..utils import ServiceRunner
from ..config import Config
from ..strategies.riscv import RiscVStrategy
from ..strategies.arm import ArmStrategy
from ..strategies.arm64 import Arm64Strategy
from ..ui import UI


class RootFSCommand(BaseCommand):
    def __init__(self):
        self.strategies = {
            "riscv": RiscVStrategy(),
            "arm64": Arm64Strategy(),
            "arm": ArmStrategy(),
        }

    @property
    def name(self):
        return "rootfs"

    @property
    def help(self):
        return "Generates a Debian root filesystem (requires sudo)."

    def register_args(self, parser):
        parser.add_argument(
            "-f",
            "--force",
            action="store_true",
            help="Recreate image even if it exists",
        )
        parser.add_argument(
            "-a", "--arch", help="Override target architecture (e.g., arm64, riscv)"
        )

    def run(self, args):
        cfg = Config()

        # 1. Determine Architecture
        target_arch = args.arch if args.arch else cfg.get("TARGET_ARCH", "riscv")

        if target_arch not in self.strategies:
            UI.error(f"Unsupported architecture: {target_arch}")
            UI.console.print(f"Available: {', '.join(self.strategies.keys())}")
            sys.exit(1)

        # Persist selection
        if args.arch:
            cfg.set("TARGET_ARCH", target_arch)

        strategy = self.strategies[target_arch]
        UI.log(
            f"Generating RootFS for [bold]{strategy.name}[/] (Debian: {strategy.debian_arch})..."
        )
        UI.warn("This operation requires sudo privileges for 'debootstrap'.")

        # 2. Prepare Environment
        env_vars = os.environ.copy()
        env_vars.update(strategy.get_env())

        # --- FIX: Explicitly export TARGET_ARCH ---
        env_vars["TARGET_ARCH"] = strategy.name
        # ------------------------------------------

        # 3. Prepare Args
        service_args = ["create"]
        if args.force:
            service_args.append("--force")

        # 4. Execute
        ServiceRunner.run("RootFSService.sh", service_args, env=env_vars)
