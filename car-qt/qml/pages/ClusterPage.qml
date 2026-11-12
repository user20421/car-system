import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Aurora

Item {
    id: root
    property string gaugeTheme: "舒适"
    GlassPanel {
        anchors.fill: parent
        highlighted: true
        gradient: Gradient { GradientStop { position: 0; color: "#101c29" } GradientStop { position: 1; color: "#080d14" } }
        ColumnLayout {
            anchors.fill: parent; anchors.margins: 26; spacing: 12
            RowLayout {
                SectionHeader { Layout.fillWidth: true; title: "数字驾驶舱"; subtitle: vehicle.speed > 1 ? "DRIVE · 行驶数据实时更新" : "READY · 车辆已就绪" }
                StatusPill { connectionState: vehicle.faults.length ? "offline" : "online"; label: vehicle.faults.length ? "功率受限" : "系统正常" }
            }
            RowLayout {
                Layout.fillWidth: true; Layout.fillHeight: true; spacing: 16
                CircularGauge {
                    Layout.fillWidth: true; Layout.fillHeight: true; Layout.maximumWidth: 390
                    value: vehicle.rpm; maximum: 7200; valueText: vehicle.rpm.toLocaleString(); unit: "rpm"; caption: "MOTOR"; accent: Theme.violet
                }
                ColumnLayout {
                    Layout.fillWidth: true; Layout.fillHeight: true; spacing: 8
                    Text { text: Theme.speedText(vehicle.speed); color: Theme.text; font.pixelSize: Math.min(root.height * 0.22, 150); font.weight: Font.Light; font.family: Theme.monoFamily; Layout.alignment: Qt.AlignHCenter }
                    Text { text: "km/h"; color: Theme.muted; font.pixelSize: 15; font.letterSpacing: 4; Layout.alignment: Qt.AlignHCenter }
                    Rectangle { Layout.preferredWidth: 100; Layout.preferredHeight: 34; Layout.alignment: Qt.AlignHCenter; radius: 17; color: Theme.cyanSoft; Text { anchors.centerIn: parent; text: vehicle.speed > 0 ? "D  前进" : "P  驻车"; color: Theme.cyan; font.pixelSize: 13 } }
                    Item { Layout.fillHeight: true }
                    RowLayout {
                        Layout.alignment: Qt.AlignHCenter; spacing: 26
                        Column { Text { text: Math.round(vehicle.battery) + "%"; color: Theme.text; font.pixelSize: 23; font.family: Theme.monoFamily; anchors.horizontalCenter: parent.horizontalCenter } Text { text: "电量"; color: Theme.muted; font.pixelSize: 10; anchors.horizontalCenter: parent.horizontalCenter } }
                        Rectangle { width: 1; height: 36; color: Theme.stroke }
                        Column { Text { text: Math.round(vehicle.range); color: Theme.text; font.pixelSize: 23; font.family: Theme.monoFamily; anchors.horizontalCenter: parent.horizontalCenter } Text { text: "续航 km"; color: Theme.muted; font.pixelSize: 10; anchors.horizontalCenter: parent.horizontalCenter } }
                        Rectangle { width: 1; height: 36; color: Theme.stroke }
                        Column { Text { text: vehicle.cabinTemperature.toFixed(1) + "°"; color: Theme.text; font.pixelSize: 23; font.family: Theme.monoFamily; anchors.horizontalCenter: parent.horizontalCenter } Text { text: "座舱"; color: Theme.muted; font.pixelSize: 10; anchors.horizontalCenter: parent.horizontalCenter } }
                    }
                }
                CircularGauge {
                    Layout.fillWidth: true; Layout.fillHeight: true; Layout.maximumWidth: 390
                    value: vehicle.speed; maximum: 160; valueText: Math.round(vehicle.speed); unit: "km/h"; caption: "VELOCITY"; accent: Theme.cyan
                }
            }
            RowLayout {
                Layout.fillWidth: true; spacing: 12
                Repeater {
                    model: ["极简", "舒适", "运动"]
                    delegate: Button {
                        required property string modelData
                        Layout.fillWidth: true; implicitHeight: 45; text: modelData + "主题"
                        onClicked: root.gaugeTheme = modelData
                        contentItem: Text { text: parent.text; color: root.gaugeTheme === modelData ? Theme.cyan : Theme.muted; horizontalAlignment: Text.AlignHCenter; font.pixelSize: 12 }
                        background: Rectangle { radius: 14; color: root.gaugeTheme === modelData ? Theme.cyanSoft : Theme.surfaceRaised; border.color: root.gaugeTheme === modelData ? Theme.cyan : Theme.stroke }
                    }
                }
            }
        }
    }
}
