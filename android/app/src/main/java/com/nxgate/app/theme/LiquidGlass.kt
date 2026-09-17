package com.nxgate.app.theme

import android.os.Build
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.graphics.asComposeRenderEffect
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

/**
 * AndroidLiquidGlass (Backdrop) Inspired Styling.
 * Replicates the Apple VisionOS & AndroidLiquidGlass physical liquid glass material:
 * - High-transparency frosty substrate (45% - 60% opacity) so background content clearly passes through
 * - 135 deg Prismatic Specular Highlight stroke simulating glass facet refraction
 * - Hardware level RenderEffect blur on Android 12+ (API 31+)
 * - Deep diffuse atmospheric shadow
 */
object LiquidGlassDefaults {
    val DefaultCornerRadius = 32.dp
    val HighlightBorderWidth = 1.2.dp

    // Light Theme Liquid Glass Palette (Highly Translucent & Crystal Clear)
    val LightSubstrateColor = Color(0xFFFFFFFF).copy(alpha = 0.55f)
    val LightGlassTint = Color(0xFFE2EFFF).copy(alpha = 0.25f)
    val LightHighlightTop = Color.White.copy(alpha = 0.90f)
    val LightHighlightBottom = Color(0xFF8FB3DD).copy(alpha = 0.35f)

    // Dark Theme Liquid Glass Palette (Translucent Obsidian)
    val DarkSubstrateColor = Color(0xFF0E1622).copy(alpha = 0.65f)
    val DarkGlassTint = Color(0xFF1E2E44).copy(alpha = 0.35f)
    val DarkHighlightTop = Color.White.copy(alpha = 0.50f)
    val DarkHighlightBottom = Color(0xFF1C2C40).copy(alpha = 0.25f)
}

/**
 * Applies physical Liquid Glass material effect to any Composable:
 * - Real hardware-accelerated frosted blur on Android 12+ (RenderEffect)
 * - Highly translucent crystal substrate allowing content underneath to show through
 * - Prismatic 135-degree specular highlight reflection rim
 * - Soft diffuse atmospheric elevation
 */
@Composable
fun Modifier.liquidGlass(
    shape: Shape = RoundedCornerShape(LiquidGlassDefaults.DefaultCornerRadius),
    isDark: Boolean = false,
    elevation: Dp = 8.dp,
    tintColor: Color? = null,
    blurRadius: Float = 24f
): Modifier {
    val substrate = if (isDark) LiquidGlassDefaults.DarkSubstrateColor else LiquidGlassDefaults.LightSubstrateColor
    val tint = tintColor?.copy(alpha = if (isDark) 0.15f else 0.08f)
        ?: (if (isDark) LiquidGlassDefaults.DarkGlassTint else LiquidGlassDefaults.LightGlassTint)

    val highlightBrush = Brush.linearGradient(
        colors = listOf(
            if (isDark) LiquidGlassDefaults.DarkHighlightTop else LiquidGlassDefaults.LightHighlightTop,
            if (isDark) LiquidGlassDefaults.DarkHighlightBottom else LiquidGlassDefaults.LightHighlightBottom
        ),
        start = Offset(0f, 0f),
        end = Offset(Float.POSITIVE_INFINITY, Float.POSITIVE_INFINITY)
    )

    // Glass refraction gloss overlay (subtle diagonal sheen)
    val glossBrush = Brush.linearGradient(
        colors = listOf(
            Color.White.copy(alpha = if (isDark) 0.12f else 0.28f),
            Color.White.copy(alpha = if (isDark) 0.02f else 0.08f),
            Color.Transparent
        ),
        start = Offset(0f, 0f),
        end = Offset(Float.POSITIVE_INFINITY, Float.POSITIVE_INFINITY)
    )

    return this
        .shadow(
            elevation = elevation,
            shape = shape,
            ambientColor = if (isDark) Color(0xFF001529).copy(alpha = 0.50f) else Color(0xFF526D92).copy(alpha = 0.20f),
            spotColor = if (isDark) Color(0xFF002244).copy(alpha = 0.60f) else Color(0xFF38557A).copy(alpha = 0.25f)
        )
        .graphicsLayer {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
                renderEffect = android.graphics.RenderEffect.createBlurEffect(
                    blurRadius,
                    blurRadius,
                    android.graphics.Shader.TileMode.CLAMP
                ).asComposeRenderEffect()
            }
        }
        .clip(shape)
        .background(substrate)
        .background(tint)
        .background(glossBrush)
        .border(
            border = BorderStroke(LiquidGlassDefaults.HighlightBorderWidth, highlightBrush),
            shape = shape
        )
}
