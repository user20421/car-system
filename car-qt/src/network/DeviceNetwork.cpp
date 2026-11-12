#include "network/DeviceNetwork.h"
#include "network/MqttTransport.h"
#include "network/VehicleProtobuf.h"

#include <QFile>
#include <QJsonDocument>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QNetworkRequest>
#include <QSslCertificate>
#include <QSslConfiguration>
#include <QSslKey>
#include <QTimer>
#include <QUrl>
#include <QUuid>

namespace aurora {

class DeviceNetworkWorker final : public QObject {
    Q_OBJECT
public:
    explicit DeviceNetworkWorker(NetworkOptions options)
        : m_options(std::move(options)),
          m_bootId(QUuid::createUuid().toString(QUuid::WithoutBraces)) {}

public slots:
    void start() {
        m_nam = new QNetworkAccessManager(this);
        m_tls = QSslConfiguration::defaultConfiguration();
        m_tls.setProtocol(QSsl::TlsV1_2OrLater);
        m_tls.setPeerVerifyMode(QSslSocket::VerifyPeer);
        QString certificateError;
        loadCertificates(&certificateError);

        MqttTransportOptions transportOptions;
        transportOptions.vin = m_options.vin;
        transportOptions.host = m_options.mqttHost;
        transportOptions.port = m_options.mqttPort;
        transportOptions.tls = m_tls;
        transportOptions.clientId = QStringLiteral("%1-%2").arg(
            m_options.vin, QUuid::createUuid().toString(QUuid::WithoutBraces));
        transportOptions.willPayload = protobuf::presence(
            false, m_bootId, nextEnvelope());
        m_mqtt = new QtMqttTransport(transportOptions, this);
        if (!certificateError.isEmpty()) {
            emit networkStateChanged(QStringLiteral("offline"), certificateError);
            return;
        }
        connect(m_mqtt, &MqttTransport::connected, this, [this]() {
            m_connected = true;
            const QByteArray online = protobuf::presence(true, m_bootId, nextEnvelope());
            if (!m_mqtt->publish(statusTopic(), online, 1, true)) {
                m_connected = false;
                emit networkStateChanged(QStringLiteral("offline"), QStringLiteral("ONLINE Presence 发布失败"));
                return;
            }
            qInfo() << "MQTTS connected" << m_options.vin << m_options.mqttHost;
            emit networkStateChanged(QStringLiteral("online"), QStringLiteral("MQTTS 已连接 · TLS 1.2+ · Protobuf"));
        });
        connect(m_mqtt, &MqttTransport::disconnected, this, [this](const QString &reason) {
            m_connected = false;
            qWarning() << reason;
            emit networkStateChanged(QStringLiteral("offline"), reason);
        });
        connect(m_mqtt, &MqttTransport::binaryMessage, this, &DeviceNetworkWorker::handleCommand);
        m_flush = new QTimer(this);
        m_flush->setInterval(250);
        connect(m_flush, &QTimer::timeout, this, &DeviceNetworkWorker::flushTelemetry);
        m_flush->start();
        emit networkStateChanged(QStringLiteral("connecting"), QStringLiteral("正在建立 MQTTS 连接"));
        m_mqtt->connectSecure();
    }

    void publishLatest(const Telemetry &telemetry) {
        if (telemetry.vin != m_options.vin) return;
        m_latest = telemetry;
        m_hasLatest = true; // latest-value coalescing; stale telemetry is never queued
    }

    void publishAck(const CommandAck &ack) {
        if (ack.vin != m_options.vin || ack.commandId.isEmpty()) return;
        if (!m_connected || !m_mqtt) {
            emit ackDeliveryFailed(ack.commandId, QStringLiteral("MQTTS 未连接，ACK 未发送"));
            return;
        }
        const QByteArray payload = protobuf::acknowledgement(
            ack, nextEnvelope(ack.occurredAtMs));
        if (payload.isEmpty() || !m_mqtt->publish(ackTopic(), payload, 1, false))
            emit ackDeliveryFailed(ack.commandId, QStringLiteral("Protobuf ACK 发布失败"));
    }

