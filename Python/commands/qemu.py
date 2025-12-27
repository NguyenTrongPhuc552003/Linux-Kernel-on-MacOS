import sys
import os
import stat
from .base import BaseCommand
from ..utils import ServiceRunner
from ..config import Config
from ..managers.module_state import ModuleState
from ..strategies.riscv import RiscVStrategy
from ..strategies.arm import ArmStrategy
from ..strategies.arm64 import Arm64Strategy
from ..ui import UI


class QemuCommand(BaseCommand):
    def __init__(self):
        self.module_state = ModuleState()
        self.strategies = {
            "riscv": RiscVStrategy(),
            "arm": ArmStrategy(),
            "arm64": Arm64Strategy(),
        }

    @property
    def name(self):
        return "qemu"

    @property
    def help(self):
        return "Runs the kernel in QEMU (defaults to console/nographic)."

    def register_args(self, parser):
        parser.add_argument(
            "-g",
            "--gui",
            action="store_true",
            help="Enable graphical output (Cocoa window)",
        )
        parser.add_argument(
            "-d",
            "--debug",
            nargs="?",
            const="0",
            default=None,
            help="Start GDB Server (default) or connect Client (-d 1)",
        )

    def run(self, args):
        cfg = Config()
        target_arch = cfg.get("TARGET_ARCH", "riscv")

        if target_arch not in self.strategies:
            UI.error(f"Unsupported architecture: {target_arch}")
            sys.exit(1)

        strategy = self.strategies[target_arch]

        # 1. Configuration Checks
        if args.gui:
            self._check_gpu_config()

        # 2. Debug Logic
        # Case A: Client Mode (--debug 1)
        if args.debug == "1":
            env_vars = os.environ.copy()
            env_vars.update(strategy.get_env())
            env_vars["TARGET_ARCH"] = strategy.name  # Explicitly pass arch

            UI.log(f"Launching GDB Client for [bold]{target_arch}[/]...")
            ServiceRunner.run("VirtualizationService.sh", ["client"], env=env_vars)
            return

        # Case B: Server Mode (--debug or --debug 0)
        is_debug_server = args.debug == "0"
        if is_debug_server:
            self._check_debug_config()

        # 3. Generate Guest Sync Script
        self._generate_guest_script(cfg.project_root)

        # 4. Prepare Environment
        env_vars = os.environ.copy()
        env_vars.update(strategy.get_env())
        env_vars["QEMU_FLAGS"] = " ".join(strategy.qemu_machine_flags)

        # 5. Execute Service (Server/Run Mode)
        service_args = ["run"]

        if args.gui:
            service_args.append("--gui")
        else:
            service_args.append("--nographic")

        if is_debug_server:
            service_args.append("--debug")

        ServiceRunner.run("VirtualizationService.sh", service_args, env=env_vars)

    def _get_config_path(self):
        return os.path.join("/Volumes/kernel-dev/linux", ".config")

    def _check_gpu_config(self):
        """Checks if .config has DRM/FB enabled before allowing GUI."""
        config_path = self._get_config_path()
        if not os.path.exists(config_path):
            UI.warn("Kernel .config not found. GUI mode might fail.")
            return

        required_configs = [
            "CONFIG_DRM=y",
            "CONFIG_DRM_VIRTIO_GPU=y",
            "CONFIG_FB=y",
            "CONFIG_FRAMEBUFFER_CONSOLE=y",
        ]
        missing = []
        try:
            with open(config_path, "r") as f:
                content = f.read()
                for req in required_configs:
                    if req not in content:
                        missing.append(req)
        except Exception:
            pass

        if missing:
            UI.error("Missing Kernel GPU configuration for GUI mode.")
            UI.warn("Falling back to console mode to prevent black screen.")
            sys.exit(1)

    def _check_debug_config(self):
        """Checks if debug symbols are enabled before starting GDB stub."""
        config_path = self._get_config_path()
        if not os.path.exists(config_path):
            UI.warn("Kernel .config not found. Debugging might fail.")
            return

        has_debug = False
        try:
            with open(config_path, "r") as f:
                for line in f:
                    line = line.strip()
                    if line == "CONFIG_DEBUG_KERNEL=y":
                        has_debug = True
                    if line.startswith("CONFIG_DEBUG_INFO") and line.endswith("=y"):
                        has_debug = True
        except Exception:
            pass

        if not has_debug:
            UI.error("Kernel debugging symbols not properly enabled!")
            UI.console.print(
                "  Required: CONFIG_DEBUG_KERNEL=y or CONFIG_DEBUG_INFO*=y"
            )
            sys.exit(1)

    def _generate_guest_script(self, project_root):
        modules_dir = os.path.join(project_root, "modules")
        sync_file = os.path.join(modules_dir, "guesync.sh")
        os.makedirs(modules_dir, exist_ok=True)
        state = self.module_state.get_status()

        content = ["#!/bin/sh", "echo '  [GUEST] Processing module queues...'"]
        for mod in state["remove"]:
            content.append(f"rmmod {mod} 2>/dev/null")
        for mod in state["install"]:
            content.append(f"insmod /mnt/modules/{mod}/{mod}.ko 2>/dev/null")
        content.append("echo '  [GUEST] Sync complete.'")

        try:
            with open(sync_file, "w") as f:
                f.write("\n".join(content))
            st = os.stat(sync_file)
            os.chmod(sync_file, st.st_mode | stat.S_IEXEC)
        except Exception as e:
            UI.error(f"Failed to generate guesync.sh: {e}")
