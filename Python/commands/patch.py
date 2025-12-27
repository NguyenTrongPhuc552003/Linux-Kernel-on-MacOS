import sys
import os
from .base import BaseCommand
from ..utils import ServiceRunner


class PatchCommand(BaseCommand):
    @property
    def name(self):
        return "patch"

    @property
    def help(self):
        return "Manages and applies kernel patches for macOS compatibility."

    def register_args(self, parser):
        parser.add_argument(
            "action", choices=["list", "apply"], help="Action to perform"
        )
        parser.add_argument(
            "target",
            nargs="?",
            help="Specific patch file or 'auto' to apply version-matched patches",
        )

    def run(self, args):
        # Resolve paths
        project_root = os.environ.get("PROJECT_ROOT", ".")
        patch_dir = os.path.join(project_root, "patches")

        # 1. LIST Action
        if args.action == "list":
            self._list_patches(patch_dir)
            return

        # 2. APPLY Action
        if args.action == "apply":
            if not args.target:
                print("  [ERROR] Please specify a patch file or 'auto'.")
                print("  Usage: km patch apply <filename> | auto")
                sys.exit(1)

            # Logic for 'auto' or specific file is handled here or passed to bash.
            # Let's handle the smarts in Python.

            target_file = args.target

            if args.target == "auto":
                # We could try to detect kernel version here, but simpler to just
                # ask the Service to apply the whole v6.xx folder if it matches.
                # For now, let's pass 'auto' to the service.
                pass
            elif not os.path.exists(target_file) and not os.path.exists(
                os.path.join(patch_dir, target_file)
            ):
                # Try finding it recursively if user just gave the name
                found = self._find_patch(patch_dir, target_file)
                if found:
                    target_file = found
                else:
                    print(f"  [ERROR] Patch file not found: {target_file}")
                    sys.exit(1)

            # Execute
            ServiceRunner.run("PatchService.sh", ["apply", target_file])

    def _list_patches(self, root):
        print(f"\n  [AVAILABLE PATCHES] ({root})")
        print("  ────────────────────────────────────────")
        if not os.path.exists(root):
            print("  (No patches directory found)")
            return

        for dirpath, _, filenames in os.walk(root):
            for f in filenames:
                if f.endswith(".patch") or f.endswith(".diff"):
                    full_path = os.path.join(dirpath, f)
                    rel_path = os.path.relpath(full_path, root)
                    print(f"  - {rel_path}")
        print()

    def _find_patch(self, root, filename):
        for dirpath, _, filenames in os.walk(root):
            if filename in filenames:
                return os.path.join(dirpath, filename)
        return None
