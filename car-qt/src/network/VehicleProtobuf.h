#pragma once

#include "core/VehicleTypes.h"
#include "vehicle_v1.qpb.h"

#include <QByteArray>

namespace aurora::protobuf {

struct DecodedCommand {
    bool valid = false;
    bool supported = false;
    VehicleCommand command;
    QString messageId;
    QString sourceBootId;
    quint64 sourceSequence = 0;
    QString error;
};

car::vehicle::v1::Envelope envelope(const QString &vin, const QString &bootId,
                                    quint64 sequence, qint64 occurredAtMs);
QByteArray telemetry(const Telemetry &source, const car::vehicle::v1::Envelope &envelope);
QByteArray acknowledgement(const CommandAck &source, const car::vehicle::v1::Envelope &envelope);
QByteArray presence(bool online, const QString &connectionId,
                    const car::vehicle::v1::Envelope &envelope);
DecodedCommand decodeCommand(const QByteArray &payload, const QString &expectedVin);

} // namespace aurora::protobuf
