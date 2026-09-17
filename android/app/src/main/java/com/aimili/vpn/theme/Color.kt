package com.aimili.vpn.theme

import androidx.compose.material3.ColorScheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.ui.graphics.Color

data class AppThemePalette(
    val id: String,
    val name: String,
    val description: String,
    val primaryColor: Color
)

val AVAILABLE_PALETTES = listOf(
    AppThemePalette("monet", "壁纸莫奈动态取色", "从系统壁纸提取动态色彩 (Android 12+)", Color(0xFF6750A4)),
    AppThemePalette("miuix", "小米澎湃 (HyperOS · MIUIX)", "移植自 flutter_miuix 的经典超凡蓝与层级卡片", Color(0xFF3482FF)),
    AppThemePalette("teal", "默认青翠 (Teal)", "Aimili 标志性青碧护眼色系 (默认)", Color(0xFF00696E)),
    AppThemePalette("ocean", "极客苍蓝 (Ocean)", "沉稳专业深海科技纯蓝色系", Color(0xFF0061A4)),
    AppThemePalette("emerald", "原野翡翠 (Emerald)", "自然原野翡翠绿色系", Color(0xFF1B6D36)),
    AppThemePalette("purple", "暮光紫罗 (Amethyst)", "优雅高贵紫罗兰色系", Color(0xFF7A4B95)),
    AppThemePalette("amber", "晚霞琥珀 (Amber)", "温暖夕阳琥珀橙色系", Color(0xFF9C4300)),
    AppThemePalette("rose", "暗夜樱粉 (Rose)", "现代活力粉樱浪漫色系", Color(0xFF984061))
)

// ==========================================
// 1. Teal (Aimili Default)
// ==========================================
val TealLightPrimary = Color(0xFF00696E)
val TealLightOnPrimary = Color(0xFFFFFFFF)
val TealLightPrimaryContainer = Color(0xFF9CF1F6)
val TealLightOnPrimaryContainer = Color(0xFF002022)
val TealLightSecondary = Color(0xFF4D6263)
val TealLightOnSecondary = Color(0xFFFFFFFF)
val TealLightSecondaryContainer = Color(0xFFCCE8E9)
val TealLightOnSecondaryContainer = Color(0xFF051F20)
val TealLightTertiary = Color(0xFF3F608F)
val TealLightOnTertiary = Color(0xFFFFFFFF)
val TealLightTertiaryContainer = Color(0xFFD2E4FF)
val TealLightOnTertiaryContainer = Color(0xFF001C3B)
val TealLightSurface = Color(0xFFF4F6F8)
val TealLightSurfaceContainerLow = Color(0xFFFFFFFF)
val TealLightSurfaceContainer = Color(0xFFFFFFFF)
val TealLightSurfaceContainerHigh = Color(0xFFECEFF2)
val TealLightSurfaceContainerHighest = Color(0xFFFFFFFF)
val TealLightOnSurface = Color(0xFF000000)
val TealLightOnSurfaceVariant = Color(0xFF1E242B)
val TealLightOutline = Color(0xFFCAD1DC)
val TealLightOutlineVariant = Color(0xFFE2E7ED)
val TealLightInverseSurface = Color(0xFF2B3232)
val TealLightInverseOnSurface = Color(0xFFECF2F2)
val TealLightInversePrimary = Color(0xFF80D5DA)

val TealDarkPrimary = Color(0xFF83D4D8)
val TealDarkOnPrimary = Color(0xFF063639)
val TealDarkPrimaryContainer = Color(0xFF0B4F52)
val TealDarkOnPrimaryContainer = Color(0xFF9FF0F5)
val TealDarkSecondary = Color(0xFFB3CBCC)
val TealDarkOnSecondary = Color(0xFF1E3335)
val TealDarkSecondaryContainer = Color(0xFF354A4C)
val TealDarkOnSecondaryContainer = Color(0xFFCFE7E8)
val TealDarkTertiary = Color(0xFFA5C8F6)
val TealDarkOnTertiary = Color(0xFF05315E)
val TealDarkTertiaryContainer = Color(0xFF38485A)
val TealDarkOnTertiaryContainer = Color(0xFFD3E4FA)
val TealDarkSurface = Color(0xFF0D1515)
val TealDarkSurfaceContainerLow = Color(0xFF161D1D)
val TealDarkSurfaceContainer = Color(0xFF1A2121)
val TealDarkSurfaceContainerHigh = Color(0xFF252B2B)
val TealDarkSurfaceContainerHighest = Color(0xFF2F3636)
val TealDarkOnSurface = Color(0xFFDCE4E4)
val TealDarkOnSurfaceVariant = Color(0xFFB7CACB)
val TealDarkOutline = Color(0xFF819495)
val TealDarkOutlineVariant = Color(0xFF394A4B)
val TealDarkInverseSurface = Color(0xFFDCE4E4)
val TealDarkInverseOnSurface = Color(0xFF2B3232)
val TealDarkInversePrimary = Color(0xFF00696E)

