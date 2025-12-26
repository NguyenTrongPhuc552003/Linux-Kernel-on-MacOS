from .base import BaseCommand
from ..utils import ServiceRunner


class MountCommand(BaseCommand):
    @property
    def name(self):
        return "mount"

    @property
    def help(self):
        return "Mounts the case-sensitive disk image."

    def register_args(self, parser):
        pass  # No args needed

    def run(self, args):
        # Call ImageService.sh with 'mount'
        ServiceRunner.run("ImageService.sh", ["mount"])


class UnmountCommand(BaseCommand):
    @property
    def name(self):
        return "unmount"

    @property
    def help(self):
        return "Unmounts the disk image."

    def register_args(self, parser):
        pass

    def run(self, args):
        # Call ImageService.sh with 'unmount'
        ServiceRunner.run("ImageService.sh", ["unmount"])
