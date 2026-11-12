#pragma once

#include "hal/IVehicleHardware.h"
#include <QElapsedTimer>
#include <QTimer>

namespace aurora {

class SimulatedVehicleHardware final : public IVehicleHardware {
    Q_OBJECT
public:
    explicit SimulatedVehicleHardware(QString vin, QObject *parent = nullptr);

public slots:
    void start() override;
    void stop() override;
    void execute(const aurora::VehicleCommand &command) override;
    void setScenario(const QString &scenario);

private:
    void tick();
    void finish(const VehicleCommand &command, CommandAck::Status status,
                const QString &code = {}, const QString &message = {});
    Telemetry snapshot() const;

    const QString m_vin;
    const QString m_bootId;
    QTimer *m_timer = nullptr;
    QElapsedTimer m_clock;
    Telemetry m_state;
    QString m_scenario = QStringLiteral("city");
    qint64 m_lastTickMs = 0;
    double m_distance = 0.0;
    bool m_running = false;
};

} // namespace aurora