val AimiliLightColorScheme: ColorScheme = lightColorScheme(
    primary = TealLightPrimary,
    onPrimary = TealLightOnPrimary,
    primaryContainer = TealLightPrimaryContainer,
    onPrimaryContainer = TealLightOnPrimaryContainer,
    secondary = TealLightSecondary,
    onSecondary = TealLightOnSecondary,
    secondaryContainer = TealLightSecondaryContainer,
    onSecondaryContainer = TealLightOnSecondaryContainer,
    tertiary = TealLightTertiary,
    onTertiary = TealLightOnTertiary,
    tertiaryContainer = TealLightTertiaryContainer,
    onTertiaryContainer = TealLightOnTertiaryContainer,
    surface = TealLightSurface,
    surfaceContainerLow = TealLightSurfaceContainerLow,
    surfaceContainer = TealLightSurfaceContainer,
    surfaceContainerHigh = TealLightSurfaceContainerHigh,
    surfaceContainerHighest = TealLightSurfaceContainerHighest,
    onSurface = TealLightOnSurface,
    onSurfaceVariant = TealLightOnSurfaceVariant,
    outline = TealLightOutline,
    outlineVariant = TealLightOutlineVariant,
    inverseSurface = TealLightInverseSurface,
    inverseOnSurface = TealLightInverseOnSurface,
    inversePrimary = TealLightInversePrimary,
    error = Color(0xFFBA1A1A),
    onError = Color.White,
    errorContainer = Color(0xFFFFDAD6),
    onErrorContainer = Color(0xFF410002)
)

val AimiliDarkColorScheme: ColorScheme = darkColorScheme(
    primary = TealDarkPrimary,
    onPrimary = TealDarkOnPrimary,
    primaryContainer = TealDarkPrimaryContainer,
    onPrimaryContainer = TealDarkOnPrimaryContainer,
    secondary = TealDarkSecondary,
    onSecondary = TealDarkOnSecondary,
    secondaryContainer = TealDarkSecondaryContainer,
    onSecondaryContainer = TealDarkOnSecondaryContainer,
    tertiary = TealDarkTertiary,
    onTertiary = TealDarkOnTertiary,
    tertiaryContainer = TealDarkTertiaryContainer,
    onTertiaryContainer = TealDarkOnTertiaryContainer,
    surface = TealDarkSurface,
    surfaceContainerLow = TealDarkSurfaceContainerLow,
    surfaceContainer = TealDarkSurfaceContainer,
    surfaceContainerHigh = TealDarkSurfaceContainerHigh,
    surfaceContainerHighest = TealDarkSurfaceContainerHighest,
    onSurface = TealDarkOnSurface,
    onSurfaceVariant = TealDarkOnSurfaceVariant,
    outline = TealDarkOutline,
    outlineVariant = TealDarkOutlineVariant,
    inverseSurface = TealDarkInverseSurface,
    inverseOnSurface = TealDarkInverseOnSurface,
    inversePrimary = TealDarkInversePrimary,
    error = Color(0xFFFFB4AB),
    onError = Color(0xFF690005),
    errorContainer = Color(0xFF93000A),
    onErrorContainer = Color(0xFFFFDAD6)
)

