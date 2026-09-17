package com.nxgate.app.ui.screens

import android.widget.Toast
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.QrCodeScanner
import androidx.compose.material.icons.rounded.Settings
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.FloatingActionButtonDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.nxgate.app.NXGateApplication
import com.nxgate.app.model.ClusterSummary
import com.nxgate.app.model.ServerProfile
import com.nxgate.app.ui.components.ConnectedButtonItem
import com.nxgate.app.ui.components.ConnectedButtonGroup
import com.nxgate.app.ui.components.ConnectedButtonStyle
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

    val configuration = LocalConfiguration.current
    val isTablet = configuration.screenWidthDp >= 600

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
                            imageVector = Icons.Rounded.Settings,
                            contentDescription = "配置与安全设置",
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
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(horizontal = 16.dp)
        ) {
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .widthIn(max = if (isTablet) 920.dp else 500.dp)
                    .align(Alignment.TopCenter)
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
                            text = "纳管主机：${servers.size} 台在线 / ${servers.count { !it.isOnline }} 台离线\n实时吞吐：下行 ${summary.downSpeedStr}，上行 ${summary.upSpeedStr}\n今日总流量：${summary.todayTrafficStr}",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onPrimaryContainer.copy(alpha = 0.9f),
                            lineHeight = MaterialTheme.typography.bodyMedium.lineHeight * 1.35f
                        )
                    }
                }

                // 2. 动态渲染各服务器卡片及其对应操作组
                if (servers.isEmpty()) {
                    Surface(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(16.dp),
                        color = MaterialTheme.colorScheme.surfaceContainerLow
                    ) {
                        Text(
                            text = "当前暂未纳管任何服务器，请点击右下角按钮或右上角扫码添加 VPS。",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(16.dp)
                        )
                    }
                } else {
                    servers.forEachIndexed { index, server ->
                        val cardHeight = if (index == 0) 174.dp else 160.dp

                        Card(
                            modifier = Modifier
                                .fillMaxWidth()
                                .height(cardHeight)
                                .clip(RoundedCornerShape(20.dp))
                                .clickable { onSelectServerAndOpenConsole(server) },
                            shape = RoundedCornerShape(20.dp),
                            colors = CardDefaults.cardColors(
                                containerColor = MaterialTheme.colorScheme.surfaceContainer
                            ),
                            border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant),
                            elevation = CardDefaults.cardElevation(defaultElevation = 1.5.dp)
                        ) {
                            Column(
                                modifier = Modifier
                                    .fillMaxSize()
                                    .padding(20.dp),
                                verticalArrangement = Arrangement.Center
                            ) {
                                Text(
                                    text = "${server.name} 在线 ${server.latencyMs}ms",
                                    style = MaterialTheme.typography.titleMedium,
                                    fontWeight = FontWeight.Bold,
                                    color = MaterialTheme.colorScheme.onSurface
                                )
                                Spacer(Modifier.height(8.dp))
                                Text(
                                    text = "物理出口：${server.exitIp} ${server.ipType}\n智能解锁：${server.unlockStatus}\n下行 ${server.downSpeedStr}，总计 ${server.totalTrafficStr}，活跃连接 ${server.activeConns}",
                                    style = MaterialTheme.typography.bodyMedium,
                                    color = MaterialTheme.colorScheme.onSurface,
                                    lineHeight = MaterialTheme.typography.bodyMedium.lineHeight * 1.35f
                                )
                            }
                        }

                        // Connected button group below the card
                        ConnectedButtonGroup(
                            items = listOf(
                                ConnectedButtonItem(
                                    text = "一键换线",
                                    style = ConnectedButtonStyle.Tonal,
                                    onClick = {
                                        scope.launch {
                                            val res = NXGateApplication.instance.apiClient.triggerRotate(server)
                                            Toast.makeText(
                                                context,
                                                res.getOrDefault("已触发 [${server.name}] 动态重评换线！"),
                                                Toast.LENGTH_SHORT
                                            ).show()
                                        }
                                    }
                                ),
                                ConnectedButtonItem(
                                    text = "复制订阅",
                                    style = ConnectedButtonStyle.Tonal,
                                    onClick = {
                                        scope.launch {
                                            val subRes = NXGateApplication.instance.apiClient.fetchClashSubscription(server)
                                            val url = subRes.getOrDefault("${server.baseUrl}/api/singbox/subscription/clash")
                                            clipboardManager.setText(AnnotatedString(url))
                                            Toast.makeText(context, "已复制 [${server.name}] Clash Meta 分流订阅！", Toast.LENGTH_SHORT).show()
                                        }
                                    }
                                ),
                                ConnectedButtonItem(
                                    text = "控制台",
                                    style = ConnectedButtonStyle.Filled,
                                    onClick = { onSelectServerAndOpenConsole(server) }
                                )
                            )
                        )
                    }
                }

                Spacer(Modifier.height(80.dp))
            }
        }
    }
}
