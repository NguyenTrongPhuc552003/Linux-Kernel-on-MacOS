import sys
import shutil
import os
import subprocess
from .base import BaseCommand
from ..config import Config


class DoctorCommand(BaseCommand):
	@property
	def name(self):
		return "doctor"

	@property
	def help(self):
		return "Checks the environment for necessary tools and configurations."

	def register_args(self, parser):
		# No arguments needed for doctor
		pass

	def run(self, args):
		print(f"  [PYTHON] Running Environment Doctor...\n")

		checks = [
			self.check_directories,
			self.check_dependencies,
			self.check_disk_image,
			self.check_headers,
		]

		issues = 0
		for check in checks:
			if not check():
				issues += 1

		print("\n" + "=" * 50)
		if issues == 0:
			print(f"  \033[92m[PASS] All checks passed. System is ready.\033[0m")
		else:
			print(f"  \033[91m[FAIL] Found {issues} issues.\033[0m")
			sys.exit(1)

	def _log(self, name, status, error=None):
		if status:
			print(f"  [\033[92mOK\033[0m] {name}")
		else:
			print(f"  [\033[91mFAIL\033[0m] {name}")
			if error:
				print(f"         └─ {error}")

	def check_directories(self):
		# We can read paths from Config or EnvironmentService
		# For now, simplistic checks
		required_dirs = ["libraries", "tools", "modules"]
		root = os.environ.get("PROJECT_ROOT", ".")
		all_ok = True

		for d in required_dirs:
			path = os.path.join(root, d)
			if os.path.isdir(path):
				self._log(f"Directory exists: {d}", True)
			else:
				self._log(f"Directory missing: {d}", False, "Create this directory.")
				all_ok = False
		return all_ok

	def check_dependencies(self):
		tools = ["git", "make", "python3", "hdiutil"]
		all_ok = True
		for tool in tools:
			if shutil.which(tool):
				self._log(f"Tool found: {tool}", True)
			else:
				self._log(f"Tool missing: {tool}", False, "Install via Homebrew/Xcode.")
				all_ok = False

		# Check specific cross compilers based on Config?
		# Optional, but good practice
		return all_ok

	def check_disk_image(self):
		cfg = Config()
		root = os.environ.get("PROJECT_ROOT", ".")
		# Build path to var/state/img.sparseimage
		img_path = os.path.join(root, "var", "state", "img.sparseimage")

		if os.path.exists(img_path):
			self._log("Disk Image exists", True)
			return True
		else:
			self._log(
				"Disk Image missing",
				False,
				"Run 'km mount' or 'km image' to create it.",
			)
			return False

	def check_headers(self):
		# Check for elf.h which is often missing on macOS
		root = os.environ.get("PROJECT_ROOT", ".")
		elf_h = os.path.join(root, "libraries", "elf.h")

		if os.path.exists(elf_h):
			self._log("Header: elf.h", True)
			return True
		else:
			self._log(
				"Header: elf.h",
				False,
				"Missing. Run legacy doctor to fetch or download manually.",
			)
			return False
