#include "concurrency/BoundedTaskQueue.h"
#include <QRunnable>

namespace aurora {
namespace {
class FunctionTask final : public QRunnable {
public:
    explicit FunctionTask(std::function<void()> fn) : m_fn(std::move(fn)) { setAutoDelete(true); }
    void run() override { m_fn(); }
private:
    std::function<void()> m_fn;
};
}

BoundedTaskQueue::BoundedTaskQueue(int maxThreads, int maxPending, QObject *parent)
    : QObject(parent), m_slots(maxPending), m_maxPending(maxPending) {
    m_pool.setMaxThreadCount(qMax(1, maxThreads));
    m_pool.setExpiryTimeout(30000);
}

BoundedTaskQueue::~BoundedTaskQueue() { clearAndWait(); }

bool BoundedTaskQueue::trySubmit(std::function<void()> task) {
    if (!task || !m_slots.tryAcquire(1)) return false;
    m_pool.start(new FunctionTask([this, task = std::move(task)]() mutable {
        task();
        m_slots.release(1);
    }));
    return true;
}

int BoundedTaskQueue::activeCount() const { return m_pool.activeThreadCount(); }
int BoundedTaskQueue::maxPending() const { return m_maxPending; }
void BoundedTaskQueue::clearAndWait() {
    m_pool.clear();
    m_pool.waitForDone();
    // Cleared tasks never ran their release callback.
    m_slots.release(m_maxPending - m_slots.available());
}

} // namespace aurora
