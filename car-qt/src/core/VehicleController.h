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

public:
    explicit VehicleController(hal::IVehicleData *dataSource,
                               net::ApiClient *api,
                               QObject *parent = nullptr);

    double speed() const { return m_frame.speed; }
    int rpm() const { return m_frame.rpm; }
    double battery() const { return m_frame.battery; }
    double lat() const { return m_frame.lat; }
    double lng() const { return m_frame.lng; }
    QString faultCode() const { return m_frame.faultCode; }
    bool online() const { return m_api->online(); }
    int pendingCount() const { return m_api->pendingCount(); }

signals:
    void dataChanged();
    void onlineChanged();
    void pendingChanged();

private:
    hal::TelemetryFrame m_frame;
    hal::IVehicleData *m_dataSource;
    net::ApiClient *m_api;
};

} // namespace core
