import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Aurora

Item {
    id: root
    property string destination: "家 · 世纪大道"
    property bool routeActive: true
    property int selectedRoute: 0
    RowLayout {
        anchors.fill: parent; spacing: 18
        GlassPanel {
            Layout.fillWidth: true; Layout.fillHeight: true; clip: true; highlighted: true
            MapCanvas { anchors.fill: parent; latitude: vehicle.latitude; longitude: vehicle.longitude; heading: vehicle.heading; navigating: root.routeActive }
            GlassPanel {
                anchors.left: parent.left; anchors.top: parent.top; anchors.margins: 18
                width: Math.min(420, parent.width - 36); height: 70; radius: 18
                RowLayout { anchors.fill: parent; anchors.margins: 12; spacing: 10; Text { text: "⌕"; color: Theme.cyan; font.pixelSize: 24 } TextField { Layout.fillWidth: true; text: root.destination; color: Theme.text; font.pixelSize: 14; selectByMouse: true; background: null; onAccepted: { root.destination = text; root.routeActive = true } placeholderText: "搜索目的地" } ActionButton { text: "搜索"; implicitWidth: 70; implicitHeight: 40; onClicked: root.routeActive = true } }
            }
            GlassPanel {
                visible: root.routeActive
                anchors.left: parent.left; anchors.bottom: parent.bottom; anchors.margins: 22
                width: 350; height: 104
                RowLayout { anchors.fill: parent; anchors.margins: 16; spacing: 16; Rectangle { width: 52; height: 52; radius: 16; color: Theme.cyanSoft; Text { anchors.centerIn: parent; text: "↰"; color: Theme.cyan; font.pixelSize: 30 } } ColumnLayout { Text { text: "前方 300 米左转"; color: Theme.text; font.pixelSize: 18; font.weight: Font.DemiBold } Text { text: "进入世纪大道 · 随后靠右行驶"; color: Theme.muted; font.pixelSize: 11 } } }
            }
            Column {
                anchors.right: parent.right; anchors.bottom: parent.bottom; anchors.margins: 18; spacing: 8
                ActionButton { text: "+"; implicitWidth: 44; implicitHeight: 44 }
                ActionButton { text: "−"; implicitWidth: 44; implicitHeight: 44 }
                ActionButton { text: "◎"; implicitWidth: 44; implicitHeight: 44; onClicked: root.routeActive = true }
            }
        }
        GlassPanel {
            Layout.preferredWidth: root.width > 1120 ? 330 : 270; Layout.fillHeight: true
            ColumnLayout {
                anchors.fill: parent; anchors.margins: 20; spacing: 14
                SectionHeader { title: "路线规划"; subtitle: "实时路况 · 新能源优先" }
                Text { text: root.destination; color: Theme.text; font.pixelSize: 18; wrapMode: Text.Wrap; Layout.fillWidth: true }
                Text { text: "18 分钟 · 8.6 km"; color: Theme.cyan; font.pixelSize: 25; font.family: Theme.monoFamily }
                Repeater {
                    model: [{name:"推荐路线",detail:"18 分钟 · 8.6 km",tag:"畅通"},{name:"避开拥堵",detail:"22 分钟 · 9.1 km",tag:"少红灯"},{name:"风景路线",detail:"27 分钟 · 10.4 km",tag:"滨江"}]
                    delegate: GlassPanel {
                        required property var modelData; required property int index
                        Layout.fillWidth: true; height: 78; highlighted: root.selectedRoute === index
                        MouseArea { anchors.fill: parent; onClicked: { root.selectedRoute = index; root.routeActive = true } }
                        RowLayout { anchors.fill: parent; anchors.margins: 12; ColumnLayout { Text { text: modelData.name; color: Theme.text; font.pixelSize: 13 } Text { text: modelData.detail; color: Theme.muted; font.pixelSize: 10 } } Item { Layout.fillWidth: true } Text { text: modelData.tag; color: root.selectedRoute === index ? Theme.cyan : Theme.muted; font.pixelSize: 10 } }
                    }
                }
                ActionButton { Layout.fillWidth: true; prominent: true; text: root.routeActive ? "重新规划" : "开始导航"; glyph: "⌁"; onClicked: root.routeActive = !root.routeActive }
                Item { Layout.fillHeight: true }
                Rectangle { Layout.fillWidth: true; height: 1; color: Theme.stroke }
                Text { text: "沿途充电"; color: Theme.text; font.pixelSize: 13 }
                Text { text: "●  世纪汇超充站  2.4 km\n●  国金中心充电站  4.1 km"; color: Theme.muted; font.pixelSize: 11; lineHeight: 1.6 }
            }
        }
    }
}
