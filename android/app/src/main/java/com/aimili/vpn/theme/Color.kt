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
    AppThemePalette("monet", "✨ 壁纸莫奈动态取色", "从系统壁纸提取动态色彩 (Android 12+)", Color(0xFF6750A4)),
    AppThemePalette("teal", "🌿 默认青翠 (Teal)", "Aimili 标志性青碧护眼色系 (默认)", Color(0xFF00696E)),
    AppThemePalette("ocean", "🌊 极客苍蓝 (Ocean)", "沉稳专业深海科技纯蓝色系", Color(0xFF0061A4)),
    AppThemePalette("emerald", "🌲 原野翡翠 (Emerald)", "自然原野翡翠绿色系", Color(0xFF1B6D36)),
    AppThemePalette("purple", "🔮 暮光紫罗 (Amethyst)", "优雅高贵紫罗兰色系", Color(0xFF7A4B95)),
    AppThemePalette("amber", "🌅 晚霞琥珀 (Amber)", "温暖夕阳琥珀橙色系", Color(0xFF9C4300)),
    AppThemePalette("rose", "🌸 暗夜樱粉 (Rose)", "现代活力粉樱浪漫色系", Color(0xFF984061))
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
val TealLightSurface = Color(0xFFF4FBFB)
val TealLightSurfaceContainerLow = Color(0xFFEEF5F5)
val TealLightSurfaceContainer = Color(0xFFE8EFEF)
val TealLightSurfaceContainerHigh = Color(0xFFE2EAEA)
val TealLightSurfaceContainerHighest = Color(0xFFDDE4E4)
val TealLightOnSurface = Color(0xFF161D1D)
val TealLightOnSurfaceVariant = Color(0xFF3F4948)
val TealLightOutline = Color(0xFF6F7979)
val TealLightOutlineVariant = Color(0xFFBEC8C8)
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
)

// ==========================================
// 2. Ocean Blue (极客苍蓝)
// ==========================================
val OceanLightColorScheme: ColorScheme = lightColorScheme(
    primary = Color(0xFF0061A4),
    onPrimary = Color(0xFFFFFFFF),
    primaryContainer = Color(0xFFD1E4FF),
    onPrimaryContainer = Color(0xFF001D36),
    secondary = Color(0xFF535F70),
    onSecondary = Color(0xFFFFFFFF),
    secondaryContainer = Color(0xFFD7E3F7),
    onSecondaryContainer = Color(0xFF101C2B),
    tertiary = Color(0xFF6B5778),
    onTertiary = Color(0xFFFFFFFF),
    tertiaryContainer = Color(0xFFF2DAFF),
    onTertiaryContainer = Color(0xFF251432),
    surface = Color(0xFFFDFBFF),
    surfaceContainerLow = Color(0xFFF2F4FA),
    surfaceContainer = Color(0xFFECEEF4),
    surfaceContainerHigh = Color(0xFFE6E8EE),
    surfaceContainerHighest = Color(0xFFE0E2E8),
    onSurface = Color(0xFF1A1C1E),
    onSurfaceVariant = Color(0xFF43474E),
    outline = Color(0xFF73777F),
    outlineVariant = Color(0xFFC3C7CF)
)

val OceanDarkColorScheme: ColorScheme = darkColorScheme(
    primary = Color(0xFF9ECAFF),
    onPrimary = Color(0xFF003258),
    primaryContainer = Color(0xFF00497D),
    onPrimaryContainer = Color(0xFFD1E4FF),
    secondary = Color(0xFFBBC7DB),
    onSecondary = Color(0xFF253140),
    secondaryContainer = Color(0xFF3B4858),
    onSecondaryContainer = Color(0xFFD7E3F7),
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
    outlineVariant = Color(0xFF43474E)
)

// ==========================================
// 3. Emerald Green (原野翡翠)
// ==========================================
val EmeraldLightColorScheme: ColorScheme = lightColorScheme(
    primary = Color(0xFF1B6D36),
    onPrimary = Color(0xFFFFFFFF),
    primaryContainer = Color(0xFFA5F4AF),
    onPrimaryContainer = Color(0xFF00210A),
    secondary = Color(0xFF516351),
    onSecondary = Color(0xFFFFFFFF),
    secondaryContainer = Color(0xFFD4E8D2),
    onSecondaryContainer = Color(0xFF0F1F11),
    tertiary = Color(0xFF39656D),
    onTertiary = Color(0xFFFFFFFF),
    tertiaryContainer = Color(0xFFBCEBF4),
    onTertiaryContainer = Color(0xFF001F25),
    surface = Color(0xFFFCFDF6),
    surfaceContainerLow = Color(0xFFF1F5EC),
    surfaceContainer = Color(0xFFEBEFE6),
    surfaceContainerHigh = Color(0xFFE5E9E0),
    surfaceContainerHighest = Color(0xFFE0E4DB),
    onSurface = Color(0xFF1A1C19),
    onSurfaceVariant = Color(0xFF424940),
    outline = Color(0xFF72796F),
    outlineVariant = Color(0xFFC2C9BD)
)

val EmeraldDarkColorScheme: ColorScheme = darkColorScheme(
    primary = Color(0xFF8AD795),
    onPrimary = Color(0xFF003916),
    primaryContainer = Color(0xFF005223),
    onPrimaryContainer = Color(0xFFA5F4AF),
    secondary = Color(0xFFB8CCB5),
    onSecondary = Color(0xFF243425),
    secondaryContainer = Color(0xFF3A4B3A),
    onSecondaryContainer = Color(0xFFD4E8D2),
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
    outlineVariant = Color(0xFF424940)
)

