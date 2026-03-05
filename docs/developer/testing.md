# Testing

ELMOS uses [Catch2](https://github.com/catchorg/Catch2/) (v3) for unit testing with mock implementations for infra interfaces.

## Structure

- Test files in `tests/` directory
- Mock implementations in `src/infra/executor/mock.cpp`
- Catch2 `TEST_CASE` / `SECTION` based test organization

## Running Tests

```bash
# Via CMake
cmake --build build --target test

# Via Task
task test

# Verbose output
task test:verbose
```

## Mocking

Infra interfaces are mocked for domain testing:

```cpp
// src/infra/executor/mock.hpp
class MockExecutor : public Executor {
public:
    std::vector<CommandCall> calls;
    std::optional<Error> run_error;
    std::map<std::string, std::string> output_responses;
    std::map<std::string, Error> output_errors;
    std::map<std::string, std::string> look_path_responses;

    auto run(std::stop_token, const std::string& cmd,
             const std::vector<std::string>& args) -> VoidResult override;
    // ...
    void reset();
};
```

Used in domain tests to verify command calls without invoking real shell.

## Writing Tests

```cpp
// tests/test_example.cpp
#include <catch2/catch_test_macros.hpp>
#include <infra/executor/mock.hpp>

TEST_CASE("KernelBuilder builds with correct flags", "[builder]") {
    MockExecutor mock;
    // ... setup

    SECTION("default build") {
        auto result = builder.build(token, {});
        REQUIRE(result.has_value());
        REQUIRE(mock.calls.size() == 1);
        CHECK(mock.calls[0].cmd == "make");
    }

    SECTION("parallel build") {
        auto result = builder.build(token, {.jobs = 4});
        REQUIRE(result.has_value());
    }
}
```

## Integration Tests

Limited; focus on unit tests. Manual testing for full workflows via:
```bash
./build/bin/elmos doctor
```

## Best Practices

- Test public APIs and domain logic
- Use `MockExecutor` for all shell commands
- Use `SECTION` blocks for related test cases
- Verify both success and error paths
- Use `Result<T>` / `VoidResult` checks (not exceptions)