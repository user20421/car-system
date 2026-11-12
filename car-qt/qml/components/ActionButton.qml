import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Aurora

Button {
    id: root
    property string glyph: ""
    property string subtitle: ""
    property color accent: Theme.cyan
    property bool prominent: false
    hoverEnabled: true
    implicitHeight: subtitle.length ? 70 : 48
    implicitWidth: 150
    contentItem: RowLayout {
        spacing: 11
        Text { text: root.glyph; visible: text.length > 0; color: root.enabled ? root.accent : Theme.muted; font.pixelSize: 20; Layout.leftMargin: 4 }
        ColumnLayout {
            spacing: 2
            Layout.fillWidth: true
            Text { text: root.text; color: root.enabled ? Theme.text : Theme.muted; font.pixelSize: 14; font.weight: Font.DemiBold; font.family: Theme.fontFamily }
            Text { text: root.subtitle; visible: text.length > 0; color: Theme.muted; font.pixelSize: 10; font.family: Theme.fontFamily; elide: Text.ElideRight; Layout.fillWidth: true }
        }
        Text { text: "›"; visible: root.subtitle.length > 0; color: Theme.muted; font.pixelSize: 22; Layout.rightMargin: 4 }
    }
    background: Rectangle {
        radius: Theme.smallRadius
        color: !root.enabled ? "#10151d" : root.down ? Theme.cyanSoft : root.hovered ? "#202c3b" : root.prominent ? "#225ce1e6" : Theme.surfaceRaised
        border.width: 1
        border.color: root.hovered && root.enabled ? root.accent : Theme.stroke
        Behavior on color { ColorAnimation { duration: 140 } }
        Behavior on border.color { ColorAnimation { duration: 140 } }
    }
}
