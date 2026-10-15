import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    id: window
    width: 1024
    height: 600
    visible: true
    title: "Car-System 车机模拟器"
    color: "#0d1117"

    // ===== 顶部标题栏 =====
    Rectangle {
        id: header
        anchors.top: parent.top
        width: parent.width
        height: 52
        color: "#161b22"

        Text {
            anchors.left: parent.left
            anchors.leftMargin: 20
            anchors.verticalCenter: parent.verticalCenter
            text: "🚗 Car-System | VIN: LVSFACBE8MF012345"
            color: "#e6edf3"
            font.pixelSize: 17
            font.bold: true
        }

        // 连接状态指示
        Row {
            anchors.right: parent.right
            anchors.rightMargin: 20
            anchors.verticalCenter: parent.verticalCenter
            spacing: 8

            Rectangle {
                width: 12; height: 12; radius: 6
                anchors.verticalCenter: parent.verticalCenter
                color: vehicle.online ? "#3fb950" : "#ff5252"
                // 呼吸灯效果
                SequentialAnimation on opacity {
                    running: vehicle.online
                    loops: Animation.Infinite
                    NumberAnimation { to: 0.3; duration: 800 }
                    NumberAnimation { to: 1.0; duration: 800 }
                }
            }
            Text {
                text: vehicle.online ? "在线" : "离线（缓存 " + vehicle.pendingCount + " 条）"
                color: vehicle.online ? "#3fb950" : "#ff5252"
                font.pixelSize: 14
                anchors.verticalCenter: parent.verticalCenter
            }
        }
    }

    // ===== 故障告警横幅 =====
    Rectangle {
        id: faultBanner
        anchors.top: header.bottom
        width: parent.width
        height: vehicle.faultCode !== "" ? 40 : 0
        visible: height > 0
        color: "#5c1a1a"
        Behavior on height { NumberAnimation { duration: 200 } }

        Text {
            anchors.centerIn: parent
            text: "⚠ 检测到故障码：" + vehicle.faultCode
            color: "#ff8a80"
            font.pixelSize: 15
            font.bold: true
        }
    }

    // ===== 主体区域 =====
    RowLayout {
        anchors.top: faultBanner.bottom
        anchors.bottom: footer.top
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.margins: 24
        spacing: 24

        // 车速表
        Gauge {
            Layout.fillWidth: true
            Layout.fillHeight: true
            Layout.minimumWidth: 260
            Layout.minimumHeight: 260
            title: "车速"
            unit: "km/h"
            minValue: 0
            maxValue: 240
            value: vehicle.speed
            accentColor: "#4fc3f7"
        }

        // 转速表
        Gauge {
            Layout.fillWidth: true
            Layout.fillHeight: true
            Layout.minimumWidth: 260
            Layout.minimumHeight: 260
            title: "转速"
            unit: "rpm"
            minValue: 0
            maxValue: 8000
            value: vehicle.rpm
            accentColor: "#d2a8ff"
            warning: vehicle.rpm > 5000
        }

        // 右侧信息栏
        ColumnLayout {
            Layout.preferredWidth: 300
            Layout.fillHeight: true
            spacing: 24

            BatteryBar {
                Layout.fillWidth: true
                Layout.preferredHeight: 60
                value: vehicle.battery
            }

            // GPS 坐标
            Rectangle {
                Layout.fillWidth: true
                Layout.preferredHeight: 110
                radius: 8
                color: "#161b22"
                border.color: "#30363d"

                Column {
                    anchors.centerIn: parent
                    spacing: 10
                    Text { text: "📍 当前位置"; color: "#8b949e"; font.pixelSize: 14; anchors.horizontalCenter: parent.horizontalCenter }
                    Text { text: "纬度  " + vehicle.lat.toFixed(5); color: "#e6edf3"; font.pixelSize: 15; font.family: "Consolas" }
                    Text { text: "经度  " + vehicle.lng.toFixed(5); color: "#e6edf3"; font.pixelSize: 15; font.family: "Consolas" }
                }
            }

            Item { Layout.fillHeight: true } // 弹簧
        }
    }

    // ===== 底部状态栏 =====
    Rectangle {
        id: footer
        anchors.bottom: parent.bottom
        width: parent.width
        height: 34
        color: "#161b22"

        Text {
            anchors.centerIn: parent
            text: "HAL: VehicleSimulator (模拟硬件)  |  上报周期: 1s  |  通道: HTTP → RabbitMQ → MySQL / Redis / WebSocket"
            color: "#6e7681"
            font.pixelSize: 12
        }
    }
}
