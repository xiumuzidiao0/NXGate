package com.nxgate.app.theme

import androidx.compose.material3.ColorScheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.ui.graphics.Color

// ==========================================
// 小米澎湃 (HyperOS · MIUIX 风格)
// 与 HyperOS 设计规范一致
// ==========================================
val MiuixLightPrimary = Color(0xFF3482FF) // HyperOS 超凡蔚蓝
val MiuixLightOnPrimary = Color(0xFFFFFFFF)
val MiuixLightPrimaryContainer = Color(0xFFEAF2FF) // HyperOS TertiaryContainer
val MiuixLightOnPrimaryContainer = Color(0xFF3482FF)
val MiuixLightSecondary = Color(0xFF4B5C78)
val MiuixLightOnSecondary = Color(0xFFFFFFFF)
val MiuixLightSecondaryContainer = Color(0xFFF0F0F0) // HyperOS SecondaryVariant
val MiuixLightOnSecondaryContainer = Color(0xFF303030)
val MiuixLightTertiary = Color(0xFF2E6DD8)
val MiuixLightOnTertiary = Color(0xFFFFFFFF)
val MiuixLightTertiaryContainer = Color(0xFFDCE8FF)
val MiuixLightOnTertiaryContainer = Color(0xFF002255)
val MiuixLightSurface = Color(0xFFF4F5F7) // HyperOS 极简浅灰底色
val MiuixLightSurfaceContainerLow = Color(0xFFFFFFFF)
val MiuixLightSurfaceContainer = Color(0xFFFFFFFF) // 纯白高光卡片 (HyperOS Card)
val MiuixLightSurfaceContainerHigh = Color(0xFFECEFF2)
val MiuixLightSurfaceContainerHighest = Color(0xFFFFFFFF) // 纯白高光卡片
val MiuixLightOnSurface = Color(0xFF000000)
val MiuixLightOnSurfaceVariant = Color(0xFF1F1F1F)
val MiuixLightOutline = Color(0xFFCAD1DC)
val MiuixLightOutlineVariant = Color(0xFFE2E7ED)
val MiuixLightInverseSurface = Color(0xFF242424)
val MiuixLightInverseOnSurface = Color(0xFFF2F2F2)
val MiuixLightInversePrimary = Color(0xFF277AF7)
val MiuixLightError = Color(0xFFE94634)
val MiuixLightOnError = Color(0xFFFFFFFF)
val MiuixLightErrorContainer = Color(0xFFFDF6F4)
val MiuixLightOnErrorContainer = Color(0xFF410002)

val MiuixDarkPrimary = Color(0xFF277AF7) // HyperOS 暗夜蔚蓝
val MiuixDarkOnPrimary = Color(0xFFFFFFFF)
val MiuixDarkPrimaryContainer = Color(0xFF2B3B54) // HyperOS 暗夜 TertiaryContainer
val MiuixDarkOnPrimaryContainer = Color(0xFF76A9FF)
val MiuixDarkSecondary = Color(0xFF8B9CB8)
val MiuixDarkOnSecondary = Color(0xFF141E2D)
val MiuixDarkSecondaryContainer = Color(0xFF383838) // HyperOS 暗夜 SecondaryVariant
val MiuixDarkOnSecondaryContainer = Color(0xFFD9D9D9)
val MiuixDarkTertiary = Color(0xFF85B1FF)
val MiuixDarkOnTertiary = Color(0xFF00296A)
val MiuixDarkTertiaryContainer = Color(0xFF1D3E78)
val MiuixDarkOnTertiaryContainer = Color(0xFFD6E3FF)
val MiuixDarkSurface = Color(0xFF000000) // 纯粹深邃黑 (OLED 友好)
val MiuixDarkSurfaceContainerLow = Color(0xFF181818)
val MiuixDarkSurfaceContainer = Color(0xFF242424) // HyperOS 标志性 242424 悬浮卡片
val MiuixDarkSurfaceContainerHigh = Color(0xFF2D2D2D)
val MiuixDarkSurfaceContainerHighest = Color(0xFF353535)
val MiuixDarkOnSurface = Color(0xFFF2F2F2)
val MiuixDarkOnSurfaceVariant = Color(0xFFC8C8CC)
val MiuixDarkOutline = Color(0xFF404040)
val MiuixDarkOutlineVariant = Color(0xFF393939)
val MiuixDarkInverseSurface = Color(0xFFF7F7F7)
val MiuixDarkInverseOnSurface = Color(0xFF000000)
val MiuixDarkInversePrimary = Color(0xFF3482FF)
val MiuixDarkError = Color(0xFFF12522)
val MiuixDarkOnError = Color(0xFFFFFFFF)
val MiuixDarkErrorContainer = Color(0xFF2E0603)
val MiuixDarkOnErrorContainer = Color(0xFFFFDAD6)