// ==========================================
// 2. Ocean Blue (极客苍蓝)
// ==========================================
val OceanLightColorScheme: ColorScheme = lightColorScheme(
    primary = Color(0xFF0061A4),
    onPrimary = Color(0xFFFFFFFF),
    primaryContainer = Color(0xFFD1E4FF),
    onPrimaryContainer = Color(0xFF001D36),
    secondary = Color(0xFF0061A4),
    onSecondary = Color(0xFFFFFFFF),
    secondaryContainer = Color(0xFFD1E4FF),
    onSecondaryContainer = Color(0xFF001D36),
    tertiary = Color(0xFF6B5778),
    onTertiary = Color(0xFFFFFFFF),
    tertiaryContainer = Color(0xFFF2DAFF),
    onTertiaryContainer = Color(0xFF251432),
    surface = Color(0xFFF4F6F8),
    surfaceContainerLow = Color(0xFFFFFFFF),
    surfaceContainer = Color(0xFFFFFFFF),
    surfaceContainerHigh = Color(0xFFECEFF2),
    surfaceContainerHighest = Color(0xFFFFFFFF),
    onSurface = Color(0xFF000000),
    onSurfaceVariant = Color(0xFF1A1C1E),
    outline = Color(0xFFCAD1DC),
    outlineVariant = Color(0xFFE2E7ED),
    error = Color(0xFFBA1A1A),
    onError = Color.White,
    errorContainer = Color(0xFFFFDAD6),
    onErrorContainer = Color(0xFF410002)
)

val OceanDarkColorScheme: ColorScheme = darkColorScheme(
    primary = Color(0xFF9ECAFF),
    onPrimary = Color(0xFF003258),
    primaryContainer = Color(0xFF00497D),
    onPrimaryContainer = Color(0xFFD1E4FF),
    secondary = Color(0xFF9ECAFF),
    onSecondary = Color(0xFF003258),
    secondaryContainer = Color(0xFF00497D),
    onSecondaryContainer = Color(0xFFD1E4FF),
    tertiary = Color(0xFFD6BEE4),
    onTertiary = Color(0xFF3B2948),
    tertiaryContainer = Color(0xFF523F5F),
    onTertiaryContainer = Color(0xFFF2DAFF),
    surface = Color(0xFF101418),
    surfaceContainerLow = Color(0xFF181C20),
    surfaceContainer = Color(0xFF1C2024),
    surfaceContainerHigh = Color(0xFF272B2F),
    surfaceContainerHighest = Color(0xFF32363A),
    onSurface = Color(0xFFE2E2E6),
    onSurfaceVariant = Color(0xFFC3C7CF),
    outline = Color(0xFF8D9199),
    outlineVariant = Color(0xFF43474E),
    error = Color(0xFFFFB4AB),
    onError = Color(0xFF690005),
    errorContainer = Color(0xFF93000A),
    onErrorContainer = Color(0xFFFFDAD6)
)

// ==========================================
// 3. Emerald Green (原野翡翠)
// ==========================================
val EmeraldLightColorScheme: ColorScheme = lightColorScheme(
    primary = Color(0xFF1B6D36),
    onPrimary = Color(0xFFFFFFFF),
    primaryContainer = Color(0xFFA5F4AF),
    onPrimaryContainer = Color(0xFF00210A),
    secondary = Color(0xFF1B6D36),
    onSecondary = Color(0xFFFFFFFF),
    secondaryContainer = Color(0xFFA5F4AF),
    onSecondaryContainer = Color(0xFF00210A),
    tertiary = Color(0xFF39656D),
    onTertiary = Color(0xFFFFFFFF),
    tertiaryContainer = Color(0xFFBCEBF4),
    onTertiaryContainer = Color(0xFF001F25),
    surface = Color(0xFFF4F6F8),
    surfaceContainerLow = Color(0xFFFFFFFF),
    surfaceContainer = Color(0xFFFFFFFF),
    surfaceContainerHigh = Color(0xFFECEFF2),
    surfaceContainerHighest = Color(0xFFFFFFFF),
    onSurface = Color(0xFF000000),
    onSurfaceVariant = Color(0xFF1A1D1A),
    outline = Color(0xFFCAD1DC),
    outlineVariant = Color(0xFFE2E7ED),
    error = Color(0xFFBA1A1A),
    onError = Color.White,
    errorContainer = Color(0xFFFFDAD6),
    onErrorContainer = Color(0xFF410002)
)

