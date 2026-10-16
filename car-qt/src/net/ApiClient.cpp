#include "net/ApiClient.h"
#include <QJsonDocument>
#include <QJsonObject>
#include <QNetworkReply>
#include <QNetworkRequest>

namespace net {

ApiClient::ApiClient(QObject *parent) : QObject(parent)
{
    m_heartbeatTimer.setInterval(5000);
    m_flushTimer.setInterval(3000);
    m_wsReconnectTimer.setInterval(3000);
    connect(&m_heartbeatTimer, &QTimer::timeout, this, &ApiClient::heartbeat);
    connect(&m_flushTimer, &QTimer::timeout, this, &ApiClient::flushPending);
    connect(&m_wsReconnectTimer, &QTimer::timeout, this, &ApiClient::connectWs);
    connect(&m_ws, &QWebSocket::textMessageReceived, this, &ApiClient::onWsTextMessage);
    connect(&m_ws, &QWebSocket::disconnected, this, &ApiClient::onWsDisconnected);
}

void ApiClient::configure(const QString &baseUrl, const QString &vin,
                          const QString &username, const QString &password)
{
    m_baseUrl = baseUrl;
    m_vin = vin;
    m_username = username;
    m_password = password;
}

void ApiClient::start()
{
    login();
    connectWs();
    m_heartbeatTimer.start();
    m_flushTimer.start();
    m_wsReconnectTimer.start();
}

// ===== WebSocket 指令通道 =====

void ApiClient::connectWs()
{
    if (m_ws.state() == QAbstractSocket::UnconnectedState) {
        if (m_token.isEmpty()) {
            return; // 未登录不连，登录成功后由重连定时器触发
        }
        // WS 握手鉴权：token 走查询参数，连接后按 VIN 过滤自己的指令
        m_ws.open(QUrl(QStringLiteral("ws://%1/ws?token=%2")
                           .arg(m_baseUrl.mid(7), QString(QUrl::toPercentEncoding(m_token))))); // http:// → ws://
    }
}

void ApiClient::onWsTextMessage(const QString &message)
{
    const auto doc = QJsonDocument::fromJson(message.toUtf8());
    if (!doc.isObject())
        return;
    const auto obj = doc.object();
    const QString type = obj.value("type").toString();

    if (type == QLatin1String("alarm")) {
        // 告警消息：{"type":"alarm","data":{vin,title,detail,severity}}
        const auto data = obj.value("data").toObject();
        if (data.value("vin").toString() != m_vin)
            return; // 不是本车的告警
        emit alarmReceived(data.value("title").toString(),
                           data.value("detail").toString(),
                           data.value("severity").toString());
        return;
    }

    if (type != QLatin1String("command"))
        return;                                    // 只处理控车指令与告警
    if (obj.value("vin").toString() != m_vin)
        return;                                    // 不是给本车的
    const uint id = static_cast<uint>(obj.value("id").toInt());
    const QString cmd = obj.value("command").toString();
    emit commandReceived(id, cmd);
}

void ApiClient::onWsDisconnected()
{
    // 重连由 m_wsReconnectTimer 周期触发
}

// ===== 本车本地操作（走统一指令链路） =====

void ApiClient::sendCommand(const QString &command)
{
    QNetworkRequest req(QUrl(QStringLiteral("%1/api/v1/vehicle/%2/command").arg(m_baseUrl, m_vin)));
    req.setHeader(QNetworkRequest::ContentTypeHeader, "application/json");
    req.setRawHeader("Authorization", "Bearer " + m_token.toUtf8());
    QJsonObject body{{"command", command}};
    auto *reply = m_nam.post(req, QJsonDocument(body).toJson(QJsonDocument::Compact));
    connect(reply, &QNetworkReply::finished, reply, &QNetworkReply::deleteLater);
}

// ===== 指令回执 =====

void ApiClient::ackCommand(uint id, const QString &status)
{
    QNetworkRequest req(QUrl(QStringLiteral("%1/api/v1/vehicle/%2/command/%3/ack")
                                 .arg(m_baseUrl, m_vin).arg(id)));
    req.setHeader(QNetworkRequest::ContentTypeHeader, "application/json");
    req.setRawHeader("Authorization", "Bearer " + m_token.toUtf8());
    QJsonObject body{{"status", status}};
    auto *reply = m_nam.post(req, QJsonDocument(body).toJson(QJsonDocument::Compact));
    connect(reply, &QNetworkReply::finished, reply, &QNetworkReply::deleteLater);
}

void ApiClient::setOnline(bool on)
{
    if (m_online != on) {
        m_online = on;
        emit onlineChanged(on);
    }
}

void ApiClient::setPendingCount(int n)
{
    if (pendingCount() != n)
        emit pendingCountChanged(n);
}

void ApiClient::login()
{
    if (m_loggingIn)
        return;
    m_loggingIn = true;

    QNetworkRequest req(QUrl(m_baseUrl + "/api/v1/auth/login"));
    req.setHeader(QNetworkRequest::ContentTypeHeader, "application/json");
    QJsonObject body{{"username", m_username}, {"password", m_password}};

    auto *reply = m_nam.post(req, QJsonDocument(body).toJson(QJsonDocument::Compact));
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        m_loggingIn = false;
        reply->deleteLater();
        if (reply->error() == QNetworkReply::NoError) {
            const auto doc = QJsonDocument::fromJson(reply->readAll());
            m_token = doc.object().value("data").toObject().value("token").toString();
            setOnline(!m_token.isEmpty());
        } else {
            setOnline(false);
        }
    });
}

