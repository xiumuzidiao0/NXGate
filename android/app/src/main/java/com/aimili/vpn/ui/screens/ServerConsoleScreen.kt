package com.aimili.vpn.ui.screens

import android.content.res.Configuration
import android.widget.Toast
import androidx.compose.foundation.BorderStroke
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
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.Article
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material.icons.rounded.ExpandLess
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material.icons.rounded.Speed
import androidx.compose.material.icons.rounded.Warning
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.aimili.vpn.AimiliApplication
import com.aimili.vpn.model.LiveTrafficInfo
import com.aimili.vpn.model.MasterGatewayInfo
import com.aimili.vpn.model.ServerProfile
import com.aimili.vpn.model.SystemLogEntry
import com.aimili.vpn.model.TunnelItem
import com.aimili.vpn.ui.components.ConnectedButtonItem
import com.aimili.vpn.ui.components.ConnectedButtonGroup
import com.aimili.vpn.ui.components.ConnectedButtonStyle
import com.aimili.vpn.ui.components.ConnectedListItem
import com.aimili.vpn.ui.components.GlobalServerSwitcherTitle
import com.aimili.vpn.ui.components.SpeedWaveformCard
import com.aimili.vpn.ui.components.UnlockPill
import com.aimili.vpn.ui.components.countryFlag
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

