package com.nxgate.app.ui.screens.matrix

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.PathEffect
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

@Composable
fun UnlockMiniPill(label: String, status: String) {
    val isOk = status == "unlocked"
    val textColor = if (isOk) Color(0xFF10B981) else MaterialTheme.colorScheme.outline
    val bgColor = if (isOk) Color(0xFF10B981).copy(alpha = 0.12f) else MaterialTheme.colorScheme.surfaceContainerHighest
    Surface(
        shape = RoundedCornerShape(4.dp),
        color = bgColor
    ) {
        Text(
            text = if (isOk) "$label✓" else label,
            style = MaterialTheme.typography.labelSmall,
            fontWeight = FontWeight.SemiBold,
            color = textColor,
            fontSize = 9.sp,
            modifier = Modifier.padding(horizontal = 4.dp, vertical = 1.dp)
        )
    }
}

@Composable
fun PipelineCurvedConnector(
    exitsCount: Int = 1,
    isFallback: Boolean = false,
    modifier: Modifier = Modifier,
    color: Color = Color(0xFF4DCAEC)
) {
    val pathEffect = remember { PathEffect.dashPathEffect(floatArrayOf(12f, 8f), 0f) }
    Canvas(modifier = modifier) {
        val stroke = 2.dp.toPx()
        val startY = size.height / 2f
        val exitCardHeight = 84.dp.toPx()
        val gap = 10.dp.toPx()
        val count = exitsCount.coerceAtLeast(1)
        val wireColor = if (isFallback) Color(0xFFF59E0B) else color
        for (i in 0 until count) {
            val endY = if (count == 1) startY else (i * (exitCardHeight + gap) + exitCardHeight / 2f)
            val path = Path().apply {
                moveTo(0f, startY)
                cubicTo(size.width * 0.5f, startY, size.width * 0.5f, endY, size.width, endY)
            }
            drawPath(path, wireColor, style = Stroke(width = stroke, cap = StrokeCap.Round, pathEffect = pathEffect))
            drawCircle(wireColor, radius = 3.dp.toPx(), center = Offset(size.width, endY))
        }
        drawCircle(wireColor, radius = 3.5.dp.toPx(), center = Offset(0f, startY))
    }
}
