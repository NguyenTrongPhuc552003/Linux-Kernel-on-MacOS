from abc import ABC, abstractmethod


class BaseCommand(ABC):
	"""Abstract base class for all km commands."""

	@property
	@abstractmethod
	def name(self):
		"""The CLI command name (e.g., 'build')."""
		pass

	@property
	@abstractmethod
	def help(self):
		"""Short help description."""
		pass

	@abstractmethod
	def register_args(self, parser):
		"""Register arguments for this command's subparser."""
		pass

	@abstractmethod
	def run(self, args):
		"""Execute the command logic."""
		pass
