#pragma once

#include <QObject>
#include <QString>
#include "hal/IVehicleData.h"
#include "net/ApiClient.h"

namespace core {

/**
 * @brief 车机业务控制器：QML 界面 ⇄ HAL 硬件层 ⇄ 网络层 的桥接
 *
 * 职责：订阅硬件数据帧 → 更新界面属性 → 上报后端
 */
class VehicleController : public QObject
{
    Q_OBJECT
    Q_PROPERTY(double speed READ speed NOTIFY dataChanged)
    Q_PROPERTY(int rpm READ rpm NOTIFY dataChanged)
    Q_PROPERTY(double battery READ battery NOTIFY dataChanged)
    Q_PROPERTY(double lat READ lat NOTIFY dataChanged)
    Q_PROPERTY(double lng READ lng NOTIFY dataChanged)
    Q_PROPERTY(QString faultCode READ faultCode NOTIFY dataChanged)
    Q_PROPERTY(bool online READ online NOTIFY onlineChanged)
    Q_PROPERTY(int pendingCount READ pendingCount NOTIFY pendingChanged)
    Q_PROPERTY(bool locked READ locked NOTIFY stateChanged)
    Q_PROPERTY(bool acOn READ acOn NOTIFY stateChanged)
    Q_PROPERTY(QString lastCommand READ lastCommand NOTIFY stateChanged)
    Q_PROPERTY(QString lastAlarm READ lastAlarm NOTIFY stateChanged)
    Q_PROPERTY(QString lastAlarmSeverity READ lastAlarmSeverity NOTIFY stateChanged)
    Q_PROPERTY(QString tripTime READ tripTime NOTIFY dataChanged)
    Q_PROPERTY(double tripDistance READ tripDistance NOTIFY dataChanged)
    Q_PROPERTY(double tripAvgSpeed READ tripAvgSpeed NOTIFY dataChanged)

public:
    explicit VehicleController(hal::IVehicleData *dataSource,
                               net::ApiClient *api,
                               QObject *parent = nullptr);

    // 中控屏本地操作（仅限空调等车内合理操作，锁车/鸣笛仅远程）
    Q_INVOKABLE void requestCommand(const QString &cmd) { m_api->sendCommand(cmd); }

    double speed() const { return m_frame.speed; }
    int rpm() const { return m_frame.rpm; }
    double battery() const { return m_frame.battery; }
    double lat() const { return m_frame.lat; }
    double lng() const { return m_frame.lng; }
    QString faultCode() const { return m_frame.faultCode; }
    bool online() const { return m_api->online(); }
    int pendingCount() const { return m_api->pendingCount(); }
    bool locked() const { return m_locked; }
    bool acOn() const { return m_acOn; }
    QString lastCommand() const { return m_lastCommand; }
    QString lastAlarm() const { return m_lastAlarm; }
    QString lastAlarmSeverity() const { return m_lastAlarmSeverity; }
    QString tripTime() const;
    double tripDistance() const { return m_tripDistance; }
    double tripAvgSpeed() const { return m_tripSeconds > 0 ? m_tripDistance / (m_tripSeconds / 3600.0) : 0.0; }

signals:
    void dataChanged();
    void onlineChanged();
    void pendingChanged();
    void stateChanged();

private:
    hal::TelemetryFrame m_frame;
    hal::IVehicleData *m_dataSource;
    net::ApiClient *m_api;
    bool m_locked = true;
    bool m_acOn = false;
    QString m_lastCommand;
    QString m_lastAlarm;         // 最近一次告警提示文本（触发 Toast）
    QString m_lastAlarmSeverity; // 告警级别：info | warning | critical
    int m_tripSeconds = 0;        // 本次行程时长（秒）
    double m_tripDistance = 0.0;  // 本次行程里程（km）
};

} // namespace core
