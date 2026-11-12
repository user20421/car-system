import QtQuick
import Aurora

Rectangle {
    id: root
    property string connectionState: "offline"
    property string label
    width: row.implicitWidth + 24
    height: 34
    radius: 17
    color: connectionState === "online" ? "#226ce5ae" : connectionState === "connecting" ? "#22ffc76a" : "#22ff7085"
    border.color: connectionState === "online" ? Theme.green : connectionState === "connecting" ? Theme.amber : Theme.red
    Row {
        id: row
        anchors.centerIn: parent
        spacing: 8
        Rectangle { width: 7; height: 7; radius: 4; anchors.verticalCenter: parent.verticalCenter; color: root.connectionState === "online" ? Theme.green : root.connectionState === "connecting" ? Theme.amber : Theme.red
            SequentialAnimation on opacity { running: root.connectionState === "connecting"; loops: Animation.Infinite; NumberAnimation { to: 0.25; duration: 550 } NumberAnimation { to: 1; duration: 550 } }
        }
        Text { text: root.label; color: Theme.text; font.pixelSize: 11; font.family: Theme.fontFamily }
    }
}
