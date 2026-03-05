# ELMOS Testing Strategy

This directory contains comprehensive unit and integration tests for the ELMOS Embedded Linux SDK.

## Testing Framework: Catch2 3

**Why Catch2?**
- Header-only (included via CMake's FetchContent)
- Excellent assertion messages
- Simple lambda-based test definition
- Built-in parameterized tests
- Works with C++20 and C++23
- No external dependencies

**Documentation:** https://github.com/catchorg/Catch2

## Test Organization

```
tests/
├── CMakeLists.txt                    # Test build configuration
├── unit/                             # Unit tests (fast, isolated)
│   ├── config_loader_tests.cpp       # Config loading and validation
│   ├── context_tests.cpp             # Build context management
│   ├── executor_tests.cpp            # Command execution abstraction
│   ├── filesystem_tests.cpp          # Filesystem I/O abstraction
│   └── platform_tests.cpp            # Platform detection and abstraction
├── integration/                      # Integration tests (slower, full stack)
│   └── kernel_build_integration_tests.cpp  # End-to-end kernel build
└── README.md                         # This file
```

## Test Categories

### Unit Tests (`tests/unit/`)

**Purpose:** Test individual components in isolation

**Characteristics:**
- Use **dependency injection** to inject mocks
- Test single class/function behavior
- Execute in <1ms per test
- Should not access filesystem, network, or execute commands

**Key Testing Patterns:**

1. **Mock Executor** — Test builders without running real commands
   ```cpp
   auto mock = std::make_unique<MockExecutor>();
   auto builder = KernelBuilder(context.get(), mock.get());
   // Test with mocked commands
   ```

2. **Temporary Directories** — Test filesystem operations safely
   ```cpp
   // Create temp dir with RAII cleanup
   // Test read/write operations
   ```

3. **Result Type Assertions** — Test error handling
   ```cpp
   auto result = load_config("path");
   REQUIRE(!result); // Should fail
   REQUIRE_THAT(result.error().message(), 
                ContainsSubstring("not found"));
   ```

### Integration Tests (`tests/integration/`)

**Purpose:** Test component interactions and full workflows

**Characteristics:**
- Test multiple components together
- Can be slower (seconds to minutes)
- Simulate real scenarios with real dependencies
- Execute in isolation (no side effects on system)

**Key Test Scenarios:**
- Complete kernel build workflow
- Configuration loading and validation
- Plugin system initialization and hook execution
- Cross-platform differences
- Error recovery and rollback

## Running Tests

### Build project with tests
```bash
cmake --preset default
cmake --build build --parallel
```

### Run all tests
```bash
cmake --build build --target test
# or
ctest --test-dir build --output-on-failure
```

### Run specific test file
```bash
ctest --test-dir build -R config_loader --output-on-failure
```

### Run with verbose output
```bash
ctest --test-dir build -VV
```

### Run test executable directly
```bash
./build/bin/elmos_unit_tests
./build/bin/elmos_integration_tests
```

### Run with test filter
```bash
./build/bin/elmos_unit_tests "[executor]"
./build/bin/elmos_unit_tests "[integration]"
```

## Test Structure Template

Each test file follows this pattern:

```cpp
#include <catch2/catch_test_macros.hpp>
// ... other includes

TEST_CASE("Feature name — description", "[tags]") {
    SECTION("Scenario 1") {
        // Given: setup
        // When: action
        // Then: assertion
        
        REQUIRE(condition);
    }
    
    SECTION("Scenario 2") {
        // Isolated section — setup repeated
        
        REQUIRE(other_condition);
    }
}
```

## Dependency Injection for Testability

**Key Pattern:** Domain classes accept infrastructure as pointer parameters

```cpp
// Testable — accepts Executor* dependency
class KernelBuilder {
    KernelBuilder(context::Context* ctx, infra::executor::Executor* exec);
};

// Usage in tests:
auto mock_exec = std::make_unique<infra::executor::MockExecutor>();
auto kernel = KernelBuilder(context.get(), mock_exec.get());
```

**Benefits:**
- Easy to swap real implementations with mocks
- No hidden dependencies
- Clear what each component needs
- Improves code testability

## Test Naming Conventions

**Test Suite Name Format:**
```
"Feature name — description", "[category][subcategory]"
```

**Examples:**
```cpp
TEST_CASE("Config Loader — YAML parsing", "[config][loader]")
TEST_CASE("Kernel Builder — Cross-compilation", "[domain][builder][arch]")
TEST_CASE("Platform Abstraction — macOS specific", "[infra][platform][darwin]")
```

**Tags:**
- `[unit]` — Unit tests (fast)
- `[integration]` — Integration tests (slower)
- `[config]` — Configuration layer
- `[context]` — Build context
- `[domain]` — Domain business logic
- `[infra]` — Infrastructure layer
- `[executor]` — Command execution
- `[filesystem]` — File I/O
- `[platform]` — OS abstraction
- `[kernel]` — Kernel-specific
- `[arch]` — Architecture-specific
- `[arm]`, `[aarch64]`, `[riscv]` — Specific architectures
- `[errors]` — Error handling scenarios
- `[mock]` — Tests using mocks

## Assertions and Matchers

### Basic Assertions
```cpp
REQUIRE(condition);           // Test fails if false
REQUIRE(!result);             // Negation
REQUIRE(value == expected);   // Equality
REQUIRE(value > 5);           // Comparisons
```

### Result Type Assertions
```cpp
auto result = some_operation();
REQUIRE(result);                    // Check if success
REQUIRE_FALSE(result);              // Check if error
REQUIRE(result.value() == expected);  // Access value
REQUIRE(!result);                   // Check if error
```

### String Matchers
```cpp
using namespace Catch::Matchers;

REQUIRE_THAT(text, ContainsSubstring("keyword"));
REQUIRE_THAT(text, StartsWith("prefix"));
REQUIRE_THAT(text, EndsWith("suffix"));
REQUIRE_THAT(text, Matches(regex));
```

## Test Coverage Goals

**Targets:**
- Infrastructure layer: 80%+ coverage
  - Executor, Filesystem, Platform
  - These are most critical for robustness
  
- Domain layer: 70%+ coverage
  - Builder, Orchestrator, BSP registry
  - Plugin system
  
- Configuration layer: 90%+ coverage
  - Validation is critical
  
- Application layer: 60%+
  - CLI command wiring less critical

**Excluded from coverage:**
- `main.cpp` (binary entry point)
- Third-party libraries
- Platform-specific code (Windows tests run on Windows only)

## Mocking Strategy

### Mock Executor Pattern
```cpp
class MockExecutor : public infra::executor::Executor {
    std::map<std::string, VoidResult> results;
    
public:
    void set_result(const std::string& cmd, VoidResult result) {
        results[cmd] = result;
    }
    
    VoidResult run(std::stop_token, const std::string& cmd,
                   const std::vector<std::string>&) override {
        return results[cmd];
    }
};
```

### Usage in Tests
```cpp
SECTION("Handle compilation failure") {
    auto mock = std::make_unique<MockExecutor>();
    mock->set_result("make", 
        std::unexpected(Error::generic("compilation failed")));
    
    auto builder = KernelBuilder(context.get(), mock.get());
    auto result = builder.build(token, opts);
    
    REQUIRE(!result);  // Should fail
}
```

## Continuous Integration

Tests run automatically on:
- Every commit (pre-commit hook)
- Every pull request
- Before release builds

**CI Configuration:** See `.github/workflows/` (when created)

## Future Test Expansion

Placeholder test files are prepared for:
- Kernel module builder tests
- Rootfs builder tests
- QEMU emulator tests
- Plugin loader and hook executor tests
- Cross-architecture compilation tests
- Patch application tests
- Fingerprint and caching tests

Each placeholder test uses the pattern:
```cpp
SECTION("Feature description") {
    // Given: setup description
    // When: action description
    // Then: assertion description
    
    REQUIRE(true); // Placeholder for actual test
}
```

To implement:
1. Replace `REQUIRE(true)` with actual assertions
2. Add necessary setup in Given section
3. Add actual action code in When section
4. Add real assertions in Then section

## Troubleshooting Tests

### Test discovery not working
```bash
# Force CMake rebuild
rm -rf build/
cmake --preset default
cmake --build build --parallel
```

### Test executable not found
```bash
# Check build output
cmake --build build --verbose

# Look for elmos_unit_tests and elmos_integration_tests binaries
find build -name "elmos_*_tests"
```

### Individual test failing
```bash
# Run with verbose output
./build/bin/elmos_unit_tests -vv "[failing_test_name]"

# Get stack trace
./build/bin/elmos_unit_tests -vv "[failing_test_name]" --result-format=xml
```

### Mock executor not behaving
- Verify command name matches exactly
- Check stop_token isn't being provided (affects behavior)
- Ensure mock is injected to correct dependency

## Contributing Tests

When adding new features:

1. **Write tests first** (TDD approach)
   - Define behavior in tests
   - Implement to pass tests

2. **Follow existing patterns**
   - Use SECTION for scenarios
   - Include Given/When/Then comments
   - Use appropriate tags

3. **Test error paths**
   - What happens on failure?
   - Are error messages helpful?
   - Can user recover?

4. **Consider mocking strategy**
   - Unit tests: mock external dependencies
   - Integration tests: use real implementations
   - Avoid testing test infrastructure

## Performance Considerations

**Unit Tests Target:** <50ms total
**Integration Tests Target:** <5s total

Optimize slow tests by:
- Using mocks instead of real executables
- Reducing test data size
- Parallelizing independent tests
- Skipping integration tests in fast-path CI

## References

- [Catch2 Documentation](https://github.com/catchorg/Catch2)
- [C++ Testing Best Practices](https://github.com/catchorg/Catch2/blob/devel/docs/test-cases-and-sections.md)
- [Dependency Injection in C++](https://www.cppstories.com/2016/03/dependency-injection-in-cpp.html)
- [Result Type (std::expected)](https://cplusplus.github.io/CWG/lwg-issues/3629.html)

---

**Maintainer's Note:** This testing infrastructure is designed to catch regressions early and ensure ELMOS remains stable across platforms and use cases. Invest time in good tests now to save debugging time later!
