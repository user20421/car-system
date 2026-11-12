#pragma once

#include "core/VehicleTypes.h"
#include <QObject>

namespace aurora {

// Hardware boundary: implementations own device I/O and simulation. The UI and
// controller never mutate hardware state or invent command results.
class IVehicleHardware : public QObject {
    Q_OBJECT
public:
    using QObject::QObject;
    ~IVehicleHardware() override = default;

public slots:
    virtual void start() = 0;
    virtual void stop() = 0;
    virtual void execute(const aurora::VehicleCommand &command) = 0;

signals:
    void telemetryProduced(const aurora::Telemetry &telemetry);
    void commandCompleted(const aurora::CommandAck &ack);
    void hardwareError(const QString &code, const QString &message);
};

} // namespace aurora
