package com.nxgate.app.ui.screens.matrix

import com.nxgate.app.util.groupDescription

import com.nxgate.app.util.localized

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.Block
import androidx.compose.material.icons.rounded.Delete
import androidx.compose.material.icons.rounded.Edit
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material.icons.rounded.Star
import androidx.compose.material.icons.rounded.StarBorder
import androidx.compose.material.icons.rounded.Warning
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.nxgate.app.model.DynamicGroupCard
import com.nxgate.app.model.MasterGatewayInfo
import com.nxgate.app.model.ServerProfile
import com.nxgate.app.model.TunnelItem
import com.nxgate.app.ui.components.ConnectedButtonGroup
import com.nxgate.app.ui.components.ConnectedButtonItem
import com.nxgate.app.ui.components.ConnectedButtonStyle
import com.nxgate.app.ui.components.countryFlag
import com.nxgate.app.util.LocalAppStrings

private data class GroupExitRow(
    val devName: String,
    val nodeId: String,
    val nodeIp: String,
    val nodePort: Int,
    val country: String,
    val latencyMs: Int,
    val isFavorite: Boolean
)

@Composable
fun DynamicGroupTab(
    dynamicGroups: List<DynamicGroupCard>,
    activeServer: ServerProfile?,
    isEn: Boolean,
    tunnels: List<TunnelItem> = emptyList(),
    masterInfo: MasterGatewayInfo = MasterGatewayInfo(),
    favoriteNodeIds: Set<String> = emptySet(),
    onToggleFavoriteNode: (String) -> Unit = {},
    onBlacklistNode: (nodeId: String, ip: String, country: String, groupId: String) -> Unit = { _, _, _, _ -> },
    onDeleteGroup: (DynamicGroupCard) -> Unit,
    onEditGroup: (DynamicGroupCard) -> Unit,
    onEvaluateAndRotateClick: () -> Unit,
    onAddGroupClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    val strings = LocalAppStrings.current

    Column(
        modifier = modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        if (dynamicGroups.isEmpty()) {
            Surface(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(16.dp),
                color = MaterialTheme.colorScheme.surfaceContainerLow
            ) {
                Text(
                    text = if (activeServer == null) {
                        if (isEn) "No servers managed yet. Go to Dashboard or Settings to add a VPS gateway." else "当前尚未纳管任何服务器，请前往「概览」或「设置」添加 VPS 网关。"
                    } else {
                        if (isEn) "No dynamic exit groups configured. Tap \"Add Exit Group\" below to create one." else "当前服务器尚未配置动态出口组，请点击下方「新建出口组」添加。"
                    },
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurface,
                    modifier = Modifier.padding(20.dp)
                )
            }
        } else {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                dynamicGroups.forEach { group ->
                    Card(
                        modifier = Modifier.fillMaxWidth(),
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
                                .padding(20.dp),
                            verticalArrangement = Arrangement.Center
                        ) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Text(
                                    text = group.title,
                                    style = MaterialTheme.typography.titleMedium,
                                    fontWeight = FontWeight.Bold,
                                    color = MaterialTheme.colorScheme.onSurface
                                )
                                Row(verticalAlignment = Alignment.CenterVertically) {
                                    val modeSuffix = when (group.connectMode) {
                                        "zero_tun" -> strings.localized(" [免TUN]", " [No TUN]")
                                        "kernel" -> strings.localized(" [内核TUN]", " [Kernel TUN]")
                                        else -> ""
                                    }
                                    val badgeText = if (group.isSystem) {
                                        val sysMode = if (group.connectMode == "zero_tun") strings.localized("免TUN", "No TUN") else "tun0"
                                        if (isEn) "Primary ($sysMode)" else "系统主出口 ($sysMode)"
                                    } else if (group.country.equals("FAVORITES", true)) {
                                        "${strings.favoritesGroupBadge} (${group.targetCount}${if (isEn) " NICs" else "网卡"})$modeSuffix"
                                    } else {
                                        if (isEn) "Exit Group (${group.targetCount} NICs)$modeSuffix" else "出口组 (${group.targetCount}网卡)$modeSuffix"
                                    }
                                    Text(
                                        text = badgeText,
                                        style = MaterialTheme.typography.labelSmall,
                                        color = MaterialTheme.colorScheme.primary
                                    )
                                    if (!group.isSystem) {
                                        IconButton(
                                            onClick = { onDeleteGroup(group) },
                                            modifier = Modifier.size(32.dp)
                                        ) {
                                            Icon(
                                                Icons.Rounded.Delete,
                                                contentDescription = if (isEn) "Delete" else "删除组",
                                                tint = MaterialTheme.colorScheme.error,
                                                modifier = Modifier.size(18.dp)
                                            )
                                        }
                                    }
                                }
                            }
                            Spacer(Modifier.height(8.dp))
                            Text(
                                text = strings.groupDescription(group),
                                style = MaterialTheme.typography.bodyMedium,
                                color = MaterialTheme.colorScheme.onSurface,
                                lineHeight = MaterialTheme.typography.bodyMedium.lineHeight * 1.35f
                            )
                            if (group.inFallback) {
                                Spacer(Modifier.height(6.dp))
                                Surface(
                                    shape = RoundedCornerShape(8.dp),
                                    color = MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.85f),
                                    modifier = Modifier.fillMaxWidth()
                                ) {
                                    Row(
                                        modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp),
                                        verticalAlignment = Alignment.CenterVertically
                                    ) {
                                        Icon(
                                            Icons.Rounded.Warning,
                                            contentDescription = null,
                                            tint = MaterialTheme.colorScheme.error,
                                            modifier = Modifier.size(16.dp)
                                        )
                                        Spacer(Modifier.width(6.dp))
                                        Text(
                                            text = group.fallbackReason.ifEmpty { if (isEn) "Fallback Active" else "已触发级联降级策略运行中" },
                                            style = MaterialTheme.typography.labelSmall,
                                            color = MaterialTheme.colorScheme.onErrorContainer,
                                            fontWeight = FontWeight.SemiBold
                                        )
                                    }
                                }
                            }
                            Spacer(Modifier.height(10.dp))
                            val groupExits = tunnels.filter { t ->
                                group.activeTunnelIds.contains(t.id) ||
                                    group.activeTunnelIds.contains(t.devName) ||
                                    (group.isSystem && (t.isMaster || t.id == "primary-userland" || t.devIndex == 0))
                            }.map { t ->
                                GroupExitRow(
                                    devName = t.devName,
                                    nodeId = t.nodeId,
                                    nodeIp = t.nodeIp,
                                    nodePort = t.nodePort,
                                    country = t.country,
                                    latencyMs = t.latencyMs,
                                    isFavorite = favoriteNodeIds.contains(t.nodeId) || t.isFavorite
                                )
                            }.ifEmpty {
                                val hasPrimaryNode = group.isSystem && masterInfo.isConnected && masterInfo.nodeIp.isNotEmpty()
                                val resolvedId = masterInfo.nodeName.takeIf { it.isNotEmpty() && it != "未连接" && it != "Disconnected" } ?: ""
                                if (hasPrimaryNode) {
                                    listOf(
                                        GroupExitRow(
                                            devName = if (masterInfo.isZeroTun) "sstp" else "tun0",
                                            nodeId = resolvedId,
                                            nodeIp = masterInfo.nodeIp,
                                            nodePort = 443,
                                            country = masterInfo.country,
                                            latencyMs = masterInfo.latencyMs,
                                            isFavorite = (resolvedId.isNotEmpty() && favoriteNodeIds.contains(resolvedId)) || masterInfo.isFavorite
                                        )
                                    )
                                } else emptyList()
                            }

                            if (groupExits.isNotEmpty()) {
                                HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.6f))
                                Spacer(Modifier.height(8.dp))
                                Text(
                                    text = "${strings.groupActiveNodes} (${groupExits.size})",
                                    style = MaterialTheme.typography.labelSmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    fontWeight = FontWeight.SemiBold
                                )
                                Spacer(Modifier.height(6.dp))
                                Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                                    groupExits.forEach { exit ->
                                        Surface(
                                            shape = RoundedCornerShape(10.dp),
                                            color = MaterialTheme.colorScheme.surfaceContainerHigh,
                                            modifier = Modifier.fillMaxWidth()
                                        ) {
                                            Row(
                                                modifier = Modifier
                                                    .fillMaxWidth()
                                                    .padding(horizontal = 10.dp, vertical = 6.dp),
                                                horizontalArrangement = Arrangement.SpaceBetween,
                                                verticalAlignment = Alignment.CenterVertically
                                            ) {
                                                Row(
                                                    modifier = Modifier.weight(1f),
                                                    verticalAlignment = Alignment.CenterVertically,
                                                    horizontalArrangement = Arrangement.spacedBy(6.dp)
                                                ) {
                                                    Text(
                                                        text = exit.devName,
                                                        style = MaterialTheme.typography.labelSmall,
                                                        fontFamily = FontFamily.Monospace,
                                                        fontWeight = FontWeight.Bold,
                                                        color = MaterialTheme.colorScheme.primary
                                                    )
                                                    Text(
                                                        text = "${countryFlag(exit.country)} ${exit.nodeIp}${if (exit.nodePort > 0) ":${exit.nodePort}" else ""}",
                                                        style = MaterialTheme.typography.bodySmall,
                                                        fontFamily = FontFamily.Monospace,
                                                        color = MaterialTheme.colorScheme.onSurface
                                                    )
                                                    if (exit.latencyMs > 0) {
                                                        Text(
                                                            text = "${exit.latencyMs}ms",
                                                            style = MaterialTheme.typography.labelSmall,
                                                            color = MaterialTheme.colorScheme.secondary
                                                        )
                                                    }
                                                }
                                                if (exit.nodeId.isNotEmpty()) {
                                                    Row(verticalAlignment = Alignment.CenterVertically) {
                                                        IconButton(
                                                            onClick = { onToggleFavoriteNode(exit.nodeId) },
                                                            modifier = Modifier.size(32.dp)
                                                        ) {
                                                            Icon(
                                                                imageVector = if (exit.isFavorite) Icons.Rounded.Star else Icons.Rounded.StarBorder,
                                                                contentDescription = if (exit.isFavorite) strings.favorited else strings.favorite,
                                                                tint = if (exit.isFavorite) Color(0xFFF59E0B) else MaterialTheme.colorScheme.onSurfaceVariant,
                                                                modifier = Modifier.size(18.dp)
                                                            )
                                                        }
                                                        IconButton(
                                                            onClick = { onBlacklistNode(exit.nodeId, exit.nodeIp, exit.country, group.id) },
                                                            modifier = Modifier.size(32.dp)
                                                        ) {
                                                            Icon(
                                                                imageVector = Icons.Rounded.Block,
                                                                contentDescription = strings.blockAndRotate,
                                                                tint = MaterialTheme.colorScheme.error,
                                                                modifier = Modifier.size(18.dp)
                                                            )
                                                        }
                                                    }
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                            Spacer(Modifier.height(12.dp))
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.End
                            ) {
                                OutlinedButton(
                                    onClick = { onEditGroup(group) },
                                    modifier = Modifier.height(36.dp),
                                    shape = RoundedCornerShape(18.dp)
                                ) {
                                    Icon(Icons.Rounded.Edit, contentDescription = null, modifier = Modifier.size(16.dp))
                                    Spacer(Modifier.width(4.dp))
                                    Text(if (isEn) "Edit Rule" else "编辑规则", style = MaterialTheme.typography.labelMedium)
                                }
                            }
                        }
                    }
                }
            }
        }

        // 出口组操作组
        ConnectedButtonGroup(
            items = listOf(
                ConnectedButtonItem(
                    text = if (isEn) "Evaluate & Rotate" else "立即重评换线",
                    style = ConnectedButtonStyle.Filled,
                    icon = Icons.Rounded.Refresh,
                    onClick = onEvaluateAndRotateClick
                ),
                ConnectedButtonItem(
                    text = if (isEn) "Add Exit Group" else "新建出口组",
                    style = ConnectedButtonStyle.Tonal,
                    icon = Icons.Rounded.Add,
                    onClick = onAddGroupClick
                )
            )
        )
    }
}
