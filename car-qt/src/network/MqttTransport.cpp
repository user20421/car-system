#include "network/MqttTransport.h"

#include <QMqttClient>
#include <QMqttMessage>
#include <QMqttSubscription>
#include <QMetaEnum>
#include <QSslKey>

namespace aurora {

QtMqttTransport::QtMqttTransport(MqttTransportOptions options, QObject *parent)
    : MqttTransport(parent), m_options(std::move(options)) {
    m_client = new QMqttClient(this);
    m_client->setProtocolVersion(QMqttClient::MQTT_5_0);
    m_client->setClientId(m_options.clientId);
    m_client->setHostname(m_options.host);
    m_client->setPort(m_options.port);
    m_client->setKeepAlive(30);
    m_client->setAutoKeepAlive(true);
    m_client->setCleanSession(false);
    QMqttConnectionProperties connectionProperties;
    connectionProperties.setSessionExpiryInterval(24 * 60 * 60);
    m_client->setConnectionProperties(connectionProperties);

    m_client->setWillTopic(QStringLiteral("vehicles/%1/status/v1").arg(m_options.vin));
    m_client->setWillMessage(m_options.willPayload);
    m_client->setWillQoS(1);
    m_client->setWillRetain(true);
    QMqttLastWillProperties willProperties;
    willProperties.setContentType(QStringLiteral("application/x-protobuf"));
    m_client->setLastWillProperties(willProperties);

    m_available = m_options.port == 8883 && !m_options.host.isEmpty()
        && !m_options.vin.isEmpty() && !m_options.clientId.isEmpty()
        && !m_options.willPayload.isEmpty()
        && m_options.tls.protocol() == QSsl::TlsV1_2OrLater
        && m_options.tls.peerVerifyMode() == QSslSocket::VerifyPeer
        && !m_options.tls.caCertificates().isEmpty()
        && !m_options.tls.localCertificate().isNull()
        && !m_options.tls.privateKey().isNull();
    m_availabilityDetail = m_available ? QStringLiteral("MQTTS/Protobuf 已就绪")
        : QStringLiteral("MQTTS 配置不完整：需要 8883、CA、VIN 证书和 TLS 1.2+");

    m_reconnect = new QTimer(this);
    m_reconnect->setSingleShot(true);
    connect(m_reconnect, &QTimer::timeout, this, &QtMqttTransport::connectSecure);
    connect(m_client, &QMqttClient::connected, this, [this]() {
        m_retryMs = 1000;
        m_subscription = m_client->subscribe(
            QMqttTopicFilter(QStringLiteral("vehicles/%1/command/v1").arg(m_options.vin)), 1);
        connect(m_subscription, &QMqttSubscription::messageReceived, this,
                [this](const QMqttMessage &message) {
            const QString topic = message.topic().name();
            if (topic == QStringLiteral("vehicles/%1/command/v1").arg(m_options.vin))
                emit binaryMessage(topic, message.payload());
        });
        emit connected();
    });
    connect(m_client, &QMqttClient::disconnected, this, [this]() {
        m_subscription = nullptr;
        emit disconnected(errorDetail());
        scheduleReconnect();
    });
}

QtMqttTransport::~QtMqttTransport() { disconnectFromHost(); }

void QtMqttTransport::connectSecure() {
    if (m_stopping || !m_available) return;
    m_reconnect->stop();
    if (m_client->state() != QMqttClient::Disconnected) return;
    // SecureSocket is implicit in connectToHostEncrypted; never call the
    // plaintext connectToHost() overload and never downgrade TLS on retry.
    m_client->connectToHostEncrypted(m_options.tls);
}

void QtMqttTransport::disconnectFromHost() {
    if (m_stopping) return;
    m_stopping = true;
    if (m_reconnect) m_reconnect->stop();
    if (m_client && m_client->state() != QMqttClient::Disconnected)
        m_client->disconnectFromHost();
}

bool QtMqttTransport::publish(const QString &topic, const QByteArray &protobuf, int qos, bool retain) {
    if (!m_available || m_client->state() != QMqttClient::Connected || protobuf.isEmpty()
        || (qos != 0 && qos != 1)) return false;
    const QString prefix = QStringLiteral("vehicles/%1/").arg(m_options.vin);
    const QString kind = topic.mid(prefix.size());
    if (!topic.startsWith(prefix)
        || (kind != QStringLiteral("telemetry/v1") && kind != QStringLiteral("ack/v1")
            && kind != QStringLiteral("status/v1"))) return false;
    QMqttPublishProperties properties;
    properties.setContentType(QStringLiteral("application/x-protobuf"));
    return m_client->publish(QMqttTopicName(topic), properties, protobuf,
                             static_cast<quint8>(qos), retain) >= 0;
}

void QtMqttTransport::scheduleReconnect() {
    if (m_stopping || !m_available || m_reconnect->isActive()) return;
    m_reconnect->start(m_retryMs);
    m_retryMs = qMin(m_retryMs * 2, 30000);
}

QString QtMqttTransport::errorDetail() const {
    const auto error = m_client->error();
    const char *name = QMetaEnum::fromType<QMqttClient::ClientError>().valueToKey(error);
    return QStringLiteral("MQTTS 已断开 · %1").arg(name ? QString::fromLatin1(name)
                                                      : QStringLiteral("UnknownError"));
}

} // namespace aurora
