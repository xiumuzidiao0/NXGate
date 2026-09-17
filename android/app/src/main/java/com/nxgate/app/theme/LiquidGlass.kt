package com.nxgate.app.theme

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.BlendMode
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.graphics.drawOutline
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

/**
 * AndroidLiquidGlass (Backdrop) Inspired Styling.
 * Replicates the Apple/VisionOS Liquid Glass material effect for Android Compose:
 * - Translucent frosty glass substrate with subtle refractive tint
 * - Specular highlight rim reflecting environmental light (135 deg linear gradient)
 * - Atmospheric elevation and soft light transmission
 */
object LiquidGlassDefaults {
    val DefaultCornerRadius = 20.dp
    val HighlightBorderWidth = 1.dp

    // Light Theme Liquid Glass Palette
    val LightSubstrateColor = Color(0xFFFFFFFF).copy(alpha = 0.88f)
    val LightGlassTint = Color(0xFFE8F2FF).copy(alpha = 0.35f)
    val LightHighlightTop = Color.White.copy(alpha = 0.95f)
    val LightHighlightBottom = Color(0xFFB0C8E8).copy(alpha = 0.40f)

    // Dark Theme Liquid Glass Palette
    val DarkSubstrateColor = Color(0xFF141922).copy(alpha = 0.82f)
    val DarkGlassTint = Color(0xFF1A2638).copy(alpha = 0.50f)
    val DarkHighlightTop = Color.White.copy(alpha = 0.35f)
    val DarkHighlightBottom = Color(0xFF2C3E55).copy(alpha = 0.20f)
}

/**
 * Applies a Liquid Glass effect to any Composable:
 * - Frosted glass layer with subtle translucent refraction
 * - Prismatic highlight stroke with directional glare (inspired by AndroidLiquidGlass Highlight.kt)
 * - Atmospheric elevation shadow
 */
@Composable
fun Modifier.liquidGlass(
    shape: Shape = RoundedCornerShape(LiquidGlassDefaults.DefaultCornerRadius),
    isDark: Boolean = false,
    elevation: Dp = 2.dp,
    tintColor: Color? = null
): Modifier {
    val substrate = if (isDark) LiquidGlassDefaults.DarkSubstrateColor else LiquidGlassDefaults.LightSubstrateColor
    val tint = tintColor?.copy(alpha = if (isDark) 0.15f else 0.08f) ?: (if (isDark) LiquidGlassDefaults.DarkGlassTint else LiquidGlassDefaults.LightGlassTint)

    val highlightBrush = Brush.linearGradient(
        colors = listOf(
            if (isDark) LiquidGlassDefaults.DarkHighlightTop else LiquidGlassDefaults.LightHighlightTop,
            if (isDark) LiquidGlassDefaults.DarkHighlightBottom else LiquidGlassDefaults.LightHighlightBottom
        ),
        start = Offset(0f, 0f),
        end = Offset(Float.POSITIVE_INFINITY, Float.POSITIVE_INFINITY)
    )

    return this
        .shadow(
            elevation = elevation,
            shape = shape,
            ambientColor = if (isDark) Color(0xFF001529).copy(alpha = 0.5f) else Color(0xFF6B82A6).copy(alpha = 0.25f),
            spotColor = if (isDark) Color(0xFF002244).copy(alpha = 0.6f) else Color(0xFF4A6B99).copy(alpha = 0.30f)
        )
        .clip(shape)
        .background(substrate)
        .background(tint)
        .border(
            border = BorderStroke(LiquidGlassDefaults.HighlightBorderWidth, highlightBrush),
            shape = shape
        )
}
