import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Aurora

Item {
    id: root
    property int zone: 0
    RowLayout {
        anchors.fill: parent; spacing: 18
        GlassPanel {
            Layout.fillWidth: true; Layout.fillHeight: true; highlighted: true
            ColumnLayout {
                anchors.fill: parent; anchors.margins: 28; spacing: 18
                SectionHeader { title: "座舱气候"; subtitle: "双区智能恒温 · PM2.5 已净化"; trailing: vehicle.hvacOn ? "AUTO" : "OFF" }
                RowLayout {
                    Layout.fillWidth: true; spacing: 12
                    Repeater { model: ["主驾", "前排", "全车"]; delegate: Button { required property string modelData; required property int index; text: modelData; Layout.fillWidth: true; onClicked: root.zone = index; contentItem: Text { text: modelData; color: root.zone === index ? Theme.cyan : Theme.muted; horizontalAlignment: Text.AlignHCenter } background: Rectangle { radius: 13; color: root.zone === index ? Theme.cyanSoft : Theme.surfaceRaised; border.color: root.zone === index ? Theme.cyan : Theme.stroke } } }
                }
                Item {
                    Layout.fillWidth: true; Layout.fillHeight: true
                    Text { text: vehicle.cabinTemperature.toFixed(1) + "°"; anchors.centerIn: parent; color: "#0e5ce1e6"; font.pixelSize: Math.min(parent.width,parent.height)*0.45; font.weight: Font.Black }
                    Column {
                        anchors.centerIn: parent; spacing: 8
                        Text { text: vehicle.targetTemperature.toFixed(1); color: Theme.text; font.pixelSize: 94; font.weight: Font.Light; font.family: Theme.monoFamily; anchors.horizontalCenter: parent.horizontalCenter }
                        Text { text: "目标温度 °C  ·  " + (vehicle.hvacOn ? "智能调节中" : "空调已关闭"); color: Theme.muted; font.pixelSize: 12; anchors.horizontalCenter: parent.horizontalCenter }
                    }
                    ActionButton { anchors.left: parent.left; anchors.verticalCenter: parent.verticalCenter; text: "−"; implicitWidth: 58; implicitHeight: 58; enabled: !vehicle.commandBusy; onClicked: vehicle.setTemperature(Math.max(16, vehicle.targetTemperature - 0.5)) }
                    ActionButton { anchors.right: parent.right; anchors.verticalCenter: parent.verticalCenter; text: "+"; implicitWidth: 58; implicitHeight: 58; enabled: !vehicle.commandBusy; onClicked: vehicle.setTemperature(Math.min(30, vehicle.targetTemperature + 0.5)) }
                }
                ColumnLayout {
                    Layout.fillWidth: true; spacing: 6
                    RowLayout { Text { text: "16°C"; color: Theme.muted; font.pixelSize: 10 } Item { Layout.fillWidth: true } Text { text: "30°C"; color: Theme.muted; font.pixelSize: 10 } }
                    Slider { id: tempSlider; Layout.fillWidth: true; from: 16; to: 30; stepSize: 0.5; value: vehicle.targetTemperature; onPressedChanged: if(!pressed && value !== vehicle.targetTemperature) vehicle.setTemperature(value) }
                }
                RowLayout {
                    Layout.fillWidth: true; spacing: 12
                    ActionButton { Layout.fillWidth: true; glyph: vehicle.hvacOn ? "❄" : "○"; text: vehicle.hvacOn ? "关闭空调" : "开启空调"; subtitle: vehicle.hvacOn ? "AUTO · 风量 " + vehicle.fanLevel : "座舱自然通风"; enabled: !vehicle.commandBusy; prominent: vehicle.hvacOn; onClicked: vehicle.setClimate(!vehicle.hvacOn) }
                    ActionButton { Layout.fillWidth: true; glyph: "▧"; text: "前挡除霜"; subtitle: vehicle.defrost ? "正在高效除霜" : "快速恢复清晰视野"; enabled: !vehicle.commandBusy; prominent: vehicle.defrost; accent: Theme.amber; onClicked: vehicle.setDefrost(!vehicle.defrost) }
                }
            }
        }
        GlassPanel {
            visible: root.width > 1020; Layout.preferredWidth: 290; Layout.fillHeight: true
            ColumnLayout {
                anchors.fill: parent; anchors.margins: 22; spacing: 18
                SectionHeader { title: "空气品质"; subtitle: "座舱传感器实时监测" }
                Item {
                    Layout.fillWidth: true; Layout.preferredHeight: 140
                    Rectangle { anchors.centerIn: parent; width: 120; height: 120; radius: 60; color: "#186ce5ae"; border.color: Theme.green; border.width: 1; Column { anchors.centerIn: parent; Text { text: "12"; color: Theme.text; font.pixelSize: 38; font.family: Theme.monoFamily; anchors.horizontalCenter: parent.horizontalCenter } Text { text: "PM2.5  μg/m³"; color: Theme.muted; font.pixelSize: 10; anchors.horizontalCenter: parent.horizontalCenter } } }
                }
                Text { text: "空气洁净"; color: Theme.green; font.pixelSize: 17 }
                Text { text: "CN95 滤芯运行正常\n车外 PM2.5：31 μg/m³\n负离子净化已开启"; color: Theme.muted; font.pixelSize: 12; lineHeight: 1.65 }
                Rectangle { Layout.fillWidth: true; height: 1; color: Theme.stroke }
                Text { text: "智能场景"; color: Theme.text; font.pixelSize: 14 }
                ActionButton { Layout.fillWidth: true; text: "小憩模式"; subtitle: "22°C · 微风 · 座椅放松"; glyph: "☾"; onClicked: vehicle.setTemperature(22) }
                ActionButton { Layout.fillWidth: true; text: "快速降温"; subtitle: "18°C · 最大制冷效率"; glyph: "❄"; onClicked: vehicle.setTemperature(18) }
                Item { Layout.fillHeight: true }
                Text { text: vehicle.commandBusy ? "执行中 · " + vehicle.commandState : vehicle.commandMessage; color: vehicle.commandState === "failed" ? Theme.red : Theme.muted; font.pixelSize: 10; wrapMode: Text.Wrap; Layout.fillWidth: true }
            }
        }
    }
}
