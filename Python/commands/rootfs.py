import sys
import os
from .base import BaseCommand
from ..utils import ServiceRunner
from ..config import Config
from ..strategies.riscv import RiscVStrategy
from ..strategies.arm64 import Arm64Strategy


class RootFSCommand(BaseCommand):
    def __init__(self):
        self.strategies = {"riscv": RiscVStrategy(), "arm64": Arm64Strategy()}

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
        # Added --arch support
        parser.add_argument(
            "-a", "--arch", help="Override target architecture (e.g., arm64, riscv)"
        )

    def run(self, args):
        cfg = Config()

        # 1. Determine Architecture
        # Priority: CLI flag > Config file > Default (riscv)
        target_arch = args.arch if args.arch else cfg.get("TARGET_ARCH", "riscv")

        if target_arch not in self.strategies:
            print(f"  [ERROR] Unsupported architecture: {target_arch}")
            print(f"  Available: {', '.join(self.strategies.keys())}")
            sys.exit(1)

        # 2. Persist selection if changed via CLI (Consistency with build command)
        if args.arch:
            cfg.set("TARGET_ARCH", target_arch)

        strategy = self.strategies[target_arch]
        print(
            f"  [PYTHON] Generating RootFS for {strategy.name} (Debian: {strategy.debian_arch})..."
        )
        print("  [INFO] This operation requires sudo privileges for 'debootstrap'.")

        # 3. Prepare Environment
        env_vars = os.environ.copy()
        env_vars.update(strategy.get_env())

        # 4. Prepare Args
        service_args = ["create"]
        if args.force:
            service_args.append("--force")

        # 5. Execute
        ServiceRunner.run("RootFSService.sh", service_args, env=env_vars)