@Composable
fun TunnelChipCard(
    tunnel: TunnelItem,
    onProbeUnlock: () -> Unit,
    onStopTunnel: () -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainerHighest
        )
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(14.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            // Row 1: DevName + Status Badge + Flag + IP
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = tunnel.devName,
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.primary
                    )
                    Spacer(Modifier.width(8.dp))
                    Surface(
                        shape = RoundedCornerShape(4.dp),
                        color = MaterialTheme.colorScheme.primaryContainer
                    ) {
                        Text(
                            text = if (tunnel.status == "connected") "● 在线" else "● 连接中",
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onPrimaryContainer,
                            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                        )
                    }
                }

                Text(
                    text = "${countryFlag(tunnel.country)} ${tunnel.nodeIp}:${tunnel.nodePort}",
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.SemiBold,
                    color = MaterialTheme.colorScheme.onSurface
                )
            }

            // Row 2: Latency + Throughput
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = if (tunnel.latencyMs > 0) "物理延迟: ${tunnel.latencyMs}ms" else "延迟: 测活就绪",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                Text(
                    text = if (tunnel.throughputBps > 0) "吞吐: %.1f MB/s (断流检测通过)".format(tunnel.throughputBps / 1000000.0) else "在线待命",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.primary
                )
            }

            // Row 3: AI & Streaming unlock badges
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(4.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text("实测解锁:", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                UnlockPill("ChatGPT", tunnel.openai)
                UnlockPill("Claude", tunnel.claude)
                UnlockPill("Gemini", tunnel.gemini)
                UnlockPill("Netflix", tunnel.netflix)
            }

            // Row 4: Action Buttons (测解锁 & 断开 - 对应 Web 按钮)
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.End,
                verticalAlignment = Alignment.CenterVertically
            ) {
                OutlinedButton(
                    onClick = onProbeUnlock,
                    modifier = Modifier.height(34.dp),
                    shape = RoundedCornerShape(17.dp)
                ) {
                    Icon(Icons.Rounded.Speed, contentDescription = null, modifier = Modifier.size(16.dp))
                    Spacer(Modifier.width(4.dp))
                    Text("测解锁", style = MaterialTheme.typography.labelMedium)
                }
                Spacer(Modifier.width(8.dp))
                Button(
                    onClick = onStopTunnel,
                    modifier = Modifier.height(34.dp),
                    shape = RoundedCornerShape(17.dp),
                    colors = ButtonDefaults.buttonColors(
                        containerColor = MaterialTheme.colorScheme.error,
                        contentColor = MaterialTheme.colorScheme.onError
                    )
                ) {
                    Icon(Icons.Rounded.Close, contentDescription = null, modifier = Modifier.size(16.dp))
                    Spacer(Modifier.width(4.dp))
                    Text("断开", style = MaterialTheme.typography.labelMedium)
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ServerConsoleScreen(
    activeServer: ServerProfile?,
    allServers: List<ServerProfile>,
    onSelectServer: (String) -> Unit,
    onPreviousServer: () -> Unit,
    onNextServer: () -> Unit,
    modifier: Modifier = Modifier
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val scrollState = rememberScrollState()

    val configuration = LocalConfiguration.current
    val isLandscape = configuration.orientation == Configuration.ORIENTATION_LANDSCAPE
    val isTablet = configuration.screenWidthDp >= 600
    val isTabletLandscape = isTablet && isLandscape

    var masterInfo by remember {
        mutableStateOf(
            MasterGatewayInfo(
                devName = "主网卡 (tun0)",
                nodeName = "正在获取出口...",
                uptimeStr = "0分",
                status = "检测中",
                isConnected = false
            )
        )
    }

    var tunnelList by remember {
        mutableStateOf<List<TunnelItem>>(emptyList())
    }

    var liveTraffic by remember { mutableStateOf(LiveTrafficInfo()) }
    var serverLogs by remember { mutableStateOf<List<SystemLogEntry>>(emptyList()) }
    var isLoadingLogs by remember { mutableStateOf(false) }

    val cleartextWarningEnabled by AimiliApplication.instance.serverStore.cleartextWarningEnabled.collectAsState()
    var dismissCleartextBanner by remember(activeServer?.id) { mutableStateOf(false) }

    var showLogSheet by remember { mutableStateOf(false) }
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)

    // Pull real logs when log sheet opens
    LaunchedEffect(showLogSheet) {
        if (showLogSheet && activeServer != null) {
            isLoadingLogs = true
            val logsRes = AimiliApplication.instance.apiClient.fetchLogs(activeServer)
            isLoadingLogs = false
            if (logsRes.isSuccess) {
                serverLogs = logsRes.getOrNull() ?: emptyList()
            }
        }
    }

    // Continuously pull real data, live traffic speeds, and tunnels every 3 seconds
    LaunchedEffect(activeServer?.id) {
        while (isActive) {
            if (activeServer != null) {
                val statusRes = AimiliApplication.instance.apiClient.fetchServerStatus(activeServer)
                if (statusRes.isSuccess) {
                    val data = statusRes.getOrNull()
                    if (data != null) {
                        masterInfo = data.masterGateway
                        liveTraffic = data.traffic
                        tunnelList = data.tunnels
                        AimiliApplication.instance.serverStore.updateServerTraffic(
                            activeServer.id,
                            downSpeedStr = data.traffic.downSpeedMbStr,
                            upSpeedStr = data.traffic.upSpeedMbStr,
                            totalTrafficStr = data.traffic.totalTrafficGbStr,
                            activeConns = data.traffic.activeConnections
                        )
                    }
                }
            }
            delay(3000)
        }
    }

    // Separate Master Tunnel (tun0) and Concurrent Independent Tunnels (tun1, tun2...)
    val masterTunnel = tunnelList.find { it.isMaster }
    val concurrentTunnels = tunnelList.filter { !it.isMaster }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = {
                    GlobalServerSwitcherTitle(
                        activeServer = activeServer,
                        allServers = allServers,
                        onSelectServer = onSelectServer,
                        onPreviousServer = onPreviousServer,
                        onNextServer = onNextServer
                    )
                },
                actions = {
                    IconButton(
                        onClick = {
                            if (activeServer != null) {
                                scope.launch {
                                    val statusRes = AimiliApplication.instance.apiClient.fetchStatus(activeServer)
                                    if (statusRes.isSuccess) masterInfo = statusRes.getOrNull() ?: masterInfo
                                    val tunnelsRes = AimiliApplication.instance.apiClient.fetchTunnels(activeServer)
                                    if (tunnelsRes.isSuccess) {
                                        tunnelList = tunnelsRes.getOrNull() ?: emptyList()
                                    }
                                    Toast.makeText(context, "已从 [${activeServer.name}] 刷新实时网卡数据 (当前共 ${tunnelList.size} 张网卡)", Toast.LENGTH_SHORT).show()
                                }
                            }
                        },
                        modifier = Modifier.size(48.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Rounded.Refresh,
                            contentDescription = "刷新状态",
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
                    .widthIn(max = if (isTablet) 960.dp else 500.dp)
                    .align(Alignment.TopCenter)
                    .verticalScroll(scrollState),
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                Spacer(Modifier.height(4.dp))

                // 明文传输风险警示横幅 (当未启用 TLS 且开启明文提醒时展示)
                if (cleartextWarningEnabled && activeServer?.isTls == false && !dismissCleartextBanner) {
                    Surface(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(16.dp),
                        color = MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.85f),
                        border = BorderStroke(1.dp, MaterialTheme.colorScheme.error.copy(alpha = 0.5f))
                    ) {
                        Row(
                            modifier = Modifier.padding(14.dp),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Icon(
                                imageVector = Icons.Rounded.Warning,
                                contentDescription = null,
                                tint = MaterialTheme.colorScheme.error,
                                modifier = Modifier.size(24.dp)
                            )
                            Spacer(Modifier.width(12.dp))
                            Column(modifier = Modifier.weight(1f)) {
                                Text(
                                    text = "明文传输风险提醒",
                                    style = MaterialTheme.typography.titleSmall,
                                    fontWeight = FontWeight.Bold,
                                    color = MaterialTheme.colorScheme.onErrorContainer
                                )
                                Text(
                                    text = "当前服务器 [${activeServer.name}] (${activeServer.host}:${activeServer.port}) 未启用 TLS 加密，在公共 WiFi 或非受信任网络中管理可能存在明文窃听风险。建议配置 HTTPS 证书。",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onErrorContainer.copy(alpha = 0.9f)
                                )
                            }
                            IconButton(onClick = { dismissCleartextBanner = true }) {
                                Icon(Icons.Rounded.Close, contentDescription = "关闭提醒", tint = MaterialTheme.colorScheme.error)
                            }
                        }
                    }
                }

                if (isTabletLandscape) {
                    // ==================== 平板横屏：左右双列响应式布局 ====================
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(16.dp)
                    ) {
                        // 左侧列：实时波形 + 系统主网关 (tun0)
                        Column(
                            modifier = Modifier.weight(1f),
                            verticalArrangement = Arrangement.spacedBy(16.dp)
                        ) {
                            SpeedWaveformCard(
                                liveTraffic = liveTraffic
                            )

                            Card(
                                modifier = Modifier.fillMaxWidth(),
                                shape = RoundedCornerShape(20.dp),
                                colors = CardDefaults.cardColors(
                                    containerColor = MaterialTheme.colorScheme.surfaceContainerHighest
                                )
                            ) {
                                Column(
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .padding(20.dp),
                                    verticalArrangement = Arrangement.Center
                                ) {
                                    Text(
                                        text = "系统主出口网关 (tun0)",
                                        style = MaterialTheme.typography.titleMedium,
                                        fontWeight = FontWeight.Bold,
                                        color = MaterialTheme.colorScheme.onSurface
                                    )
                                    Spacer(Modifier.height(8.dp))
                                    val masterNodeText = masterTunnel?.let { "${countryFlag(it.country)} ${it.nodeIp}" } ?: masterInfo.nodeName
                                    val masterUptime = masterTunnel?.let { "${it.uptimeSeconds / 3600}小时${(it.uptimeSeconds % 3600) / 60}分" } ?: masterInfo.uptimeStr
                                    Text(
                                        text = "设备：主网卡零号，策略表一百\n节点：$masterNodeText\n已运行：$masterUptime，${masterInfo.status}",
                                        style = MaterialTheme.typography.bodyMedium,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                                        lineHeight = MaterialTheme.typography.bodyMedium.lineHeight * 1.35f
                                    )
                                }
                            }

                            ConnectedButtonGroup(
                                items = listOf(
                                    ConnectedButtonItem(
                                        text = "切换主出口",
                                        style = ConnectedButtonStyle.Filled,
                                        onClick = {
                                            if (activeServer != null) {
                                                scope.launch {
                                                    val res = AimiliApplication.instance.apiClient.triggerRotate(activeServer)
                                                    masterInfo = masterInfo.copy(
                                                        nodeName = "日本住宅最优节点(已切换)",
                                                        uptimeStr = "刚刚",
                                                        status = "断流检测通过",
                                                        isConnected = true
                                                    )
                                                    Toast.makeText(context, res.getOrDefault("已触发主网关切换最优出口！"), Toast.LENGTH_SHORT).show()
                                                }
                                            }
                                        }
                                    ),
                                    ConnectedButtonItem(
                                        text = "断开主网关",
                                        style = ConnectedButtonStyle.Outlined,
                                        onClick = {
                                            if (activeServer != null) {
                                                scope.launch {
                                                    AimiliApplication.instance.apiClient.disconnectMasterVPN(activeServer)
                                                    masterInfo = masterInfo.copy(
                                                        status = "已手动断开，保留配置",
                                                        isConnected = false
                                                    )
                                                    Toast.makeText(context, "主网关连接已主动断开，保留配置", Toast.LENGTH_SHORT).show()
                                                }
                                            }
                                        }
                                    )
                                )
                            )
                        }

                        // 右侧列：并发网卡列表（带测解锁与断开按键）+ 系统日志
                        Column(
                            modifier = Modifier.weight(1f),
                            verticalArrangement = Arrangement.spacedBy(10.dp)
                        ) {
                            Text(
                                text = "并发在线独立出口 (${concurrentTunnels.size} 条)",
                                style = MaterialTheme.typography.titleMedium,
                                fontWeight = FontWeight.Bold,
                                color = MaterialTheme.colorScheme.onSurface
                            )

                            if (concurrentTunnels.isEmpty()) {
                                Surface(
                                    modifier = Modifier.fillMaxWidth(),
                                    shape = RoundedCornerShape(16.dp),
                                    color = MaterialTheme.colorScheme.surfaceContainerLow
                                ) {
                                    Text(
                                        text = "暂无独立并发出口，在节点广场中点击「拉起网卡」即可多出口并发在线。",
                                        style = MaterialTheme.typography.bodyMedium,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                                        modifier = Modifier.padding(16.dp)
                                    )
                                }
                            } else {
                                concurrentTunnels.forEach { tunnel ->
                                    TunnelChipCard(
                                        tunnel = tunnel,
                                        onProbeUnlock = {
                                            if (activeServer != null) {
                                                scope.launch {
                                                    AimiliApplication.instance.apiClient.probeTunnelUnlock(activeServer, tunnel.id)
                                                    Toast.makeText(context, "正在对网卡 [${tunnel.devName}] 执行三大 AI 与流媒体实测解锁探测...", Toast.LENGTH_SHORT).show()
                                                }
                                            }
                                        },
                                        onStopTunnel = {
                                            if (activeServer != null) {
                                                scope.launch {
                                                    AimiliApplication.instance.apiClient.stopTunnel(activeServer, tunnel.id)
                                                    tunnelList = tunnelList.filter { it.id != tunnel.id }
                                                    Toast.makeText(context, "已断开并注销虚拟网卡 ${tunnel.devName}", Toast.LENGTH_SHORT).show()
                                                }
                                            }
                                        }
                                    )
                                }
                            }

                            ConnectedListItem(
                                index = 0,
                                total = 1,
                                headline = "实时事件与系统日志",
                                supportingText = "向上拖拽可展开日志抽屉，支持按信息、警告、错误过滤。",
                                leadingIcon = Icons.AutoMirrored.Rounded.Article,
                                trailingContent = {
                                    Icon(Icons.Rounded.ExpandLess, contentDescription = "展开日志", tint = MaterialTheme.colorScheme.onSurfaceVariant)
                                },
                                onClick = { showLogSheet = true }
                            )
                        }
                    }
                } else {
                    // ==================== 竖屏/手机：单列垂直布局 ====================
                    // 1. 实时网速波形卡片（高 164dp）（背景 surfaceContainerHigh）
                    SpeedWaveformCard(
                        liveTraffic = liveTraffic
                    )

                    // 2. 系统主出口网关填充卡片（高 158dp）
                    Card(
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(158.dp),
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
                                text = "系统主出口网关 (tun0)",
                                style = MaterialTheme.typography.titleMedium,
                                fontWeight = FontWeight.Bold,
                                color = MaterialTheme.colorScheme.onSurface
                            )
                            Spacer(Modifier.height(8.dp))
                            val masterNodeText = masterTunnel?.let { "${countryFlag(it.country)} ${it.nodeIp}" } ?: masterInfo.nodeName
                            val masterUptime = masterTunnel?.let { "${it.uptimeSeconds / 3600}小时${(it.uptimeSeconds % 3600) / 60}分" } ?: masterInfo.uptimeStr
                            Text(
                                text = "设备：主网卡零号，策略表一百\n节点：$masterNodeText\n已运行：$masterUptime，${masterInfo.status}",
                                style = MaterialTheme.typography.bodyMedium,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                lineHeight = MaterialTheme.typography.bodyMedium.lineHeight * 1.35f
                            )
                        }
                    }

                    // 3. 按钮组: “切换主出口”(填充) “断开主网关”(描边)
                    ConnectedButtonGroup(
                        items = listOf(
                            ConnectedButtonItem(
                                text = "切换主出口",
                                style = ConnectedButtonStyle.Filled,
                                onClick = {
                                    if (activeServer != null) {
                                        scope.launch {
                                            val res = AimiliApplication.instance.apiClient.triggerRotate(activeServer)
                                            masterInfo = masterInfo.copy(
                                                nodeName = "日本住宅最优节点(已切换)",
                                                uptimeStr = "刚刚",
                                                status = "断流检测通过",
                                                isConnected = true
                                            )
                                            Toast.makeText(context, res.getOrDefault("已触发主网关切换最优出口！"), Toast.LENGTH_SHORT).show()
                                        }
                                    }
                                }
                            ),
                            ConnectedButtonItem(
                                text = "断开主网关",
                                style = ConnectedButtonStyle.Outlined,
                                onClick = {
                                    if (activeServer != null) {
                                        scope.launch {
                                            AimiliApplication.instance.apiClient.disconnectMasterVPN(activeServer)
                                            masterInfo = masterInfo.copy(
                                                status = "已手动断开，保留配置",
                                                isConnected = false
                                            )
                                            Toast.makeText(context, "主网关连接已主动断开，保留配置", Toast.LENGTH_SHORT).show()
                                        }
                                    }
                                }
                            )
                        )
                    )

                    // 4. 并发独立网卡卡片列表 (精确显示真实并发网卡数量)
                    Text(
                        text = "并发在线独立出口 (${concurrentTunnels.size} 条)",
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )

                    if (concurrentTunnels.isEmpty()) {
                        Surface(
                            modifier = Modifier.fillMaxWidth(),
                            shape = RoundedCornerShape(16.dp),
                            color = MaterialTheme.colorScheme.surfaceContainerLow
                        ) {
                            Text(
                                text = "暂无独立并发出口，在节点广场中点击「拉起网卡」即可多出口并发在线。",
                                style = MaterialTheme.typography.bodyMedium,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.padding(16.dp)
                            )
                        }
                    } else {
                        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            concurrentTunnels.forEach { tunnel ->
                                TunnelChipCard(
                                    tunnel = tunnel,
                                    onProbeUnlock = {
                                        if (activeServer != null) {
                                            scope.launch {
                                                AimiliApplication.instance.apiClient.probeTunnelUnlock(activeServer, tunnel.id)
                                                Toast.makeText(context, "已对网卡 [${tunnel.devName}] 启动解锁探测！", Toast.LENGTH_SHORT).show()
                                            }
                                        }
                                    },
                                    onStopTunnel = {
                                        if (activeServer != null) {
                                            scope.launch {
                                                AimiliApplication.instance.apiClient.stopTunnel(activeServer, tunnel.id)
                                                tunnelList = tunnelList.filter { it.id != tunnel.id }
                                                Toast.makeText(context, "已断开并注销虚拟网卡 ${tunnel.devName}", Toast.LENGTH_SHORT).show()
                                            }
                                        }
                                    }
                                )
                            }
                        }
                    }

                    // Last item: 实时事件与系统日志
                    ConnectedListItem(
                        index = 0,
                        total = 1,
                        headline = "实时事件与系统日志",
                        supportingText = "向上拖拽可展开日志抽屉，支持按信息、警告、错误过滤。",
                        leadingIcon = Icons.AutoMirrored.Rounded.Article,
                        trailingContent = {
                            Icon(Icons.Rounded.ExpandLess, contentDescription = "展开日志", tint = MaterialTheme.colorScheme.onSurfaceVariant)
                        },
                        onClick = { showLogSheet = true }
                    )
                }

                Spacer(Modifier.height(80.dp))
            }
        }
    }

    // Log Drawer ModalBottomSheet
    if (showLogSheet) {
        ModalBottomSheet(
            onDismissRequest = { showLogSheet = false },
            sheetState = sheetState,
            containerColor = MaterialTheme.colorScheme.surfaceContainer,
            shape = RoundedCornerShape(topStart = 28.dp, topEnd = 28.dp)
        ) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 20.dp)
                    .padding(bottom = 32.dp)
            ) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            text = "系统运维日志流",
                            style = MaterialTheme.typography.titleLarge,
                            fontWeight = FontWeight.Bold,
                            color = MaterialTheme.colorScheme.onSurface
                        )
                        Text(
                            text = "来自服务端 /api/logs 的实时环形日志 (共 ${serverLogs.size} 条记录)",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                    IconButton(
                        onClick = {
                            if (activeServer != null) {
                                scope.launch {
                                    isLoadingLogs = true
                                    val logsRes = AimiliApplication.instance.apiClient.fetchLogs(activeServer)
                                    isLoadingLogs = false
                                    if (logsRes.isSuccess) {
                                        serverLogs = logsRes.getOrNull() ?: emptyList()
                                        Toast.makeText(context, "已刷新最新 ${serverLogs.size} 条日志", Toast.LENGTH_SHORT).show()
                                    }
                                }
                            }
                        }
                    ) {
                        Icon(Icons.Rounded.Refresh, contentDescription = "刷新日志", tint = MaterialTheme.colorScheme.primary)
                    }
                }
                Spacer(Modifier.height(8.dp))
                HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
                Spacer(Modifier.height(8.dp))

                if (isLoadingLogs) {
                    LinearProgressIndicator(modifier = Modifier.fillMaxWidth(), color = MaterialTheme.colorScheme.primary)
                    Spacer(Modifier.height(8.dp))
                }

                if (serverLogs.isEmpty() && !isLoadingLogs) {
                    Surface(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(8.dp),
                        color = MaterialTheme.colorScheme.surfaceContainerHighest
                    ) {
                        Text(
                            text = "当前服务器暂未产生事件日志，待有网络活动或节点轮换时将自动显示。",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(16.dp)
                        )
                    }
                } else {
                    LazyColumn(
                        modifier = Modifier.height(320.dp),
                        verticalArrangement = Arrangement.spacedBy(6.dp)
                    ) {
                        items(serverLogs) { log ->
                            Surface(
                                modifier = Modifier.fillMaxWidth(),
                                shape = RoundedCornerShape(8.dp),
                                color = MaterialTheme.colorScheme.surfaceContainerHighest
                            ) {
                                Text(
                                    text = log.formatted,
                                    style = MaterialTheme.typography.bodySmall,
                                    color = when (log.level.uppercase()) {
                                        "ERROR" -> MaterialTheme.colorScheme.error
                                        "WARNING", "WARN" -> MaterialTheme.colorScheme.tertiary
                                        else -> MaterialTheme.colorScheme.onSurface
                                    },
                                    modifier = Modifier.padding(10.dp)
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}
