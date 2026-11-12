import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Aurora

GlassPanel {
    id: root
    signal close()
    highlighted: true
    color: "#f0101722"
    ColumnLayout {
        anchors.fill: parent; anchors.margins: 20; spacing: 14
        RowLayout {
            ColumnLayout { Text { text: "SIMULATION LAB"; color: Theme.amber; font.pixelSize: 10; font.letterSpacing: 2 } Text { text: "硬件场景调试"; color: Theme.text; font.pixelSize: 19; font.weight: Font.DemiBold } }
            Item { Layout.fillWidth: true }
            ActionButton { text: "×"; implicitWidth: 38; implicitHeight: 38; onClicked: root.close() }
        }
        Text { text: "场景直接发送至独立模拟 HAL 线程。UI 不修改车辆状态，所有反馈仍经 QueuedConnection 返回。"; color: Theme.muted; font.pixelSize: 11; wrapMode: Text.Wrap; Layout.fillWidth: true }
        Rectangle { Layout.fillWidth: true; height: 1; color: Theme.stroke }
        Repeater {
            model: [{id:"city",name:"城市巡航",detail:"动态车速与能耗",glyph:"⌂"},{id:"highway",name:"高速行驶",detail:"高车速与电机转速",glyph:"↗"},{id:"parked",name:"驻车",detail:"零车速与空调消耗",glyph:"P"},{id:"charging",name:"充电",detail:"电量逐步回升",glyph:"▰"},{id:"fault",name:"电池告警",detail:"BMS 活跃故障",glyph:"!"}]
            delegate: ActionButton { required property var modelData; Layout.fillWidth: true; glyph: modelData.glyph; text: modelData.name; subtitle: modelData.detail; prominent: vehicle.scenario === modelData.id; accent: modelData.id === "fault" ? Theme.amber : Theme.cyan; onClicked: vehicle.setScenario(modelData.id) }
        }
        Item { Layout.fillHeight: true }
        GlassPanel {
            Layout.fillWidth: true; height: 116
            ColumnLayout { anchors.fill: parent; anchors.margins: 12; Text { text: "LIVE HAL SNAPSHOT"; color: Theme.cyan; font.pixelSize: 9; font.letterSpacing: 1 } Text { text: "场景  " + vehicle.scenario + "\n速度  " + vehicle.speed.toFixed(1) + " km/h   RPM  " + vehicle.rpm + "\n坐标  " + vehicle.latitude.toFixed(5) + ", " + vehicle.longitude.toFixed(5); color: Theme.muted; font.pixelSize: 10; font.family: Theme.monoFamily; lineHeight: 1.5 } }
        }
        Text { text: "GUI · main\nHAL · simulated-hal\nNetwork · device-network\nStorage · storage-worker"; color: "#687d90"; font.pixelSize: 9; font.family: Theme.monoFamily; lineHeight: 1.45 }
    }
}
