import QtQuick
import QtQuick.Layouts
import Aurora

RowLayout {
    id: root
    property string title
    property string subtitle
    property string trailing
    spacing: 12
    ColumnLayout {
        spacing: 3
        Text { text: root.title; color: Theme.text; font.pixelSize: 19; font.weight: Font.DemiBold; font.family: Theme.fontFamily }
        Text { text: root.subtitle; visible: text.length > 0; color: Theme.muted; font.pixelSize: 12; font.family: Theme.fontFamily }
    }
    Item { Layout.fillWidth: true }
    Text { text: root.trailing; color: Theme.cyan; font.pixelSize: 12; font.family: Theme.fontFamily }
}
