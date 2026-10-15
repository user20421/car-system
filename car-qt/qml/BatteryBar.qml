import QtQuick

// 电量条组件
Item {
    id: root
    property real value: 0      // 0~100
    property string title: "电量"

    readonly property color barColor: value > 50 ? "#3fb950" : (value > 20 ? "#d29922" : "#ff5252")

    Column {
        anchors.fill: parent
        spacing: 6

        Item {
            width: parent.width
            height: 20
            Text {
                text: root.title
                color: "#8b949e"
                font.pixelSize: 14
                anchors.verticalCenter: parent.verticalCenter
            }
            Text {
                anchors.right: parent.right
                anchors.verticalCenter: parent.verticalCenter
                text: root.value.toFixed(1) + " %"
                color: root.barColor
                font.pixelSize: 16
                font.bold: true
            }
        }

        Item {
            width: parent.width
            height: 26

            // 电池主体
            Rectangle {
                id: body
                anchors.left: parent.left
                anchors.verticalCenter: parent.verticalCenter
                width: parent.width - 10
                height: parent.height
                radius: 5
                color: "#2a2f3a"
                border.color: "#444c56"
                border.width: 1

                Rectangle {
                    anchors.left: parent.left
                    anchors.leftMargin: 2
                    anchors.verticalCenter: parent.verticalCenter
                    width: Math.max(0, (parent.width - 4) * root.value / 100)
                    height: parent.height - 4
                    radius: 4
                    color: root.barColor
                    Behavior on width { NumberAnimation { duration: 300 } }
                }
            }

            // 电池正极凸起
            Rectangle {
                anchors.left: body.right
                anchors.leftMargin: 2
                anchors.verticalCenter: parent.verticalCenter
                width: 6
                height: 12
                radius: 2
                color: "#444c56"
            }
        }
    }
}