val EmeraldDarkColorScheme: ColorScheme = darkColorScheme(
    primary = Color(0xFF8AD795),
    onPrimary = Color(0xFF003916),
    primaryContainer = Color(0xFF005223),
    onPrimaryContainer = Color(0xFFA5F4AF),
    secondary = Color(0xFF8AD795),
    onSecondary = Color(0xFF003916),
    secondaryContainer = Color(0xFF005223),
    onSecondaryContainer = Color(0xFFA5F4AF),
    tertiary = Color(0xFFA1CED7),
    onTertiary = Color(0xFF00363E),
    tertiaryContainer = Color(0xFF1F4D55),
    onTertiaryContainer = Color(0xFFBCEBF4),
    surface = Color(0xFF101511),
    surfaceContainerLow = Color(0xFF171D18),
    surfaceContainer = Color(0xFF1B211C),
    surfaceContainerHigh = Color(0xFF262C26),
    surfaceContainerHighest = Color(0xFF313731),
    onSurface = Color(0xFFE2E3DD),
    onSurfaceVariant = Color(0xFFC2C9BD),
    outline = Color(0xFF8C9388),
    outlineVariant = Color(0xFF424940),
    error = Color(0xFFFFB4AB),
    onError = Color(0xFF690005),
    errorContainer = Color(0xFF93000A),
    onErrorContainer = Color(0xFFFFDAD6)
)

// ==========================================
// 4. Amethyst Purple (暮光紫罗)
// ==========================================
val PurpleLightColorScheme: ColorScheme = lightColorScheme(
    primary = Color(0xFF7A4B95),
    onPrimary = Color(0xFFFFFFFF),
    primaryContainer = Color(0xFFF6D8FF),
    onPrimaryContainer = Color(0xFF30034B),
    secondary = Color(0xFF7A4B95),
    onSecondary = Color(0xFFFFFFFF),
    secondaryContainer = Color(0xFFF6D8FF),
    onSecondaryContainer = Color(0xFF30034B),
    tertiary = Color(0xFF815250),
    onTertiary = Color(0xFFFFFFFF),
    tertiaryContainer = Color(0xFFFFDAD8),
    onTertiaryContainer = Color(0xFF331111),
    surface = Color(0xFFF4F6F8),
    surfaceContainerLow = Color(0xFFFFFFFF),
    surfaceContainer = Color(0xFFFFFFFF),
    surfaceContainerHigh = Color(0xFFECEFF2),
    surfaceContainerHighest = Color(0xFFFFFFFF),
    onSurface = Color(0xFF000000),
    onSurfaceVariant = Color(0xFF1E1A20),
    outline = Color(0xFFCAD1DC),
    outlineVariant = Color(0xFFE2E7ED),
    error = Color(0xFFBA1A1A),
    onError = Color.White,
    errorContainer = Color(0xFFFFDAD6),
    onErrorContainer = Color(0xFF410002)
)

val PurpleDarkColorScheme: ColorScheme = darkColorScheme(
    primary = Color(0xFFE5B5FF),
    onPrimary = Color(0xFF481A63),
    primaryContainer = Color(0xFF60327B),
    onPrimaryContainer = Color(0xFFF6D8FF),
    secondary = Color(0xFFE5B5FF),
    onSecondary = Color(0xFF481A63),
    secondaryContainer = Color(0xFF60327B),
    onSecondaryContainer = Color(0xFFF6D8FF),
    tertiary = Color(0xFFF5B7B5),
    onTertiary = Color(0xFF4C2524),
    tertiaryContainer = Color(0xFF663B39),
    onTertiaryContainer = Color(0xFFFFDAD8),
    surface = Color(0xFF161218),
    surfaceContainerLow = Color(0xFF1E1A20),
    surfaceContainer = Color(0xFF221E24),
    surfaceContainerHigh = Color(0xFF2D292F),
    surfaceContainerHighest = Color(0xFF38343A),
    onSurface = Color(0xFFE7E0E8),
    onSurfaceVariant = Color(0xFFCDC4CE),
    outline = Color(0xFF968E98),
    outlineVariant = Color(0xFF4B454D),
    error = Color(0xFFFFB4AB),
    onError = Color(0xFF690005),
    errorContainer = Color(0xFF93000A),
    onErrorContainer = Color(0xFFFFDAD6)
)

