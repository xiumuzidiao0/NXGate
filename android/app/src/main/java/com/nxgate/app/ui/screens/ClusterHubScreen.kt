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
import com.nxgate.app.ui.components.SubscriptionDialog
import com.nxgate.app.util.LocalAppStrings
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
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
    val strings = LocalAppStrings.current
    val clipboardManager = LocalClipboardManager.current
    val scope = rememberCoroutineScope()
    val scrollState = rememberScrollState()
    var subscriptionServerTarget by remember { mutableStateOf<ServerProfile?>(null) }

    val configuration = LocalConfiguration.current
    val isTablet = configuration.screenWidthDp >= 600

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        text = strings.clusterMonitor,
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
                            contentDescription = strings.settingsTitle,
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
                    contentDescription = strings.addServer,
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
                            text = strings.clusterSummary,
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold,
                            color = MaterialTheme.colorScheme.onPrimaryContainer
                        )
                        Spacer(Modifier.height(8.dp))
                        Text(
                            text = "${strings.managedHosts}: ${servers.size} ${strings.hostsOnline} / ${servers.count { !it.isOnline }} ${strings.hostsOffline}\n${strings.liveThroughput}: ${strings.downSpeed} ${summary.downSpeedStr}, ${strings.upSpeed} ${summary.upSpeedStr}\n${strings.todayTraffic}: ${summary.todayTrafficStr}",
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
                            text = strings.noServers,
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
                                    text = "${server.name} ${if (server.isOnline && server.latencyMs > 0) "${strings.online} (${server.latencyMs}ms)" else "${strings.offline}"}",
                                    style = MaterialTheme.typography.titleMedium,
                                    fontWeight = FontWeight.Bold,
                                    color = MaterialTheme.colorScheme.onSurface
                                )
                                Spacer(Modifier.height(8.dp))
                                val exitText = if (server.exitIp.isNotEmpty()) "${server.exitIp} ${server.ipType}" else strings.unknown
                                val unlockText = if (server.unlockStatus.isNotEmpty()) server.unlockStatus else "-"
                                Text(
                                    text = "${strings.physicalExit}: $exitText\n${strings.aiUnlock}: $unlockText\n${strings.downSpeed} ${server.downSpeedStr}, ${strings.totalTraffic} ${server.totalTrafficStr}, ${strings.activeConns} ${server.activeConns}",
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
                                    text = strings.quickRotate,
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
                                    text = strings.getSub,
                                    style = ConnectedButtonStyle.Tonal,
                                    onClick = {
                                        subscriptionServerTarget = server
                                    }
                                ),
                                ConnectedButtonItem(
                                    text = strings.console,
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

    subscriptionServerTarget?.let { s ->
        SubscriptionDialog(
            server = s,
            onDismissRequest = { subscriptionServerTarget = null }
        )
    }
}
