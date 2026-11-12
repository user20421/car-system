#pragma once

#include <QObject>
#include <QSemaphore>
#include <QThreadPool>
#include <functional>

namespace aurora {

// A bounded admission queue for short non-UI jobs. trySubmit never blocks a
// caller and reports backpressure instead of growing memory without limit.
class BoundedTaskQueue final : public QObject {
    Q_OBJECT
public:
    explicit BoundedTaskQueue(int maxThreads = 2, int maxPending = 16,
                              QObject *parent = nullptr);
    ~BoundedTaskQueue() override;
    bool trySubmit(std::function<void()> task);
    int activeCount() const;
    int maxPending() const;
    void clearAndWait();

private:
    QThreadPool m_pool;
    QSemaphore m_slots;
    const int m_maxPending;
};

} // namespace aurora
