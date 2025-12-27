import json
import os
from ..config import Config


class ModuleState:
    """
    Manages the queue of modules to be installed or removed.
    Persists data to var/state/modules.json
    """

    def __init__(self):
        self.config = Config()
        self.state_file = os.path.join(
            self.config.project_root, "var", "state", "modules.json"
        )
        self.data = self._load()

    def _load(self):
        if os.path.exists(self.state_file):
            try:
                with open(self.state_file, "r") as f:
                    return json.load(f)
            except json.JSONDecodeError:
                return {"install": [], "remove": []}
        return {"install": [], "remove": []}

    def _save(self):
        # Ensure directory exists
        os.makedirs(os.path.dirname(self.state_file), exist_ok=True)
        with open(self.state_file, "w") as f:
            json.dump(self.data, f, indent=4)

    def add_install(self, module_name):
        if module_name not in self.data["install"]:
            self.data["install"].append(module_name)
            # Cannot install and remove same mod at once
            if module_name in self.data["remove"]:
                self.data["remove"].remove(module_name)
            self._save()

    def add_remove(self, module_name):
        if module_name not in self.data["remove"]:
            self.data["remove"].append(module_name)
            if module_name in self.data["install"]:
                self.data["install"].remove(module_name)
            self._save()

    def clear(self):
        """Clears all pending module actions."""
        self.data = {"install": [], "remove": []}
        self._save()

    def get_status(self):
        return self.data
