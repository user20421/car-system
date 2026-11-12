import QtQuick
import Aurora

Canvas {
    id: root
    property real value: 0
    property real maximum: 160
    property string valueText: Math.round(value)
    property string unit: "km/h"
    property string caption: "SPEED"
    property color accent: Theme.cyan
    onValueChanged: requestPaint()
    onWidthChanged: requestPaint()
    onHeightChanged: requestPaint()
    onPaint: {
        var ctx = getContext("2d"); ctx.reset()
        var cx=width/2, cy=height/2, radius=Math.min(width,height)*0.40
        var start=Math.PI*0.75, total=Math.PI*1.5
        ctx.lineCap="round"
        ctx.beginPath(); ctx.arc(cx,cy,radius,start,start+total); ctx.strokeStyle="#223142"; ctx.lineWidth=10; ctx.stroke()
        ctx.beginPath(); ctx.arc(cx,cy,radius,start,start+total*Math.min(value/maximum,1)); ctx.strokeStyle=accent; ctx.lineWidth=10; ctx.stroke()
        for(var i=0;i<=40;i++) {
            var a=start+total*i/40, major=i%5===0, r1=radius-20, r2=radius-(major?32:27)
            ctx.beginPath(); ctx.moveTo(cx+Math.cos(a)*r1,cy+Math.sin(a)*r1); ctx.lineTo(cx+Math.cos(a)*r2,cy+Math.sin(a)*r2)
            ctx.strokeStyle=major?"#7991a5":"#354555"; ctx.lineWidth=major?2:1; ctx.stroke()
        }
    }
    Column {
        anchors.centerIn: parent; spacing: 2
        Text { text: root.valueText; color: Theme.text; font.pixelSize: Math.min(root.width,root.height)*0.18; font.weight: Font.Light; font.family: Theme.monoFamily; anchors.horizontalCenter: parent.horizontalCenter }
        Text { text: root.unit; color: Theme.muted; font.pixelSize: 12; anchors.horizontalCenter: parent.horizontalCenter }
        Text { text: root.caption; color: root.accent; font.pixelSize: 9; font.letterSpacing: 3; anchors.horizontalCenter: parent.horizontalCenter }
    }
    Behavior on value { NumberAnimation { duration: 160; easing.type: Easing.OutQuad } }
}
