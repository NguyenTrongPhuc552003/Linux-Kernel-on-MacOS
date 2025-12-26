import os
import subprocess
import sys


class ServiceRunner:
    @staticmethod
    def get_service_path(service_name):
        base_dir = os.path.dirname(os.path.abspath(__file__))
        return os.path.join(base_dir, "Services", service_name)

    @staticmethod
    def run(service_name, args=None, env=None):
        """
        Executes a Bash service script.
        :param env: Optional dictionary of environment variables to inject.
        """
        if args is None:
            args = []

        script_path = ServiceRunner.get_service_path(service_name)
        cmd = [script_path] + args

        print(f"  [PYTHON] Delegating to {service_name}...")

        try:
            # Merge current env with injected env if provided
            run_env = os.environ.copy()
            if env:
                run_env.update(env)

            subprocess.run(cmd, check=True, env=run_env)
        except subprocess.CalledProcessError as e:
            print(
                f"  [PYTHON] Service {service_name} failed with exit code {e.returncode}"
            )
            sys.exit(e.returncode)
        except KeyboardInterrupt:
            print("\n  [PYTHON] Operation cancelled by user.")
            sys.exit(130)


# Quick helper for colors if needed later
class Colors:
    HEADER = "\033[95m"
    BLUE = "\033[94m"
    GREEN = "\033[92m"
    WARNING = "\033[93m"
    FAIL = "\033[91m"
    ENDC = "\033[0m"
