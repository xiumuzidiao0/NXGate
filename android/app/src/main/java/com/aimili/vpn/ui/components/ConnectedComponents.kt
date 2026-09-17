package com.aimili.vpn.ui.components

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.aimili.vpn.theme.connectedButtonShape
import com.aimili.vpn.theme.connectedChipShape
import com.aimili.vpn.theme.connectedListShape

enum class ConnectedButtonStyle {
    Filled,
    Tonal,
    Outlined
}

data class ConnectedButtonItem(
    val text: String,
    val style: ConnectedButtonStyle = ConnectedButtonStyle.Tonal,
    val icon: ImageVector? = null,
    val onClick: () -> Unit
)

/**
 * M3 Expressive Connected Button Group.
 * Arranged horizontally with 3dp spacing, inner corners 8dp, outer corners capsule shape.
 * Standard M3 Medium height: 56dp.
 */
@Composable
fun ConnectedButtonGroup(
    items: List<ConnectedButtonItem>,
    modifier: Modifier = Modifier,
    height: Dp = 56.dp
) {
    Row(
        modifier = modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(3.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        val isCompact = items.size >= 4
        val contentPadding = if (isCompact) PaddingValues(horizontal = 4.dp, vertical = 0.dp) else ButtonDefaults.ContentPadding
        val textStyle = if (isCompact) MaterialTheme.typography.labelMedium else MaterialTheme.typography.labelLarge
        val iconSize = if (isCompact) 14.dp else 18.dp
        val iconSpacing = if (isCompact) 3.dp else 6.dp

        items.forEachIndexed { index, item ->
            val shape = connectedButtonShape(index, items.size)
            when (item.style) {
                ConnectedButtonStyle.Filled -> {
                    Button(
                        onClick = item.onClick,
                        modifier = Modifier
                            .weight(1f)
                            .height(height),
                        shape = shape,
                        contentPadding = contentPadding,
                        colors = ButtonDefaults.buttonColors(
                            containerColor = MaterialTheme.colorScheme.primary,
                            contentColor = MaterialTheme.colorScheme.onPrimary
                        )
                    ) {
                        if (item.icon != null) {
                            Icon(item.icon, contentDescription = null, modifier = Modifier.size(iconSize))
                            Spacer(Modifier.width(iconSpacing))
                        }
                        Text(
                            text = item.text,
                            style = textStyle,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis
                        )
                    }
                }
                ConnectedButtonStyle.Tonal -> {
                    Button(
                        onClick = item.onClick,
                        modifier = Modifier
                            .weight(1f)
                            .height(height),
                        shape = shape,
                        contentPadding = contentPadding,
                        colors = ButtonDefaults.buttonColors(
                            containerColor = MaterialTheme.colorScheme.secondaryContainer,
                            contentColor = MaterialTheme.colorScheme.onSecondaryContainer
                        )
                    ) {
                        if (item.icon != null) {
                            Icon(item.icon, contentDescription = null, modifier = Modifier.size(iconSize))
                            Spacer(Modifier.width(iconSpacing))
                        }
                        Text(
                            text = item.text,
                            style = textStyle,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis
                        )
                    }
                }
                ConnectedButtonStyle.Outlined -> {
                    OutlinedButton(
                        onClick = item.onClick,
                        modifier = Modifier
                            .weight(1f)
                            .height(height),
                        shape = shape,
                        contentPadding = contentPadding,
                        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline),
                        colors = ButtonDefaults.outlinedButtonColors(
                            contentColor = MaterialTheme.colorScheme.primary
                        )
                    ) {
                        if (item.icon != null) {
                            Icon(item.icon, contentDescription = null, modifier = Modifier.size(iconSize))
                            Spacer(Modifier.width(iconSpacing))
                        }
                        Text(
                            text = item.text,
                            style = textStyle,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis
                        )
                    }
                }
            }
        }
    }
}

/**
 * M3 Expressive Connected Chip Group.
 * Arranged with 3dp spacing, inner corners 4dp, outer corners 8dp, height 32dp.
 */
@Composable
fun ConnectedChipGroup(
    chips: List<String>,
    selectedIndex: Int,
    onSelected: (Int) -> Unit,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(3.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        chips.forEachIndexed { index, label ->
            val isSelected = index == selectedIndex
            val shape = connectedChipShape(index, chips.size)
            Surface(
                modifier = Modifier
                    .height(32.dp)
                    .clip(shape)
                    .clickable { onSelected(index) },
                shape = shape,
                color = if (isSelected) MaterialTheme.colorScheme.secondaryContainer else MaterialTheme.colorScheme.surfaceContainerHigh,
                border = if (!isSelected) BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant) else null
            ) {
                Row(
                    modifier = Modifier.padding(horizontal = 12.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.Center
                ) {
                    if (isSelected) {
                        Icon(
                            imageVector = Icons.Rounded.Check,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.onSecondaryContainer,
                            modifier = Modifier.size(14.dp)
                        )
                        Spacer(Modifier.width(4.dp))
                    }
                    Text(
                        text = label,
                        style = MaterialTheme.typography.labelMedium,
                        fontWeight = if (isSelected) FontWeight.Bold else FontWeight.Normal,
                        color = if (isSelected) MaterialTheme.colorScheme.onSecondaryContainer else MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }
        }
    }
}

/**
 * M3 Expressive Connected List Item.
 * Item height: 72dp.
 * Arranged with 3dp gap, outer corners 28dp, adjacent inner corners 8dp.
 * Left icon 24dp (inside 40dp primaryContainer circular badge if specified).
 */
@Composable
fun ConnectedListItem(
    index: Int,
    total: Int,
    headline: String,
    supportingText: String,
    leadingIcon: ImageVector,
    trailingContent: @Composable () -> Unit,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    backgroundColor: Color = MaterialTheme.colorScheme.surfaceContainerLow,
    iconContainerColor: Color = MaterialTheme.colorScheme.primaryContainer
) {
    val shape = connectedListShape(index, total)

    Surface(
        modifier = modifier
            .fillMaxWidth()
            .height(72.dp)
            .clip(shape)
            .clickable { onClick() },
        shape = shape,
        color = backgroundColor
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Left 40dp circular badge with 24dp icon
            Box(
                modifier = Modifier
                    .size(40.dp)
                    .background(iconContainerColor, CircleShape),
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    imageVector = leadingIcon,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.size(24.dp)
                )
            }

            Spacer(Modifier.width(16.dp))

            // Center texts
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.Center
            ) {
                Text(
                    text = headline,
                    style = MaterialTheme.typography.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurface,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                Text(
                    text = supportingText,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurface,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
            }

            Spacer(Modifier.width(12.dp))

            // Trailing content
            trailingContent()
        }
    }
}
