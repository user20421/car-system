#include "hal/VehicleSimulator.h"
#include <QRandomGenerator>
#include <QtMath>

namespace hal {

VehicleSimulator::VehicleSimulator(QObject *parent)
    : IVehicleData(parent)
{
    m_timer.setInterval(1000); // 每秒一帧，模拟 1Hz CAN 数据
    connect(&m_timer, &QTimer::timeout, this, &VehicleSimulator::tick);
}

void VehicleSimulator::start() { m_timer.start(); }
void VehicleSimulator::stop()  { m_timer.stop(); }

void VehicleSimulator::tick()
{
    ++m_tickCount;
    auto *rng = QRandomGenerator::global();

    // ---- 车速：120s 大周期的巡航正弦 + 噪声，部分周期停车 ----
    const double phase = m_tickCount / 120.0 * 2 * M_PI;
    double base = 60.0 * (1.0 + qSin(phase));   // 0 ~ 120 巡航
    if (qSin(phase * 0.5) < -0.7)               // 周期性"停车"
        base = 0.0;
    const double speed = qMax(0.0, base + (rng->generateDouble() - 0.5) * 10.0);

    // ---- 转速：与车速联动 ----
    const int rpm = speed < 1.0 ? 800 + rng->bounded(100)
                                : int(1200 + speed * 25 + rng->bounded(-200, 200));

    // ---- 电量：行驶耗电 0.02%/s，停车回充 0.05%/s ----
    m_battery += (speed < 1.0 ? 0.05 : -0.02);
    m_battery = qBound(15.0, m_battery, 100.0);

    // ---- GPS：小范围漂移 ----
    m_lat += rng->bounded(-20, 21) / 100000.0;
    m_lng += rng->bounded(-20, 21) / 100000.0;

    // ---- 故障码：0.5% 概率触发，持续 5~15 帧后自动清除 ----
    if (m_faultCountdown > 0) {
        if (--m_faultCountdown == 0)
            m_currentFault.clear();
    } else if (rng->bounded(200) == 0) {
        const QStringList codes = {"P0300", "P0420", "B1001", "U0100"};
        m_currentFault = codes.at(rng->bounded(codes.size()));
        m_faultCountdown = 5 + rng->bounded(10);
    }

    TelemetryFrame frame;
    frame.speed = qRound(speed * 10.0) / 10.0;
    frame.rpm = rpm;
    frame.battery = qRound(m_battery * 10.0) / 10.0;
    frame.lat = m_lat;
    frame.lng = m_lng;
    frame.faultCode = m_currentFault;
    emit frameReady(frame);
}

} // namespace hal