// ==========================================
// 5. Sunset Amber (晚霞琥珀)
// ==========================================
val AmberLightColorScheme: ColorScheme = lightColorScheme(
    primary = Color(0xFF9C4300),
    onPrimary = Color(0xFFFFFFFF),
    primaryContainer = Color(0xFFFFDBCF),
    onPrimaryContainer = Color(0xFF341100),
    secondary = Color(0xFF9C4300),
    onSecondary = Color(0xFFFFFFFF),
    secondaryContainer = Color(0xFFFFDBCF),
    onSecondaryContainer = Color(0xFF341100),
    tertiary = Color(0xFF695E2F),
    onTertiary = Color(0xFFFFFFFF),
    tertiaryContainer = Color(0xFFF2E2A7),
    onTertiaryContainer = Color(0xFF211B00),
    surface = Color(0xFFF4F6F8),
    surfaceContainerLow = Color(0xFFFFFFFF),
    surfaceContainer = Color(0xFFFFFFFF),
    surfaceContainerHigh = Color(0xFFECEFF2),
    surfaceContainerHighest = Color(0xFFFFFFFF),
    onSurface = Color(0xFF000000),
    onSurfaceVariant = Color(0xFF201A17),
    outline = Color(0xFFCAD1DC),
    outlineVariant = Color(0xFFE2E7ED),
    error = Color(0xFFBA1A1A),
    onError = Color.White,
    errorContainer = Color(0xFFFFDAD6),
    onErrorContainer = Color(0xFF410002)
)

val AmberDarkColorScheme: ColorScheme = darkColorScheme(
    primary = Color(0xFFFFB59B),
    onPrimary = Color(0xFF552000),
    primaryContainer = Color(0xFF783100),
    onPrimaryContainer = Color(0xFFFFDBCF),
    secondary = Color(0xFFFFB59B),
    onSecondary = Color(0xFF552000),
    secondaryContainer = Color(0xFF783100),
    onSecondaryContainer = Color(0xFFFFDBCF),
    tertiary = Color(0xFFD6C68E),
    onTertiary = Color(0xFF393005),
    tertiaryContainer = Color(0xFF51461A),
    onTertiaryContainer = Color(0xFFF2E2A7),
    surface = Color(0xFF181210),
    surfaceContainerLow = Color(0xFF211A17),
    surfaceContainer = Color(0xFF251E1B),
    surfaceContainerHigh = Color(0xFF302925),
    surfaceContainerHighest = Color(0xFF3B3330),
    onSurface = Color(0xFFEDE0DB),
    onSurfaceVariant = Color(0xFFD8C2B8),
    outline = Color(0xFFA08C84),
    outlineVariant = Color(0xFF53433C),
    error = Color(0xFFFFB4AB),
    onError = Color(0xFF690005),
    errorContainer = Color(0xFF93000A),
    onErrorContainer = Color(0xFFFFDAD6)
)

// ==========================================
// 6. Sakura Rose (暗夜樱粉)
// ==========================================
val RoseLightColorScheme: ColorScheme = lightColorScheme(
    primary = Color(0xFF984061),
    onPrimary = Color(0xFFFFFFFF),
    primaryContainer = Color(0xFFFFD9E2),
    onPrimaryContainer = Color(0xFF3E001D),
    secondary = Color(0xFF984061),
    onSecondary = Color(0xFFFFFFFF),
    secondaryContainer = Color(0xFFFFD9E2),
    onSecondaryContainer = Color(0xFF3E001D),
    tertiary = Color(0xFF7C5635),
    onTertiary = Color(0xFFFFFFFF),
    tertiaryContainer = Color(0xFFFFDCC1),
    onTertiaryContainer = Color(0xFF2E1500),
    surface = Color(0xFFF4F6F8),
    surfaceContainerLow = Color(0xFFFFFFFF),
    surfaceContainer = Color(0xFFFFFFFF),
    surfaceContainerHigh = Color(0xFFECEFF2),
    surfaceContainerHighest = Color(0xFFFFFFFF),
    onSurface = Color(0xFF000000),
    onSurfaceVariant = Color(0xFF201A1B),
    outline = Color(0xFFCAD1DC),
    outlineVariant = Color(0xFFE2E7ED),
    error = Color(0xFFBA1A1A),
    onError = Color.White,
    errorContainer = Color(0xFFFFDAD6),
    onErrorContainer = Color(0xFF410002)
)

