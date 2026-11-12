import QtQuick
import QtQuick.Layouts
import Aurora

GlassPanel {
    id: root
    property string label
    property string value
    property string unit
    property string glyph
    property color accent: Theme.cyan
    implicitHeight: 126
    Layout.fillWidth: true
    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 18
        spacing: 8
        RowLayout {
            Text { text: root.glyph; color: root.accent; font.pixelSize: 17 }
            Text { text: root.label; color: Theme.muted; font.pixelSize: 12; font.family: Theme.fontFamily }
            Item { Layout.fillWidth: true }
            Rectangle { width: 5; height: 5; radius: 3; color: root.accent }
        }
        Item { Layout.fillHeight: true }
        RowLayout {
            spacing: 7
            Text { text: root.value; color: Theme.text; font.pixelSize: 30; font.weight: Font.DemiBold; font.family: Theme.monoFamily }
            Text { text: root.unit; color: Theme.muted; font.pixelSize: 12; font.family: Theme.fontFamily; Layout.alignment: Qt.AlignBottom; Layout.bottomMargin: 5 }
        }
    }
}
