package com.aimili.vpn.ui.screens

import android.widget.Toast
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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.QrCodeScanner
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.FloatingActionButtonDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.aimili.vpn.AimiliApplication
import com.aimili.vpn.model.ClusterSummary
import com.aimili.vpn.model.ServerProfile
import com.aimili.vpn.ui.components.ConnectedButtonItem
import com.aimili.vpn.ui.components.ConnectedButtonGroup
import com.aimili.vpn.ui.components.ConnectedButtonStyle
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ClusterHubScreen(
    servers: List<ServerProfile>,
    activeServer: ServerProfile?,
    summary: ClusterSummary,
    onSelectServerAndOpenConsole: (ServerProfile) -> Unit,
    onOpenSettings: () -> Unit,
    modifier: Modifier = Modifier
) {
    val context = LocalContext.current
    val clipboardManager = LocalClipboardManager.current
    val scope = rememberCoroutineScope()
    val scrollState = rememberScrollState()

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        text = "VPN 集群监控",
                        style = MaterialTheme.typography.titleLarge,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                },
                actions = {
                    IconButton(
                        onClick = onOpenSettings,
                        modifier = Modifier.size(48.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Rounded.QrCodeScanner,
                            contentDescription = "扫码导入与配置",
                            tint = MaterialTheme.colorScheme.onSurface,
                            modifier = Modifier.size(24.dp)
                        )
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surface
                )
            )
        },
        floatingActionButton = {
            FloatingActionButton(
                onClick = onOpenSettings,
                shape = RoundedCornerShape(16.dp),
                containerColor = MaterialTheme.colorScheme.primary,
                contentColor = MaterialTheme.colorScheme.onPrimary,
                elevation = FloatingActionButtonDefaults.elevation(defaultElevation = 6.dp),
                modifier = Modifier.padding(16.dp)
            ) {
                Icon(
                    imageVector = Icons.Rounded.Add,
                    contentDescription = "添加新服务器",
                    modifier = Modifier.size(24.dp)
                )
            }
        },
        containerColor = MaterialTheme.colorScheme.surface
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(horizontal = 16.dp)
                .verticalScroll(scrollState),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Spacer(Modifier.height(4.dp))

            // 1. 集群全局汇总填充卡片（高 176dp）（背景 primaryContainer）
            Card(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(176.dp),
                shape = RoundedCornerShape(20.dp),
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.primaryContainer
                )
            ) {
                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(20.dp),
                    verticalArrangement = Arrangement.Center
                ) {
                    Text(
                        text = "集群全局汇总",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onPrimaryContainer
                    )
                    Spacer(Modifier.height(8.dp))
                    Text(
                        text = "纳管主机：${summary.onlineCount} 台在线 / ${summary.offlineCount} 台离线\n实时吞吐：下行 ${summary.downSpeedStr}，上行 ${summary.upSpeedStr}\n今日总流量：${summary.todayTrafficStr}",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onPrimaryContainer.copy(alpha = 0.9f),
                        lineHeight = MaterialTheme.typography.bodyMedium.lineHeight * 1.35f
                    )
                }
            }

            // 2. 东京住宅网关填充卡片（高 174dp）
            val tokyoServer = servers.firstOrNull { it.id == "tokyo-residential" } ?: servers.firstOrNull()
            if (tokyoServer != null) {
                Card(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(174.dp)
                        .clip(RoundedCornerShape(20.dp))
                        .clickable { onSelectServerAndOpenConsole(tokyoServer) },
                    shape = RoundedCornerShape(20.dp),
                    colors = CardDefaults.cardColors(
                        containerColor = MaterialTheme.colorScheme.surfaceContainerHighest
                    )
                ) {
                    Column(
                        modifier = Modifier
                            .fillMaxSize()
                            .padding(20.dp),
                        verticalArrangement = Arrangement.Center
                    ) {
                        Text(
                            text = "${tokyoServer.name} 在线 ${tokyoServer.latencyMs}毫秒",
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold,
                            color = MaterialTheme.colorScheme.onSurface
                        )
                        Spacer(Modifier.height(8.dp))
                        Text(
                            text = "物理出口：${tokyoServer.exitIp} ${tokyoServer.ipType}\n智能解锁：${tokyoServer.unlockStatus}\n下行 ${tokyoServer.downSpeedStr}，总计 ${tokyoServer.totalTrafficStr}，活跃连接 ${tokyoServer.activeConns}",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            lineHeight = MaterialTheme.typography.bodyMedium.lineHeight * 1.35f
                        )
                    }
                }

                // 3. 按钮组: “一键换线”(色调) “复制订阅”(色调) “控制台”(填充)
                ConnectedButtonGroup(
                    items = listOf(
                        ConnectedButtonItem(
                            text = "一键换线",
                            style = ConnectedButtonStyle.Tonal,
                            onClick = {
                                scope.launch {
                                    val res = AimiliApplication.instance.apiClient.triggerRotate(tokyoServer)
                                    Toast.makeText(
                                        context,
                                        res.getOrDefault("已在后台触发 [${tokyoServer.name}] 动态重评与换线！"),
                                        Toast.LENGTH_SHORT
                                    ).show()
                                }
                            }
                        ),
                        ConnectedButtonItem(
                            text = "复制订阅",
                            style = ConnectedButtonStyle.Tonal,
                            onClick = {
                                val clashUrl = "${tokyoServer.baseUrl}/api/singbox/subscription/clash"
                                clipboardManager.setText(AnnotatedString(clashUrl))
                                Toast.makeText(context, "Clash 订阅地址已复制到剪贴板！", Toast.LENGTH_SHORT).show()
                            }
                        ),
                        ConnectedButtonItem(
                            text = "控制台",
                            style = ConnectedButtonStyle.Filled,
                            onClick = { onSelectServerAndOpenConsole(tokyoServer) }
                        )
                    )
                )
            }

            // 4. 硅谷智能专属池填充卡片（高 160dp）
            val svServer = servers.find { it.id == "silicon-valley-ai" } ?: servers.getOrNull(1)
            if (svServer != null) {
                Card(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(160.dp)
                        .clip(RoundedCornerShape(20.dp))
                        .clickable { onSelectServerAndOpenConsole(svServer) },
                    shape = RoundedCornerShape(20.dp),
                    colors = CardDefaults.cardColors(
                        containerColor = MaterialTheme.colorScheme.surfaceContainerHighest
                    )
                ) {
                    Column(
                        modifier = Modifier
                            .fillMaxSize()
                            .padding(20.dp),
                        verticalArrangement = Arrangement.Center
                    ) {
                        Text(
                            text = "${svServer.name} 在线 ${svServer.latencyMs}毫秒",
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold,
                            color = MaterialTheme.colorScheme.onSurface
                        )
                        Spacer(Modifier.height(8.dp))
                        Text(
                            text = "物理出口：${svServer.exitIp} ${svServer.ipType}\n智能解锁：${svServer.unlockStatus}\n下行 ${svServer.downSpeedStr}，总计 ${svServer.totalTrafficStr}，活跃连接 ${svServer.activeConns}",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            lineHeight = MaterialTheme.typography.bodyMedium.lineHeight * 1.35f
                        )
                    }
                }
            }

            Spacer(Modifier.height(80.dp)) // Padding for bottom bar & FAB
        }
    }
}
