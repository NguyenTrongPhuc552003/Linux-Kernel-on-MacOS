import os
import sys
from .base import BaseCommand
from ..utils import ServiceRunner
from ..config import Config
from ..strategies.riscv import RiscVStrategy
from ..strategies.arm import ArmStrategy
from ..strategies.arm64 import Arm64Strategy


class ConfigCommand(BaseCommand):
    def __init__(self):
        self.strategies = {
            "riscv": RiscVStrategy(),
            "arm": ArmStrategy(),
            "arm64": Arm64Strategy(),
        }

    @property
    def name(self):
        return "config"

    @property
    def help(self):
        return "Generates the kernel configuration (defconfig, etc)."

    def register_args(self, parser):
        parser.add_argument(
            "target",
            nargs="?",
            default="defconfig",
            help="Config target (default: defconfig, e.g., allnoconfig, menuconfig)",
        )
        parser.add_argument("-a", "--arch", help="Override target architecture")

    def run(self, args):
        cfg = Config()

        # Determine Architecture
        target_arch = args.arch if args.arch else cfg.get("TARGET_ARCH", "riscv")

        if target_arch not in self.strategies:
            print(f"  [ERROR] Unsupported architecture: {target_arch}")
            sys.exit(1)

        # Save arch if changed
        if args.arch:
            cfg.set("TARGET_ARCH", target_arch)

        strategy = self.strategies[target_arch]

        # Prepare Environment
        env_vars = os.environ.copy()
        env_vars.update(strategy.get_env())
        env_vars["TARGET_ARCH"] = strategy.name

        # Construct Args: Mode ("config") + Target ("defconfig")
        service_args = ["config", args.target]

        ServiceRunner.run("KernelService.sh", service_args, env=env_vars)