val MiuixLightColorScheme: ColorScheme = lightColorScheme(
    primary = MiuixLightPrimary,
    onPrimary = MiuixLightOnPrimary,
    primaryContainer = MiuixLightPrimaryContainer,
    onPrimaryContainer = MiuixLightOnPrimaryContainer,
    secondary = MiuixLightSecondary,
    onSecondary = MiuixLightOnSecondary,
    secondaryContainer = MiuixLightSecondaryContainer,
    onSecondaryContainer = MiuixLightOnSecondaryContainer,
    tertiary = MiuixLightTertiary,
    onTertiary = MiuixLightOnTertiary,
    tertiaryContainer = MiuixLightTertiaryContainer,
    onTertiaryContainer = MiuixLightOnTertiaryContainer,
    surface = MiuixLightSurface,
    surfaceContainerLow = MiuixLightSurfaceContainerLow,
    surfaceContainer = MiuixLightSurfaceContainer,
    surfaceContainerHigh = MiuixLightSurfaceContainerHigh,
    surfaceContainerHighest = MiuixLightSurfaceContainerHighest,
    onSurface = MiuixLightOnSurface,
    onSurfaceVariant = MiuixLightOnSurfaceVariant,
    outline = MiuixLightOutline,
    outlineVariant = MiuixLightOutlineVariant,
    inverseSurface = MiuixLightInverseSurface,
    inverseOnSurface = MiuixLightInverseOnSurface,
    inversePrimary = MiuixLightInversePrimary,
    error = MiuixLightError,
    onError = MiuixLightOnError,
    errorContainer = MiuixLightErrorContainer,
    onErrorContainer = MiuixLightOnErrorContainer
)

val MiuixDarkColorScheme: ColorScheme = darkColorScheme(
    primary = MiuixDarkPrimary,
    onPrimary = MiuixDarkOnPrimary,
    primaryContainer = MiuixDarkPrimaryContainer,
    onPrimaryContainer = MiuixDarkOnPrimaryContainer,
    secondary = MiuixDarkSecondary,
    onSecondary = MiuixDarkOnSecondary,
    secondaryContainer = MiuixDarkSecondaryContainer,
    onSecondaryContainer = MiuixDarkOnSecondaryContainer,
    tertiary = MiuixDarkTertiary,
    onTertiary = MiuixDarkOnTertiary,
    tertiaryContainer = MiuixDarkTertiaryContainer,
    onTertiaryContainer = MiuixDarkOnTertiaryContainer,
    surface = MiuixDarkSurface,
    surfaceContainerLow = MiuixDarkSurfaceContainerLow,
    surfaceContainer = MiuixDarkSurfaceContainer,
    surfaceContainerHigh = MiuixDarkSurfaceContainerHigh,
    surfaceContainerHighest = MiuixDarkSurfaceContainerHighest,
    onSurface = MiuixDarkOnSurface,
    onSurfaceVariant = MiuixDarkOnSurfaceVariant,
    outline = MiuixDarkOutline,
    outlineVariant = MiuixDarkOutlineVariant,
    inverseSurface = MiuixDarkInverseSurface,
    inverseOnSurface = MiuixDarkInverseOnSurface,
    inversePrimary = MiuixDarkInversePrimary,
    error = MiuixDarkError,
    onError = MiuixDarkOnError,
    errorContainer = MiuixDarkErrorContainer,
    onErrorContainer = MiuixDarkOnErrorContainer
)

// ==========================================
// Google Pixel 桌面标准：强调色与基准底色解耦体系
// ==========================================
data class AccentColorOption(
    val id: String,
    val name: String,
    val nameEn: String,
    val lightPrimary: Color,
    val darkPrimary: Color,
    val lightContainer: Color,
    val darkContainer: Color,
    val lightOnContainer: Color,
    val darkOnContainer: Color
) {
    fun localizedName(isEn: Boolean): String = if (isEn) nameEn else name
}

data class BaseToneOption(
    val id: String,
    val name: String,
    val nameEn: String,
    val description: String,
    val descriptionEn: String,
    val lightSurface: Color,
    val darkSurface: Color,
    val lightContainer: Color,
    val darkContainer: Color,
    val lightContainerHigh: Color,
    val darkContainerHigh: Color,
    val lightOnSurface: Color,
    val darkOnSurface: Color,
    val lightOnSurfaceVariant: Color,
    val darkOnSurfaceVariant: Color,
    val lightOutline: Color,
    val darkOutline: Color
) {
    fun localizedName(isEn: Boolean): String = if (isEn) nameEn else name
    fun localizedDescription(isEn: Boolean): String = if (isEn) descriptionEn else description
}

