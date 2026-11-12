#include "hal/SimulatedVehicleHardware.h"

#include <QRandomGenerator>
#include <QUuid>
#include <QtMath>
#include <algorithm>

namespace aurora {

SimulatedVehicleHardware::SimulatedVehicleHardware(QString vin, QObject *parent)
    : IVehicleHardware(parent), m_vin(std::move(vin)),
      m_bootId(QUuid::createUuid().toString(QUuid::WithoutBraces)) {
    m_state.vin = m_vin;
    m_state.bootId = m_bootId;
    m_state.batteryPercent = 82.4;
    m_state.rangeKm = 438.0;
    m_state.cabinTempC = 26.4;
    m_state.hvac.targetTempC = 22.0;
    m_state.hvac.mode = HvacState::Mode::Auto;
    m_state.hvac.acOn = true;
    m_state.occurredAtMs = nowMs();
}

void SimulatedVehicleHardware::start() {
    if (m_running) return;
    m_running = true;
    m_lastTickMs = nowMs();
    m_clock.start();
    m_timer = new QTimer(this);
    m_timer->setInterval(100);
    m_timer->setTimerType(Qt::PreciseTimer);
    connect(m_timer, &QTimer::timeout, this, &SimulatedVehicleHardware::tick);
    m_timer->start();
    tick();
}

void SimulatedVehicleHardware::stop() {
    m_running = false;
    if (m_timer) {
        m_timer->stop();
        m_timer->deleteLater();
        m_timer = nullptr;
    }
}

void SimulatedVehicleHardware::setScenario(const QString &scenario) {
    static const QStringList supported = {QStringLiteral("city"), QStringLiteral("highway"),
        QStringLiteral("parked"), QStringLiteral("charging"), QStringLiteral("fault")};
    m_scenario = supported.contains(scenario) ? scenario : QStringLiteral("city");
    m_state.faults.clear();
    if (m_scenario == QStringLiteral("fault")) {
        m_state.faults.append({QStringLiteral("BMS_A017"), QStringLiteral("WARNING"),
                               QStringLiteral("电池温差偏高，已限制功率"), true});
    }
    tick();
}

void SimulatedVehicleHardware::tick() {
    if (!m_running) return;
    const qint64 current = nowMs();
    const double dt = qBound(0.0, (current - m_lastTickMs) / 1000.0, 0.5);
    m_lastTickMs = current;
    const double t = m_clock.elapsed() / 1000.0;
    double targetSpeed = 0.0;
    if (m_scenario == QStringLiteral("city"))
        targetSpeed = qMax(0.0, 42.0 + 25.0 * qSin(t * 0.19) + 10.0 * qSin(t * 0.71));
    else if (m_scenario == QStringLiteral("highway"))
        targetSpeed = 105.0 + 12.0 * qSin(t * 0.12);
    else if (m_scenario == QStringLiteral("fault"))
        targetSpeed = 28.0 + 8.0 * qSin(t * 0.2);

    const double response = 1.0 - qExp(-dt * (targetSpeed > m_state.speedKph ? 0.85 : 1.5));
    m_state.speedKph += (targetSpeed - m_state.speedKph) * response;
    if (m_state.speedKph < 0.05) m_state.speedKph = 0.0;
    m_state.rpm = m_state.speedKph < 0.1 ? 0 : qBound(650, int(720 + m_state.speedKph * 43), 7200);
    m_distance += m_state.speedKph * dt / 3600.0;

    if (m_scenario == QStringLiteral("charging")) {
        m_state.speedKph = 0.0;
        m_state.rpm = 0;
        m_state.batteryPercent = qMin(100.0, m_state.batteryPercent + dt * 0.012);
    } else {
        const double consumption = dt * (0.00035 + m_state.speedKph * 0.000004 +
                                         (m_state.hvac.mode != HvacState::Mode::Off ? 0.00025 : 0.0));
        m_state.batteryPercent = qMax(0.0, m_state.batteryPercent - consumption);
    }
    m_state.rangeKm = m_state.batteryPercent * 5.31;

    if (m_state.hvac.mode != HvacState::Mode::Off) {
        const double rate = m_state.defrost ? 0.12 : 0.045;
        m_state.cabinTempC += (m_state.hvac.targetTempC - m_state.cabinTempC) * qMin(1.0, dt * rate);
        m_state.hvac.fanLevel = qBound(1, int(1 + qAbs(m_state.cabinTempC - m_state.hvac.targetTempC)), 7);
    } else {
        m_state.cabinTempC += (26.0 - m_state.cabinTempC) * qMin(1.0, dt * 0.005);
        m_state.hvac.fanLevel = 0;
    }
    const double travel = m_distance * 0.002;
    m_state.location.latitude = 31.2304 + qSin(travel) * 0.018;
    m_state.location.longitude = 121.4737 + qCos(travel * 0.8) * 0.025;
    m_state.location.headingDeg = std::fmod(360.0 + travel * 35.0, 360.0);
    m_state.sequence++;
    m_state.occurredAtMs = current;
    emit telemetryProduced(snapshot());
}

void SimulatedVehicleHardware::execute(const VehicleCommand &command) {
    if (!m_running) {
        finish(command, CommandAck::Status::Failed, QStringLiteral("HW_OFFLINE"),
               QStringLiteral("车辆硬件尚未就绪"));
        return;
    }
    if (command.expiresAtMs > 0 && command.expiresAtMs <= nowMs()) {
        finish(command, CommandAck::Status::Expired, QStringLiteral("COMMAND_EXPIRED"),
               QStringLiteral("指令已过期"));
        return;
    }

    // The queued delay models an actuator round trip. Only this implementation
    // changes simulated state and emits the terminal result.
    QTimer::singleShot(180, this, [this, command]() {
        if (!m_running) {
            finish(command, CommandAck::Status::Failed, QStringLiteral("HW_OFFLINE"),
                   QStringLiteral("硬件已停止"));
            return;
        }
        if (command.expiresAtMs > 0 && command.expiresAtMs <= nowMs()) {
            finish(command, CommandAck::Status::Expired, QStringLiteral("COMMAND_EXPIRED"),
                   QStringLiteral("执行前指令已过期"));
            return;
        }
        switch (command.type) {
        case VehicleCommand::Type::Lock: m_state.locked = true; break;
        case VehicleCommand::Type::Unlock: m_state.locked = false; break;
        case VehicleCommand::Type::HvacOn:
            m_state.hvac.mode = HvacState::Mode::Auto;
            m_state.hvac.acOn = true;
            break;
        case VehicleCommand::Type::HvacOff:
            m_state.hvac.mode = HvacState::Mode::Off;
            m_state.hvac.acOn = false;
            break;
        case VehicleCommand::Type::SetTemperature: {
            bool ok = false;
            const double value = command.parameter.toDouble(&ok);
            if (!ok || value < 16.0 || value > 30.0) {
                finish(command, CommandAck::Status::Failed, QStringLiteral("INVALID_TEMPERATURE"),
                       QStringLiteral("温度须在 16–30°C 之间"));
                return;
            }
            m_state.hvac.targetTempC = value;
            m_state.hvac.mode = HvacState::Mode::Auto;
            m_state.hvac.acOn = true;
            break;
        }
        case VehicleCommand::Type::DefrostOn: m_state.defrost = true; break;
        case VehicleCommand::Type::DefrostOff: m_state.defrost = false; break;
        case VehicleCommand::Type::HonkFlash: break;
        }
        finish(command, CommandAck::Status::Succeeded, {}, QStringLiteral("车辆已执行"));
        tick();
    });
}

void SimulatedVehicleHardware::finish(const VehicleCommand &command, CommandAck::Status status,
                                      const QString &code, const QString &message) {
    CommandAck ack;
    ack.commandId = command.commandId;
    ack.vin = command.vin;
    ack.status = status;
    ack.errorCode = code;
    ack.message = message;
    ack.retryable = code == QStringLiteral("HW_OFFLINE");
    ack.occurredAtMs = nowMs();
    emit commandCompleted(ack);
}

Telemetry SimulatedVehicleHardware::snapshot() const { return m_state; }

} // namespace aurora
