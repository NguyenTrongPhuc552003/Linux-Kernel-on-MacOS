from .arch_base import ArchStrategy


class Arm64Strategy(ArchStrategy):
    @property
    def name(self):
        return "arm64"

    @property
    def cross_compile_prefix(self):
        return "aarch64-linux-gnu-"

    def get_image_name(self):
        return "Image.gz"

    # --- QEMU ---
    @property
    def qemu_binary(self):
        return "qemu-system-aarch64"

    @property
    def qemu_machine_flags(self):
        return ["-M", "virt", "-cpu", "cortex-a57", "-m", "2G", "-smp", "2"]

    @property
    def debian_arch(self):
        return "arm64"  # Linux uses 'arm64', Debian uses 'arm64'
