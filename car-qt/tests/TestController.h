#pragma once
#include "controller/VehicleController.h"
#include <QObject>
#include <QSignalSpy>
#include <QTest>

class TestController final : public QObject {
    Q_OBJECT
private slots:
    void controllerUsesHardwareAck() {
        aurora::NetworkOptions options;
        aurora::VehicleController controller(QStringLiteral("LTEST000000000002"), options);
        controller.start();
        QTRY_VERIFY_WITH_TIMEOUT(controller.battery() > 0, 1500);
        QSignalSpy changed(&controller, &aurora::VehicleController::commandChanged);
        QVERIFY(!controller.unlockVehicle().isEmpty());
        QCOMPARE(controller.commandState(), QStringLiteral("received"));
        QTRY_COMPARE_WITH_TIMEOUT(controller.commandState(), QStringLiteral("succeeded"), 2000);
        QTRY_VERIFY_WITH_TIMEOUT(!controller.locked(), 1000);
        QVERIFY(changed.count() >= 2);
        controller.shutdown();
    }
    void serializesCommands() {
        aurora::NetworkOptions options;
        aurora::VehicleController controller(QStringLiteral("LTEST000000000003"), options);
        controller.start();
        QTRY_VERIFY_WITH_TIMEOUT(controller.battery() > 0, 1500);
        controller.setTemperature(21.0);
        controller.setDefrost(true);
        controller.lockVehicle();
        QTRY_VERIFY_WITH_TIMEOUT(!controller.commandBusy(), 3500);
        QCOMPARE(controller.defrost(), true);
        QCOMPARE(controller.locked(), true);
        QCOMPARE(controller.targetTemperature(), 21.0);
        controller.shutdown();
    }
    void rejectsWrongVinAndDuplicates() {
        aurora::NetworkOptions options;
        aurora::VehicleController controller(QStringLiteral("LTEST000000000001"), options);
        controller.start();
        aurora::VehicleCommand remote;
        remote.commandId = QStringLiteral("remote-1"); remote.vin = QStringLiteral("WRONGVIN00000001");
        remote.type = aurora::VehicleCommand::Type::Unlock; remote.expiresAtMs = aurora::nowMs() + 1000;
        controller.receiveRemoteCommand(remote);
        QVERIFY(!controller.commandBusy());
        remote.vin = QStringLiteral("LTEST000000000001");
        controller.receiveRemoteCommand(remote);
        controller.receiveRemoteCommand(remote);
        QTRY_VERIFY_WITH_TIMEOUT(!controller.commandBusy(), 2000);
        QTRY_VERIFY_WITH_TIMEOUT(!controller.locked(), 1000);
        controller.shutdown();
    }
};
