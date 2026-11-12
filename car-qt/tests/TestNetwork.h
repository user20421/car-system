#pragma once
#include "network/VehicleProtobuf.h"
#include <QProtobufSerializer>
#include <QObject>
#include <QTest>

class TestNetwork final : public QObject {
    Q_OBJECT
private slots:
    void telemetryRoundTripUsesVehicleV1() {
        aurora::Telemetry source;
        source.vin = QStringLiteral("LTEST000000000001");
        source.speedKph = 42.5;
        source.rpm = 2450;
        source.batteryPercent = 82.4;
        source.location.latitude = 31.23;
        source.location.longitude = 121.47;
        source.hvac.mode = aurora::HvacState::Mode::Cool;
        source.hvac.fanLevel = 3;
        source.hvac.acOn = true;
        source.faults.append({QStringLiteral("BMS_A017"), QStringLiteral("WARNING"),
                              QStringLiteral("battery"), true});
        const auto envelope = aurora::protobuf::envelope(source.vin, QStringLiteral("boot"), 7, 123456);
        const QByteArray wire = aurora::protobuf::telemetry(source, envelope);
        QVERIFY(!wire.isEmpty());
        car::vehicle::v1::Telemetry decoded;
        QProtobufSerializer serializer;
        QVERIFY(decoded.deserialize(&serializer, wire));
        QCOMPARE(decoded.envelope().vin(), source.vin);
        QCOMPARE(decoded.envelope().sequence(), quint64(7));
        QCOMPARE(decoded.speedKph(), 42.5);
        QCOMPARE(decoded.rpm(), quint32(2450));
        QCOMPARE(decoded.hvac().mode(), car::vehicle::v1::HvacState::Mode::MODE_COOL);
        QCOMPARE(decoded.fault().first().severity(), car::vehicle::v1::Fault::Severity::SEVERITY_WARNING);
    }

    void decodesCommandMapAndRejectsWrongVin() {
        car::vehicle::v1::Command message;
        message.setEnvelope(aurora::protobuf::envelope(
            QStringLiteral("LTEST000000000001"), QStringLiteral("backend"), 9, 1000));
        message.setCommandId(QStringLiteral("command-1"));
        message.setIdempotencyKey(QStringLiteral("key-1"));
        message.setExpiresAtMs(2000);
        message.setType(car::vehicle::v1::CommandTypeGadget::CommandType::COMMAND_TYPE_SET_TEMPERATURE);
        message.setParams({{QStringLiteral("target_temp_c"), QStringLiteral("21.5")}});
        QProtobufSerializer serializer;
        const QByteArray wire = message.serialize(&serializer);
        auto decoded = aurora::protobuf::decodeCommand(wire, QStringLiteral("LTEST000000000001"));
        QVERIFY(decoded.valid);
        QVERIFY(decoded.supported);
        QCOMPARE(decoded.command.parameter, QStringLiteral("21.5"));
        QCOMPARE(decoded.sourceSequence, quint64(9));
        QVERIFY(!aurora::protobuf::decodeCommand(wire, QStringLiteral("LTEST000000000002")).valid);
        QVERIFY(!aurora::protobuf::decodeCommand(QByteArray::fromHex("ff"), QStringLiteral("LTEST000000000001")).valid);
    }

    void unknownCommandIsExplicitlyUnsupported() {
        car::vehicle::v1::Command message;
        message.setEnvelope(aurora::protobuf::envelope(
            QStringLiteral("LTEST000000000001"), QStringLiteral("backend"), 1, 1000));
        message.setCommandId(QStringLiteral("unknown"));
        message.setType(car::vehicle::v1::CommandTypeGadget::CommandType::COMMAND_TYPE_UNSPECIFIED);
        QProtobufSerializer serializer;
        auto decoded = aurora::protobuf::decodeCommand(message.serialize(&serializer),
                                                        QStringLiteral("LTEST000000000001"));
        QVERIFY(decoded.valid);
        QVERIFY(!decoded.supported);
    }
};
