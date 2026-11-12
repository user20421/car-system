#include "controller/VehicleController.h"
#include "core/VehicleTypes.h"

#include <QCommandLineParser>
#include <QFile>
#include <QGuiApplication>
#include <QJsonDocument>
#include <QJsonObject>
#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <QQuickStyle>
#include <QQuickWindow>
#include <QTimer>

using namespace aurora;

int main(int argc, char *argv[]) {
    QGuiApplication app(argc, argv);
    QGuiApplication::setApplicationName(QStringLiteral("AURORA Cockpit"));
    QGuiApplication::setOrganizationName(QStringLiteral("AURORA Mobility"));
    QQuickStyle::setStyle(QStringLiteral("Basic"));

    QCommandLineParser parser;
    parser.setApplicationDescription(QStringLiteral("AURORA Qt/QML 智能座舱"));
    parser.addHelpOption();
    const QCommandLineOption vinOption({QStringLiteral("v"), QStringLiteral("vin")}, QStringLiteral("车辆 VIN（每实例唯一）"), QStringLiteral("vin"));
    const QCommandLineOption configOption(QStringLiteral("config"), QStringLiteral("JSON 启动配置"), QStringLiteral("file"));
    const QCommandLineOption certOption(QStringLiteral("cert-dir"), QStringLiteral("CA 与 VIN 客户端证书目录"), QStringLiteral("dir"));
    const QCommandLineOption mqttOption(QStringLiteral("mqtt-host"), QStringLiteral("MQTTS 主机"), QStringLiteral("host"));
    const QCommandLineOption portOption(QStringLiteral("mqtt-port"), QStringLiteral("MQTTS 端口"), QStringLiteral("port"));
    const QCommandLineOption httpsOption(QStringLiteral("https-base"), QStringLiteral("HTTPS 设备服务根地址"), QStringLiteral("url"));
    const QCommandLineOption debugOption(QStringLiteral("debug"), QStringLiteral("显示模拟场景面板"));
    const QCommandLineOption screenshotOption(QStringLiteral("screenshot"), QStringLiteral("启动 3 秒后保存界面并退出（验证用）"), QStringLiteral("file"));
    const QCommandLineOption pageOption(QStringLiteral("page"), QStringLiteral("初始页面索引 0–7"), QStringLiteral("index"));
    parser.addOptions({vinOption, configOption, certOption, mqttOption, portOption, httpsOption, debugOption, screenshotOption, pageOption});
    parser.process(app);

    QJsonObject config;
    if (parser.isSet(configOption)) {
        QFile file(parser.value(configOption));
        if (!file.open(QIODevice::ReadOnly)) {
            qCritical("Cannot open config: %s", qPrintable(file.errorString()));
            return 2;
        }
        config = QJsonDocument::fromJson(file.readAll()).object();
    }
    const QString vin = parser.isSet(vinOption) ? parser.value(vinOption)
        : config.value(QStringLiteral("vin")).toString(QStringLiteral("LTEST000000000001"));
    if (!QRegularExpression(QStringLiteral("^[A-Z0-9]{11,17}$")).match(vin).hasMatch()) {
        qCritical("Invalid VIN: %s", qPrintable(vin));
        return 2;
    }
    NetworkOptions options;
    options.vin = vin;
    options.mqttHost = parser.isSet(mqttOption) ? parser.value(mqttOption)
        : config.value(QStringLiteral("mqtt_host")).toString(QStringLiteral("localhost"));
    options.mqttPort = parser.isSet(portOption) ? parser.value(portOption).toUShort()
        : static_cast<quint16>(config.value(QStringLiteral("mqtt_port")).toInt(8883));
    options.httpsBaseUrl = parser.isSet(httpsOption) ? parser.value(httpsOption)
        : config.value(QStringLiteral("https_base_url")).toString();
    options.certificateDirectory = parser.isSet(certOption) ? parser.value(certOption)
        : config.value(QStringLiteral("certificate_directory")).toString();

    qRegisterMetaType<Telemetry>("aurora::Telemetry");
    qRegisterMetaType<VehicleCommand>("aurora::VehicleCommand");
    qRegisterMetaType<CommandAck>("aurora::CommandAck");
    VehicleController controller(vin, options);
    controller.start();

    QQmlApplicationEngine engine;
    engine.rootContext()->setContextProperty(QStringLiteral("vehicle"), &controller);
    engine.rootContext()->setContextProperty(QStringLiteral("initialPage"), qBound(0, parser.value(pageOption).toInt(), 7));
    engine.rootContext()->setContextProperty(QStringLiteral("debugBuild"),
#ifdef QT_DEBUG
        true
#else
        parser.isSet(debugOption)
#endif
        || parser.isSet(debugOption));
    QObject::connect(&app, &QCoreApplication::aboutToQuit, &controller, &VehicleController::shutdown);
    QObject::connect(&engine, &QQmlApplicationEngine::objectCreationFailed,
                     &app, []() { QCoreApplication::exit(3); }, Qt::QueuedConnection);
    engine.loadFromModule("Aurora", "Main");
    if (parser.isSet(screenshotOption)) {
        QTimer::singleShot(3000, &app, [&app, &engine, file = parser.value(screenshotOption)]() {
            if (!engine.rootObjects().isEmpty()) {
                auto *window = qobject_cast<QQuickWindow *>(engine.rootObjects().constFirst());
                if (window && !window->grabWindow().save(file))
                    qWarning("Could not save screenshot to %s", qPrintable(file));
            }
            app.exit();
        });
    }
    const int result = app.exec();
    controller.shutdown();
    return result;
}
