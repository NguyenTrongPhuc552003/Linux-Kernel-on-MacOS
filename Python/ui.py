from rich.console import Console
from rich.theme import Theme
from rich.panel import Panel

# Define a custom theme for consistent colors
custom_theme = Theme(
    {
        "info": "cyan",
        "warning": "yellow",
        "error": "bold red",
        "success": "bold green",
        "command": "bold white on blue",
    }
)


class UI:
    """
    Central UI Manager using Rich.
    Replaces standard print() with colored, structured output.
    """

    console = Console(theme=custom_theme)

    @staticmethod
    def log(message, style="info"):
        """Print a standard log message."""
        UI.console.print(message, style=style)

    @staticmethod
    def success(message):
        """Print a success message with a checkmark."""
        UI.console.print(f":white_check_mark: {message}", style="success")

    @staticmethod
    def error(message):
        """Print an error message with a cross."""
        UI.console.print(f":x: {message}", style="error")

    @staticmethod
    def warn(message):
        """Print a warning."""
        UI.console.print(f":warning: {message}", style="warning")

    @staticmethod
    def header(title):
        """Print a styled header panel."""
        UI.console.print(Panel(title, expand=False, border_style="cyan"))

    @staticmethod
    def step(name):
        """Print a step indicator."""
        UI.console.print(f"[bold magenta]>>[/] [bold]{name}[/]")
