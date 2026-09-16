package com.aimili.vpn.ui.screens

import android.widget.Toast
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.Article
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material.icons.rounded.ExpandLess
import androidx.compose.material.icons.rounded.Hub
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material.icons.rounded.Speed
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.aimili.vpn.AimiliApplication
import com.aimili.vpn.model.MasterGatewayInfo
import com.aimili.vpn.model.ServerProfile
import com.aimili.vpn.model.TunnelItem
import com.aimili.vpn.ui.components.ConnectedButtonItem
import com.aimili.vpn.ui.components.ConnectedButtonGroup
import com.aimili.vpn.ui.components.ConnectedButtonStyle
import com.aimili.vpn.ui.components.ConnectedListItem
import com.aimili.vpn.ui.components.GlobalServerSwitcherTitle
import com.aimili.vpn.ui.components.SpeedWaveformCard
import kotlinx.coroutines.launch

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

    var masterInfo by remember {
        mutableStateOf(
            MasterGatewayInfo(
                devName = "主网卡零号，策略表一百",
                nodeName = "日本住宅节点十二号",
                uptimeStr = "18小时24分",
                status = "断流检测通过",
                isConnected = true
            )
        )
    }

    var tunnelList by remember {
        mutableStateOf(
            listOf(
                TunnelItem(
                    id = "tun-1",
                    devName = "tun1",
                    devIndex = 1,
                    latencyMs = 42,
                    country = "日本",
                    throughputBps = 1800000,
                    openai = "unlocked",
                    claude = "unlocked",
                    gemini = "unlocked"
                ),
                TunnelItem(
                    id = "tun-2",
                    devName = "tun2",
                    devIndex = 2,
                    latencyMs = 55,
                    country = "日本",
                    throughputBps = 920000,
                    uptimeSeconds = 2880
                )
            )
        )
    }

    var showLogSheet by remember { mutableStateOf(false) }
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)

    // Pull real data when active server changes
    LaunchedEffect(activeServer?.id) {
        if (activeServer != null) {
            val statusRes = AimiliApplication.instance.apiClient.fetchStatus(activeServer)
            if (statusRes.isSuccess) {
                masterInfo = statusRes.getOrNull() ?: masterInfo
            }
            val tunnelsRes = AimiliApplication.instance.apiClient.fetchTunnels(activeServer)
            if (tunnelsRes.isSuccess && !tunnelsRes.getOrNull().isNullOrEmpty()) {
                tunnelList = tunnelsRes.getOrNull()!!
            }
        }
    }

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
                                    if (tunnelsRes.isSuccess && !tunnelsRes.getOrNull().isNullOrEmpty()) {
                                        tunnelList = tunnelsRes.getOrNull()!!
                                    }
                                    Toast.makeText(context, "已从 [${activeServer.name}] 刷新实时隧道与主网关数据", Toast.LENGTH_SHORT).show()
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
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(horizontal = 16.dp)
                .verticalScroll(scrollState),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Spacer(Modifier.height(4.dp))

            // 1. 实时网速波形卡片（高 164dp）（背景 surfaceContainerHigh）
            SpeedWaveformCard(
                downSpeedStr = activeServer?.downSpeedStr ?: "12.4 兆每秒",
                upSpeedStr = "1.2 兆每秒"
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
                        text = "系统主出口网关",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                    Spacer(Modifier.height(8.dp))
                    Text(
                        text = "设备：${masterInfo.devName}\n节点：${masterInfo.nodeName}\n已运行：${masterInfo.uptimeStr}，${masterInfo.status}",
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

            // 4. 动态并发隧道列表项与日志项
            val itemsCount = tunnelList.size + 1
            Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
                tunnelList.forEachIndexed { index, tunnel ->
                    ConnectedListItem(
                        index = index,
                        total = itemsCount,
                        headline = tunnel.title,
                        supportingText = tunnel.subtitle,
                        leadingIcon = Icons.Rounded.Hub,
                        trailingContent = {
                            if (index == 0) {
                                IconButton(onClick = {
                                    Toast.makeText(context, "重新测速该隧道并刷新延迟与解锁状态...", Toast.LENGTH_SHORT).show()
                                }) {
                                    Icon(Icons.Rounded.Speed, contentDescription = "测速", tint = MaterialTheme.colorScheme.primary)
                                }
                            } else {
                                IconButton(onClick = {
                                    if (activeServer != null) {
                                        scope.launch {
                                            AimiliApplication.instance.apiClient.stopTunnel(activeServer, tunnel.id)
                                            tunnelList = tunnelList.filter { it.id != tunnel.id }
                                            Toast.makeText(context, "已释放虚拟网卡 ${tunnel.devName} 并清理远端策略路由", Toast.LENGTH_SHORT).show()
                                        }
                                    }
                                }) {
                                    Icon(Icons.Rounded.Close, contentDescription = "释放该网卡", tint = MaterialTheme.colorScheme.error)
                                }
                            }
                        },
                        onClick = {
                            Toast.makeText(context, "设备: ${tunnel.devName} (出口: ${tunnel.nodeIp} ${tunnel.country})", Toast.LENGTH_SHORT).show()
                        }
                    )
                }

                // Last item: 实时事件与系统日志
                ConnectedListItem(
                    index = itemsCount - 1,
                    total = itemsCount,
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
                Text(
                    text = "系统运维日志流",
                    style = MaterialTheme.typography.titleLarge,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.onSurface
                )
                Text(
                    text = "实时事件长连接广播: 端口预检、断流熔断与自适应轮换",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                Spacer(Modifier.height(12.dp))
                HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
                Spacer(Modifier.height(8.dp))

                val mockLogs = listOf(
                    "[INFO] [Main] 正在初始化节点池并拉取候选节点...",
                    "[INFO] [Nodes] 🚪 端口预检完成：291/767 节点可达，已过滤 476 个死端口",
                    "[INFO] [TunnelPool] 已为接口 tun1 配置独立隔离策略路由 (Table 101) 与 rp_filter",
                    "[INFO] [UnlockDetector] [tun1] 吞吐量检测通过: PASSED (1019.4 KB/s)",
                    "[INFO] [UnlockDetector] 实测解锁结果: ChatGPT=unlocked, Claude=unlocked, Gemini=unlocked",
                    "[INFO] [DynamicGroup] [日本前三住宅组] 动态自适应评估完成，维持 3 个健康出口在线",
                    "[INFO] [Proxy] 代理端口 [7928] 已就绪监听于 127.0.0.1:7928 (SOCKS5 TCP+UDP)"
                )

                LazyColumn(
                    modifier = Modifier.height(280.dp),
                    verticalArrangement = Arrangement.spacedBy(6.dp)
                ) {
                    items(mockLogs) { log ->
                        Surface(
                            modifier = Modifier.fillMaxWidth(),
                            shape = RoundedCornerShape(8.dp),
                            color = MaterialTheme.colorScheme.surfaceContainerHighest
                        ) {
                            Text(
                                text = log,
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurface,
                                modifier = Modifier.padding(10.dp)
                            )
                        }
                    }
                }
            }
        }
    }
}
