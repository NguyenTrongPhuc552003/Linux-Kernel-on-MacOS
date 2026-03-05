#pragma once
// ============================================================================
// domain/orchestrator/pipeline.hpp — DAG build pipeline
// ============================================================================

#include <elmos/common.hpp>

#include <functional>
#include <memory>
#include <stop_token>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <vector>

namespace elmos::domain::orchestrator {

enum class TaskStatus { Pending, Running, Completed, Failed, Skipped };

struct Task {
    std::string id;
    std::vector<std::string> depends_on;
    std::function<VoidResult(std::stop_token)> execute;
    TaskStatus status = TaskStatus::Pending;
    std::string error_message;
};

/// DAG-based build pipeline. Tasks run in dependency order;
/// independent branches execute in parallel.
class Pipeline {
public:
    void add_task(Task task);
    auto run(std::stop_token token, int max_parallel = 0) -> VoidResult;
    auto get_task(const std::string& id) -> Task*;
    auto get_all_tasks() const -> const std::vector<Task>&;
    auto topological_order() -> Result<std::vector<std::string>>;

private:
    std::vector<Task> tasks_;
    std::unordered_map<std::string, size_t> index_;

    auto find_ready_tasks() -> std::vector<size_t>;
    auto all_deps_completed(const Task& task) -> bool;
};

}  // namespace elmos::domain::orchestrator
