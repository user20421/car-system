import QtQuick
import Aurora

Canvas {
    id: root
    property real latitude: 31.2304
    property real longitude: 121.4737
    property real heading: 0
    property bool navigating: true
    property real zoom: 1.0
    property color routeColor: Theme.cyan
    onLatitudeChanged: requestPaint()
    onLongitudeChanged: requestPaint()
    onZoomChanged: requestPaint()
    onWidthChanged: requestPaint()
    onHeightChanged: requestPaint()
    onPaint: {
        var ctx = getContext("2d")
        ctx.reset()
        var w = width, h = height
        ctx.fillStyle = "#0b121c"
        ctx.fillRect(0, 0, w, h)
        ctx.lineWidth = 1
        ctx.strokeStyle = "#182534"
        var grid = 42 * zoom
        for (var x = (w % grid) / 2; x < w; x += grid) { ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, h); ctx.stroke() }
        for (var y = (h % grid) / 2; y < h; y += grid) { ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(w, y); ctx.stroke() }
        var roads = [[-0.1,0.75,0.35,0.58,0.63,0.65,1.1,0.22],[-0.1,0.25,0.25,0.34,0.52,0.18,1.1,0.40],[0.2,-0.1,0.28,0.28,0.17,0.63,0.34,1.1],[0.75,-0.1,0.70,0.32,0.86,0.66,0.72,1.1]]
        for (var r = 0; r < roads.length; r++) {
            var road = roads[r]
            ctx.beginPath(); ctx.moveTo(road[0]*w,road[1]*h); ctx.lineTo(road[2]*w,road[3]*h); ctx.lineTo(road[4]*w,road[5]*h); ctx.lineTo(road[6]*w,road[7]*h)
            ctx.strokeStyle = "#263747"; ctx.lineWidth = 15 * zoom; ctx.stroke()
            ctx.strokeStyle = "#111c28"; ctx.lineWidth = 11 * zoom; ctx.stroke()
            ctx.setLineDash([5, 10]); ctx.strokeStyle = "#3c4b57"; ctx.lineWidth = 1; ctx.stroke(); ctx.setLineDash([])
        }
        if (navigating) {
            var points = [[0.08,0.85],[0.18,0.69],[0.36,0.64],[0.44,0.43],[0.62,0.39],[0.70,0.20],[0.89,0.15]]
            ctx.beginPath(); ctx.moveTo(points[0][0]*w, points[0][1]*h)
            for (var i=1;i<points.length;i++) ctx.lineTo(points[i][0]*w, points[i][1]*h)
            ctx.strokeStyle = "#335ce1e6"; ctx.lineWidth = 11; ctx.lineCap = "round"; ctx.lineJoin = "round"; ctx.stroke()
            ctx.strokeStyle = routeColor; ctx.lineWidth = 4; ctx.stroke()
            ctx.beginPath(); ctx.arc(0.89*w,0.15*h,7,0,Math.PI*2); ctx.fillStyle = Theme.violet; ctx.fill()
        }
        var px = w * (0.50 + Math.sin(longitude * 13) * 0.035)
        var py = h * (0.53 + Math.cos(latitude * 17) * 0.025)
        var glow = ctx.createRadialGradient(px,py,2,px,py,26)
        glow.addColorStop(0,"#885ce1e6"); glow.addColorStop(1,"#005ce1e6")
        ctx.fillStyle = glow; ctx.beginPath(); ctx.arc(px,py,26,0,Math.PI*2); ctx.fill()
        ctx.save(); ctx.translate(px,py); ctx.rotate(heading * Math.PI / 180)
        ctx.beginPath(); ctx.moveTo(0,-12); ctx.lineTo(8,9); ctx.lineTo(0,5); ctx.lineTo(-8,9); ctx.closePath()
        ctx.fillStyle = "#eaffff"; ctx.fill(); ctx.restore()
        ctx.fillStyle = "#7890a4"; ctx.font = "11px sans-serif"
        ctx.fillText("世纪大道",w*0.08,h*0.54); ctx.fillText("陆家嘴环路",w*0.64,h*0.76); ctx.fillText("黄浦江",w*0.80,h*0.36)
    }
    Rectangle { anchors.right: parent.right; anchors.bottom: parent.bottom; anchors.margins: 14; width: 64; height: 24; radius: 6; color: "#cc101722"; Text { anchors.centerIn: parent; text: "200 m"; color: Theme.muted; font.pixelSize: 10 } }
}
