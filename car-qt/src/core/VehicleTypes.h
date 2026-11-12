#pragma once

#include <QDateTime>
#include <QMetaType>
#include <QString>
#include <QStringList>
#include <QVector>
#include <QtGlobal>

namespace aurora {

struct Location {
    double latitude = 31.2304;
    double longitude = 121.4737;
    double altitudeM = 8.0;
    double headingDeg = 0.0;
    double accuracyM = 4.0;
};

struct HvacState {
    enum class Mode { Off, Auto, Cool, Heat };
    Mode mode = Mode::Off;
    double targetTempC = 22.0;
    int fanLevel = 2;
    bool acOn = false;
    bool recirculation = false;
};

struct Fault {
    QString code;
    QString severity;
    QString message;
    bool active = true;
};

struct Telemetry {
    QString vin;
    QString bootId;
    quint64 sequence = 0;
    qint64 occurredAtMs = 0;
    double speedKph = 0.0;
    int rpm = 0;
    double batteryPercent = 100.0;
    double rangeKm = 0.0;
    Location location;
    bool locked = true;
    HvacState hvac;
    double cabinTempC = 22.0;
    bool defrost = false;
    QVector<Fault> faults;
};

struct VehicleCommand {
    enum class Type {
        Lock, Unlock, HvacOn, HvacOff, SetTemperature,
        DefrostOn, DefrostOff, HonkFlash
    };
    QString commandId;
    QString idempotencyKey;
    QString vin;
    Type type = Type::Lock;
    QString parameter;
    qint64 createdAtMs = 0;
    qint64 expiresAtMs = 0;
};

struct CommandAck {
    enum class Status { Received, Succeeded, Failed, Expired };
    QString commandId;
    QString vin;
    Status status = Status::Received;
    QString errorCode;
    QString message;
    bool retryable = false;
    qint64 occurredAtMs = 0;
};

inline qint64 nowMs() { return QDateTime::currentMSecsSinceEpoch(); }

} // namespace aurora

Q_DECLARE_METATYPE(aurora::Location)
Q_DECLARE_METATYPE(aurora::HvacState)
Q_DECLARE_METATYPE(aurora::Fault)
Q_DECLARE_METATYPE(aurora::Telemetry)
Q_DECLARE_METATYPE(aurora::VehicleCommand)
Q_DECLARE_METATYPE(aurora::CommandAck)
