#include "controller/VehicleController.h"

#include <QJsonObject>
#include <QUuid>

namespace aurora {

VehicleController::VehicleController(QString vin, NetworkOptions options, QObject *parent)
    : QObject(parent), m_vin(std::move(vin)), m_tasks(2, 16, this) {
    options.vin = m_vin;
    m_latest.vin = m_vin;
    m_hardware = new SimulatedVehicleHardware(m_vin);
    m_hardware->moveToThread(&m_hardwareThread);
    m_hardwareThread.setObjectName(QStringLiteral("simulated-hal"));
    connect(&m_hardwareThread, &QThread::started, m_hardware, &SimulatedVehicleHardware::start);
    connect(&m_hardwareThread, &QThread::finished, m_hardware, &QObject::deleteLater);
    connect(m_hardware, &SimulatedVehicleHardware::telemetryProduced,
            this, &VehicleController::receiveTelemetry, Qt::QueuedConnection);
    connect(m_hardware, &SimulatedVehicleHardware::commandCompleted,
            this, &VehicleController::completeCommand, Qt::QueuedConnection);

    m_network = new DeviceNetwork(options, this);
    m_store = new TelemetryStore(m_vin, this);
    connect(this, &VehicleController::telemetryChanged, this, [this]() {
        m_network->publishLatest(m_latest);
        m_store->storeLatest(m_latest);
    });
    connect(m_network, &DeviceNetwork::networkStateChanged,
            this, &VehicleController::setNetworkState, Qt::QueuedConnection);
    connect(m_network, &DeviceNetwork::commandReceived,
            this, &VehicleController::receiveRemoteCommand, Qt::QueuedConnection);
    connect(m_network, &DeviceNetwork::ackDeliveryFailed, this, [this](const QString &, const QString &reason) {
        addEvent(QStringLiteral("ACK 投递失败：%1").arg(reason));
    }, Qt::QueuedConnection);
    connect(m_network, &DeviceNetwork::httpsCompleted, this,
            [this](const QString &operation, bool success, const QJsonObject &, const QString &error) {
        static const QHash<QString, QString> names = {{QStringLiteral("config"), QStringLiteral("配置同步")},
            {QStringLiteral("ota"), QStringLiteral("OTA 检查")}, {QStringLiteral("diagnostic"), QStringLiteral("诊断上传")},
            {QStringLiteral("batch"), QStringLiteral("离线补传")}};
        const QString name = names.value(operation, operation);
        setRequestMessage(false, success ? QStringLiteral("%1完成").arg(name)
                                         : QStringLiteral("%1失败：%2").arg(name, error));
        addEvent(m_requestMessage);
    }, Qt::QueuedConnection);
    connect(m_store, &TelemetryStore::storageFailed, this, [this](const QString &message) {
        addEvent(QStringLiteral("本地存储失败：%1").arg(message));
    }, Qt::QueuedConnection);
}

VehicleController::~VehicleController() { shutdown(); }

void VehicleController::start() {
    if (m_started) return;
    m_started = true;
    m_stopping = false;
    m_store->start();
    m_network->start();
    m_hardwareThread.start();
}

void VehicleController::shutdown() {
    if (!m_started || m_stopping) return;
    m_stopping = true;
    m_commands.clear();
    m_activeCommand.clear();
    if (m_hardwareThread.isRunning()) {
        QMetaObject::invokeMethod(m_hardware, &SimulatedVehicleHardware::stop, Qt::BlockingQueuedConnection);
        m_hardwareThread.quit();
        m_hardwareThread.wait(3000);
    }
    m_network->stop();
    m_store->stop();
    m_tasks.clearAndWait();
    m_started = false;
}

QStringList VehicleController::faults() const {
    QStringList result;
    for (const Fault &fault : m_latest.faults)
        if (fault.active) result << QStringLiteral("%1 · %2").arg(fault.code, fault.message);
    return result;
}

QString VehicleController::enqueueLocal(VehicleCommand::Type type, const QString &parameter) {
    if (m_stopping) return {};
    VehicleCommand command;
    command.commandId = QUuid::createUuid().toString(QUuid::WithoutBraces);
    command.idempotencyKey = command.commandId;
    command.vin = m_vin;
    command.type = type;
    command.parameter = parameter;
    command.createdAtMs = nowMs();
    command.expiresAtMs = command.createdAtMs + 15000;
    enqueue(command);
    return command.commandId;
}

void VehicleController::enqueue(VehicleCommand command) {
    if (command.vin != m_vin) {
        addEvent(QStringLiteral("已拒绝 VIN 不匹配的指令"));
        return;
    }
    if (m_knownCommands.contains(command.commandId)) return; // QoS 1 duplicate
    m_knownCommands.insert(command.commandId);
    m_knownOrder.append(command.commandId);
    while (m_knownOrder.size() > 512) m_knownCommands.remove(m_knownOrder.takeFirst());
    m_commands.enqueue(command);
    m_commandState = QStringLiteral("pending");
    m_commandMessage = QStringLiteral("指令已进入串行队列 · %1").arg(m_commands.size());
    emit commandChanged();
    processNextCommand();
}

void VehicleController::processNextCommand() {
    if (m_commandBusy || m_commands.isEmpty() || m_stopping) return;
    const VehicleCommand command = m_commands.dequeue();
    if (command.expiresAtMs > 0 && command.expiresAtMs <= nowMs()) {
        CommandAck ack;
        ack.commandId = command.commandId;
        ack.vin = m_vin;
        ack.status = CommandAck::Status::Expired;
        ack.errorCode = QStringLiteral("COMMAND_EXPIRED");
        ack.message = QStringLiteral("排队期间指令已过期");
        ack.occurredAtMs = nowMs();
        m_network->publishAck(ack);
        addEvent(ack.message);
        processNextCommand();
        return;
    }
    m_activeCommand = command.commandId;
    m_commandBusy = true;
    m_commandState = QStringLiteral("received");
    m_commandMessage = QStringLiteral("硬件正在执行 · %1").arg(command.commandId.left(8));
    emit commandChanged();
    CommandAck received;
    received.commandId = command.commandId;
    received.vin = m_vin;
    received.status = CommandAck::Status::Received;
    received.message = QStringLiteral("车辆已接收");
    received.occurredAtMs = nowMs();
    m_network->publishAck(received);
    QMetaObject::invokeMethod(m_hardware, [this, command]() { m_hardware->execute(command); }, Qt::QueuedConnection);
}

void VehicleController::completeCommand(const CommandAck &ack) {
    if (ack.commandId != m_activeCommand || ack.vin != m_vin) return;
    switch (ack.status) {
    case CommandAck::Status::Succeeded: m_commandState = QStringLiteral("succeeded"); break;
    case CommandAck::Status::Failed: m_commandState = QStringLiteral("failed"); break;
    case CommandAck::Status::Expired: m_commandState = QStringLiteral("expired"); break;
    case CommandAck::Status::Received: return;
    }
    m_commandBusy = false;
    m_commandMessage = ack.message;
    m_activeCommand.clear();
    emit commandChanged();
    addEvent(QStringLiteral("%1 · %2").arg(m_commandState, ack.message));
    m_network->publishAck(ack);
    processNextCommand();
}

void VehicleController::receiveTelemetry(const Telemetry &telemetry) {
    if (telemetry.vin != m_vin || (telemetry.bootId == m_latest.bootId &&
        telemetry.sequence < m_latest.sequence)) return;
    const bool hadFault = !m_latest.faults.isEmpty();
    m_latest = telemetry;
    emit telemetryChanged();
    if (!hadFault && !telemetry.faults.isEmpty()) {
        addEvent(telemetry.faults.first().message);
        emit criticalFault(telemetry.faults.first().message);
    }
}

void VehicleController::receiveRemoteCommand(const VehicleCommand &command) { enqueue(command); }

QString VehicleController::lockVehicle() { return enqueueLocal(VehicleCommand::Type::Lock); }
QString VehicleController::unlockVehicle() { return enqueueLocal(VehicleCommand::Type::Unlock); }
QString VehicleController::setClimate(bool on) { return enqueueLocal(on ? VehicleCommand::Type::HvacOn : VehicleCommand::Type::HvacOff); }
QString VehicleController::setTemperature(double value) { return enqueueLocal(VehicleCommand::Type::SetTemperature, QString::number(value, 'f', 1)); }
QString VehicleController::setDefrost(bool on) { return enqueueLocal(on ? VehicleCommand::Type::DefrostOn : VehicleCommand::Type::DefrostOff); }
QString VehicleController::honkFlash() { return enqueueLocal(VehicleCommand::Type::HonkFlash); }

void VehicleController::setScenario(const QString &scenario) {
    m_scenario = scenario;
    emit scenarioChanged();
    addEvent(QStringLiteral("调试场景：%1").arg(scenario));
    QMetaObject::invokeMethod(m_hardware, [this, scenario]() { m_hardware->setScenario(scenario); }, Qt::QueuedConnection);
}

void VehicleController::fetchConfiguration() { setRequestMessage(true, QStringLiteral("正在同步车辆配置…")); m_network->fetchConfiguration(); }
void VehicleController::checkForUpdate() { setRequestMessage(true, QStringLiteral("正在通过 HTTPS/2 检查 OTA…")); m_network->checkForUpdate(QStringLiteral("1.0.0")); }
void VehicleController::sendDiagnostic() {
    setRequestMessage(true, QStringLiteral("正在上传诊断摘要…"));
    m_network->uploadDiagnostic({{QStringLiteral("vin"), m_vin}, {QStringLiteral("sequence"), QString::number(m_latest.sequence)},
                                 {QStringLiteral("battery_percent"), m_latest.batteryPercent}, {QStringLiteral("faults"), faults().join(';')}});
}
void VehicleController::retryBatch() { setRequestMessage(true, QStringLiteral("正在准备 Protobuf 离线批次…")); m_network->uploadBatch(QUuid::createUuid().toString(QUuid::WithoutBraces)); }

void VehicleController::setNetworkState(const QString &state, const QString &detail) {
    m_networkState = state;
    m_networkDetail = detail;
    emit networkChanged();
}
void VehicleController::setRequestMessage(bool busy, const QString &message) {
    m_requestBusy = busy;
    m_requestMessage = message;
    emit requestChanged();
}
void VehicleController::addEvent(const QString &event) {
    m_recentEvents.prepend(QStringLiteral("%1  %2").arg(QTime::currentTime().toString("HH:mm:ss"), event));
    while (m_recentEvents.size() > 8) m_recentEvents.removeLast();
    emit eventsChanged();
}

} // namespace aurora
