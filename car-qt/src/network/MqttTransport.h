#pragma once

#include <QByteArray>
#include <QObject>
#include <QSslConfiguration>
#include <QTimer>

class QMqttClient;
class QMqttSubscription;

namespace aurora {

struct MqttTransportOptions {
    QString vin;
    QString host = QStringLiteral("localhost");
    quint16 port = 8883;
    QSslConfiguration tls;
    QString clientId;
    QByteArray willPayload;
};

// Binary-only MQTTS boundary. There is deliberately no plaintext/JSON path.
class MqttTransport : public QObject {
    Q_OBJECT
public:
    using QObject::QObject;
    virtual bool available() const = 0;
    virtual QString availabilityDetail() const = 0;
    virtual void connectSecure() = 0;
    virtual void disconnectFromHost() = 0;
    virtual bool publish(const QString &topic, const QByteArray &protobuf, int qos, bool retain) = 0;

signals:
    void connected();
    void disconnected(const QString &reason);
    void binaryMessage(const QString &topic, const QByteArray &protobuf);
};

class QtMqttTransport final : public MqttTransport {
    Q_OBJECT
public:
    explicit QtMqttTransport(MqttTransportOptions options, QObject *parent = nullptr);
    ~QtMqttTransport() override;

    bool available() const override { return m_available; }
    QString availabilityDetail() const override { return m_availabilityDetail; }
    void connectSecure() override;
    void disconnectFromHost() override;
    bool publish(const QString &topic, const QByteArray &protobuf, int qos, bool retain) override;

private:
    void scheduleReconnect();
    QString errorDetail() const;

    MqttTransportOptions m_options;
    QMqttClient *m_client = nullptr;
    QMqttSubscription *m_subscription = nullptr;
    QTimer *m_reconnect = nullptr;
    bool m_available = false;
    bool m_stopping = false;
    QString m_availabilityDetail;
    int m_retryMs = 1000;
};

} // namespace aurora
