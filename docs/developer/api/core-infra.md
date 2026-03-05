# Core Infra API

Infrastructure abstract base classes and implementations.

---

## Executor Interface

`src/infra/executor/interface.hpp`

```cpp
class Executor {
public:
    virtual ~Executor() = default;

    virtual auto run(std::stop_token token, const std::string& cmd,
                     const std::vector<std::string>& args = {}) -> VoidResult = 0;

    virtual auto run_with_env(std::stop_token token, const EnvList& env,
                              const std::string& cmd,
                              const std::vector<std::string>& args = {}) -> VoidResult = 0;

    virtual auto run_in_dir(std::stop_token token, const std::string& dir,
                            const std::string& cmd,
                            const std::vector<std::string>& args = {}) -> VoidResult = 0;

    virtual auto output(std::stop_token token, const std::string& cmd,
                        const std::vector<std::string>& args = {}) -> Result<std::string> = 0;

    virtual auto look_path(const std::string& cmd) -> Result<std::string> = 0;

    virtual auto exec_replace(const std::string& cmd,
                              const std::vector<std::string>& args,
                              const EnvList& env) -> VoidResult = 0;
};
```

### Implementations

| Class           | Location                       | Purpose                  |
| --------------- | ------------------------------ | ------------------------ |
| `ShellExecutor` | `src/infra/executor/shell.cpp` | Real shell via fork/exec |
| `MockExecutor`  | `tests/mocks/mock.hpp`         | Test double              |

---

## FileSystem Interface

`src/infra/filesystem/interface.hpp`

```cpp
struct DirEntry {
    std::string name;
    bool is_directory;
    std::uintmax_t size;
};

class FileSystem {
public:
    virtual ~FileSystem() = default;

    virtual auto exists(const fs::path& path) const -> bool = 0;
    virtual auto is_dir(const fs::path& path) const -> bool = 0;
    virtual auto read_file(const fs::path& path) const -> Result<std::string> = 0;
    virtual auto write_file(const fs::path& path, std::string_view data,
                            fs::perms perm = fs::perms::owner_all) -> VoidResult = 0;
    virtual auto mkdir_all(const fs::path& path, fs::perms perm = fs::perms::owner_all) -> VoidResult = 0;
    virtual auto read_dir(const fs::path& path) const -> Result<std::vector<DirEntry>> = 0;
    virtual auto remove(const fs::path& path) -> VoidResult = 0;
    virtual auto remove_all(const fs::path& path) -> VoidResult = 0;
    virtual auto getwd() const -> Result<fs::path> = 0;
    virtual auto file_size(const fs::path& path) const -> Result<std::uintmax_t> = 0;
};
```

### Implementations

| Class            | Location                      | Purpose                |
| ---------------- | ----------------------------- | ---------------------- |
| `OSFileSystem`   | `src/infra/filesystem/os.cpp` | Real `std::filesystem` |
| `MockFileSystem` | `tests/mocks/mock.hpp`        | In-memory for tests    |

---

## Platform Interface

`src/infra/platform/interface.hpp`

```cpp
class Platform {
public:
    virtual ~Platform() = default;

    virtual auto name() const -> std::string = 0;
    virtual auto disk_image() -> DiskImageManager& = 0;
    virtual auto packages() -> PackageManager& = 0;
    virtual auto paths() -> PathProvider& = 0;
    virtual void set_executor(executor::Executor* exec) = 0;
};

auto create_platform(executor::Executor* exec = nullptr) -> std::unique_ptr<Platform>;
```

### Sub-Interfaces

- **DiskImageManager**: `create()`, `mount()`, `unmount()`, `is_mounted()`
- **PackageManager**: `is_installed()`, `install()`, `get_bin_path()`
- **PathProvider**: `workspace_root()`, `cache_dir()`, `toolchain_dir()`

### Implementations

| Platform | File                             |
| -------- | -------------------------------- |
| Linux    | `src/infra/platform/linux.cpp`   |
| macOS    | `src/infra/platform/darwin.cpp`  |
| Windows  | `src/infra/platform/windows.cpp` |

---

## Homebrew Resolver

`src/infra/homebrew/resolver.hpp` — macOS-only Homebrew formula resolution.

---

## Usage

Infra provides abstract base classes with pure virtual methods. Domain services receive pointers to these interfaces, enabling testability via mock implementations.