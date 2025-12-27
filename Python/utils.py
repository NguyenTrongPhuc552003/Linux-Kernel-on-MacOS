import os
import subprocess
import sys
import signal
from .ui import UI


class ServiceRunner:
    @staticmethod
    def get_service_path(service_name):
        # Resolve path relative to this file
        base_dir = os.path.dirname(os.path.abspath(__file__))
        return os.path.join(base_dir, "Services", service_name)

    @staticmethod
    def run(service_name, args=None, env=None, description=None):
        if args is None:
            args = []

        script_path = ServiceRunner.get_service_path(service_name)
        cmd = [script_path] + args

        if not description:
            description = f"Running [bold]{service_name}[/]"

        run_env = os.environ.copy()
        if env:
            run_env.update(env)

        # Services that require direct TTY access (Interactive)
        long_running_services = [
            "KernelService.sh",  # menuconfig
            "RootFSService.sh",  # debootstrap prompts
            "ModuleService.sh",  # build logs
            "VirtualizationService.sh",  # QEMU & GDB
        ]

        is_interactive = service_name in long_running_services

        if is_interactive:
            # We tell Python to IGNORE Ctrl+C. This allows the signal to pass
            # through to the child process (GDB/QEMU) directly.
            # GDB uses Ctrl+C to pause execution, NOT to exit.

            # 1. Save original handler
            original_sigint = signal.signal(signal.SIGINT, signal.SIG_IGN)

            try:
                UI.log(f"Delegating to {service_name}...", style="dim")
                # 2. Run process (Child receives SIGINT directly from TTY)
                subprocess.run(cmd, check=True, env=run_env)

            except subprocess.CalledProcessError as e:
                # Child exited with error (e.g. forced kill)
                sys.exit(e.returncode)
            finally:
                # 3. Restore Python's handler (Cleanup)
                signal.signal(signal.SIGINT, original_sigint)

        else:
            # --- Standard Mode (Spinner) ---
            try:
                with UI.console.status(f"[green]{description}...", spinner="dots"):
                    result = subprocess.run(
                        cmd, check=True, env=run_env, capture_output=True, text=True
                    )
                    if result.stdout.strip():
                        UI.console.print(result.stdout.strip())

            except subprocess.CalledProcessError as e:
                UI.error(f"Service {service_name} failed with exit code {e.returncode}")
                if e.stdout and e.stdout.strip():
                    UI.console.print("[bold]Output:[/]")
                    UI.console.print(e.stdout.strip(), style="dim")
                if e.stderr and e.stderr.strip():
                    UI.console.print("[bold]Error Log:[/]")
                    UI.console.print(e.stderr.strip(), style="red")
                sys.exit(e.returncode)

            except KeyboardInterrupt:
                UI.warn("Operation cancelled by user.")
                sys.exit(130)
