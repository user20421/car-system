import QtQuick
import QtQuick.Shapes
import Aurora

Item {
    id: root
    property bool locked: true
    property bool lightsOn: true
    property real heading: 0
    implicitWidth: 280
    implicitHeight: 360
    Rectangle {
        width: parent.width * 0.7; height: parent.height * 0.8; radius: width / 2
        anchors.centerIn: parent
        color: "transparent"; border.width: 1; border.color: Theme.cyanSoft
        Rectangle { anchors.fill: parent; anchors.margins: 25; radius: width / 2; color: "transparent"; border.width: 1; border.color: "#185ce1e6" }
    }
    Shape {
        id: car
        width: root.width * 0.56; height: root.height * 0.85
        anchors.centerIn: parent
        layer.enabled: true
        ShapePath {
            fillColor: "#182d3b"; strokeColor: Theme.cyan; strokeWidth: 2.2
            capStyle: ShapePath.RoundCap; joinStyle: ShapePath.RoundJoin
            startX: car.width * 0.30; startY: 8
            PathCubic { x: car.width * 0.18; y: car.height * 0.25; control1X: car.width * 0.18; control1Y: car.height * 0.03; control2X: car.width * 0.20; control2Y: car.height * 0.16 }
            PathLine { x: car.width * 0.12; y: car.height * 0.72 }
            PathCubic { x: car.width * 0.30; y: car.height - 8; control1X: car.width * 0.12; control1Y: car.height * 0.88; control2X: car.width * 0.20; control2Y: car.height * 0.96 }
            PathCubic { x: car.width * 0.70; y: car.height - 8; control1X: car.width * 0.45; control1Y: car.height; control2X: car.width * 0.55; control2Y: car.height }
            PathCubic { x: car.width * 0.88; y: car.height * 0.72; control1X: car.width * 0.80; control1Y: car.height * 0.96; control2X: car.width * 0.88; control2Y: car.height * 0.88 }
            PathLine { x: car.width * 0.82; y: car.height * 0.25 }
            PathCubic { x: car.width * 0.70; y: 8; control1X: car.width * 0.80; control1Y: car.height * 0.16; control2X: car.width * 0.82; control2Y: car.height * 0.03 }
            PathLine { x: car.width * 0.30; y: 8 }
        }
        ShapePath {
            fillColor: "#0b131e"; strokeColor: "#537185"; strokeWidth: 1.3
            startX: car.width * 0.30; startY: car.height * 0.24
            PathLine { x: car.width * 0.70; y: car.height * 0.24 }
            PathLine { x: car.width * 0.76; y: car.height * 0.44 }
            PathLine { x: car.width * 0.24; y: car.height * 0.44 }
            PathLine { x: car.width * 0.30; y: car.height * 0.24 }
        }
        ShapePath {
            fillColor: "#101b28"; strokeColor: "#537185"; strokeWidth: 1.3
            startX: car.width * 0.24; startY: car.height * 0.53
            PathLine { x: car.width * 0.76; y: car.height * 0.53 }
            PathLine { x: car.width * 0.72; y: car.height * 0.77 }
            PathLine { x: car.width * 0.28; y: car.height * 0.77 }
            PathLine { x: car.width * 0.24; y: car.height * 0.53 }
        }
        ShapePath {
            fillColor: "transparent"; strokeColor: "#355064"; strokeWidth: 1
            startX: car.width * 0.5; startY: car.height * 0.24
            PathLine { x: car.width * 0.5; y: car.height * 0.77 }
            PathMove { x: car.width * 0.24; y: car.height * 0.49 }
            PathLine { x: car.width * 0.76; y: car.height * 0.49 }
        }
    }
    Repeater {
        model: [{x: 0.16, y: 0.24}, {x: 0.78, y: 0.24}, {x: 0.16, y: 0.66}, {x: 0.78, y: 0.66}]
        delegate: Rectangle { x: root.width * modelData.x; y: root.height * modelData.y; width: root.width * 0.06; height: root.height * 0.14; radius: 5; color: "#030509"; border.color: "#354554" }
    }
    Rectangle { width: root.width * 0.12; height: 5; radius: 3; color: root.lightsOn ? "#d9ffff" : "#34444d"; anchors.left: parent.left; anchors.leftMargin: root.width * 0.33; anchors.top: parent.top; anchors.topMargin: root.height * 0.09 }
    Rectangle { width: root.width * 0.12; height: 5; radius: 3; color: root.lightsOn ? "#d9ffff" : "#34444d"; anchors.right: parent.right; anchors.rightMargin: root.width * 0.33; anchors.top: parent.top; anchors.topMargin: root.height * 0.09 }
    Rectangle {
        anchors.horizontalCenter: parent.horizontalCenter; anchors.bottom: parent.bottom; anchors.bottomMargin: 10
        width: lockRow.width + 22; height: 30; radius: 15; color: Theme.surfaceRaised; border.color: root.locked ? Theme.green : Theme.amber
        Row { id: lockRow; anchors.centerIn: parent; spacing: 6; Text { text: root.locked ? "●" : "○"; color: root.locked ? Theme.green : Theme.amber; font.pixelSize: 10 } Text { text: root.locked ? "已锁车" : "已解锁"; color: Theme.text; font.pixelSize: 11 } }
    }
}