val RoseDarkColorScheme: ColorScheme = darkColorScheme(
    primary = Color(0xFFFFB0C8),
    onPrimary = Color(0xFF5D1133),
    primaryContainer = Color(0xFF7B2949),
    onPrimaryContainer = Color(0xFFFFD9E2),
    secondary = Color(0xFFE3BDC6),
    onSecondary = Color(0xFF422931),
    secondaryContainer = Color(0xFF5A3F47),
    onSecondaryContainer = Color(0xFFFFD9E2),
    tertiary = Color(0xFFEFBD94),
    onTertiary = Color(0xFF472A0C),
    tertiaryContainer = Color(0xFF613F20),
    onTertiaryContainer = Color(0xFFFFDCC1),
    surface = Color(0xFF161113),
    surfaceContainerLow = Color(0xFF1E181A),
    surfaceContainer = Color(0xFF221C1E),
    surfaceContainerHigh = Color(0xFF2D2628),
    surfaceContainerHighest = Color(0xFF383133),
    onSurface = Color(0xFFECE0E1),
    onSurfaceVariant = Color(0xFFD6C2C6),
    outline = Color(0xFF9E8C90),
    outlineVariant = Color(0xFF514347)
)

// ==========================================
// 7. 小米澎湃 (HyperOS · MIUIX 风格)
// 源自 /home/xmzd/flutter_miuix 的 Colors.kt，与 HyperOS 规范一致
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
// 8. Google Pixel 桌面标准：强调色与基准底色解耦体系
// ==========================================
data class AccentColorOption(
    val id: String,
    val name: String,
    val lightPrimary: Color,
    val darkPrimary: Color,
    val lightContainer: Color,
    val darkContainer: Color,
    val lightOnContainer: Color,
    val darkOnContainer: Color
)

data class BaseToneOption(
    val id: String,
    val name: String,
    val description: String,
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
)

val ACCENT_OPTIONS = listOf(
    AccentColorOption("teal", "青翠绿野", Color(0xFF00696E), Color(0xFF80D5DA), Color(0xFF9CF1F6), Color(0xFF0B4F52), Color(0xFF002022), Color(0xFF9CF1F6)),
    AccentColorOption("bay_blue", "晴空海蓝", Color(0xFF1A73E8), Color(0xFF8AB4F8), Color(0xFFD3E3FD), Color(0xFF0842A0), Color(0xFF041E49), Color(0xFFD3E3FD)),
    AccentColorOption("miuix_blue", "澎湃超凡", Color(0xFF3482FF), Color(0xFF277AF7), Color(0xFFEAF2FF), Color(0xFF2B3B54), Color(0xFF001D4D), Color(0xFFDCE8FF)),
    AccentColorOption("mint", "薄荷青竹", Color(0xFF00875A), Color(0xFF48D08D), Color(0xFFCEF5E1), Color(0xFF005236), Color(0xFF00210B), Color(0xFFCEF5E1)),
    AccentColorOption("coral", "珊瑚暖橙", Color(0xFFE64A19), Color(0xFFFF8A65), Color(0xFFFFDBCF), Color(0xFF782508), Color(0xFF380D00), Color(0xFFFFDBCF)),
    AccentColorOption("iris", "紫鸢霓裳", Color(0xFF7C4DFF), Color(0xFFB388FF), Color(0xFFEDE7F6), Color(0xFF4A148C), Color(0xFF21005D), Color(0xFFEDE7F6)),
    AccentColorOption("amber", "琥珀暖阳", Color(0xFFD84315), Color(0xFFFFB74D), Color(0xFFFFF3E0), Color(0xFF6E2805), Color(0xFF330E00), Color(0xFFFFF3E0)),
    AccentColorOption("rose", "暗夜樱粉", Color(0xFFC2185B), Color(0xFFF48FB1), Color(0xFFFCE4EC), Color(0xFF68002D), Color(0xFF3B0018), Color(0xFFFCE4EC))
)

val BASE_TONE_OPTIONS = listOf(
    BaseToneOption(
        id = "neutral",
        name = "中性协调",
        description = "Google Pixel 标准中性底色",
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
        description = "科技纯净冷调白灰与玄岩",
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
        description = "温润柔和暖沙与米白质感",
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
        description = "高反差通透纯黑 (OLED 专属)",
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