val ACCENT_OPTIONS = listOf(
    AccentColorOption("teal", "青翠绿野", "Emerald Teal", Color(0xFF00696E), Color(0xFF80D5DA), Color(0xFF9CF1F6), Color(0xFF0B4F52), Color(0xFF002022), Color(0xFF9CF1F6)),
    AccentColorOption("bay_blue", "晴空海蓝", "Ocean Blue", Color(0xFF1A73E8), Color(0xFF8AB4F8), Color(0xFFD3E3FD), Color(0xFF0842A0), Color(0xFF041E49), Color(0xFFD3E3FD)),
    AccentColorOption("miuix_blue", "澎湃超凡", "Hyper Blue", Color(0xFF3482FF), Color(0xFF277AF7), Color(0xFFEAF2FF), Color(0xFF2B3B54), Color(0xFF001D4D), Color(0xFFDCE8FF)),
    AccentColorOption("mint", "薄荷青竹", "Fresh Mint", Color(0xFF00875A), Color(0xFF48D08D), Color(0xFFCEF5E1), Color(0xFF005236), Color(0xFF00210B), Color(0xFFCEF5E1)),
    AccentColorOption("coral", "珊瑚暖橙", "Warm Coral", Color(0xFFE64A19), Color(0xFFFF8A65), Color(0xFFFFDBCF), Color(0xFF782508), Color(0xFF380D00), Color(0xFFFFDBCF)),
    AccentColorOption("iris", "紫鸢霓裳", "Royal Iris", Color(0xFF7C4DFF), Color(0xFFB388FF), Color(0xFFEDE7F6), Color(0xFF4A148C), Color(0xFF21005D), Color(0xFFEDE7F6)),
    AccentColorOption("amber", "琥珀暖阳", "Golden Amber", Color(0xFFD84315), Color(0xFFFFB74D), Color(0xFFFFF3E0), Color(0xFF6E2805), Color(0xFF330E00), Color(0xFFFFF3E0)),
    AccentColorOption("rose", "暗夜樱粉", "Sakura Rose", Color(0xFFC2185B), Color(0xFFF48FB1), Color(0xFFFCE4EC), Color(0xFF68002D), Color(0xFF3B0018), Color(0xFFFCE4EC))
)

val BASE_TONE_OPTIONS = listOf(
    BaseToneOption(
        id = "neutral",
        name = "中性协调",
        nameEn = "Neutral",
        description = "Google Pixel 标准中性底色",
        descriptionEn = "Standard Pixel neutral background",
        lightSurface = Color(0xFFF8F9FA),
        darkSurface = Color(0xFF121417),
        lightContainer = Color(0xFFFFFFFF),
        darkContainer = Color(0xFF1C1F23),
        lightContainerHigh = Color(0xFFE9ECEF),
        darkContainerHigh = Color(0xFF25292E),
        lightOnSurface = Color(0xFF000000),
        darkOnSurface = Color(0xFFF0F2F5),
        lightOnSurfaceVariant = Color(0xFF1A1C1E),
        darkOnSurfaceVariant = Color(0xFFC7CDD4),
        lightOutline = Color(0xFFC5CCD4),
        darkOutline = Color(0xFF3B4149)
    ),
    BaseToneOption(
        id = "slate",
        name = "极简冷灰",
        nameEn = "Cool Slate",
        description = "科技纯净冷调白灰与玄岩",
        descriptionEn = "Clean cool slate and basalt tones",
        lightSurface = Color(0xFFF0F4F8),
        darkSurface = Color(0xFF0B1015),
        lightContainer = Color(0xFFFFFFFF),
        darkContainer = Color(0xFF141B22),
        lightContainerHigh = Color(0xFFDDE6F0),
        darkContainerHigh = Color(0xFF1C242D),
        lightOnSurface = Color(0xFF000000),
        darkOnSurface = Color(0xFFE9EFF5),
        lightOnSurfaceVariant = Color(0xFF141A21),
        darkOnSurfaceVariant = Color(0xFFBCCAD8),
        lightOutline = Color(0xFFBCCAD8),
        darkOutline = Color(0xFF2E3D4D)
    ),
    BaseToneOption(
        id = "sand",
        name = "暖阳米沙",
        nameEn = "Warm Sand",
        description = "温润柔和暖沙与米白质感",
        descriptionEn = "Soft warm sand and ivory tones",
        lightSurface = Color(0xFFFAF7F2),
        darkSurface = Color(0xFF151210),
        lightContainer = Color(0xFFFFFFFF),
        darkContainer = Color(0xFF1F1B17),
        lightContainerHigh = Color(0xFFEEE5D7),
        darkContainerHigh = Color(0xFF29241F),
        lightOnSurface = Color(0xFF000000),
        darkOnSurface = Color(0xFFF4EFEA),
        lightOnSurfaceVariant = Color(0xFF1E1813),
        darkOnSurfaceVariant = Color(0xFFD6C8B8),
        lightOutline = Color(0xFFD2C5B4),
        darkOutline = Color(0xFF453D35)
    ),
    BaseToneOption(
        id = "oled",
        name = "深邃玄黑",
        nameEn = "Pure OLED",
        description = "高反差通透纯黑 (OLED 专属)",
        descriptionEn = "High-contrast true black for OLED screens",
        lightSurface = Color(0xFFF5F5F5),
        darkSurface = Color(0xFF000000),
        lightContainer = Color(0xFFFFFFFF),
        darkContainer = Color(0xFF18181A),
        lightContainerHigh = Color(0xFFE0E0E0),
        darkContainerHigh = Color(0xFF222226),
        lightOnSurface = Color(0xFF000000),
        darkOnSurface = Color(0xFFF5F5F7),
        lightOnSurfaceVariant = Color(0xFF111111),
        darkOnSurfaceVariant = Color(0xFFD5D5D9),
        lightOutline = Color(0xFFC0C0C0),
        darkOutline = Color(0xFF38383A)
    )
)

