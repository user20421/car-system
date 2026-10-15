#pragma once

#include "hal/IVehicleData.h"
#include <QTimer>

namespace hal {

/**
 * @brief 车辆数据模拟器（无真实硬件时的 HAL 实现）
 *
 * 模拟策略（让数据"像真车"）：
 * - 车速：正弦巡航 + 随机扰动，周期性启停
 * - 转速：与车速联动 + 噪声
 * - 电量：行驶缓慢耗电，停车（速度≈0）缓慢回充
 * - GPS：围绕上海市中心坐标做小范围漂移
 * - 故障码：低概率随机产生，持续数帧后自动清除
 */
class VehicleSimulator : public IVehicleData
{
    Q_OBJECT
public:
    explicit VehicleSimulator(QObject *parent = nullptr);

    void start() override;
    void stop() override;

private slots:
    void tick();

private:
    QTimer m_timer;
    int    m_tickCount = 0;
    double m_battery = 85.0;
    double m_lat = 31.2304;     // 上海市中心
    double m_lng = 121.4737;
    int    m_faultCountdown = 0;
    QString m_currentFault;
};

} // namespace hal
