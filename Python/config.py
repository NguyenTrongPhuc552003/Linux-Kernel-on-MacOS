import os


class Config:
    """
    Manages the 'build.cfg' file in var/state.
    Parses shell-style variable assignments (export KEY="VAL").
    """

    _instance = None

    def __new__(cls):
        if cls._instance is None:
            cls._instance = super(Config, cls).__new__(cls)
            cls._instance._load()
        return cls._instance

    def _load(self):
        # Resolve path relative to project root
        self.project_root = os.environ.get("PROJECT_ROOT", os.getcwd())
        self.config_path = os.path.join(self.project_root, "var", "state", "build.cfg")
        self.data = {}

        if os.path.exists(self.config_path):
            with open(self.config_path, "r") as f:
                for line in f:
                    # simplistic parsing of 'export KEY="VALUE"'
                    line = line.strip()
                    if line.startswith("export") and "=" in line:
                        parts = line.replace("export ", "").split("=", 1)
                        key = parts[0].strip()
                        val = parts[1].strip().strip('"').strip("'")
                        self.data[key] = val

    def get(self, key, default=None):
        return self.data.get(key, default)

    def set(self, key, value):
        self.data[key] = value
        self._save()

    def _save(self):
        # Write back in a format Bash can 'source'
        with open(self.config_path, "w") as f:
            for key, val in self.data.items():
                f.write(f'export {key}="{val}"\n')
