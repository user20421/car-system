#pragma once

#include "core/VehicleTypes.h"
#include <QObject>
#include <QThread>

namespace aurora {

class TelemetryStoreWorker;

// Facade lives on the GUI thread. All file I/O is serialized on its worker.
class TelemetryStore final : public QObject {
    Q_OBJECT
public:
    explicit TelemetryStore(QString vin, QObject *parent = nullptr);
    ~TelemetryStore() override;
    void start();
    void stop();

public slots:
    void storeLatest(const aurora::Telemetry &telemetry);

signals:
    void storageFailed(const QString &message);

private:
    QThread m_thread;
    TelemetryStoreWorker *m_worker = nullptr;
};

} // namespace aurora
