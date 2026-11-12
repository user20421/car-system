#pragma once
#include "hal/SimulatedVehicleHardware.h"
#include <QObject>
#include <QSignalSpy>
#include <QTest>

class TestHal final : public QObject {
    Q_OBJECT
private slots:
    void producesTelemetry() {
        aurora::SimulatedVehicleHardware hal(QStringLiteral("LTEST000000000001"));
        QSignalSpy spy(&hal, &aurora::IVehicleHardware::telemetryProduced);
        hal.start();
        QTRY_VERIFY_WITH_TIMEOUT(spy.count() >= 3, 1500);
        const auto telemetry = qvariant_cast<aurora::Telemetry>(spy.last().at(0));
        QCOMPARE(telemetry.vin, QStringLiteral("LTEST000000000001"));
        QVERIFY(telemetry.batteryPercent > 0);
        QVERIFY(telemetry.sequence >= 3);
        hal.stop();
    }
    void executesRealStateChange() {
        aurora::SimulatedVehicleHardware hal(QStringLiteral("LTEST000000000001"));
        QSignalSpy telemetry(&hal, &aurora::IVehicleHardware::telemetryProduced);
        QSignalSpy completed(&hal, &aurora::IVehicleHardware::commandCompleted);
        hal.start();
        aurora::VehicleCommand command;
        command.commandId = QStringLiteral("unlock-1"); command.vin = QStringLiteral("LTEST000000000001");
        command.type = aurora::VehicleCommand::Type::Unlock; command.expiresAtMs = aurora::nowMs() + 2000;
        hal.execute(command);
        QTRY_COMPARE_WITH_TIMEOUT(completed.count(), 1, 1500);
        const auto ack = qvariant_cast<aurora::CommandAck>(completed.first().at(0));
        QCOMPARE(ack.status, aurora::CommandAck::Status::Succeeded);
        QTRY_VERIFY_WITH_TIMEOUT(!qvariant_cast<aurora::Telemetry>(telemetry.last().at(0)).locked, 1000);
        hal.stop();
    }
    void rejectsExpiredAndInvalidCommands() {
        aurora::SimulatedVehicleHardware hal(QStringLiteral("LTEST000000000001"));
        QSignalSpy completed(&hal, &aurora::IVehicleHardware::commandCompleted);
        hal.start();
        aurora::VehicleCommand expired;
        expired.commandId = QStringLiteral("expired"); expired.vin = QStringLiteral("LTEST000000000001");
        expired.type = aurora::VehicleCommand::Type::Lock; expired.expiresAtMs = aurora::nowMs() - 1;
        hal.execute(expired);
        QCOMPARE(completed.count(), 1);
        QCOMPARE(qvariant_cast<aurora::CommandAck>(completed.first().at(0)).status, aurora::CommandAck::Status::Expired);
        aurora::VehicleCommand invalid;
        invalid.commandId = QStringLiteral("invalid"); invalid.vin = expired.vin;
        invalid.type = aurora::VehicleCommand::Type::SetTemperature; invalid.parameter = QStringLiteral("80");
        invalid.expiresAtMs = aurora::nowMs() + 2000;
        hal.execute(invalid);
        QTRY_COMPARE_WITH_TIMEOUT(completed.count(), 2, 1500);
        QCOMPARE(qvariant_cast<aurora::CommandAck>(completed.last().at(0)).errorCode, QStringLiteral("INVALID_TEMPERATURE"));
        hal.stop();
    }
};
