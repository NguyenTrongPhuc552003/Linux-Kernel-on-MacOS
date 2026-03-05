# Contributing

Guidelines for contributing to ELMOS.

---

## Quick Start

```bash
# Clone
git clone https://github.com/NguyenTrongPhuc552003/elmos.git
cd elmos

# Install dependencies (Ubuntu/Debian)
sudo apt install libcli11-dev libyaml-cpp-dev nlohmann-json3-dev \
    libspdlog-dev libssl-dev catch2 libcpp-httplib-dev \
    pkg-config ninja-build

# Build
cmake --preset default
cmake --build build --parallel

# Test
cmake --build build --target test

# Verify
./build/bin/elmos doctor
```

---

## Commit Convention

Use the project's commit message format:

```
<scope>: <Title>

- Change 1
- Change 2

Signed-off-by: Your Name <email@example.com>
```

**Scope examples:**

| Scope                  | Example                |
| ---------------------- | ---------------------- |
| `src: domain: builder` | Kernel builder changes |
| `src: app: commands`   | CLI command changes    |
| `docs: user: qemu`     | User doc updates       |
| `cmake`                | Build system changes   |
| `tests`                | Test changes           |

**Examples:**

```bash
git commit -sm "src: domain: emulator: Add machine validation

- Add validate_machine method
- Improve error messages"

git commit -sm "docs: user: kernel-building: Add troubleshooting table

- Add common issues and solutions"
```

---

## Pull Request Process

1. **Branch from `main`**
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Make changes with tests**
   ```bash
   cmake --build build --parallel
   cmake --build build --target test
   ```

3. **Push and create PR**
   ```bash
   git push origin feature/my-feature
   ```

4. **PR requirements:**
   - Descriptive title
   - Link related issues
   - CI must pass
   - One maintainer approval

---

## Code Standards

### C++ Style

- C++23 standard (GCC 13+)
- Use `auto` for return types on method declarations
- Use `std::expected<T, Error>` for error handling (no exceptions)
- Constructor injection via raw pointers (no ownership transfer)
- Use `snake_case` for functions/variables, `PascalCase` for types
- Prefer designated initializers for struct construction

### Testing

- Catch2 v3 framework
- Mock infrastructure interfaces (Executor, FileSystem)
- Test domain logic in isolation

```bash
cmake --build build --target test
```

---

## Documentation

- Update user docs for CLI changes
- Update developer API docs for interface changes
- Add code comments only where logic isn't self-evident

---

## Issue Guidelines

### Bug Reports

Include:

- ELMOS version (`./build/bin/elmos version`)
- OS and compiler version
- Steps to reproduce
- Expected vs actual behavior

### Feature Requests

Include:

- Use case description
- Proposed solution
- Alternatives considered

---

## Code Review Checklist

- [ ] `cmake --build build --parallel` produces zero errors/warnings
- [ ] `./build/bin/elmos doctor` runs without crashes
- [ ] No `system()` or `popen()` calls in domain or plugin code
- [ ] No hardcoded OS-specific paths; use platform layer
- [ ] All errors via `Result<T>` / `VoidResult`, not exceptions
- [ ] New CLI commands wired in `commands.hpp`
- [ ] New plugins registered in builtin factory map

---

## License

All contributions are under MIT license.