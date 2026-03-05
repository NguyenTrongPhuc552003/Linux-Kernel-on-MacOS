// ============================================================================
// domain/orchestrator/pipeline.cpp — DAG build pipeline
// ============================================================================

#include "pipeline.hpp"

#include <algorithm>
#include <queue>
#include <thread>

namespace elmos::domain::orchestrator {

void Pipeline::add_task(Task task) {
    index_[task.id] = tasks_.size();
    tasks_.push_back(std::move(task));
}

auto Pipeline::get_task(const std::string& id) -> Task* {
    auto it = index_.find(id);
    if (it == index_.end())
        return nullptr;
    return &tasks_[it->second];
}

auto Pipeline::get_all_tasks() const -> const std::vector<Task>& {
    return tasks_;
}

auto Pipeline::topological_order() -> Result<std::vector<std::string>> {
    std::unordered_map<std::string, int> in_degree;
    std::unordered_map<std::string, std::vector<std::string>> dependents;

    for (const auto& task : tasks_) {
        if (!in_degree.contains(task.id))
            in_degree[task.id] = 0;
        for (const auto& dep : task.depends_on) {
            dependents[dep].push_back(task.id);
            in_degree[task.id]++;
        }
    }

    std::queue<std::string> q;
    for (const auto& [id, deg] : in_degree) {
        if (deg == 0)
            q.push(id);
    }

    std::vector<std::string> order;
    while (!q.empty()) {
        auto id = q.front();
        q.pop();
        order.push_back(id);
        for (const auto& dep : dependents[id]) {
            if (--in_degree[dep] == 0) {
                q.push(dep);
            }
        }
    }

    if (order.size() != tasks_.size()) {
        return make_error(Error::generic("circular dependency detected in pipeline"));
    }
    return order;
}

auto Pipeline::find_ready_tasks() -> std::vector<size_t> {
    std::vector<size_t> ready;
    for (size_t i = 0; i < tasks_.size(); ++i) {
        if (tasks_[i].status == TaskStatus::Pending && all_deps_completed(tasks_[i])) {
            ready.push_back(i);
        }
    }
    return ready;
}

auto Pipeline::all_deps_completed(const Task& task) -> bool {
    for (const auto& dep : task.depends_on) {
        auto it = index_.find(dep);
        if (it == index_.end())
            return false;
        if (tasks_[it->second].status != TaskStatus::Completed)
            return false;
    }
    return true;
}

auto Pipeline::run(std::stop_token token, int max_parallel) -> VoidResult {
    if (max_parallel <= 0) {
        max_parallel = static_cast<int>(std::thread::hardware_concurrency());
        if (max_parallel <= 0)
            max_parallel = 4;
    }

    // Validate DAG
    auto order = topological_order();
    if (!order)
        return make_error(order.error());

    while (true) {
        if (token.stop_requested()) {
            return make_error(Error::generic("pipeline cancelled"));
        }

        auto ready = find_ready_tasks();
        if (ready.empty()) {
            // Check if all done
            bool all_done = true;
            for (const auto& task : tasks_) {
                if (task.status == TaskStatus::Pending || task.status == TaskStatus::Running) {
                    all_done = false;
                    break;
                }
            }
            if (all_done)
                break;

            // Check for stuck tasks (dependencies failed)
            bool has_pending = false;
            for (const auto& task : tasks_) {
                if (task.status == TaskStatus::Pending) {
                    has_pending = true;
                    // Check if any dep failed
                    for (const auto& dep : task.depends_on) {
                        auto it = index_.find(dep);
                        if (it != index_.end() && tasks_[it->second].status == TaskStatus::Failed) {
                            return make_error(Error::build("task '" + task.id +
                                                           "' blocked by failed dependency '" +
                                                           dep + "'"));
                        }
                    }
                }
            }
            if (!has_pending)
                break;
            continue;
        }

        // Execute ready tasks (sequentially for now; parallel in future)
        for (auto idx : ready) {
            if (token.stop_requested())
                break;
            tasks_[idx].status = TaskStatus::Running;
            auto r = tasks_[idx].execute(token);
            if (r) {
                tasks_[idx].status = TaskStatus::Completed;
            }
            else {
                tasks_[idx].status = TaskStatus::Failed;
                tasks_[idx].error_message = r.error().message();
                return make_error(
                    Error::build("task '" + tasks_[idx].id + "' failed: " + r.error().message()));
            }
        }
    }

    return {};
}

}  // namespace elmos::domain::orchestrator
