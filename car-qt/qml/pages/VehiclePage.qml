import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Aurora

Item {
    id: root
    Flickable {
        anchors.fill: parent; contentWidth: width; contentHeight: column.implicitHeight; clip: true
        ColumnLayout {
            id: column; width: parent.width; spacing: 18
            RowLayout {
                Layout.fillWidth: true; spacing: 18
                GlassPanel {
                    Layout.fillWidth: true; Layout.preferredHeight: 480; highlighted: true
                    RowLayout {
                        anchors.fill: parent; anchors.margins: 24
                        ColumnLayout {
                            Layout.fillWidth: true; spacing: 15
                            SectionHeader { title: "AURORA S1"; subtitle: "纯电智能轿跑 · " + vehicle.vin; trailing: vehicle.locked ? "SECURED" : "UNLOCKED" }
                            Item { Layout.fillHeight: true }
                            ActionButton { Layout.fillWidth: true; glyph: vehicle.locked ? "▣" : "▢"; text: vehicle.locked ? "解锁车辆" : "锁定车辆"; subtitle: "指令经车辆硬件确认后更新状态"; prominent: true; enabled: !vehicle.commandBusy; onClicked: vehicle.locked ? vehicle.unlockVehicle() : vehicle.lockVehicle() }
                            ActionButton { Layout.fillWidth: true; glyph: "◉"; text: "鸣笛闪灯"; subtitle: "在停车场快速寻找车辆"; enabled: !vehicle.commandBusy; onClicked: vehicle.honkFlash() }
                            ActionButton { Layout.fillWidth: true; glyph: "▧"; text: vehicle.defrost ? "关闭除霜" : "开启除霜"; subtitle: "前挡风玻璃与后视区域"; enabled: !vehicle.commandBusy; onClicked: vehicle.setDefrost(!vehicle.defrost) }
                            Item { Layout.fillHeight: true }
                            Text { text: vehicle.commandBusy ? "车辆执行中 · " + vehicle.commandState : vehicle.commandMessage; color: vehicle.commandState === "failed" ? Theme.red : Theme.muted; font.pixelSize: 11; wrapMode: Text.Wrap; Layout.fillWidth: true }
                        }
                        VehicleGraphic { Layout.preferredWidth: 270; Layout.fillHeight: true; locked: vehicle.locked; lightsOn: vehicle.commandBusy }
                    }
                }
                Item {
                    Layout.preferredWidth: root.width > 1100 ? 520 : 370
                    Layout.minimumWidth: root.width > 1100 ? 520 : 370
                    Layout.maximumWidth: root.width > 1100 ? 520 : 370
                    Layout.preferredHeight: 480
                    ColumnLayout {
                    anchors.fill: parent; spacing: 18
                    GridLayout {
                        Layout.fillWidth: true; columns: 2; columnSpacing: 14; rowSpacing: 14
                        MetricTile { label: "动力电池"; value: vehicle.battery.toFixed(1); unit: "%"; glyph: "▰"; accent: Theme.green }
                        MetricTile { label: "预计续航"; value: Math.round(vehicle.range); unit: "km"; glyph: "↗"; accent: Theme.cyan }
                        MetricTile { label: "驱动转速"; value: vehicle.rpm.toLocaleString(); unit: "rpm"; glyph: "◉"; accent: Theme.violet }
                        MetricTile { label: "当前车速"; value: Math.round(vehicle.speed); unit: "km/h"; glyph: "›"; accent: Theme.amber }
                    }
                    GlassPanel {
                        Layout.fillWidth: true; Layout.preferredHeight: 196
                        ColumnLayout {
                            anchors.fill: parent; anchors.margins: 20; spacing: 12
                            SectionHeader { title: "车辆健康"; subtitle: "最近一次全车扫描 · 刚刚" }
                            Repeater {
                                model: [{name:"三电系统",status:vehicle.faults.length ? "需关注" : "正常"},{name:"制动与转向",status:"正常"},{name:"胎压监测",status:"正常"},{name:"智能传感器",status:"正常"}]
                                delegate: RowLayout { required property var modelData; Layout.fillWidth: true; Text { text: modelData.name; color: Theme.muted; font.pixelSize: 12 } Item { Layout.fillWidth: true } Text { text: modelData.status; color: modelData.status === "正常" ? Theme.green : Theme.amber; font.pixelSize: 11 } }
                            }
                        }
                    }
                    }
                }
            }
            GlassPanel {
                Layout.fillWidth: true; visible: vehicle.faults.length > 0; implicitHeight: faultColumn.implicitHeight + 36; border.color: Theme.amber
                ColumnLayout { id: faultColumn; anchors.fill: parent; anchors.margins: 18; SectionHeader { title: "活跃告警"; subtitle: "故障由硬件遥测上报，请按指引处理"; trailing: vehicle.faults.length + " 项" } Repeater { model: vehicle.faults; delegate: Text { required property string modelData; text: "⚠  " + modelData; color: Theme.amber; font.pixelSize: 12; Layout.fillWidth: true; wrapMode: Text.Wrap } } }
            }
        }
    }
}
