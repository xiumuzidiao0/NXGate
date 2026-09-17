package com.nxgate.app.ui.components

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.spring
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.nxgate.app.theme.liquidGlass
import com.nxgate.app.ui.navigation.AppNavDestination

/**
 * Liquid Glass Floating Bottom Navigation Bar.
 * Inspired by AndroidLiquidGlass (Backdrop) and VisionOS floating glass panels:
 * - Suspended capsule geometry with horizontal padding
 * - Translucent frosty glass substrate
 * - 135 deg prismatic specular highlight rim
 * - Animated liquid indicator capsule for active tab
 */
@Composable
fun LiquidGlassBottomBar(
    currentDestination: AppNavDestination,
    onDestinationSelected: (AppNavDestination) -> Unit,
    modifier: Modifier = Modifier
) {
    val isDark = isSystemInDarkTheme()

    Box(
        modifier = modifier
            .fillMaxWidth()
            .navigationBarsPadding()
            .padding(horizontal = 20.dp, vertical = 8.dp),
        contentAlignment = Alignment.Center
    ) {
        // Floating glass capsule container
        Surface(
            modifier = Modifier
                .fillMaxWidth()
                .height(64.dp)
                .liquidGlass(
                    shape = RoundedCornerShape(32.dp),
                    isDark = isDark,
                    elevation = 8.dp
                ),
            shape = RoundedCornerShape(32.dp),
            color = Color.Transparent
        ) {
            Row(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(horizontal = 6.dp, vertical = 4.dp),
                horizontalArrangement = Arrangement.SpaceAround,
                verticalAlignment = Alignment.CenterVertically
            ) {
                AppNavDestination.entries.forEach { destination ->
                    val isSelected = currentDestination == destination
                    LiquidTabItem(
                        destination = destination,
                        isSelected = isSelected,
                        isDark = isDark,
                        onClick = { onDestinationSelected(destination) },
                        modifier = Modifier.weight(1f)
                    )
                }
            }
        }
    }
}

@Composable
private fun LiquidTabItem(
    destination: AppNavDestination,
    isSelected: Boolean,
    isDark: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    val interactionSource = remember { MutableInteractionSource() }

    // Spring animations for scale and color
    val iconScale by animateFloatAsState(
        targetValue = if (isSelected) 1.08f else 1.0f,
        animationSpec = spring(dampingRatio = 0.75f, stiffness = Spring.StiffnessMedium),
        label = "TabScale"
    )

    // 适当提亮未选中状态在透光液态玻璃上的对比度与可视度
    val unselectedContentColor = if (isDark) {
        Color(0xFFF1F5F9).copy(alpha = 0.82f) // 墨曜透光基底上提亮为晶澈银白，避免被背景杂色吞噬
    } else {
        Color(0xFF1E293B).copy(alpha = 0.88f) // 浅色透光冰晶上保持深色高对比度
    }

    val contentColor by animateColorAsState(
        targetValue = if (isSelected) MaterialTheme.colorScheme.primary else unselectedContentColor,
        label = "TabContentColor"
    )

    val pillAlpha by animateFloatAsState(
        targetValue = if (isSelected) 1.0f else 0f,
        animationSpec = spring(dampingRatio = 0.85f, stiffness = Spring.StiffnessMedium),
        label = "PillAlpha"
    )

    Box(
        modifier = modifier
            .fillMaxHeight()
            .clip(RoundedCornerShape(24.dp))
            .clickable(
                interactionSource = interactionSource,
                indication = null,
                onClick = onClick
            ),
        contentAlignment = Alignment.Center
    ) {
        // Active Liquid Pill Indicator
        if (pillAlpha > 0.05f) {
            Surface(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(horizontal = 4.dp, vertical = 2.dp),
                shape = RoundedCornerShape(24.dp),
                color = MaterialTheme.colorScheme.primary.copy(alpha = if (isDark) 0.20f * pillAlpha else 0.12f * pillAlpha),
                border = BorderStroke(
                    1.dp,
                    MaterialTheme.colorScheme.primary.copy(alpha = if (isDark) 0.35f * pillAlpha else 0.22f * pillAlpha)
                )
            ) {}
        }

        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
            modifier = Modifier.scale(iconScale)
        ) {
            Icon(
                imageVector = destination.icon,
                contentDescription = destination.title,
                tint = contentColor,
                modifier = Modifier.size(22.dp)
            )
            Spacer(Modifier.height(2.dp))
            Text(
                text = destination.title,
                style = MaterialTheme.typography.labelSmall,
                fontWeight = if (isSelected) FontWeight.Bold else FontWeight.SemiBold,
                color = contentColor
            )
        }
    }
}

