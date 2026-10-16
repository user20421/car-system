#pragma once

#include <QObject>
#include <QQueue>
#include <QTimer>
#include <QNetworkAccessManager>
#include <QWebSocket>
#include "hal/IVehicleData.h"

namespace net {

/**
 * @brief 后端 API 客户端
 *
 * 能力：
 * - 启动时登录获取 JWT，401 时自动重新登录
 * - 车况上报失败时进入内存离线队列，网络恢复后按序补传
 * - 周期性心跳检测后端连通性
 */
class ApiClient : public QObject
{
    Q_OBJECT
public:
    explicit ApiClient(QObject *parent = nullptr);

    void configure(const QString &baseUrl, const QString &vin,
                   const QString &username, const QString &password);
    void start();

    bool online() const { return m_online; }
    int  pendingCount() const { return m_pending.size(); }

    void reportTelemetry(const hal::TelemetryFrame &frame);
    void ackCommand(uint id, const QString &status);

signals:
    void onlineChanged(bool online);
    void pendingCountChanged(int count);
    // 收到控车指令（id, command）
    void commandReceived(uint id, const QString &command);

private slots:
    void heartbeat();
    void flushPending();
    void onWsTextMessage(const QString &message);
    void onWsDisconnected();

private:
    void login();
    void postTelemetry(const QByteArray &payload, bool fromQueue);
    QByteArray toJson(const hal::TelemetryFrame &frame) const;
    void setOnline(bool on);
    void setPendingCount(int n);
    void connectWs();

    QNetworkAccessManager m_nam;
    QWebSocket m_ws;                       // 指令接收通道
    QTimer m_wsReconnectTimer;
    QTimer m_heartbeatTimer;
    QTimer m_flushTimer;
    QQueue<QByteArray> m_pending;      // 离线缓存队列（上限 1000 条）

    QString m_baseUrl;
    QString m_vin;
    QString m_username;
    QString m_password;
    QString m_token;
    bool m_online = false;
    bool m_loggingIn = false;
};

} // namespace net
