package com.nxgate.app.ui.screens

import android.widget.Toast
import androidx.compose.foundation.BorderStroke
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
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.Palette
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.nxgate.app.NXGateApplication
import com.nxgate.app.theme.ACCENT_OPTIONS
import com.nxgate.app.theme.BASE_TONE_OPTIONS
import com.nxgate.app.ui.components.ConnectedChipGroup
import com.nxgate.app.util.AppStringsEn
import com.nxgate.app.util.LocalAppStrings

@Composable
fun ThemeAppearanceSettingsSection(modifier: Modifier = Modifier) {
    val context = LocalContext.current
    val strings = LocalAppStrings.current
    val isEn = strings == AppStringsEn
    val serverStore = NXGateApplication.instance.serverStore
    val themeMode by serverStore.themeMode.collectAsState()
    val themePalette by serverStore.themePalette.collectAsState()
    val themeAccent by serverStore.themeAccent.collectAsState()
    val themeBase by serverStore.themeBase.collectAsState()

    Card(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(20.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainer
        ),
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant),
        elevation = CardDefaults.cardElevation(defaultElevation = 1.5.dp)
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp)
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Surface(
                    shape = CircleShape,
                    color = MaterialTheme.colorScheme.primaryContainer,
                    modifier = Modifier.size(40.dp)
                ) {
                    Box(contentAlignment = Alignment.Center) {
                        Icon(
                            imageVector = Icons.Rounded.Palette,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.primary,
                            modifier = Modifier.size(22.dp)
                        )
                    }
                }
                Spacer(Modifier.width(12.dp))
                Column {
                    Text(
                        text = strings.themeAndAppearance,
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                    Text(
                        text = if (isEn) "Decoupled accent color and base tone palette" else "参考 Google Pixel 桌面标准：强调色与基准底色解耦混色",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                }
            }

            // 1. 深浅明暗模式
            ConnectedChipGroup(
                chips = listOf(if (isEn) "System" else "跟随系统", if (isEn) "Light" else "浅色模式", if (isEn) "Dark" else "深色模式"),
                selectedIndex = when (themeMode) {
                    "light" -> 1
                    "dark" -> 2
                    else -> 0
                },
                onSelected = { idx ->
                    val newMode = when (idx) {
                        1 -> "light"
                        2 -> "dark"
                        else -> "system"
                    }
                    serverStore.setThemeMode(newMode)
                }
            )

            // 2. 主题调色架构模式切换
            val themeModeChips = listOf(if (isEn) "Pixel Mixed" else "Pixel 自由混色", if (isEn) "Monet Dynamic" else "壁纸莫奈动态", if (isEn) "HyperOS" else "小米澎湃")
            val selectedPaletteChipIdx = when (themePalette) {
                "monet" -> 1
                "miuix" -> 2
                else -> 0
            }
            ConnectedChipGroup(
                chips = themeModeChips,
                selectedIndex = selectedPaletteChipIdx,
                onSelected = { idx ->
                    when (idx) {
                        1 -> serverStore.setThemePalette("monet")
                        2 -> serverStore.setThemePalette("miuix")
                        else -> serverStore.setThemePalette("pixel")
                    }
                }
            )

            if (themePalette == "pixel" || (themePalette != "monet" && themePalette != "miuix")) {
                val currentAccent = ACCENT_OPTIONS.find { it.id == themeAccent } ?: ACCENT_OPTIONS.first()
                val currentBase = BASE_TONE_OPTIONS.find { it.id == themeBase } ?: BASE_TONE_OPTIONS.first()

                // 2.1 强调色区域 (用于主按钮、开关与高光交互)
                Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Text(
                        text = if (isEn) "Accent Color (Primary buttons & toggles)" else "强调色 (用于主按钮、开关与交互高光)",
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.primary,
                        fontWeight = FontWeight.Bold
                    )
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        ACCENT_OPTIONS.forEach { acc ->
                            val isSelected = acc.id == themeAccent
                            Surface(
                                modifier = Modifier
                                    .size(36.dp)
                                    .clip(CircleShape)
                                    .clickable {
                                        serverStore.setThemeAccent(acc.id)
                                        serverStore.setThemePalette("pixel")
                                        Toast.makeText(context, "${if (isEn) "Accent color set to " else "强调色已设为「"}${acc.localizedName(isEn)}${if (isEn) "" else "」"}", Toast.LENGTH_SHORT).show()
                                    },
                                shape = CircleShape,
                                color = acc.lightPrimary,
                                border = if (isSelected) BorderStroke(3.dp, MaterialTheme.colorScheme.onSurface) else BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant)
                            ) {
                                if (isSelected) {
                                    Box(contentAlignment = Alignment.Center) {
                                        Icon(
                                            imageVector = Icons.Rounded.Check,
                                            contentDescription = null,
                                            tint = Color.White,
                                            modifier = Modifier.size(18.dp)
                                        )
                                    }
                                }
                            }
                        }
                    }
                }

                // 2.2 基准色区域 (用于卡片底色、页面背景与边框灰阶)
                Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Text(
                        text = if (isEn) "Base Background Tone (Cards & surfaces)" else "基准底色 (用于卡片底色、页面背景与边框灰阶)",
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onSurface,
                        fontWeight = FontWeight.Bold
                    )
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        BASE_TONE_OPTIONS.forEach { baseOpt ->
                            val isSelected = baseOpt.id == themeBase
                            Surface(
                                modifier = Modifier
                                    .weight(1f)
                                    .height(42.dp)
                                    .clip(RoundedCornerShape(12.dp))
                                    .clickable {
                                        serverStore.setThemeBase(baseOpt.id)
                                        serverStore.setThemePalette("pixel")
                                        Toast.makeText(context, "${if (isEn) "Base tone set to " else "基准色已设为「"}${baseOpt.localizedName(isEn)}${if (isEn) "" else "」"}", Toast.LENGTH_SHORT).show()
                                    },
                                shape = RoundedCornerShape(12.dp),
                                color = if (isSelected) MaterialTheme.colorScheme.secondaryContainer else MaterialTheme.colorScheme.surfaceContainer,
                                border = BorderStroke(if (isSelected) 2.dp else 1.dp, if (isSelected) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outlineVariant)
                            ) {
                                Box(
                                    modifier = Modifier.fillMaxSize(),
                                    contentAlignment = Alignment.Center
                                ) {
                                    Text(
                                        text = baseOpt.localizedName(isEn),
                                        style = MaterialTheme.typography.labelMedium,
                                        fontWeight = if (isSelected) FontWeight.Bold else FontWeight.Normal,
                                        color = if (isSelected) MaterialTheme.colorScheme.onSecondaryContainer else MaterialTheme.colorScheme.onSurface
                                    )
                                }
                            }
                        }
                    }
                }

                // 2.3 Pixel 风格双色对比预览胶囊
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(14.dp),
                    color = MaterialTheme.colorScheme.surfaceContainerLow,
                    border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant)
                ) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = 14.dp, vertical = 10.dp),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Surface(
                                shape = CircleShape,
                                color = currentAccent.lightPrimary,
                                modifier = Modifier.size(16.dp)
                            ) {}
                            Spacer(Modifier.width(6.dp))
                            Text(
                                text = "${if (isEn) "Accent: " else "强调色: "}${currentAccent.localizedName(isEn)}",
                                style = MaterialTheme.typography.bodySmall,
                                fontWeight = FontWeight.SemiBold,
                                color = MaterialTheme.colorScheme.onSurface
                            )
                        }

                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Surface(
                                shape = RoundedCornerShape(4.dp),
                                color = currentBase.darkSurface,
                                modifier = Modifier.size(16.dp)
                            ) {}
                            Spacer(Modifier.width(6.dp))
                            Text(
                                text = "${if (isEn) "Base: " else "基准底色: "}${currentBase.localizedName(isEn)}",
                                style = MaterialTheme.typography.bodySmall,
                                fontWeight = FontWeight.SemiBold,
                                color = MaterialTheme.colorScheme.onSurface
                            )
                        }
                    }
                }
            } else if (themePalette == "monet") {
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(14.dp),
                    color = MaterialTheme.colorScheme.surfaceContainerLow
                ) {
                    Text(
                        text = if (isEn) "Extracting dynamic colors from system wallpaper (Monet). Base and accent tones are automatically generated." else "正在从当前系统壁纸提取动态色彩 (Monet)。基准底色与强调色均由系统壁纸自动生成。",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface,
                        modifier = Modifier.padding(14.dp)
                    )
                }
            } else {
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(14.dp),
                    color = MaterialTheme.colorScheme.surfaceContainerLow
                ) {
                    Text(
                        text = if (isEn) "Using Xiaomi HyperOS (MIUIX) theme with floating card layers and oceanic blue accents." else "正在使用小米澎湃 (HyperOS · MIUIX) 主题。基准色为纯黑/纯白双层悬浮卡片，强调色为经典超凡蔚蓝。",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface,
                        modifier = Modifier.padding(14.dp)
                    )
                }
            }
        }
    }
}
