package com.aimili.vpn.ui.components

import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.tween
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.aimili.vpn.model.LiveTrafficInfo
import com.aimili.vpn.model.SpeedUnit
import kotlinx.coroutines.delay

/**
 * M3 Expressive Speed Waveform Card (height: 164dp, background: surfaceContainerHigh).
 * Displays live download & upload speed and smooth Bézier curve graph.
 * Tap toggles speed units (Mb/s -> MB/s -> KB/s).
 */
@Composable
fun SpeedWaveformCard(
    liveTraffic: LiveTrafficInfo = LiveTrafficInfo(),
    downSpeedStr: String = "",
    upSpeedStr: String = "",
    modifier: Modifier = Modifier
) {
    var unit by remember { mutableStateOf(SpeedUnit.MBPS) }

    // Waveform live data points
    val wavePoints = remember {
        mutableStateOf(listOf(0.2f, 0.4f, 0.35f, 0.7f, 0.55f, 0.85f, 0.75f, 0.95f, 0.65f, 0.8f))
    }

    // Reflect real throughput changes onto wave curve
    LaunchedEffect(liveTraffic.downloadSpeedBps, liveTraffic.uploadSpeedBps) {
        val totalSpeedBps = liveTraffic.downloadSpeedBps + liveTraffic.uploadSpeedBps
        val targetHeight = when {
            totalSpeedBps <= 0 -> 0.15f
            totalSpeedBps < 100_000 -> 0.25f + (totalSpeedBps / 100_000f) * 0.15f
            totalSpeedBps < 1_000_000 -> 0.40f + (totalSpeedBps / 1_000_000f) * 0.25f
            else -> (0.65f + (totalSpeedBps / 10_000_000f) * 0.30f).coerceAtMost(0.95f)
        }
        val current = wavePoints.value.toMutableList()
        current.removeAt(0)
        current.add(targetHeight)
        wavePoints.value = current
    }

    val displayDown = if (downSpeedStr.isNotEmpty()) downSpeedStr else liveTraffic.formattedDownSpeed(unit)
    val displayUp = if (upSpeedStr.isNotEmpty()) upSpeedStr else liveTraffic.formattedUpSpeed(unit)

    val phaseAnim = remember { Animatable(0f) }

    LaunchedEffect(Unit) {
        while (true) {
            phaseAnim.animateTo(
                targetValue = 1f,
                animationSpec = tween(durationMillis = 1800, easing = LinearEasing)
            )
            phaseAnim.snapTo(0f)
            // Shift points subtly to simulate real stream
            val current = wavePoints.value.toMutableList()
            val nextPoint = (0.3f + Math.random().toFloat() * 0.65f)
            current.removeAt(0)
            current.add(nextPoint)
            wavePoints.value = current
        }
    }

    Card(
        modifier = modifier
            .fillMaxWidth()
            .height(164.dp)
            .clip(RoundedCornerShape(20.dp))
            .clickable {
                unit = when (unit) {
                    SpeedUnit.MB_S -> SpeedUnit.MBPS
                    SpeedUnit.MBPS -> SpeedUnit.KB_S
                    SpeedUnit.KB_S -> SpeedUnit.MB_S
                }
            },
        shape = RoundedCornerShape(20.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainer
        ),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant),
        elevation = CardDefaults.cardElevation(defaultElevation = 1.5.dp)
    ) {
        Box(modifier = Modifier.fillMaxSize()) {
            // Background Bézier curve Canvas
            val primaryColor = MaterialTheme.colorScheme.primary
            val tertiaryColor = MaterialTheme.colorScheme.tertiary
            val points = wavePoints.value

            Canvas(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(top = 40.dp)
            ) {
                val width = size.width
                val height = size.height

                val path = Path()
                val fillPath = Path()

                val step = width / (points.size - 1)

                path.moveTo(0f, height * (1f - points[0] * 0.7f))
                fillPath.moveTo(0f, height)
                fillPath.lineTo(0f, height * (1f - points[0] * 0.7f))

                for (i in 1 until points.size) {
                    val prevX = (i - 1) * step
                    val prevY = height * (1f - points[i - 1] * 0.7f)
                    val currX = i * step
                    val currY = height * (1f - points[i] * 0.7f)

                    val cx = (prevX + currX) / 2
                    path.cubicTo(cx, prevY, cx, currY, currX, currY)
                    fillPath.cubicTo(cx, prevY, cx, currY, currX, currY)
                }

                fillPath.lineTo(width, height)
                fillPath.close()

                // Draw gradient under curve
                drawPath(
                    path = fillPath,
                    brush = Brush.verticalGradient(
                        colors = listOf(
                            primaryColor.copy(alpha = 0.25f),
                            Color.Transparent
                        )
                    )
                )

                // Draw curve stroke
                drawPath(
                    path = path,
                    color = primaryColor,
                    style = Stroke(width = 3.dp.toPx())
                )
            }

            // Foreground Text Content
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(20.dp),
                verticalArrangement = Arrangement.SpaceBetween
            ) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "实时网速波形",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                    Text(
                        text = when (unit) {
                            SpeedUnit.MB_S -> "单位: MB/s"
                            SpeedUnit.MBPS -> "单位: Mb/s"
                            SpeedUnit.KB_S -> "单位: KB/s"
                        },
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.primary
                    )
                }

                Column {
                    Text(
                        text = "下行：$displayDown",
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.SemiBold,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                    Text(
                        text = "上行：$displayUp",
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.SemiBold,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                    Spacer(Modifier.height(4.dp))
                    Text(
                        text = "活跃连接：${liveTraffic.activeConnections} 个 • 实时采样率：3秒/次 • 点击卡片切换单位",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                }
            }
        }
    }
}
