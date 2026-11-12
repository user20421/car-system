#include "TestHal.h"
#include "TestController.h"
#include "TestConcurrency.h"
#include "TestNetwork.h"
#include <QCoreApplication>
#include <QTest>

int main(int argc, char **argv) {
    QCoreApplication app(argc, argv);
    qRegisterMetaType<aurora::Telemetry>("aurora::Telemetry");
    qRegisterMetaType<aurora::VehicleCommand>("aurora::VehicleCommand");
    qRegisterMetaType<aurora::CommandAck>("aurora::CommandAck");
    int result = 0;
    TestHal hal;
    result |= QTest::qExec(&hal, argc, argv);
    TestController controller;
    result |= QTest::qExec(&controller, argc, argv);
    TestConcurrency concurrency;
    result |= QTest::qExec(&concurrency, argc, argv);
    TestNetwork network;
    result |= QTest::qExec(&network, argc, argv);
    return result;
}
