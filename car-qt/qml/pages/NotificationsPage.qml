import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Aurora

Item {
    id: root
    RowLayout {
        anchors.fill: parent; spacing: 18
        GlassPanel {
            Layout.fillWidth: true; Layout.fillHeight: true
            ColumnLayout {
                anchors.fill: parent; anchors.margins: 24; spacing: 14
                SectionHeader { title: "消息中心"; subtitle: "车辆事件、服务请求与安全通知"; trailing: vehicle.recentEvents.length + " 条近期事件" }
                RowLayout {
                    spacing: 10
                    StatusPill { connectionState: vehicle.faults.length ? "offline" : "online"; label: vehicle.faults.length ? vehicle.faults.length + " 项告警" : "无活跃故障" }
                    StatusPill { connectionState: vehicle.networkState; label: vehicle.networkState === "online" ? "云端同步" : "离线记录" }
                    Item { Layout.fillWidth: true }
                }
                Rectangle { Layout.fillWidth: true; height: 1; color: Theme.stroke }
                ListView {
                    Layout.fillWidth: true; Layout.fillHeight: true; clip: true; spacing: 10
                    model: vehicle.recentEvents
                    delegate: GlassPanel {
                        required property string modelData; required property int index
                        width: ListView.view.width; height: 68; radius: 15
                        RowLayout {
                            anchors.fill: parent; anchors.margins: 14; spacing: 13
                            Rectangle { width: 36; height: 36; radius: 12; color: index === 0 ? Theme.cyanSoft : Theme.surfaceRaised; Text { anchors.centerIn: parent; text: modelData.indexOf("失败") >= 0 || modelData.indexOf("故障") >= 0 ? "!" : "·"; color: modelData.indexOf("失败") >= 0 ? Theme.red : Theme.cyan; font.pixelSize: 20 } }
                            Text { text: modelData; color: Theme.text; font.pixelSize: 12; elide: Text.ElideRight; Layout.fillWidth: true }
                            Text { text: index === 0 ? "最新" : "已记录"; color: Theme.muted; font.pixelSize: 10 }
                        }
                    }
                    GlassPanel {
                        visible: vehicle.recentEvents.length === 0; anchors.centerIn: parent; width: 330; height: 150
                        Column { anchors.centerIn: parent; spacing: 10; Text { text: "◌"; color: Theme.cyan; font.pixelSize: 38; anchors.horizontalCenter: parent.horizontalCenter } Text { text: "暂无新消息"; color: Theme.text; font.pixelSize: 16; anchors.horizontalCenter: parent.horizontalCenter } Text { text: "车辆事件会安全地保存在这里"; color: Theme.muted; font.pixelSize: 11; anchors.horizontalCenter: parent.horizontalCenter } }
                    }
                }
            }
        }
        GlassPanel {
            visible: root.width > 1100; Layout.preferredWidth: 300; Layout.fillHeight: true
            ColumnLayout {
                anchors.fill: parent; anchors.margins: 22; spacing: 16
                SectionHeader { title: "通知偏好"; subtitle: "仅影响本车机显示" }
                Repeater {
                    model: [{title:"车辆安全",detail:"门锁与异常移动"},{title:"故障告警",detail:"始终优先显示"},{title:"能源与充电",detail:"电量和充电进度"},{title:"行程摘要",detail:"到达目的地后"}]
                    delegate: RowLayout { required property var modelData; Layout.fillWidth: true; ColumnLayout { Layout.fillWidth: true; Text { text: modelData.title; color: Theme.text; font.pixelSize: 12 } Text { text: modelData.detail; color: Theme.muted; font.pixelSize: 10 } } Switch { checked: true } }
                }
                Item { Layout.fillHeight: true }
                Text { text: "关键安全告警无法关闭，并会在驾驶时以最小干扰方式展示。"; color: Theme.muted; font.pixelSize: 10; wrapMode: Text.Wrap; Layout.fillWidth: true }
            }
        }
    }
}
