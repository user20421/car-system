#include "core/VehicleController.h"

namespace core {

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
        emit stateChanged();
        m_api->ackCommand(id, "executed");
    });
}

} // namespace core