    void flushTelemetry() {
        if (!m_hasLatest || !m_connected || !m_mqtt) return;
        const QByteArray payload = protobuf::telemetry(
            m_latest, nextEnvelope(m_latest.occurredAtMs));
        if (!payload.isEmpty() && m_mqtt->publish(telemetryTopic(), payload, 0, false))
            m_hasLatest = false;
    }

    void fetchConfiguration() { getJson(QStringLiteral("config"), QStringLiteral("/device/config")); }
    void checkForUpdate(const QString &currentVersion) {
        getJson(QStringLiteral("ota"), QStringLiteral("/device/ota?current=%1").arg(currentVersion));
    }
    void uploadDiagnostic(const QJsonObject &diagnostic) {
        postJson(QStringLiteral("diagnostic"), QStringLiteral("/device/diagnostic"), diagnostic);
    }
    void uploadBatch(const QString &batchId) {
        Q_UNUSED(batchId)
        emit httpsCompleted(QStringLiteral("batch"), false, {},
                            QStringLiteral("当前设备服务批量接口与 Protobuf 契约不一致，未发送"));
    }

    void stop() {
        if (m_flush) m_flush->stop();
        if (m_mqtt) {
            if (m_connected) {
                const QByteArray offline = protobuf::presence(false, m_bootId, nextEnvelope());
                m_mqtt->publish(statusTopic(), offline, 1, true);
            }
            m_connected = false;
            m_mqtt->disconnectFromHost();
        }
    }

signals:
    void networkStateChanged(const QString &state, const QString &detail);
    void commandReceived(const VehicleCommand &command);
    void httpsCompleted(const QString &operation, bool success, const QJsonObject &body, const QString &error);
    void ackDeliveryFailed(const QString &commandId, const QString &reason);

private:
    using ProtoEnvelope = car::vehicle::v1::Envelope;
    ProtoEnvelope nextEnvelope(qint64 occurredAtMs = 0) {
        return protobuf::envelope(m_options.vin, m_bootId, ++m_sequence, occurredAtMs);
    }
    QString telemetryTopic() const { return QStringLiteral("vehicles/%1/telemetry/v1").arg(m_options.vin); }
    QString commandTopic() const { return QStringLiteral("vehicles/%1/command/v1").arg(m_options.vin); }
    QString ackTopic() const { return QStringLiteral("vehicles/%1/ack/v1").arg(m_options.vin); }
    QString statusTopic() const { return QStringLiteral("vehicles/%1/status/v1").arg(m_options.vin); }

    void handleCommand(const QString &topic, const QByteArray &payload) {
        if (topic != commandTopic()) return;
        protobuf::DecodedCommand decoded = protobuf::decodeCommand(payload, m_options.vin);
        if (!decoded.valid) {
            qWarning() << "Rejected command:" << decoded.error;
            return;
        }
        if (m_seenMessages.contains(decoded.messageId)) return; // MQTT QoS 1 duplicate
        m_seenMessages.insert(decoded.messageId);
        m_seenMessageOrder.append(decoded.messageId);
        while (m_seenMessageOrder.size() > 512)
            m_seenMessages.remove(m_seenMessageOrder.takeFirst());

        if (decoded.sourceBootId != m_sourceBootId) {
            m_sourceBootId = decoded.sourceBootId;
            m_sourceSequence = 0;
        }
        if (decoded.sourceSequence <= m_sourceSequence) {
            qWarning() << "Rejected stale command sequence" << decoded.sourceSequence;
            return;
        }
        m_sourceSequence = decoded.sourceSequence;

        if (decoded.command.expiresAtMs <= nowMs()) {
            CommandAck ack;
            ack.commandId = decoded.command.commandId;
            ack.vin = m_options.vin;
            ack.status = CommandAck::Status::Expired;
            ack.errorCode = QStringLiteral("COMMAND_EXPIRED");
            ack.message = QStringLiteral("指令已过期");
            ack.occurredAtMs = nowMs();
            publishAck(ack);
            return;
        }
        if (!decoded.supported) {
            CommandAck ack;
            ack.commandId = decoded.command.commandId;
            ack.vin = m_options.vin;
            ack.status = CommandAck::Status::Failed;
            ack.errorCode = QStringLiteral("UNSUPPORTED_COMMAND");
            ack.message = QStringLiteral("车机不支持该指令类型");
            ack.occurredAtMs = nowMs();
            publishAck(ack);
            return;
        }
        emit commandReceived(decoded.command);
    }

