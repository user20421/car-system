#include "network/VehicleProtobuf.h"

#include <QProtobufSerializer>
#include <QUuid>

namespace aurora::protobuf {
namespace proto = car::vehicle::v1;

proto::Envelope envelope(const QString &vin, const QString &bootId,
                         quint64 sequence, qint64 occurredAtMs) {
    proto::Envelope result;
    result.setVersion(QStringLiteral("v1"));
    result.setMessageId(QUuid::createUuid().toString(QUuid::WithoutBraces));
    result.setVin(vin);
    result.setDeviceBootId(bootId);
    result.setSequence(sequence);
    result.setOccurredAtMs(occurredAtMs > 0 ? occurredAtMs : nowMs());
    return result;
}

QByteArray telemetry(const Telemetry &source, const proto::Envelope &envelope) {
    proto::Telemetry message;
    message.setEnvelope(envelope);
    message.setSpeedKph(source.speedKph);
    message.setRpm(static_cast<quint32>(qMax(0, source.rpm)));
    message.setBatteryPercent(source.batteryPercent);
    message.setRangeKm(source.rangeKm);
    message.setLocked(source.locked);
    message.setCabinTempC(source.cabinTempC);
    message.setDefrost(source.defrost);

    proto::Location location;
    location.setLatitude(source.location.latitude);
    location.setLongitude(source.location.longitude);
    location.setAltitudeM(source.location.altitudeM);
    location.setHeadingDeg(source.location.headingDeg);
    location.setHorizontalAccuracyM(source.location.accuracyM);
    message.setLocation(location);

    proto::HvacState hvac;
    using Mode = proto::HvacState::Mode;
    switch (source.hvac.mode) {
    case HvacState::Mode::Off: hvac.setMode(Mode::MODE_OFF); break;
    case HvacState::Mode::Auto: hvac.setMode(Mode::MODE_AUTO); break;
    case HvacState::Mode::Cool: hvac.setMode(Mode::MODE_COOL); break;
    case HvacState::Mode::Heat: hvac.setMode(Mode::MODE_HEAT); break;
    }
    hvac.setTargetTempC(source.hvac.targetTempC);
    hvac.setFanLevel(static_cast<quint32>(qMax(0, source.hvac.fanLevel)));
    hvac.setAcOn(source.hvac.acOn);
    hvac.setRecirculation(source.hvac.recirculation);
    message.setHvac(hvac);

    QList<proto::Fault> faults;
    faults.reserve(source.faults.size());
    for (const Fault &sourceFault : source.faults) {
        proto::Fault fault;
        fault.setCode(sourceFault.code);
        fault.setMessage(sourceFault.message);
        fault.setActive(sourceFault.active);
        using Severity = proto::Fault::Severity;
        const QString severity = sourceFault.severity.toUpper();
        if (severity == QStringLiteral("INFO")) fault.setSeverity(Severity::SEVERITY_INFO);
        else if (severity == QStringLiteral("WARNING")) fault.setSeverity(Severity::SEVERITY_WARNING);
        else if (severity == QStringLiteral("CRITICAL")) fault.setSeverity(Severity::SEVERITY_CRITICAL);
        else fault.setSeverity(Severity::SEVERITY_UNSPECIFIED);
        faults.append(fault);
    }
    message.setFault(faults);
    QProtobufSerializer serializer;
    return message.serialize(&serializer);
}

QByteArray acknowledgement(const CommandAck &source, const proto::Envelope &envelope) {
    proto::CommandAck message;
    message.setEnvelope(envelope);
    message.setCommandId(source.commandId);
    using Status = proto::CommandAck::Status;
    switch (source.status) {
    case CommandAck::Status::Received: message.setStatus(Status::STATUS_RECEIVED); break;
    case CommandAck::Status::Succeeded: message.setStatus(Status::STATUS_SUCCEEDED); break;
    case CommandAck::Status::Failed: message.setStatus(Status::STATUS_FAILED); break;
    case CommandAck::Status::Expired: message.setStatus(Status::STATUS_EXPIRED); break;
    }
    if (source.status == CommandAck::Status::Failed || source.status == CommandAck::Status::Expired) {
        proto::CommandError error;
        error.setCode(source.errorCode);
        error.setMessage(source.message);
        error.setRetryable(source.retryable);
        message.setError(error);
    }
    QProtobufSerializer serializer;
    return message.serialize(&serializer);
}

QByteArray presence(bool online, const QString &connectionId,
                    const proto::Envelope &envelope) {
    proto::Presence message;
    message.setEnvelope(envelope);
    message.setState(online ? proto::Presence::State::STATE_ONLINE
                            : proto::Presence::State::STATE_OFFLINE);
    message.setConnectionId(connectionId);
    message.setFirmwareVersion(QStringLiteral("1.0.0-qt"));
    QProtobufSerializer serializer;
    return message.serialize(&serializer);
}

DecodedCommand decodeCommand(const QByteArray &payload, const QString &expectedVin) {
    DecodedCommand result;
    proto::Command message;
    QProtobufSerializer serializer;
    if (!message.deserialize(&serializer, payload)) {
        result.error = QStringLiteral("Protobuf 指令解析失败：%1").arg(serializer.lastErrorString());
        return result;
    }
    if (!message.hasEnvelope()) {
        result.error = QStringLiteral("指令缺少 Envelope");
        return result;
    }
    const proto::Envelope &envelope = message.envelope();
    if (envelope.version() != QStringLiteral("v1") || envelope.messageId().isEmpty()
        || envelope.vin() != expectedVin || envelope.deviceBootId().isEmpty()
        || envelope.sequence() == 0 || envelope.occurredAtMs() <= 0
        || message.commandId().isEmpty()) {
        result.error = QStringLiteral("指令 Envelope/VIN/序列校验失败");
        return result;
    }

    result.valid = true;
    result.messageId = envelope.messageId();
    result.sourceBootId = envelope.deviceBootId();
    result.sourceSequence = envelope.sequence();
    VehicleCommand &command = result.command;
    command.commandId = message.commandId();
    command.idempotencyKey = message.idempotencyKey();
    command.vin = envelope.vin();
    command.createdAtMs = envelope.occurredAtMs();
    command.expiresAtMs = message.expiresAtMs();

    using Type = proto::CommandTypeGadget::CommandType;
    result.supported = true;
    switch (message.type()) {
    case Type::COMMAND_TYPE_LOCK: command.type = VehicleCommand::Type::Lock; break;
    case Type::COMMAND_TYPE_UNLOCK: command.type = VehicleCommand::Type::Unlock; break;
    case Type::COMMAND_TYPE_HVAC_ON: command.type = VehicleCommand::Type::HvacOn; break;
    case Type::COMMAND_TYPE_HVAC_OFF: command.type = VehicleCommand::Type::HvacOff; break;
    case Type::COMMAND_TYPE_SET_TEMPERATURE:
        command.type = VehicleCommand::Type::SetTemperature;
        command.parameter = message.params().value(QStringLiteral("target_temp_c"));
        break;
    case Type::COMMAND_TYPE_DEFROST_ON: command.type = VehicleCommand::Type::DefrostOn; break;
    case Type::COMMAND_TYPE_DEFROST_OFF: command.type = VehicleCommand::Type::DefrostOff; break;
    case Type::COMMAND_TYPE_HONK_FLASH: command.type = VehicleCommand::Type::HonkFlash; break;
    default:
        result.supported = false;
        result.error = QStringLiteral("未知指令类型");
        break;
    }
    return result;
}

} // namespace aurora::protobuf
