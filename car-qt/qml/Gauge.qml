import QtQuick

// 可复用圆形仪表盘组件
Item {
    id: root
    property real value: 0
    property real minValue: 0
    property real maxValue: 240
    property string title: ""
    property string unit: ""
    property color accentColor: "#4fc3f7"
    property bool warning: false

    // 刻度范围：-210° 到 30°（240° 扫掠）
    readonly property real startAngle: -210
    readonly property real endAngle: 30
    readonly property real ratio: Math.max(0, Math.min(1, (value - minValue) / (maxValue - minValue)))

    Canvas {
        id: canvas
        anchors.fill: parent
        antialiasing: true

        onPaint: {
            const ctx = getContext("2d")
            const w = width, h = height
            const cx = w / 2, cy = h / 2
            const radius = Math.min(w, h) / 2 - 12
            ctx.clearRect(0, 0, w, h)
            if (radius <= 0)   // 尺寸未就绪时跳过绘制，避免 arc() 参数非法
                return

            const toRad = (deg) => deg * Math.PI / 180
            const drawArc = (from, to, color, lineWidth) => {
                ctx.beginPath()
                ctx.arc(cx, cy, radius, toRad(from), toRad(to))
                ctx.strokeStyle = color
                ctx.lineWidth = lineWidth
                ctx.lineCap = "round"
                ctx.stroke()
            }

            // 背景弧
            drawArc(root.startAngle, root.endAngle, "#2a2f3a", 14)
            // 数值弧
            const valueAngle = root.startAngle + (root.endAngle - root.startAngle) * root.ratio
            if (root.ratio > 0.005)
                drawArc(root.startAngle, valueAngle, root.warning ? "#ff5252" : root.accentColor, 14)

            // 刻度
            ctx.strokeStyle = "#5a6372"
            ctx.lineWidth = 2
            const ticks = 12
            for (let i = 0; i <= ticks; i++) {
                const a = toRad(root.startAngle + (root.endAngle - root.startAngle) * i / ticks)
                const r1 = radius - 20, r2 = radius - 30
                ctx.beginPath()
                ctx.moveTo(cx + r1 * Math.cos(a), cy + r1 * Math.sin(a))
                ctx.lineTo(cx + r2 * Math.cos(a), cy + r2 * Math.sin(a))
                ctx.stroke()
            }

            // 指针
            const needleAngle = toRad(root.startAngle + (root.endAngle - root.startAngle) * root.ratio)
            ctx.beginPath()
            ctx.moveTo(cx, cy)
            ctx.lineTo(cx + (radius - 45) * Math.cos(needleAngle), cy + (radius - 45) * Math.sin(needleAngle))
            ctx.strokeStyle = root.warning ? "#ff5252" : "#e6edf3"
            ctx.lineWidth = 4
            ctx.lineCap = "round"
            ctx.stroke()

            // 中心圆点
            ctx.beginPath()
            ctx.arc(cx, cy, 8, 0, 2 * Math.PI)
            ctx.fillStyle = root.warning ? "#ff5252" : root.accentColor
            ctx.fill()
        }

        // 尺寸变化时重绘
        onWidthChanged: requestPaint()
        onHeightChanged: requestPaint()

        Connections {
            target: root
            function onValueChanged() { canvas.requestPaint() }
            function onWarningChanged() { canvas.requestPaint() }
        }
    }

    // 数字读数
    Column {
        anchors.centerIn: parent
        anchors.verticalCenterOffset: 28
        spacing: 2
        Text {
            anchors.horizontalCenter: parent.horizontalCenter
            text: root.value.toFixed(root.maxValue > 100 ? 1 : 0)
            color: root.warning ? "#ff5252" : "#e6edf3"
            font.pixelSize: 34
            font.bold: true
        }
        Text {
            anchors.horizontalCenter: parent.horizontalCenter
            text: root.unit
            color: "#8b949e"
            font.pixelSize: 13
        }
    }

    Text {
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.bottom: parent.bottom
        anchors.bottomMargin: 6
        text: root.title
        color: "#8b949e"
        font.pixelSize: 14
    }
}
