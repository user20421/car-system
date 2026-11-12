#pragma once
#include "concurrency/BoundedTaskQueue.h"
#include <QAtomicInt>
#include <QObject>
#include <QSemaphore>
#include <QTest>

class TestConcurrency final : public QObject {
    Q_OBJECT
private slots:
    void appliesBackpressure() {
        aurora::BoundedTaskQueue queue(1, 1);
        QSemaphore entered;
        QSemaphore release;
        QVERIFY(queue.trySubmit([&]() { entered.release(); release.acquire(); }));
        QVERIFY(entered.tryAcquire(1, 1000));
        QVERIFY(!queue.trySubmit([]() {}));
        release.release();
        queue.clearAndWait();
        QVERIFY(queue.trySubmit([]() {}));
        queue.clearAndWait();
    }
    void completesBoundedWork() {
        aurora::BoundedTaskQueue queue(2, 8);
        QAtomicInt count = 0;
        for (int i = 0; i < 8; ++i) QVERIFY(queue.trySubmit([&]() { count.fetchAndAddRelaxed(1); }));
        QTRY_COMPARE_WITH_TIMEOUT(count.loadRelaxed(), 8, 2000);
        queue.clearAndWait();
        QCOMPARE(count.loadRelaxed(), 8);
    }
};
