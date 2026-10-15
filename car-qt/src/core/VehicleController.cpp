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
}

} // namespace core
