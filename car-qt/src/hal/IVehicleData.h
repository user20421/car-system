#pragma once

#include <QObject>
#include <QString>

namespace hal {

// 车况数据快照（模拟 CAN 总线一帧解析后的结果）
struct TelemetryFrame {
    double speed = 0.0;     // km/h
    int    rpm = 0;         // 发动机转速
    double battery = 0.0;   // 电量 %
    double lat = 0.0;       // 纬度
    double lng = 0.0;       // 经度
    QString faultCode;      // 故障码，空串 = 无故障
};

/**
 * @brief 硬件抽象接口（HAL）
 *
 * 真实场景：由 CAN 驱动 / 串口驱动实现本接口。
 * 本地开发：由 VehicleSimulator（模拟器）实现。
 * 应用层只依赖本接口，换真车硬件时业务代码零改动。
 */
class IVehicleData : public QObject
{
    Q_OBJECT
public:
    explicit IVehicleData(QObject *parent = nullptr) : QObject(parent) {}
    ~IVehicleData() override = default;

    virtual void start() = 0;   // 启动硬件采集
    virtual void stop() = 0;    // 停止采集

signals:
    // 有新一帧车况数据
    void frameReady(const hal::TelemetryFrame &frame);
};

} // namespace hal

Q_DECLARE_METATYPE(hal::TelemetryFrame)
