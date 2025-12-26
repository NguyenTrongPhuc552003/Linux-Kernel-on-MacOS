from abc import ABC, abstractmethod


class ArchStrategy(ABC):
    """Abstract Base Class for Architecture Strategies."""

    @property
    @abstractmethod
    def name(self):
        """Returns the architecture name (e.g., 'arm64')."""
        pass

    @property
    @abstractmethod
    def cross_compile_prefix(self):
        """Returns the cross-compiler prefix (e.g., 'aarch64-linux-gnu-')."""
        pass

    @abstractmethod
    def get_image_name(self):
        """Returns the kernel image filename (e.g., 'Image.gz')."""
        pass

    @property
    @abstractmethod
    def qemu_binary(self):
        """Returns the QEMU binary name (e.g., qemu-system-riscv64)."""
        pass

    @property
    @abstractmethod
    def qemu_machine_flags(self):
        """Returns a list of specific flags for the machine (e.g., -M virt)."""
        pass

    @property
    @abstractmethod
    def debian_arch(self):
        """Returns the Debian architecture string (e.g., riscv64)."""
        pass

    def get_env(self):
        return {
            "ARCH": self.name,
            "CROSS_COMPILE": self.cross_compile_prefix,
            "QEMU_BIN": self.qemu_binary,
            "DEBIAN_ARCH": self.debian_arch,
        }