/**
 * Liquid Glass Adaptive Side Navigation Rail (for Tablet Landscape).
 */
@Composable
fun LiquidGlassNavigationRail(
    currentDestination: AppNavDestination,
    onDestinationSelected: (AppNavDestination) -> Unit,
    modifier: Modifier = Modifier
) {
    val isDark = isSystemInDarkTheme()

    Box(
        modifier = modifier
            .fillMaxHeight()
            .padding(start = 16.dp, end = 8.dp, top = 16.dp, bottom = 16.dp),
        contentAlignment = Alignment.CenterStart
    ) {
        Surface(
            modifier = Modifier
                .width(76.dp)
                .fillMaxHeight()
                .liquidGlass(
                    shape = RoundedCornerShape(36.dp),
                    isDark = isDark,
                    elevation = 8.dp
                ),
            shape = RoundedCornerShape(36.dp),
            color = Color.Transparent
        ) {
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(vertical = 16.dp, horizontal = 6.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.SpaceEvenly
            ) {
                AppNavDestination.entries.forEach { destination ->
                    val isSelected = currentDestination == destination
                    LiquidRailItem(
                        destination = destination,
                        isSelected = isSelected,
                        isDark = isDark,
                        onClick = { onDestinationSelected(destination) }
                    )
                }
            }
        }
    }
}

@Composable
private fun LiquidRailItem(
    destination: AppNavDestination,
    isSelected: Boolean,
    isDark: Boolean,
    onClick: () -> Unit
) {
    val interactionSource = remember { MutableInteractionSource() }

    val unselectedContentColor = if (isDark) {
        Color(0xFFF1F5F9).copy(alpha = 0.82f)
    } else {
        Color(0xFF1E293B).copy(alpha = 0.88f)
    }

    val contentColor by animateColorAsState(
        targetValue = if (isSelected) MaterialTheme.colorScheme.primary else unselectedContentColor,
        label = "RailContentColor"
    )

    Box(
        modifier = Modifier
            .size(width = 64.dp, height = 56.dp)
            .clip(RoundedCornerShape(20.dp))
            .clickable(
                interactionSource = interactionSource,
                indication = null,
                onClick = onClick
            ),
        contentAlignment = Alignment.Center
    ) {
        if (isSelected) {
            Surface(
                modifier = Modifier.fillMaxSize(),
                shape = RoundedCornerShape(20.dp),
                color = MaterialTheme.colorScheme.primary.copy(alpha = if (isDark) 0.20f else 0.12f),
                border = BorderStroke(
                    1.dp,
                    MaterialTheme.colorScheme.primary.copy(alpha = if (isDark) 0.35f else 0.22f)
                )
            ) {}
        }

        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center
        ) {
            Icon(
                imageVector = destination.icon,
                contentDescription = destination.title,
                tint = contentColor,
                modifier = Modifier.size(24.dp)
            )
            Spacer(Modifier.height(2.dp))
            Text(
                text = destination.title,
                style = MaterialTheme.typography.labelSmall,
                fontWeight = if (isSelected) FontWeight.Bold else FontWeight.SemiBold,
                color = contentColor
            )
        }
    }
}
