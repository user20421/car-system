#include "storage/TelemetryStore.h"

#include <QCoreApplication>
#include <QDir>
#include <QJsonDocument>
#include <QJsonObject>
#include <QSaveFile>
#include <QStandardPaths>
#include <QTimer>

namespace aurora {

class TelemetryStoreWorker final : public QObject {
    Q_OBJECT
public:
    explicit TelemetryStoreWorker(QString vin) : m_vin(std::move(vin)) {}

public slots:
    void start() {
        m_timer = new QTimer(this);
        m_timer->setInterval(1000);
        connect(m_timer, &QTimer::timeout, this, &TelemetryStoreWorker::flush);
        m_timer->start();
    }
    void storeLatest(const Telemetry &telemetry) {
        m_latest = telemetry; // latest-value coalescing, never an unbounded queue
        m_dirty = true;
    }
    void flush() {
        if (!m_dirty) return;
        const QString root = QStandardPaths::writableLocation(QStandardPaths::AppLocalDataLocation);
        QDir().mkpath(root + QStringLiteral("/telemetry"));
        QSaveFile file(root + QStringLiteral("/telemetry/%1-latest.json").arg(m_vin));
        if (!file.open(QIODevice::WriteOnly)) {
            emit storageFailed(file.errorString());
            return;
        }
        const QJsonObject object{
            {QStringLiteral("vin"), m_latest.vin}, {QStringLiteral("sequence"), QString::number(m_latest.sequence)},
            {QStringLiteral("occurred_at_ms"), QString::number(m_latest.occurredAtMs)},
            {QStringLiteral("speed_kph"), m_latest.speedKph}, {QStringLiteral("rpm"), m_latest.rpm},
            {QStringLiteral("battery_percent"), m_latest.batteryPercent}, {QStringLiteral("range_km"), m_latest.rangeKm},
            {QStringLiteral("latitude"), m_latest.location.latitude}, {QStringLiteral("longitude"), m_latest.location.longitude},
            {QStringLiteral("locked"), m_latest.locked}, {QStringLiteral("cabin_temp_c"), m_latest.cabinTempC},
            {QStringLiteral("defrost"), m_latest.defrost}
        };
        file.write(QJsonDocument(object).toJson(QJsonDocument::Compact));
        if (!file.commit()) {
            emit storageFailed(file.errorString());
            return;
        }
        m_dirty = false;
    }
    void stop() {
        if (m_timer) m_timer->stop();
        flush();
    }
signals:
    void storageFailed(const QString &message);
private:
    QString m_vin;
    QTimer *m_timer = nullptr;
    Telemetry m_latest;
    bool m_dirty = false;
};

TelemetryStore::TelemetryStore(QString vin, QObject *parent) : QObject(parent) {
    m_worker = new TelemetryStoreWorker(std::move(vin));
    m_worker->moveToThread(&m_thread);
    m_thread.setObjectName(QStringLiteral("storage-worker"));
    connect(&m_thread, &QThread::started, m_worker, &TelemetryStoreWorker::start);
    connect(&m_thread, &QThread::finished, m_worker, &QObject::deleteLater);
    connect(m_worker, &TelemetryStoreWorker::storageFailed, this, &TelemetryStore::storageFailed,
            Qt::QueuedConnection);
}

TelemetryStore::~TelemetryStore() { stop(); }
void TelemetryStore::start() { if (!m_thread.isRunning()) m_thread.start(); }
void TelemetryStore::stop() {
    if (!m_thread.isRunning()) return;
    QMetaObject::invokeMethod(m_worker, &TelemetryStoreWorker::stop, Qt::BlockingQueuedConnection);
    m_thread.quit();
    m_thread.wait(3000);
}
void TelemetryStore::storeLatest(const Telemetry &telemetry) {
    if (m_thread.isRunning())
        QMetaObject::invokeMethod(m_worker, [this, telemetry]() { m_worker->storeLatest(telemetry); },
                                  Qt::QueuedConnection);
}

} // namespace aurora

#include "TelemetryStore.moc"
