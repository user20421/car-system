#pragma once

#include "core/VehicleTypes.h"
#include <QJsonObject>
#include <QObject>
#include <QThread>

namespace aurora {

struct NetworkOptions {
    QString vin;
    QString mqttHost = QStringLiteral("localhost");
    quint16 mqttPort = 8883;
    QString httpsBaseUrl;
    QString certificateDirectory;
};

class DeviceNetworkWorker;

class DeviceNetwork final : public QObject {
    Q_OBJECT
public:
    explicit DeviceNetwork(NetworkOptions options, QObject *parent = nullptr);
    ~DeviceNetwork() override;
    void start();
    void stop();

public slots:
    void publishLatest(const aurora::Telemetry &telemetry);
    void publishAck(const aurora::CommandAck &ack);
    void fetchConfiguration();
    void checkForUpdate(const QString &currentVersion);
    void uploadDiagnostic(const QJsonObject &diagnostic);
    void uploadBatch(const QString &batchId);

signals:
    void networkStateChanged(const QString &state, const QString &detail);
    void commandReceived(const aurora::VehicleCommand &command);
    void httpsCompleted(const QString &operation, bool success,
                        const QJsonObject &body, const QString &error);
    void ackDeliveryFailed(const QString &commandId, const QString &reason);

private:
    NetworkOptions m_options;
    QThread m_thread;
    DeviceNetworkWorker *m_worker = nullptr;
};

} // namespace aurora
