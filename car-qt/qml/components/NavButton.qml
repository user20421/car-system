import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Aurora

Button {
    id: root
    property string glyph
    property string label
    property bool active: false
    hoverEnabled: true
    implicitWidth: 68
    implicitHeight: 62
    contentItem: ColumnLayout {
        spacing: 4
        Text { text: root.glyph; color: root.active ? Theme.cyan : Theme.muted; font.pixelSize: 20; Layout.alignment: Qt.AlignHCenter }
        Text { text: root.label; color: root.active ? Theme.text : Theme.muted; font.pixelSize: 10; font.family: Theme.fontFamily; Layout.alignment: Qt.AlignHCenter }
    }
    background: Rectangle {
        radius: 15
        color: root.active ? Theme.cyanSoft : root.hovered ? Theme.surfaceRaised : "transparent"
        Rectangle { visible: root.active; width: 3; height: 22; radius: 2; color: Theme.cyan; anchors.left: parent.left; anchors.leftMargin: -10; anchors.verticalCenter: parent.verticalCenter }
    }
}
