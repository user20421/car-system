import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Aurora

Item {
    id: root
    property bool animations: true
    property bool developerInfo: debugBuild
    Flickable {
        anchors.fill: parent; contentWidth: width; contentHeight: column.implicitHeight; clip: true
        ColumnLayout {
            id: column; width: parent.width; spacing: 18
            GridLayout {
                Layout.fillWidth: true; columns: root.width > 1100 ? 2 : 1; columnSpacing: 18; rowSpacing: 18
                GlassPanel {
                    Layout.fillWidth: true; Layout.preferredHeight: 360
                    ColumnLayout {
                        anchors.fill: parent; anchors.margins: 22; spacing: 14
                        SectionHeader { title: "连接与设备服务"; subtitle: "MQTTS + Protobuf · HTTPS/2 · mTLS"; trailing: vehicle.networkState.toUpperCase() }
                        GlassPanel {
                            Layout.fillWidth: true; height: 76; border.color: vehicle.networkState === "online" ? Theme.green : Theme.amber
                            RowLayout { anchors.fill: parent; anchors.margins: 14; StatusPill { connectionState: vehicle.networkState; label: vehicle.networkState === "online" ? "在线" : vehicle.networkState === "connecting" ? "连接中" : "离线" } Text { text: vehicle.networkDetail; color: Theme.muted; font.pixelSize: 10; wrapMode: Text.Wrap; Layout.fillWidth: true } }
                        }
                        RowLayout { Layout.fillWidth: true; spacing: 10; ActionButton { Layout.fillWidth: true; text: "同步配置"; glyph: "↻"; enabled: !vehicle.requestBusy; onClicked: vehicle.fetchConfiguration() } ActionButton { Layout.fillWidth: true; text: "检查 OTA"; glyph: "⇧"; enabled: !vehicle.requestBusy; onClicked: vehicle.checkForUpdate() } }
                        RowLayout { Layout.fillWidth: true; spacing: 10; ActionButton { Layout.fillWidth: true; text: "上传诊断"; glyph: "↗"; enabled: !vehicle.requestBusy; onClicked: vehicle.sendDiagnostic() } ActionButton { Layout.fillWidth: true; text: "离线补传"; glyph: "▤"; enabled: !vehicle.requestBusy; onClicked: vehicle.retryBatch() } }
                        BusyIndicator { visible: vehicle.requestBusy; running: visible; Layout.alignment: Qt.AlignHCenter }
                        Text { text: vehicle.requestMessage; color: vehicle.requestMessage.indexOf("失败") >= 0 ? Theme.red : Theme.muted; font.pixelSize: 11; wrapMode: Text.Wrap; Layout.fillWidth: true }
                    }
                }
                GlassPanel {
                    Layout.fillWidth: true; Layout.preferredHeight: 360
                    ColumnLayout {
                        anchors.fill: parent; anchors.margins: 22; spacing: 16
                        SectionHeader { title: "显示与交互"; subtitle: "个性化 AURORA 座舱体验" }
                        RowLayout { ColumnLayout { Text { text: "界面动态效果"; color: Theme.text; font.pixelSize: 13 } Text { text: "平滑过渡和仪表动画"; color: Theme.muted; font.pixelSize: 10 } } Item { Layout.fillWidth: true } Switch { checked: root.animations; onToggled: root.animations = checked } }
                        RowLayout { ColumnLayout { Text { text: "驾驶专注模式"; color: Theme.text; font.pixelSize: 13 } Text { text: "行驶时减少非关键信息"; color: Theme.muted; font.pixelSize: 10 } } Item { Layout.fillWidth: true } Switch { checked: vehicle.speed > 1; enabled: false } }
                        RowLayout { ColumnLayout { Text { text: "开发诊断信息"; color: Theme.text; font.pixelSize: 13 } Text { text: "显示场景与线程调试面板"; color: Theme.muted; font.pixelSize: 10 } } Item { Layout.fillWidth: true } Switch { checked: root.developerInfo; enabled: false } }
                        Rectangle { Layout.fillWidth: true; height: 1; color: Theme.stroke }
                        Text { text: "界面语言"; color: Theme.muted; font.pixelSize: 11 }
                        ActionButton { Layout.fillWidth: true; text: "简体中文"; glyph: "文" }
                    }
                }
            }
            GlassPanel {
                Layout.fillWidth: true; Layout.preferredHeight: 170
                RowLayout {
                    anchors.fill: parent; anchors.margins: 24; spacing: 24
                    Rectangle { width: 72; height: 72; radius: 23; color: Theme.cyan; Text { anchors.centerIn: parent; text: "A"; color: Theme.background; font.pixelSize: 42; font.weight: Font.Black } }
                    ColumnLayout { Text { text: "AURORA OS"; color: Theme.text; font.pixelSize: 22; font.weight: Font.DemiBold } Text { text: "版本 1.0.0  ·  Qt 6 / QML  ·  C++17"; color: Theme.muted; font.pixelSize: 11 } Text { text: "VIN  " + vehicle.vin + "  ·  独立车机实例"; color: Theme.cyan; font.pixelSize: 10; font.family: Theme.monoFamily } }
                    Item { Layout.fillWidth: true }
                    Text { text: "安全设计\nTLS 1.2+ / 对等证书校验\n指令过期 / 去重 / 串行 ACK"; color: Theme.muted; font.pixelSize: 11; lineHeight: 1.55 }
                }
            }
        }
    }
}
