#include "core/VehicleController.h"

namespace core {

QString VehicleController::tripTime() const
{
    const int h = m_tripSeconds / 3600;
    const int m = (m_tripSeconds % 3600) / 60;
    const int s = m_tripSeconds % 60;
    return QStringLiteral("%1:%2:%3")
        .arg(h, 2, 10, QLatin1Char('0'))
        .arg(m, 2, 10, QLatin1Char('0'))
        .arg(s, 2, 10, QLatin1Char('0'));
}

VehicleController::VehicleController(hal::IVehicleData *dataSource,
                                     net::ApiClient *api,
                                     QObject *parent)
    : QObject(parent), m_dataSource(dataSource), m_api(api)
{
    qRegisterMetaType<hal::TelemetryFrame>("hal::TelemetryFrame");

    // 硬件帧 → 更新界面 + 上报后端
    connect(m_dataSource, &hal::IVehicleData::frameReady,
            this, [this](const hal::TelemetryFrame &frame) {
        m_frame = frame;
        // 行程统计：每秒一帧，里程 += 速度/3600
        m_tripSeconds++;
        m_tripDistance += frame.speed / 3600.0;
        emit dataChanged();
        m_api->reportTelemetry(frame);
    });

    // 网络状态 → 界面
    connect(m_api, &net::ApiClient::onlineChanged, this, &VehicleController::onlineChanged);
    connect(m_api, &net::ApiClient::pendingCountChanged, this, &VehicleController::pendingChanged);

    // 控车指令 → 模拟执行 → 回执
    connect(m_api, &net::ApiClient::commandReceived,
            this, [this](uint id, const QString &cmd) {
        // 模拟车机执行指令
        if (cmd == "lock")        { m_locked = true;  m_lastCommand = "🔒 车门已上锁"; }
        else if (cmd == "unlock") { m_locked = false; m_lastCommand = "🔓 车门已解锁"; }
        else if (cmd == "ac_on")  { m_acOn = true;    m_lastCommand = "❄️ 空调已开启"; }
        else if (cmd == "ac_off") { m_acOn = false;   m_lastCommand = "💨 空调已关闭"; }
        else if (cmd == "honk")   { m_lastCommand = "📢 鸣笛！"; }
        // 空调负载联动：影响模拟器能耗
        m_dataSource->setAccessoryLoad(m_acOn);
        emit stateChanged();
        m_api->ackCommand(id, "executed");
    });

    // 服务端告警（规则/围栏）→ 车机屏幕提醒驾驶员
    connect(m_api, &net::ApiClient::alarmReceived,
            this, [this](const QString &title, const QString &detail, const QString &severity) {
        m_lastAlarm = QStringLiteral("🚨 %1：%2").arg(title, detail);
        m_lastAlarmSeverity = severity;
        emit stateChanged();
    });
}

} // namespace core
