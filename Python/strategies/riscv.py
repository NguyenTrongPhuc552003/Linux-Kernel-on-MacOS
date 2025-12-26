from .arch_base import ArchStrategy


class RiscVStrategy(ArchStrategy):
    @property
    def name(self):
        return "riscv"

    @property
    def cross_compile_prefix(self):
        return "riscv64-linux-gnu-"

    def get_image_name(self):
        return "Image"

    # --- QEMU ---
    @property
    def qemu_binary(self):
        return "qemu-system-riscv64"

    @property
    def qemu_machine_flags(self):
        return ["-M", "virt", "-m", "2G", "-smp", "2"]

    @property
    def debian_arch(self):
        return "riscv64"  # Linux uses 'riscv', Debian uses 'riscv64'