QByteArray ApiClient::toJson(const hal::TelemetryFrame &f) const
{
    QJsonObject obj;
    obj["vin"] = m_vin;
    obj["speed"] = f.speed;
    obj["rpm"] = f.rpm;
    obj["battery"] = f.battery;
    obj["lat"] = f.lat;
    obj["lng"] = f.lng;
    obj["fault_code"] = f.faultCode;
    return QJsonDocument(obj).toJson(QJsonDocument::Compact);
}

void ApiClient::reportTelemetry(const hal::TelemetryFrame &frame)
{
    const QByteArray payload = toJson(frame);
    if (!m_online) {                        // 离线 → 入队缓存
        if (m_pending.size() < 1000)
            m_pending.enqueue(payload);
        setPendingCount(m_pending.size());
        return;
    }
    postTelemetry(payload, false);
}

void ApiClient::postTelemetry(const QByteArray &payload, bool fromQueue)
{
    QNetworkRequest req(QUrl(m_baseUrl + "/api/v1/vehicle/telemetry"));
    req.setHeader(QNetworkRequest::ContentTypeHeader, "application/json");
    req.setRawHeader("Authorization", "Bearer " + m_token.toUtf8());

    auto *reply = m_nam.post(req, payload);
    connect(reply, &QNetworkReply::finished, this, [this, reply, payload, fromQueue]() {
        reply->deleteLater();
        const int status = reply->attribute(QNetworkRequest::HttpStatusCodeAttribute).toInt();
        if (reply->error() == QNetworkReply::NoError && status == 200) {
            if (fromQueue)
                setPendingCount(m_pending.size());
            setOnline(true);
        } else if (status == 401) {
            setOnline(false);
            login();                         // Token 过期 → 重新登录
        } else {
            setOnline(false);                // 网络错误 → 进入离线
            if (m_pending.size() < 1000)
                m_pending.enqueue(payload);  // 回队等待补传
            setPendingCount(m_pending.size());
        }
    });
}

void ApiClient::flushPending()
{
    if (m_online && !m_pending.isEmpty())
        postTelemetry(m_pending.dequeue(), true);
    setPendingCount(m_pending.size());
}

void ApiClient::heartbeat()
{
    QNetworkRequest req(QUrl(m_baseUrl + "/health"));
    auto *reply = m_nam.get(req);
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        reply->deleteLater();
        const bool up = (reply->error() == QNetworkReply::NoError);
        if (up && m_token.isEmpty())
            login();
        else
            setOnline(up);
    });
}

} // namespace net
