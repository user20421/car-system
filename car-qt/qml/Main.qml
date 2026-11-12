import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Aurora

ApplicationWindow {
    id: window
    visible: true
    width: 1440
    height: 900
    minimumWidth: 960
    minimumHeight: 640
    title: "AURORA · " + vehicle.vin
    color: Theme.background
    font.family: Theme.fontFamily
    property bool compact: width < 1100
    property int pageIndex: initialPage
    property bool debugOpen: false
    property string toastText: ""

    Rectangle {
        anchors.fill: parent
        gradient: Gradient {
            GradientStop { position: 0.0; color: "#101a27" }
            GradientStop { position: 0.45; color: Theme.background }
            GradientStop { position: 1.0; color: "#05070c" }
        }
        Rectangle { width: 520; height: 520; x: parent.width * 0.52; y: -280; radius: 260; color: "#0a5ce1e6" }
        Rectangle { width: 380; height: 380; x: -160; y: parent.height - 180; radius: 190; color: "#089b8cff" }
    }

    GlassPanel {
        id: rail
        visible: !window.compact
        width: 88
        anchors.left: parent.left
        anchors.top: parent.top
        anchors.bottom: parent.bottom
        anchors.margins: 16
        radius: 26
        ColumnLayout {
            anchors.fill: parent
            anchors.topMargin: 22
            anchors.bottomMargin: 18
            spacing: 8
            Rectangle {
                Layout.alignment: Qt.AlignHCenter
                Layout.bottomMargin: 17
                width: 42; height: 42; radius: 14; color: Theme.cyan
                Text { anchors.centerIn: parent; text: "A"; color: "#071217"; font.pixelSize: 25; font.weight: Font.Black }
            }
            Repeater {
                model: [
                    {glyph:"⌂", label:"桌面"}, {glyph:"◉", label:"仪表"}, {glyph:"⌁", label:"导航"},
                    {glyph:"♫", label:"媒体"}, {glyph:"❄", label:"空调"}, {glyph:"◇", label:"车辆"},
                    {glyph:"◌", label:"通知"}, {glyph:"⚙", label:"设置"}
                ]
                delegate: NavButton {
                    required property var modelData
                    required property int index
                    glyph: modelData.glyph; label: modelData.label
                    active: window.pageIndex === index
                    Layout.alignment: Qt.AlignHCenter
                    onClicked: window.pageIndex = index
                }
            }
            Item { Layout.fillHeight: true }
            Button {
                id: debugButton
                visible: debugBuild
                Layout.alignment: Qt.AlignHCenter
                implicitWidth: 42; implicitHeight: 42
                text: "∆"
                onClicked: window.debugOpen = !window.debugOpen
                contentItem: Text { text: debugButton.text; color: window.debugOpen ? Theme.amber : Theme.muted; font.pixelSize: 20; horizontalAlignment: Text.AlignHCenter }
                background: Rectangle { radius: 13; color: debugButton.hovered ? Theme.surfaceRaised : "transparent"; border.color: Theme.stroke }
            }
        }
    }

    ColumnLayout {
        id: shell
        anchors.left: window.compact ? parent.left : rail.right
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.bottom: window.compact ? bottomNav.top : parent.bottom
        anchors.margins: window.compact ? 16 : 12
        anchors.leftMargin: window.compact ? 16 : 24
        anchors.rightMargin: 24
        spacing: 18
        RowLayout {
            Layout.fillWidth: true
            Layout.preferredHeight: 48
            spacing: 16
            ColumnLayout {
                spacing: 1
                Text { text: ["智能桌面","数字仪表","导航","媒体空间","座舱气候","车辆中心","消息中心","系统设置"][window.pageIndex]; color: Theme.text; font.pixelSize: 22; font.weight: Font.DemiBold }
                Text { text: "AURORA OS  ·  " + vehicle.vin; color: Theme.muted; font.pixelSize: 10; font.letterSpacing: 1.2 }
            }
            Item { Layout.fillWidth: true }
            ColumnLayout {
                visible: window.width > 1180
                Text { id: clockText; text: "00:00"; color: Theme.text; font.pixelSize: 21; font.weight: Font.DemiBold; Layout.alignment: Qt.AlignRight }
                Text { id: dateText; text: ""; color: Theme.muted; font.pixelSize: 10; Layout.alignment: Qt.AlignRight }
            }
            StatusPill { connectionState: vehicle.networkState; label: vehicle.networkState === "online" ? "车辆在线" : vehicle.networkState === "connecting" ? "正在连接" : "本地模式" }
        }

        StackLayout {
            id: pages
            Layout.fillWidth: true
            Layout.fillHeight: true
            currentIndex: window.pageIndex
            HomePage { onNavigateTo: function(index) { window.pageIndex = index } }
            ClusterPage {}
            NavigationPage {}
            MediaPage {}
            ClimatePage {}
            VehiclePage {}
            NotificationsPage {}
            SettingsPage {}
        }
    }

    GlassPanel {
        id: bottomNav
        visible: window.compact
        height: 76
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        anchors.margins: 12
        radius: 24
        RowLayout {
            anchors.fill: parent
            anchors.leftMargin: 10; anchors.rightMargin: 10
            spacing: 2
            Repeater {
                model: [
                    {glyph:"⌂", label:"桌面"}, {glyph:"◉", label:"仪表"}, {glyph:"⌁", label:"导航"},
                    {glyph:"♫", label:"媒体"}, {glyph:"❄", label:"空调"}, {glyph:"◇", label:"车辆"},
                    {glyph:"◌", label:"通知"}, {glyph:"⚙", label:"设置"}
                ]
                delegate: NavButton {
                    required property var modelData
                    required property int index
                    glyph: modelData.glyph; label: modelData.label
                    active: window.pageIndex === index
                    Layout.fillWidth: true
                    onClicked: window.pageIndex = index
                }
            }
        }
    }

    DebugPanel {
        visible: debugBuild && window.debugOpen
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.bottom: parent.bottom
        anchors.margins: 16
        width: Math.min(340, parent.width * 0.32)
        onClose: window.debugOpen = false
    }

    Rectangle {
        id: toast
        visible: opacity > 0
        opacity: 0
        width: Math.min(toastLabel.implicitWidth + 46, window.width - 80)
        height: 48
        radius: 16
        color: "#ee172330"
        border.color: Theme.cyanSoft
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.bottom: parent.bottom
        anchors.bottomMargin: window.compact ? 104 : 28
        Text { id: toastLabel; anchors.centerIn: parent; width: Math.min(implicitWidth, window.width - 126); text: window.toastText; color: Theme.text; font.pixelSize: 12; elide: Text.ElideRight }
        SequentialAnimation {
            id: toastAnimation
            running: false
            NumberAnimation { target: toast; property: "opacity"; to: 1; duration: 180 }
            PauseAnimation { duration: 2600 }
            NumberAnimation { target: toast; property: "opacity"; to: 0; duration: 260 }
        }
    }
    Connections {
        target: vehicle
        function onCriticalFault(message) { window.toastText = message; toastAnimation.restart() }
        function onCommandChanged() {
            if (vehicle.commandState === "succeeded" || vehicle.commandState === "failed" || vehicle.commandState === "expired") {
                window.toastText = vehicle.commandMessage; toastAnimation.restart()
            }
        }
    }
    Timer {
        interval: 1000; running: true; repeat: true; triggeredOnStart: true
        onTriggered: { var d = new Date(); clockText.text = Qt.formatTime(d, "HH:mm"); dateText.text = Qt.formatDate(d, "yyyy.MM.dd  dddd") }
    }
}
