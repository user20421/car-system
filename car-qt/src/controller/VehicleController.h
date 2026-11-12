#pragma once

#include "concurrency/BoundedTaskQueue.h"
#include "core/VehicleTypes.h"
#include "hal/SimulatedVehicleHardware.h"
#include "network/DeviceNetwork.h"
#include "storage/TelemetryStore.h"

#include <QObject>
#include <QQueue>
#include <QSet>
#include <QThread>

namespace aurora {

class VehicleController final : public QObject {
    Q_OBJECT
    Q_PROPERTY(QString vin READ vin CONSTANT)
    Q_PROPERTY(double speed READ speed NOTIFY telemetryChanged)
    Q_PROPERTY(int rpm READ rpm NOTIFY telemetryChanged)
    Q_PROPERTY(double battery READ battery NOTIFY telemetryChanged)
    Q_PROPERTY(double range READ range NOTIFY telemetryChanged)
    Q_PROPERTY(double cabinTemperature READ cabinTemperature NOTIFY telemetryChanged)
    Q_PROPERTY(double targetTemperature READ targetTemperature NOTIFY telemetryChanged)
    Q_PROPERTY(double latitude READ latitude NOTIFY telemetryChanged)
    Q_PROPERTY(double longitude READ longitude NOTIFY telemetryChanged)
    Q_PROPERTY(double heading READ heading NOTIFY telemetryChanged)
    Q_PROPERTY(bool locked READ locked NOTIFY telemetryChanged)
    Q_PROPERTY(bool hvacOn READ hvacOn NOTIFY telemetryChanged)
    Q_PROPERTY(bool defrost READ defrost NOTIFY telemetryChanged)
    Q_PROPERTY(int fanLevel READ fanLevel NOTIFY telemetryChanged)
    Q_PROPERTY(QStringList faults READ faults NOTIFY telemetryChanged)
    Q_PROPERTY(QString scenario READ scenario NOTIFY scenarioChanged)
    Q_PROPERTY(QString networkState READ networkState NOTIFY networkChanged)
    Q_PROPERTY(QString networkDetail READ networkDetail NOTIFY networkChanged)
    Q_PROPERTY(bool commandBusy READ commandBusy NOTIFY commandChanged)
    Q_PROPERTY(QString commandState READ commandState NOTIFY commandChanged)
    Q_PROPERTY(QString commandMessage READ commandMessage NOTIFY commandChanged)
    Q_PROPERTY(bool requestBusy READ requestBusy NOTIFY requestChanged)
    Q_PROPERTY(QString requestMessage READ requestMessage NOTIFY requestChanged)
    Q_PROPERTY(QStringList recentEvents READ recentEvents NOTIFY eventsChanged)

public:
    explicit VehicleController(QString vin, NetworkOptions options, QObject *parent = nullptr);
    ~VehicleController() override;

    void start();
    void shutdown();
    QString vin() const { return m_vin; }
    double speed() const { return m_latest.speedKph; }
    int rpm() const { return m_latest.rpm; }
    double battery() const { return m_latest.batteryPercent; }
    double range() const { return m_latest.rangeKm; }
    double cabinTemperature() const { return m_latest.cabinTempC; }
    double targetTemperature() const { return m_latest.hvac.targetTempC; }
    double latitude() const { return m_latest.location.latitude; }
    double longitude() const { return m_latest.location.longitude; }
    double heading() const { return m_latest.location.headingDeg; }
    bool locked() const { return m_latest.locked; }
    bool hvacOn() const { return m_latest.hvac.mode != HvacState::Mode::Off; }
    bool defrost() const { return m_latest.defrost; }
    int fanLevel() const { return m_latest.hvac.fanLevel; }
    QStringList faults() const;
    QString scenario() const { return m_scenario; }
    QString networkState() const { return m_networkState; }
    QString networkDetail() const { return m_networkDetail; }
    bool commandBusy() const { return m_commandBusy; }
    QString commandState() const { return m_commandState; }
    QString commandMessage() const { return m_commandMessage; }
    bool requestBusy() const { return m_requestBusy; }
    QString requestMessage() const { return m_requestMessage; }
    QStringList recentEvents() const { return m_recentEvents; }

    Q_INVOKABLE QString lockVehicle();
    Q_INVOKABLE QString unlockVehicle();
    Q_INVOKABLE QString setClimate(bool on);
    Q_INVOKABLE QString setTemperature(double celsius);
    Q_INVOKABLE QString setDefrost(bool on);
    Q_INVOKABLE QString honkFlash();
    Q_INVOKABLE void setScenario(const QString &scenario);
    Q_INVOKABLE void fetchConfiguration();
    Q_INVOKABLE void checkForUpdate();
    Q_INVOKABLE void sendDiagnostic();
    Q_INVOKABLE void retryBatch();

public slots:
    void receiveTelemetry(const aurora::Telemetry &telemetry);
    void receiveRemoteCommand(const aurora::VehicleCommand &command);
    void completeCommand(const aurora::CommandAck &ack);

signals:
    void telemetryChanged();
    void scenarioChanged();
    void networkChanged();
    void commandChanged();
    void requestChanged();
    void eventsChanged();
    void criticalFault(const QString &message);

private:
    QString enqueueLocal(VehicleCommand::Type type, const QString &parameter = {});
    void enqueue(VehicleCommand command);
    void processNextCommand();
    void addEvent(const QString &event);
    void setNetworkState(const QString &state, const QString &detail);
    void setRequestMessage(bool busy, const QString &message);

    const QString m_vin;
    QThread m_hardwareThread;
    SimulatedVehicleHardware *m_hardware = nullptr;
    DeviceNetwork *m_network = nullptr;
    TelemetryStore *m_store = nullptr;
    BoundedTaskQueue m_tasks;
    Telemetry m_latest;
    QQueue<VehicleCommand> m_commands;
    QSet<QString> m_knownCommands;
    QStringList m_knownOrder;
    QString m_activeCommand;
    QString m_scenario = QStringLiteral("city");
    QString m_networkState = QStringLiteral("offline");
    QString m_networkDetail = QStringLiteral("网络服务启动中");
    bool m_commandBusy = false;
    QString m_commandState = QStringLiteral("idle");
    QString m_commandMessage = QStringLiteral("车辆控制已就绪");
    bool m_requestBusy = false;
    QString m_requestMessage = QStringLiteral("尚未请求设备服务");
    QStringList m_recentEvents;
    bool m_started = false;
    bool m_stopping = false;
};

} // namespace aurora
