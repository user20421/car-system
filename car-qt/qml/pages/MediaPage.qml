import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Aurora

Item {
    id: root
    property bool playing: false
    property real position: 48
    property int duration: 245
    property int trackIndex: 0
    property var tracks: [
        {title:"Midnight City",artist:"M83",album:"Hurry Up, We're Dreaming",duration:245,color:"#9b8cff"},
        {title:"Nightcall",artist:"Kavinsky",album:"OutRun",duration:258,color:"#ff7085"},
        {title:"Weightless",artist:"Marconi Union",album:"Ambient Transmissions",duration:480,color:"#5ce1e6"},
        {title:"Sunset Drive",artist:"AURORA Ensemble",album:"Electric Horizons",duration:212,color:"#ffc76a"}
    ]
    function formatTime(seconds) { return Math.floor(seconds / 60) + ":" + Math.floor(seconds % 60).toString().padStart(2,"0") }
    function selectTrack(index) { trackIndex = (index + tracks.length) % tracks.length; position = 0; duration = tracks[trackIndex].duration; playing = true }
    Timer { interval: 1000; running: root.playing; repeat: true; onTriggered: { if(root.position >= root.duration) root.selectTrack(root.trackIndex + 1); else root.position += 1 } }
    RowLayout {
        anchors.fill: parent; spacing: 20
        GlassPanel {
            Layout.fillWidth: true; Layout.fillHeight: true; highlighted: true
            ColumnLayout {
                anchors.fill: parent; anchors.margins: 30; spacing: 20
                SectionHeader { title: "媒体空间"; subtitle: "AURORA Sound · 12 扬声器沉浸声场"; trailing: root.playing ? "正在播放" : "已暂停" }
                Item { Layout.fillHeight: true }
                Rectangle {
                    Layout.preferredWidth: Math.min(root.height * 0.36, 300); Layout.preferredHeight: Layout.preferredWidth; Layout.alignment: Qt.AlignHCenter
                    radius: 34; rotation: root.playing ? 2 : 0
                    gradient: Gradient { GradientStop { position: 0; color: root.tracks[root.trackIndex].color } GradientStop { position: 1; color: "#132936" } }
                    Behavior on rotation { NumberAnimation { duration: 700; easing.type: Easing.OutBack } }
                    Rectangle { anchors.fill: parent; anchors.margins: 25; radius: 25; border.color: "#44ffffff"; color: "transparent" }
                    Text { anchors.centerIn: parent; text: "AURORA"; color: "#ddffffff"; font.pixelSize: 25; font.letterSpacing: 6; font.weight: Font.Light }
                }
                ColumnLayout {
                    Layout.fillWidth: true; spacing: 7
                    Text { text: root.tracks[root.trackIndex].title; color: Theme.text; font.pixelSize: 30; font.weight: Font.DemiBold; Layout.alignment: Qt.AlignHCenter }
                    Text { text: root.tracks[root.trackIndex].artist + "  ·  " + root.tracks[root.trackIndex].album; color: Theme.muted; font.pixelSize: 13; Layout.alignment: Qt.AlignHCenter }
                }
                ColumnLayout {
                    Layout.fillWidth: true; spacing: 4
                    Slider { id: seek; Layout.fillWidth: true; from: 0; to: root.duration; value: root.position; onMoved: root.position = value }
                    RowLayout { Text { text: root.formatTime(root.position); color: Theme.muted; font.pixelSize: 10 } Item { Layout.fillWidth: true } Text { text: root.formatTime(root.duration); color: Theme.muted; font.pixelSize: 10 } }
                }
                RowLayout {
                    Layout.alignment: Qt.AlignHCenter; spacing: 26
                    ActionButton { text: "‹‹"; implicitWidth: 55; onClicked: root.selectTrack(root.trackIndex - 1) }
                    Button {
                        implicitWidth: 66; implicitHeight: 66; onClicked: root.playing = !root.playing
                        contentItem: Text { text: root.playing ? "Ⅱ" : "▶"; color: Theme.background; font.pixelSize: 22; horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter }
                        background: Rectangle { radius: 33; color: parent.hovered ? Theme.cyan : Theme.text }
                    }
                    ActionButton { text: "››"; implicitWidth: 55; onClicked: root.selectTrack(root.trackIndex + 1) }
                }
                RowLayout { Layout.fillWidth: true; Text { text: "音量"; color: Theme.muted; font.pixelSize: 11 } Slider { Layout.fillWidth: true; from: 0; to: 100; value: 62 } Text { text: "♫"; color: Theme.cyan } }
                Item { Layout.fillHeight: true }
            }
        }
        GlassPanel {
            visible: root.width > 1020; Layout.preferredWidth: 310; Layout.fillHeight: true
            ColumnLayout {
                anchors.fill: parent; anchors.margins: 20; spacing: 12
                SectionHeader { title: "接下来播放"; subtitle: "夜间巡航 · 4 首" }
                ListView {
                    Layout.fillWidth: true; Layout.fillHeight: true; clip: true; spacing: 8
                    model: root.tracks
                    delegate: GlassPanel {
                        required property var modelData; required property int index
                        width: ListView.view.width; height: 78; highlighted: root.trackIndex === index
                        MouseArea { anchors.fill: parent; onClicked: root.selectTrack(index) }
                        RowLayout { anchors.fill: parent; anchors.margins: 10; spacing: 10; Rectangle { width: 46; height: 46; radius: 12; color: modelData.color; Text { anchors.centerIn: parent; text: root.trackIndex === index && root.playing ? "Ⅱ" : "♫"; color: "#ffffff" } } ColumnLayout { Layout.fillWidth: true; Text { text: modelData.title; color: Theme.text; font.pixelSize: 12; elide: Text.ElideRight; Layout.fillWidth: true } Text { text: modelData.artist; color: Theme.muted; font.pixelSize: 10 } } Text { text: root.formatTime(modelData.duration); color: Theme.muted; font.pixelSize: 9 } }
                    }
                }
                ActionButton { Layout.fillWidth: true; text: "音场设置"; glyph: "◉" }
            }
        }
    }
}
