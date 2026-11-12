pragma Singleton
import QtQuick

QtObject {
    readonly property color background: "#070b12"
    readonly property color surface: "#101722"
    readonly property color surfaceRaised: "#151f2d"
    readonly property color panel: "#cc101722"
    readonly property color panelSoft: "#88172231"
    readonly property color stroke: "#263445"
    readonly property color text: "#f2f6fa"
    readonly property color muted: "#8b9aac"
    readonly property color cyan: "#5ce1e6"
    readonly property color cyanSoft: "#265ce1e6"
    readonly property color violet: "#9b8cff"
    readonly property color amber: "#ffc76a"
    readonly property color red: "#ff7085"
    readonly property color green: "#6ce5ae"
    readonly property int radius: 22
    readonly property int smallRadius: 14
    readonly property int pageMargin: 28
    readonly property string fontFamily: "Microsoft YaHei UI"
    readonly property string monoFamily: "Cascadia Mono"
    function speedText(value) { return Math.round(value).toString().padStart(3, "0") }
}