// ==========================================
// 4. Amethyst Purple (暮光紫罗)
// ==========================================
val PurpleLightColorScheme: ColorScheme = lightColorScheme(
    primary = Color(0xFF7A4B95),
    onPrimary = Color(0xFFFFFFFF),
    primaryContainer = Color(0xFFF6D8FF),
    onPrimaryContainer = Color(0xFF30034B),
    secondary = Color(0xFF68586E),
    onSecondary = Color(0xFFFFFFFF),
    secondaryContainer = Color(0xFFF1DBF5),
    onSecondaryContainer = Color(0xFF231629),
    tertiary = Color(0xFF815250),
    onTertiary = Color(0xFFFFFFFF),
    tertiaryContainer = Color(0xFFFFDAD8),
    onTertiaryContainer = Color(0xFF331111),
    surface = Color(0xFFFCF7FD),
    surfaceContainerLow = Color(0xFFF4EEF6),
    surfaceContainer = Color(0xFFEEE8F0),
    surfaceContainerHigh = Color(0xFFE8E2EA),
    surfaceContainerHighest = Color(0xFFE2DCE4),
    onSurface = Color(0xFF1E1A20),
    onSurfaceVariant = Color(0xFF4B454D),
    outline = Color(0xFF7C757F),
    outlineVariant = Color(0xFFCDC4CE)
)

val PurpleDarkColorScheme: ColorScheme = darkColorScheme(
    primary = Color(0xFFE5B5FF),
    onPrimary = Color(0xFF481A63),
    primaryContainer = Color(0xFF60327B),
    onPrimaryContainer = Color(0xFFF6D8FF),
    secondary = Color(0xFFD4C0D8),
    onSecondary = Color(0xFF382B3E),
    secondaryContainer = Color(0xFF504155),
    onSecondaryContainer = Color(0xFFF1DBF5),
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
    outlineVariant = Color(0xFF4B454D)
)

// ==========================================
// 5. Sunset Amber (晚霞琥珀)
// ==========================================
val AmberLightColorScheme: ColorScheme = lightColorScheme(
    primary = Color(0xFF9C4300),
    onPrimary = Color(0xFFFFFFFF),
    primaryContainer = Color(0xFFFFDBCF),
    onPrimaryContainer = Color(0xFF341100),
    secondary = Color(0xFF77574B),
    onSecondary = Color(0xFFFFFFFF),
    secondaryContainer = Color(0xFFFFDBCE),
    onSecondaryContainer = Color(0xFF2C160D),
    tertiary = Color(0xFF695E2F),
    onTertiary = Color(0xFFFFFFFF),
    tertiaryContainer = Color(0xFFF2E2A7),
    onTertiaryContainer = Color(0xFF211B00),
    surface = Color(0xFFFCF8F6),
    surfaceContainerLow = Color(0xFFF6EFEA),
    surfaceContainer = Color(0xFFF0E9E4),
    surfaceContainerHigh = Color(0xFFEAE3DE),
    surfaceContainerHighest = Color(0xFFE4DDD8),
    onSurface = Color(0xFF201A17),
    onSurfaceVariant = Color(0xFF53433C),
    outline = Color(0xFF85736B),
    outlineVariant = Color(0xFFD8C2B8)
)

val AmberDarkColorScheme: ColorScheme = darkColorScheme(
    primary = Color(0xFFFFB59B),
    onPrimary = Color(0xFF552000),
    primaryContainer = Color(0xFF783100),
    onPrimaryContainer = Color(0xFFFFDBCF),
    secondary = Color(0xFFE7BEAF),
    onSecondary = Color(0xFF442A20),
    secondaryContainer = Color(0xFF5D4035),
    onSecondaryContainer = Color(0xFFFFDBCE),
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
    outlineVariant = Color(0xFF53433C)
)

// ==========================================
// 6. Sakura Rose (暗夜樱粉)
// ==========================================
val RoseLightColorScheme: ColorScheme = lightColorScheme(
    primary = Color(0xFF984061),
    onPrimary = Color(0xFFFFFFFF),
    primaryContainer = Color(0xFFFFD9E2),
    onPrimaryContainer = Color(0xFF3E001D),
    secondary = Color(0xFF74565F),
    onSecondary = Color(0xFFFFFFFF),
    secondaryContainer = Color(0xFFFFD9E2),
    onSecondaryContainer = Color(0xFF2B151C),
    tertiary = Color(0xFF7C5635),
    onTertiary = Color(0xFFFFFFFF),
    tertiaryContainer = Color(0xFFFFDCC1),
    onTertiaryContainer = Color(0xFF2E1500),
    surface = Color(0xFFFAECEF),
    surfaceContainerLow = Color(0xFFF3E2E6),
    surfaceContainer = Color(0xFFEDDCE1),
    surfaceContainerHigh = Color(0xFFE7D6DB),
    surfaceContainerHighest = Color(0xFFE1D0D5),
    onSurface = Color(0xFF201A1B),
    onSurfaceVariant = Color(0xFF514347),
    outline = Color(0xFF847377),
    outlineVariant = Color(0xFFD6C2C6)
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
