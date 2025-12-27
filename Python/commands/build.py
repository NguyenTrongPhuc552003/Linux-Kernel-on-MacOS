import os
import sys
from .base import BaseCommand
from ..utils import ServiceRunner
from ..config import Config
from ..strategies.riscv import RiscVStrategy
from ..strategies.arm import ArmStrategy
from ..strategies.arm64 import Arm64Strategy


class BuildCommand(BaseCommand):
    def __init__(self):
        # Register available strategies
        self.strategies = {"riscv": RiscVStrategy(), "arm": ArmStrategy(), "arm64": Arm64Strategy()}

    @property
    def name(self):
        return "build"

    @property
    def help(self):
        return "Compiles the Linux kernel (Image, dtbs, modules)."

    def register_args(self, parser):
        parser.add_argument("-j", "--jobs", help="Number of parallel jobs", default="")
        parser.add_argument("-a", "--arch", help="Override target architecture")
        parser.add_argument(
            "targets", nargs="*", default=[], help="Specific make targets"
        )

    def run(self, args):
        cfg = Config()

        # 1. Determine Architecture
        target_arch = args.arch if args.arch else cfg.get("TARGET_ARCH", "riscv")

        if target_arch not in self.strategies:
            print(f"  [ERROR] Unsupported architecture: {target_arch}")
            sys.exit(1)

        if args.arch:
            cfg.set("TARGET_ARCH", target_arch)

        strategy = self.strategies[target_arch]
        print(
            f"  [PYTHON] Strategy: {strategy.name} | Cross: {strategy.cross_compile_prefix}"
        )

        # 2. Prepare Environment
        env_vars = os.environ.copy()
        env_vars.update(strategy.get_env())
        # Also pass the TARGET_ARCH variable so bash can read it easily
        env_vars["TARGET_ARCH"] = strategy.name

        # 3. Construct Arguments
        service_args = ["build"]
        service_args.append(args.jobs if args.jobs else "")
        service_args.extend(args.targets)

        # 4. Execute
        ServiceRunner.run("KernelService.sh", service_args, env=env_vars)
