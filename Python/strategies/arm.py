from .arch_base import ArchStrategy


class ArmStrategy(ArchStrategy):
    @property
    def name(self):
        return "arm"

    @property
    def cross_compile_prefix(self):
        return "arm-none-eabi-"

    def get_image_name(self):
        return "zImage"  # 32-bit ARM usually uses zImage

    @property
    def qemu_binary(self):
        return "qemu-system-arm"

    @property
    def debian_arch(self):
        return "armhf"  # Debian standard for 32-bit ARM (Hard Float)

    @property
    def qemu_machine_flags(self):
        # Matches your legacy script: cortex-a15, virt machine
        return ["-M", "virt", "-cpu", "cortex-a15", "-m", "2G", "-smp", "2"]
