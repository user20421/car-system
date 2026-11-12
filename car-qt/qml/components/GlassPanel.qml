import QtQuick
import Aurora

Rectangle {
    id: root
    property bool highlighted: false
    color: Theme.panel
    radius: Theme.radius
    border.width: 1
    border.color: highlighted ? Theme.cyanSoft : Theme.stroke
    Behavior on border.color { ColorAnimation { duration: 220 } }
}
