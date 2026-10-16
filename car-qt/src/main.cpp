#include <QGuiApplication>
#include <QQmlApplicationEngine>
#include <QQmlContext>

#include "hal/VehicleSimulator.h"
#include "net/ApiClient.h"
#include "core/VehicleController.h"

int main(int argc, char *argv[])
{
    QGuiApplication app(argc, argv);
    app.setApplicationName("car-qt");
    app.setOrganizationName("car-system");

    // 硬件层：本地开发使用模拟器（换真车时替换为 CAN/串口驱动实现）
    hal::VehicleSimulator simulator;

    // 网络层：后端 API 客户端（地址/VIN/账号走环境变量，缺省为本地开发默认值）
    net::ApiClient api;
    api.configure(qEnvironmentVariable("CAR_API_BASE", QStringLiteral("http://127.0.0.1:8080")),
                  qEnvironmentVariable("CAR_VIN", QStringLiteral("LVSFACBE8MF012345")),
                  qEnvironmentVariable("CAR_USERNAME", QStringLiteral("driver1")),
                  qEnvironmentVariable("CAR_PASSWORD", QStringLiteral("driver123456")));

    // 控制器：桥接 QML
    core::VehicleController controller(&simulator, &api);

    QQmlApplicationEngine engine;
    engine.rootContext()->setContextProperty(QStringLiteral("vehicle"), &controller);
    engine.loadFromModule("CarQt", "Main");
    if (engine.rootObjects().isEmpty())
        return -1;

    simulator.start();
    api.start();
    return app.exec();
}
