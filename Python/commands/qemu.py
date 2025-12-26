import sys
import os
from .base import BaseCommand
from ..utils import ServiceRunner
from ..config import Config
from ..strategies.riscv import RiscVStrategy
from ..strategies.arm64 import Arm64Strategy


class QemuCommand(BaseCommand):
    def __init__(self):
        self.strategies = {"riscv": RiscVStrategy(), "arm64": Arm64Strategy()}

    @property
    def name(self):
        return "qemu"

    @property
    def help(self):
        return "Runs the kernel in QEMU virtualization."

    def register_args(self, parser):
        parser.add_argument(
            "-d", "--debug", action="store_true", help="Pause for GDB connection"
        )
        parser.add_argument(
            "-n",
            "--nogui",
            action="store_true",
            help="Run without graphical window (nographic)",
        )

    def run(self, args):
        cfg = Config()
        target_arch = cfg.get("TARGET_ARCH", "riscv")

        if target_arch not in self.strategies:
            print(f"  [ERROR] Unsupported architecture: {target_arch}")
            sys.exit(1)

        strategy = self.strategies[target_arch]
        print(f"  [PYTHON] Launching QEMU for {strategy.name}...")

        # 1. Prepare Environment
        env_vars = os.environ.copy()
        env_vars.update(strategy.get_env())

        # Pass the machine flags as a single string to Bash
        # (Bash will split it, or we can handle it in Python.
        # Simpler to pass components via Env or Args)
        env_vars["QEMU_FLAGS"] = " ".join(strategy.qemu_machine_flags)

        # 2. Prepare Arguments for the Service
        service_args = ["run"]  # Dispatcher mode

        if args.debug:
            service_args.append("--debug")
        if args.nogui:
            service_args.append("--nographic")

        # 3. Execute
        ServiceRunner.run("VirtualizationService.sh", service_args, env=env_vars)