    bool loadCertificates(QString *error) {
        const QString dir = m_options.certificateDirectory;
        if (dir.isEmpty()) {
            *error = QStringLiteral("未配置 CA/VIN 客户端证书，MQTTS 已禁用");
            return false;
        }
        QFile caFile(dir + QStringLiteral("/ca.crt"));
        QFile certFile(dir + QStringLiteral("/%1.crt").arg(m_options.vin));
        QFile keyFile(dir + QStringLiteral("/%1.key").arg(m_options.vin));
        if (!caFile.open(QIODevice::ReadOnly) || !certFile.open(QIODevice::ReadOnly)
            || !keyFile.open(QIODevice::ReadOnly)) {
            *error = QStringLiteral("无法读取 ca.crt、%1.crt 或 %1.key").arg(m_options.vin);
            return false;
        }
        const auto certificates = QSslCertificate::fromData(caFile.readAll(), QSsl::Pem);
        const QSslCertificate cert(certFile.readAll(), QSsl::Pem);
        const QSslKey key(keyFile.readAll(), QSsl::Rsa, QSsl::Pem);
        if (certificates.isEmpty() || cert.isNull() || key.isNull()
            || !cert.subjectInfo(QSslCertificate::CommonName).contains(m_options.vin)) {
            *error = QStringLiteral("CA/客户端证书无效，或证书 CN 与 VIN 不一致");
            return false;
        }
        m_tls.setCaCertificates(certificates);
        m_tls.setLocalCertificate(cert);
        m_tls.setPrivateKey(key);
        return true;
    }

    QNetworkRequest request(const QString &path) const {
        QUrl url(m_options.httpsBaseUrl + path);
        QNetworkRequest req(url);
        req.setAttribute(QNetworkRequest::Http2AllowedAttribute, true);
        req.setHeader(QNetworkRequest::UserAgentHeader, QStringLiteral("AURORA-Vehicle/1.0"));
        req.setRawHeader("X-VIN", m_options.vin.toUtf8());
        req.setSslConfiguration(m_tls);
        return req;
    }
    bool httpsReady(const QString &operation) {
        const QUrl base(m_options.httpsBaseUrl);
        if (base.scheme() != QStringLiteral("https") || base.host().isEmpty()) {
            emit httpsCompleted(operation, false, {}, QStringLiteral("仅允许配置有效 HTTPS 设备服务地址"));
            return false;
        }
        return true;
    }
    void getJson(const QString &operation, const QString &path) {
        if (!httpsReady(operation)) return;
        QNetworkReply *reply = m_nam->get(request(path));
        observe(reply, operation);
    }
    void postJson(const QString &operation, const QString &path, const QJsonObject &body) {
        if (!httpsReady(operation)) return;
        QNetworkRequest req = request(path);
        req.setHeader(QNetworkRequest::ContentTypeHeader, QStringLiteral("application/json"));
        QNetworkReply *reply = m_nam->post(req, QJsonDocument(body).toJson(QJsonDocument::Compact));
        observe(reply, operation);
    }
    void observe(QNetworkReply *reply, const QString &operation) {
        connect(reply, &QNetworkReply::finished, this, [this, reply, operation]() {
            const QByteArray payload = reply->readAll();
            const QJsonDocument doc = QJsonDocument::fromJson(payload);
            const QString error = reply->error() == QNetworkReply::NoError ? QString() : reply->errorString();
            emit httpsCompleted(operation, reply->error() == QNetworkReply::NoError,
                                doc.isObject() ? doc.object() : QJsonObject{}, error);
            reply->deleteLater();
        });
    }

