package com.nxgate.app.theme

import android.app.Activity
import android.os.Build
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.ColorScheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.dynamicDarkColorScheme
import androidx.compose.material3.dynamicLightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.SideEffect
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalView
import androidx.core.view.WindowCompat

@Composable
fun NXGateTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    paletteId: String = "pixel",
    accentId: String = "teal",
    baseId: String = "neutral",
    content: @Composable () -> Unit
) {
    val context = LocalContext.current
    val colorScheme: ColorScheme = when {
        paletteId == "liquid_glass" -> if (darkTheme) LiquidGlassDarkColorScheme else LiquidGlassLightColorScheme
        paletteId == "monet" && Build.VERSION.SDK_INT >= Build.VERSION_CODES.S -> {
            if (darkTheme) dynamicDarkColorScheme(context) else dynamicLightColorScheme(context)
        }
        paletteId == "miuix" -> if (darkTheme) MiuixDarkColorScheme else MiuixLightColorScheme
        paletteId == "pixel" || paletteId == "custom" -> {
            buildPixelColorScheme(accentId, baseId, darkTheme)
        }
        paletteId == "ocean" -> if (darkTheme) OceanDarkColorScheme else OceanLightColorScheme
        paletteId == "emerald" -> if (darkTheme) EmeraldDarkColorScheme else EmeraldLightColorScheme
        paletteId == "purple" -> if (darkTheme) PurpleDarkColorScheme else PurpleLightColorScheme
        paletteId == "amber" -> if (darkTheme) AmberDarkColorScheme else AmberLightColorScheme
        paletteId == "rose" -> if (darkTheme) RoseDarkColorScheme else RoseLightColorScheme
        else -> buildPixelColorScheme(accentId, baseId, darkTheme)
    }

    val view = LocalView.current
    if (!view.isInEditMode) {
        SideEffect {
            val window = (view.context as? Activity)?.window
            if (window != null) {
                window.statusBarColor = colorScheme.surface.toArgb()
                window.navigationBarColor = colorScheme.surfaceContainer.toArgb()
                val controller = WindowCompat.getInsetsController(window, view)
                controller.isAppearanceLightStatusBars = !darkTheme
                controller.isAppearanceLightNavigationBars = !darkTheme
            }
        }
    }

    MaterialTheme(
        colorScheme = colorScheme,
        typography = AimiliTypography,
        shapes = AimiliShapes,
        content = content
    )
}
