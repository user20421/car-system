import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Aurora

Item {
    id: root
    signal navigateTo(int index)
    Flickable {
        anchors.fill: parent
        contentWidth: width
        contentHeight: content.implicitHeight
        clip: true
        boundsBehavior: Flickable.StopAtBounds
        ScrollBar.vertical: ScrollBar { policy: ScrollBar.AsNeeded }
        ColumnLayout {
            id: content
            width: parent.width
            spacing: 18
            RowLayout {
                Layout.fillWidth: true
                spacing: 18
                GlassPanel {
                    Layout.fillWidth: true
                    Layout.preferredHeight: 350
                    highlighted: true
                    RowLayout {
                        anchors.fill: parent; anchors.margins: 28; spacing: 12
                        ColumnLayout {
                            Layout.fillWidth: true; Layout.alignment: Qt.AlignTop; spacing: 13
                            Text { text: "晚上好，驾驶者"; color: Theme.cyan; font.pixelSize: 12; font.letterSpacing: 2 }
                            Text { text: "从容启程，\n让每段旅程发生。"; color: Theme.text; font.pixelSize: 31; lineHeight: 1.12; font.weight: Font.DemiBold }
                            Text { text: vehicle.speed > 1 ? "车辆行进中 · 保持专注，安全抵达" : "车辆已就绪 · 上海  26°C  空气优"; color: Theme.muted; font.pixelSize: 13; topPadding: 5 }
                            Item { Layout.fillHeight: true }
                            RowLayout {
                                spacing: 10
                                ActionButton { text: "开始导航"; glyph: "⌁"; prominent: true; onClicked: root.navigateTo(2) }
                                ActionButton { text: vehicle.hvacOn ? "空调运行中" : "开启空调"; glyph: "❄"; enabled: !vehicle.commandBusy; onClicked: vehicle.setClimate(true) }
                            }
                            RowLayout {
                                spacing: 8; Layout.topMargin: 6
                                Rectangle { width: 7; height: 7; radius: 4; color: vehicle.locked ? Theme.green : Theme.amber }
                                Text { text: (vehicle.locked ? "车辆安全锁定" : "车辆已解锁") + "  ·  " + vehicle.faults.length + " 项活跃告警"; color: Theme.muted; font.pixelSize: 11 }
                            }
                        }
                        VehicleGraphic {
                            Layout.preferredWidth: 260; Layout.fillHeight: true
                            locked: vehicle.locked; lightsOn: true; heading: vehicle.heading
                            scale: 0.92
                        }
                    }
                }
                GlassPanel {
                    Layout.preferredWidth: root.width > 1120 ? 310 : 280
                    Layout.preferredHeight: 350
                    ColumnLayout {
                        anchors.fill: parent; anchors.margins: 22; spacing: 16
                        SectionHeader { title: "能量状态"; subtitle: "动态预估 · 舒适模式"; trailing: "正常" }
                        Item {
                            Layout.fillWidth: true; Layout.preferredHeight: 145
                            Rectangle {
                                anchors.centerIn: parent; width: 140; height: 140; radius: 70; color: "transparent"; border.width: 10; border.color: "#203040"
                                Rectangle { anchors.centerIn: parent; width: 118; height: 118; radius: 59; color: "#185ce1e6"; border.width: 2; border.color: Theme.cyan }
                                Column { anchors.centerIn: parent; Text { text: Math.round(vehicle.battery) + "%"; color: Theme.text; font.pixelSize: 30; font.family: Theme.monoFamily; anchors.horizontalCenter: parent.horizontalCenter } Text { text: "剩余电量"; color: Theme.muted; font.pixelSize: 10; anchors.horizontalCenter: parent.horizontalCenter } }
                            }
                        }
                        RowLayout { Text { text: "预计续航"; color: Theme.muted; font.pixelSize: 12 } Item { Layout.fillWidth: true } Text { text: Math.round(vehicle.range) + " km"; color: Theme.text; font.pixelSize: 21; font.family: Theme.monoFamily } }
                        Rectangle { Layout.fillWidth: true; height: 5; radius: 3; color: "#223142"; Rectangle { width: parent.width * vehicle.battery / 100; height: parent.height; radius: 3; color: Theme.cyan; Behavior on width { NumberAnimation { duration: 300 } } } }
                        Text { text: "家 · 超级充电站\n2.4 km · 当前空闲"; color: Theme.muted; font.pixelSize: 11; lineHeight: 1.4 }
                    }
                }
            }
            GridLayout {
                Layout.fillWidth: true
                columns: 2
                columnSpacing: 18
                MetricTile { label: "当前车速"; value: Math.round(vehicle.speed); unit: "km/h"; glyph: "↗"; accent: Theme.cyan }
                MetricTile { label: "座舱温度"; value: vehicle.cabinTemperature.toFixed(1); unit: "°C"; glyph: "❄"; accent: Theme.violet }
            }
            GridLayout {
                Layout.fillWidth: true
                columns: root.width > 1050 ? 2 : 1
                columnSpacing: 18; rowSpacing: 18
                GlassPanel {
                    Layout.fillWidth: true; Layout.preferredHeight: 290
                    clip: true
                    MapCanvas { anchors.fill: parent; latitude: vehicle.latitude; longitude: vehicle.longitude; heading: vehicle.heading }
                    Rectangle { anchors.fill: parent; color: "transparent"; border.width: 0; gradient: Gradient { GradientStop { position: 0; color: "#dd0b121c" } GradientStop { position: 0.48; color: "#00000000" } } }
                    Column { anchors.left: parent.left; anchors.top: parent.top; anchors.margins: 20; spacing: 5; Text { text: "常用路线"; color: Theme.cyan; font.pixelSize: 11 } Text { text: "回家 · 世纪大道"; color: Theme.text; font.pixelSize: 20; font.weight: Font.DemiBold } Text { text: "18 分钟  ·  8.6 km  ·  路况畅通"; color: Theme.muted; font.pixelSize: 11 } }
                    ActionButton { anchors.right: parent.right; anchors.bottom: parent.bottom; anchors.margins: 18; text: "展开地图"; glyph: "↗"; onClicked: root.navigateTo(2) }
                }
                GlassPanel {
                    Layout.fillWidth: true; Layout.preferredHeight: 290
                    ColumnLayout {
                        anchors.fill: parent; anchors.margins: 22; spacing: 16
                        SectionHeader { title: "此刻播放"; subtitle: "AURORA Sound · 蓝牙音频"; trailing: "Hi-Res" }
                        RowLayout {
                            Layout.fillWidth: true; spacing: 18
                            Rectangle { Layout.preferredWidth: 110; Layout.preferredHeight: 110; radius: 22; gradient: Gradient { GradientStop { position: 0; color: Theme.violet } GradientStop { position: 1; color: "#193947" } } Text { anchors.centerIn: parent; text: "A"; color: "#ccffffff"; font.pixelSize: 48; font.weight: Font.Light } }
                            ColumnLayout { Layout.fillWidth: true; spacing: 8; Text { text: "Midnight City"; color: Theme.text; font.pixelSize: 21; font.weight: Font.DemiBold } Text { text: "M83 · Hurry Up, We're Dreaming"; color: Theme.muted; font.pixelSize: 12; elide: Text.ElideRight; Layout.fillWidth: true } Text { text: "沉浸声场  ·  24 bit / 96 kHz"; color: Theme.cyan; font.pixelSize: 10 } }
                        }
                        Item { Layout.fillHeight: true }
                        RowLayout {
                            Layout.alignment: Qt.AlignHCenter; spacing: 24
                            Text { text: "‹‹"; color: Theme.text; font.pixelSize: 25; MouseArea { anchors.fill: parent; onClicked: root.navigateTo(3) } }
                            Rectangle { width: 50; height: 50; radius: 25; color: Theme.text; Text { anchors.centerIn: parent; text: "▶"; color: Theme.background; font.pixelSize: 17 } MouseArea { anchors.fill: parent; onClicked: root.navigateTo(3) } }
                            Text { text: "››"; color: Theme.text; font.pixelSize: 25; MouseArea { anchors.fill: parent; onClicked: root.navigateTo(3) } }
                        }
                    }
                }
            }
        }
    }
}
