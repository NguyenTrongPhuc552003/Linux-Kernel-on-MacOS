import sys
import shutil
import os
import subprocess
import urllib.request
from .base import BaseCommand
from ..ui import UI


class DoctorCommand(BaseCommand):
    @property
    def name(self):
        return "doctor"

    @property
    def help(self):
        return "Checks environment health (Taps, GDB, Headers, Dependencies)."

    def register_args(self, parser):
        pass

    def run(self, args):
        UI.header("Environment Doctor")

        # We run checks in a logical order
        checks = [
            self.check_directories,
            self.check_taps,  # New: Check Homebrew Taps
            self.check_dependencies,  # Updated: Check GDB/Tools
            self.check_debootstrap,  # New: Check Submodule
            self.check_disk_image,
            self.check_headers,  # Updated: Auto-download elf.h
            self.check_rust,
        ]

        issues = 0
        for check in checks:
            if not check():
                issues += 1

        UI.console.print()
        if issues == 0:
            UI.console.print(
                f"[bold green on black] PASS [/] [bold green]All checks passed. System is ready.[/]"
            )
        else:
            UI.console.print(
                f"[bold red on black] FAIL [/] [bold red]Found {issues} issue(s).[/]"
            )
            sys.exit(1)

    # ---------------------------------------------------------
    # 1. Directory Checks
    # ---------------------------------------------------------
    def check_directories(self):
        required_dirs = ["libraries", "tools", "modules"]
        root = os.environ.get("PROJECT_ROOT", ".")
        all_ok = True

        UI.step("Checking Directories")
        for d in required_dirs:
            path = os.path.join(root, d)
            if os.path.isdir(path):
                UI.success(f"Directory exists: [bold]{d}[/]")
            else:
                UI.error(f"Directory missing: [bold]{d}[/]")
                UI.warn("-> Action: Create this directory.")
                all_ok = False
        return all_ok

    # ---------------------------------------------------------
    # 2. Homebrew Taps (Legacy Port)
    # ---------------------------------------------------------
    def check_taps(self):
        UI.step("Checking Homebrew Taps")
        required_tap = "messense/macos-cross-toolchains"

        try:
            result = subprocess.run(["brew", "tap"], capture_output=True, text=True)
            if required_tap in result.stdout:
                UI.success(f"Tap found: [bold]{required_tap}[/]")
                return True
            else:
                UI.error(f"Tap missing: [bold]{required_tap}[/]")
                UI.warn(f"-> Action: brew tap {required_tap}")
                return False
        except FileNotFoundError:
            UI.error("Homebrew ('brew') not found.")
            return False

    # ---------------------------------------------------------
    # 3. Dependencies (GDB & Tools)
    # ---------------------------------------------------------
    def check_dependencies(self):
        UI.step("Checking Tools & GDB")

        # CHANGED: 'e2fsprogs' -> 'mkfs.ext4'
        core_tools = ["git", "make", "python3", "hdiutil", "mkfs.ext4", "fakeroot"]
        gdb_tools = ["riscv64-elf-gdb", "aarch64-elf-gdb"]

        all_ok = True

        # Add Homebrew sbin to PATH temporarily for this check
        # (e2fsprogs is often in /opt/homebrew/opt/e2fsprogs/sbin)
        env = os.environ.copy()
        possible_paths = [
            "/opt/homebrew/opt/e2fsprogs/sbin",
            "/usr/local/opt/e2fsprogs/sbin",
        ]
        for p in possible_paths:
            if os.path.isdir(p):
                env["PATH"] = f"{p}:{env['PATH']}"

        for tool in core_tools + gdb_tools:
            # use shutil.which with the modified env path logic if needed,
            # but shutil.which uses os.environ usually.
            # Simpler: just check if we can find it.

            found_path = shutil.which(tool, path=env["PATH"])

            if found_path:
                UI.success(f"Tool found: [bold]{tool}[/]")
            else:
                UI.error(f"Tool missing: [bold]{tool}[/]")
                if tool == "mkfs.ext4":
                    UI.warn(
                        "-> Action: brew install e2fsprogs (and ensure it is in PATH)"
                    )
                    UI.warn(
                        '   Hint: export PATH="$(brew --prefix e2fsprogs)/sbin:$PATH"'
                    )
                elif "gdb" in tool:
                    UI.warn(f"-> Action: brew install {tool} (Requires messense tap)")
                else:
                    UI.warn(f"-> Action: brew install {tool}")
                all_ok = False
        return all_ok

    # ---------------------------------------------------------
    # 4. Debootstrap Submodule (Legacy Port)
    # ---------------------------------------------------------
    def check_debootstrap(self):
        UI.step("Checking Debootstrap")
        root = os.environ.get("PROJECT_ROOT", ".")
        deboot_bin = os.path.join(root, "tools", "debootstrap", "debootstrap")

        if os.path.isfile(deboot_bin):
            UI.success("Debootstrap submodule present")
            return True
        else:
            UI.error("Debootstrap missing")
            UI.warn(
                "-> Action: git submodule update --init --recursive tools/debootstrap"
            )
            return False

    # ---------------------------------------------------------
    # 5. Disk Image
    # ---------------------------------------------------------
    def check_disk_image(self):
        UI.step("Checking Storage")
        root = os.environ.get("PROJECT_ROOT", ".")
        img_path = os.path.join(root, "var", "state", "img.sparseimage")

        if os.path.exists(img_path):
            UI.success("Disk Image exists")
            return True
        else:
            UI.error("Disk Image missing")
            UI.warn("-> Action: Run 'km mount' to create it.")
            return False

    # ---------------------------------------------------------
    # 6. Headers & Auto-Fix (Legacy Port)
    # ---------------------------------------------------------
    def check_headers(self):
        UI.step("Checking Headers")
        root = os.environ.get("PROJECT_ROOT", ".")
        elf_h = os.path.join(root, "libraries", "elf.h")

        if os.path.exists(elf_h):
            UI.success("Header found: elf.h")
            return True
        else:
            UI.error("Header missing: elf.h")

            # Interactive Fix
            # We use Python input/urllib so we don't depend on wget
            try:
                UI.console.print(
                    "[yellow]elf.h is required for macOS builds.[/] Download now? [Y/n]: ",
                    end="",
                )
                choice = input().lower()
                if choice in ["", "y", "yes"]:
                    url = "https://raw.githubusercontent.com/bminor/glibc/glibc-2.42/elf/elf.h"
                    UI.log(f"Downloading from {url}...")
                    urllib.request.urlretrieve(url, elf_h)
                    UI.success("elf.h downloaded successfully.")
                    return True
            except Exception as e:
                UI.error(f"Download failed: {e}")

            return False

    # ---------------------------------------------------------
    # 7. Rust
    # ---------------------------------------------------------
    def check_rust(self):
        UI.step("Checking Rust Support")
        tools = ["rustc", "cargo", "bindgen"]
        all_ok = True

        for tool in tools:
            if shutil.which(tool):
                UI.success(f"Rust Tool: [bold]{tool}[/]")
            else:
                UI.warn(f"Rust Tool missing: [bold]{tool}[/] (Optional)")
                # all_ok = False # Optional for now
        return all_ok