    NetworkOptions m_options;
    QNetworkAccessManager *m_nam = nullptr;
    MqttTransport *m_mqtt = nullptr;
    QSslConfiguration m_tls;
    QTimer *m_flush = nullptr;
    Telemetry m_latest;
    QString m_bootId;
    quint64 m_sequence = 0;
    bool m_hasLatest = false;
    bool m_connected = false;
    QString m_sourceBootId;
    quint64 m_sourceSequence = 0;
    QSet<QString> m_seenMessages;
    QStringList m_seenMessageOrder;
};

DeviceNetwork::DeviceNetwork(NetworkOptions options, QObject *parent)
    : QObject(parent), m_options(std::move(options)) {
    m_worker = new DeviceNetworkWorker(m_options);
    m_worker->moveToThread(&m_thread);
    m_thread.setObjectName(QStringLiteral("device-network"));
    connect(&m_thread, &QThread::started, m_worker, &DeviceNetworkWorker::start);
    connect(&m_thread, &QThread::finished, m_worker, &QObject::deleteLater);
    connect(m_worker, &DeviceNetworkWorker::networkStateChanged, this, &DeviceNetwork::networkStateChanged, Qt::QueuedConnection);
    connect(m_worker, &DeviceNetworkWorker::commandReceived, this, &DeviceNetwork::commandReceived, Qt::QueuedConnection);
    connect(m_worker, &DeviceNetworkWorker::httpsCompleted, this, &DeviceNetwork::httpsCompleted, Qt::QueuedConnection);
    connect(m_worker, &DeviceNetworkWorker::ackDeliveryFailed, this, &DeviceNetwork::ackDeliveryFailed, Qt::QueuedConnection);
}

DeviceNetwork::~DeviceNetwork() { stop(); }
void DeviceNetwork::start() { if (!m_thread.isRunning()) m_thread.start(); }
void DeviceNetwork::stop() {
    if (!m_thread.isRunning()) return;
    QMetaObject::invokeMethod(m_worker, &DeviceNetworkWorker::stop, Qt::BlockingQueuedConnection);
    m_thread.quit();
    m_thread.wait(3000);
}
void DeviceNetwork::publishLatest(const Telemetry &telemetry) {
    if (m_thread.isRunning()) QMetaObject::invokeMethod(m_worker, [this, telemetry]() { m_worker->publishLatest(telemetry); }, Qt::QueuedConnection);
}
void DeviceNetwork::publishAck(const CommandAck &ack) {
    if (m_thread.isRunning()) QMetaObject::invokeMethod(m_worker, [this, ack]() { m_worker->publishAck(ack); }, Qt::QueuedConnection);
}
void DeviceNetwork::fetchConfiguration() { if (m_thread.isRunning()) QMetaObject::invokeMethod(m_worker, &DeviceNetworkWorker::fetchConfiguration, Qt::QueuedConnection); }
void DeviceNetwork::checkForUpdate(const QString &version) { if (m_thread.isRunning()) QMetaObject::invokeMethod(m_worker, [this, version]() { m_worker->checkForUpdate(version); }, Qt::QueuedConnection); }
void DeviceNetwork::uploadDiagnostic(const QJsonObject &data) { if (m_thread.isRunning()) QMetaObject::invokeMethod(m_worker, [this, data]() { m_worker->uploadDiagnostic(data); }, Qt::QueuedConnection); }
void DeviceNetwork::uploadBatch(const QString &id) { if (m_thread.isRunning()) QMetaObject::invokeMethod(m_worker, [this, id]() { m_worker->uploadBatch(id); }, Qt::QueuedConnection); }

} // namespace aurora

#include "DeviceNetwork.moc"