fun buildPixelColorScheme(
    accentId: String,
    baseId: String,
    darkTheme: Boolean
): ColorScheme {
    val accent = ACCENT_OPTIONS.find { it.id == accentId } ?: ACCENT_OPTIONS.first()
    val base = BASE_TONE_OPTIONS.find { it.id == baseId } ?: BASE_TONE_OPTIONS.first()

    return if (darkTheme) {
        darkColorScheme(
            primary = accent.darkPrimary,
            onPrimary = Color(0xFF001F25),
            primaryContainer = accent.darkContainer,
            onPrimaryContainer = accent.darkOnContainer,
            secondary = accent.darkPrimary,
            onSecondary = Color(0xFF001F25),
            secondaryContainer = accent.darkContainer,
            onSecondaryContainer = accent.darkOnContainer,
            tertiary = accent.darkPrimary,
            onTertiary = Color.White,
            tertiaryContainer = accent.darkContainer,
            onTertiaryContainer = accent.darkOnContainer,
            surface = base.darkSurface,
            surfaceContainerLow = base.darkSurface,
            surfaceContainer = base.darkContainer,
            surfaceContainerHigh = base.darkContainerHigh,
            surfaceContainerHighest = base.darkContainerHigh,
            onSurface = base.darkOnSurface,
            onSurfaceVariant = base.darkOnSurfaceVariant,
            outline = base.darkOutline,
            outlineVariant = base.darkContainerHigh,
            background = base.darkSurface,
            onBackground = base.darkOnSurface,
            error = Color(0xFFFFB4AB),
            onError = Color(0xFF690005),
            errorContainer = Color(0xFF93000A),
            onErrorContainer = Color(0xFFFFDAD6)
        )
    } else {
        lightColorScheme(
            primary = accent.lightPrimary,
            onPrimary = Color.White,
            primaryContainer = accent.lightContainer,
            onPrimaryContainer = accent.lightOnContainer,
            secondary = accent.lightPrimary,
            onSecondary = Color.White,
            secondaryContainer = accent.lightContainer,
            onSecondaryContainer = accent.lightOnContainer,
            tertiary = accent.lightPrimary,
            onTertiary = Color.White,
            tertiaryContainer = accent.lightContainer,
            onTertiaryContainer = accent.lightOnContainer,
            surface = base.lightSurface,
            surfaceContainerLow = Color.White,
            surfaceContainer = Color.White,
            surfaceContainerHigh = base.lightContainerHigh,
            surfaceContainerHighest = Color.White,
            onSurface = base.lightOnSurface,
            onSurfaceVariant = base.lightOnSurfaceVariant,
            outline = base.lightOutline,
            outlineVariant = Color(0xFFE2E7ED),
            background = base.lightSurface,
            onBackground = base.lightOnSurface,
            error = Color(0xFFBA1A1A),
            onError = Color.White,
            errorContainer = Color(0xFFFFDAD6),
            onErrorContainer = Color(0xFF410002)
        )
    }
}
