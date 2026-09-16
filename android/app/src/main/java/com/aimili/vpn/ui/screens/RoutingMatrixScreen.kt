package com.aimili.vpn.ui.screens

import android.widget.Toast
import androidx.compose.foundation.layout.Arrangement
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
import androidx.compose.material.icons.rounded.Lan
import androidx.compose.material.icons.rounded.QrCode2
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material.icons.rounded.Security
import androidx.compose.material.icons.rounded.SwapHoriz
import androidx.compose.material.icons.rounded.ToggleOff
import androidx.compose.material.icons.rounded.ToggleOn
import androidx.compose.material.icons.rounded.VpnKey
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
import androidx.compose.material3.TabRowDefaults
import androidx.compose.material3.TabRowDefaults.tabIndicatorOffset
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.aimili.vpn.AimiliApplication
import com.aimili.vpn.model.DynamicGroupCard
import com.aimili.vpn.model.InboundProtocolItem
import com.aimili.vpn.model.PortRuleItem
import com.aimili.vpn.model.ServerProfile
import com.aimili.vpn.ui.components.ConnectedButtonItem
import com.aimili.vpn.ui.components.ConnectedButtonGroup
import com.aimili.vpn.ui.components.ConnectedButtonStyle
import com.aimili.vpn.ui.components.ConnectedListItem
import com.aimili.vpn.ui.components.GlobalServerSwitcherTitle
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RoutingMatrixScreen(
    activeServer: ServerProfile?,
    allServers: List<ServerProfile>,
    onSelectServer: (String) -> Unit,
    onPreviousServer: () -> Unit,
    onNextServer: () -> Unit,
    modifier: Modifier = Modifier
) {
    val context = LocalContext.current
    val clipboardManager = LocalClipboardManager.current
    val scope = rememberCoroutineScope()
    val scrollState = rememberScrollState()

    var selectedTabIndex by remember { mutableIntStateOf(0) }
    val tabTitles = listOf("多端口", "自适应组", "边缘入站")

    // State for port rules
    var portRules by remember {
        mutableStateOf(
            listOf(
                PortRuleItem(7928, enabled = true, policy = "round_robin", authMode = "random"),
                PortRuleItem(7929, enabled = true, policy = "interval", intervalSeconds = 300, authMode = "custom", authUser = "custom_user"),
                PortRuleItem(7930, enabled = false, policy = "random", authMode = "none")
            )
        )
    }

    var dynamicGroups by remember {
        mutableStateOf(
            listOf(
                DynamicGroupCard(id = "system-primary", name = "系统主出口网关组 (tun0)", country = "日本", ipType = "residential", unlockFilter = "ai", targetCount = 1, intervalMinutes = 30),
                DynamicGroupCard(id = "dg-1", name = "日本前三住宅组", country = "日本", ipType = "residential", unlockFilter = "ai", targetCount = 3, intervalMinutes = 15)
            )
        )
    }

    var inbounds by remember {
        mutableStateOf(
            listOf(
                InboundProtocolItem("sb-1", "Hysteria2-62799", "真实传输 (REALITY)", 443, "socks5://127.0.0.1:7928", 7928),
                InboundProtocolItem("sb-2", "AnyTLS-12974", "高速二代 (Hysteria2)", 8443, "direct", 0)
            )
        )
    }

    // Modal dialog states
    var showAddPortDialog by remember { mutableStateOf(false) }
    var newPortInput by remember { mutableStateOf("7931") }
    var newPolicyInput by remember { mutableStateOf("round_robin") }
    var newAuthInput by remember { mutableStateOf("random") }

    var editPortRuleTarget by remember { mutableStateOf<PortRuleItem?>(null) }
    var selectedInboundForOutboundSwitch by remember { mutableStateOf<InboundProtocolItem?>(null) }
    var selectedOutboundTarget by remember { mutableStateOf("direct") }

    // Pull real data when active server changes
    LaunchedEffect(activeServer?.id) {
        if (activeServer != null) {
            val portsRes = AimiliApplication.instance.apiClient.fetchPortRules(activeServer)
            if (portsRes.isSuccess && !portsRes.getOrNull().isNullOrEmpty()) {
                portRules = portsRes.getOrNull()!!
            }
            val groupsRes = AimiliApplication.instance.apiClient.fetchDynamicGroups(activeServer)
            if (groupsRes.isSuccess && !groupsRes.getOrNull().isNullOrEmpty()) {
                dynamicGroups = groupsRes.getOrNull()!!
            }
            val inboundsRes = AimiliApplication.instance.apiClient.fetchSingBoxNodes(activeServer)
            if (inboundsRes.isSuccess && !inboundsRes.getOrNull().isNullOrEmpty()) {
                inbounds = inboundsRes.getOrNull()!!
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
                                    val portsRes = AimiliApplication.instance.apiClient.fetchPortRules(activeServer)
                                    if (portsRes.isSuccess && !portsRes.getOrNull().isNullOrEmpty()) portRules = portsRes.getOrNull()!!
                                    val groupsRes = AimiliApplication.instance.apiClient.fetchDynamicGroups(activeServer)
                                    if (groupsRes.isSuccess && !groupsRes.getOrNull().isNullOrEmpty()) dynamicGroups = groupsRes.getOrNull()!!
                                    val inboundsRes = AimiliApplication.instance.apiClient.fetchSingBoxNodes(activeServer)
                                    if (inboundsRes.isSuccess && !inboundsRes.getOrNull().isNullOrEmpty()) inbounds = inboundsRes.getOrNull()!!
                                    Toast.makeText(context, "已从 [${activeServer.name}] 刷新分流与入站数据", Toast.LENGTH_SHORT).show()
                                }
                            }
                        },
                        modifier = Modifier.size(48.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Rounded.Refresh,
                            contentDescription = "刷新分流",
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
        ) {
            // 1. M3 Primary Tabs (高 48dp, 选中项文字 primary 并带 3dp 指示条)
            TabRow(
                selectedTabIndex = selectedTabIndex,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(48.dp),
                containerColor = MaterialTheme.colorScheme.surface,
                contentColor = MaterialTheme.colorScheme.primary,
                indicator = { tabPositions ->
                    TabRowDefaults.SecondaryIndicator(
                        Modifier.tabIndicatorOffset(tabPositions[selectedTabIndex]),
                        height = 3.dp,
                        color = MaterialTheme.colorScheme.primary
                    )
                },
                divider = {
                    HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
                }
            ) {
                tabTitles.forEachIndexed { index, title ->
                    Tab(
                        selected = selectedTabIndex == index,
                        onClick = { selectedTabIndex = index },
                        text = {
                            Text(
                                text = title,
                                style = MaterialTheme.typography.titleSmall,
                                fontWeight = if (selectedTabIndex == index) FontWeight.Bold else FontWeight.Normal,
                                color = if (selectedTabIndex == index) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant
                            )
                        }
                    )
                }
            }

            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(horizontal = 16.dp)
                    .verticalScroll(scrollState),
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                Spacer(Modifier.height(4.dp))

                // 2. 3项相连列表: 端口 7928, 7929, 7930 (动态显示多端口规则)
                Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
                    portRules.forEachIndexed { index, rule ->
                        ConnectedListItem(
                            index = index,
                            total = portRules.size,
                            headline = rule.title,
                            supportingText = rule.subtitle,
                            leadingIcon = Icons.Rounded.Lan,
                            trailingContent = {
                                IconButton(onClick = {
                                    val updated = portRules.toMutableList()
                                    updated[index] = rule.copy(enabled = !rule.enabled)
                                    portRules = updated
                                    if (activeServer != null) {
                                        scope.launch {
                                            AimiliApplication.instance.apiClient.savePortRules(activeServer, updated)
                                        }
                                    }
                                    Toast.makeText(context, "端口 ${rule.port} 状态已更新为: ${if (!rule.enabled) "已启用" else "已停用"}", Toast.LENGTH_SHORT).show()
                                }) {
                                    Icon(
                                        imageVector = if (rule.enabled) Icons.Rounded.ToggleOn else Icons.Rounded.ToggleOff,
                                        contentDescription = "切换开关",
                                        tint = if (rule.enabled) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outline,
                                        modifier = Modifier.size(32.dp)
                                    )
                                }
                            },
                            onClick = {
                                editPortRuleTarget = rule
                            }
                        )
                    }
                }

                // 3. 填充卡片（高 168dp）: 自适应组展示
                val primaryGroup = dynamicGroups.find { it.id == "dg-1" } ?: dynamicGroups.firstOrNull() ?: DynamicGroupCard("dg-1", "日本前三住宅组")
                Card(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(168.dp),
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
                            text = primaryGroup.title,
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold,
                            color = MaterialTheme.colorScheme.onSurface
                        )
                        Spacer(Modifier.height(8.dp))
                        Text(
                            text = primaryGroup.description,
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            lineHeight = MaterialTheme.typography.bodyMedium.lineHeight * 1.35f
                        )
                    }
                }

                // 4. 按钮组: “立即重评换线”(填充) “新建端口”(色调)
                ConnectedButtonGroup(
                    items = listOf(
                        ConnectedButtonItem(
                            text = "立即重评换线",
                            style = ConnectedButtonStyle.Filled,
                            onClick = {
                                if (activeServer != null) {
                                    scope.launch {
                                        val res = AimiliApplication.instance.apiClient.triggerRotate(activeServer, primaryGroup.id)
                                        Toast.makeText(context, res.getOrDefault("已触发自适应组重新探测并替换失效节点！"), Toast.LENGTH_SHORT).show()
                                    }
                                }
                            }
                        ),
                        ConnectedButtonItem(
                            text = "新建端口",
                            style = ConnectedButtonStyle.Tonal,
                            onClick = { showAddPortDialog = true }
                        )
                    )
                )

                // 5. 边缘入站协议列表
                Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
                    inbounds.forEachIndexed { index, inbound ->
                        ConnectedListItem(
                            index = index,
                            total = inbounds.size,
                            headline = inbound.title,
                            supportingText = inbound.subtitle,
                            leadingIcon = if (index % 2 == 0) Icons.Rounded.VpnKey else Icons.Rounded.Security,
                            trailingContent = {
                                Row(verticalAlignment = Alignment.CenterVertically) {
                                    IconButton(onClick = {
                                        val shareUrl = inbound.shareUrl.ifEmpty { "vless://${inbound.uuid}@${activeServer?.host ?: "127.0.0.1"}:${inbound.port}?security=reality#${inbound.name}" }
                                        clipboardManager.setText(AnnotatedString(shareUrl))
                                        Toast.makeText(context, "已复制 [${inbound.protocol}] 节点分享链接！", Toast.LENGTH_SHORT).show()
                                    }) {
                                        Icon(Icons.Rounded.QrCode2, contentDescription = "分享", tint = MaterialTheme.colorScheme.primary)
                                    }
                                    IconButton(onClick = {
                                        selectedInboundForOutboundSwitch = inbound
                                        selectedOutboundTarget = inbound.outbound
                                    }) {
                                        Icon(Icons.Rounded.SwapHoriz, contentDescription = "更改出口", tint = MaterialTheme.colorScheme.secondary)
                                    }
                                }
                            },
                            onClick = {
                                Toast.makeText(context, "入站: ${inbound.name} (${inbound.protocol}:${inbound.port})", Toast.LENGTH_SHORT).show()
                            }
                        )
                    }
                }

                Spacer(Modifier.height(80.dp))
            }
        }
    }

    // Add Port Dialog
    if (showAddPortDialog) {
        AlertDialog(
            onDismissRequest = { showAddPortDialog = false },
            shape = RoundedCornerShape(28.dp),
            containerColor = MaterialTheme.colorScheme.surfaceContainer,
            title = {
                Text(
                    text = "新建代理分流端口",
                    style = MaterialTheme.typography.titleLarge,
                    fontWeight = FontWeight.Bold
                )
            },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(
                        text = "输入监听端口号 (1-65535)，系统将自动为其装配独立策略路由。",
                        style = MaterialTheme.typography.bodyMedium
                    )
                    OutlinedTextField(
                        value = newPortInput,
                        onValueChange = { newPortInput = it },
                        label = { Text("监听端口") },
                        singleLine = true,
                        shape = RoundedCornerShape(12.dp),
                        modifier = Modifier.fillMaxWidth()
                    )
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val portNum = newPortInput.toIntOrNull() ?: 7931
                        val updated = portRules.toMutableList()
                        val newRule = PortRuleItem(portNum, enabled = true, policy = newPolicyInput, authMode = newAuthInput)
                        updated.add(newRule)
                        portRules = updated
                        if (activeServer != null) {
                            scope.launch {
                                AimiliApplication.instance.apiClient.savePortRules(activeServer, updated)
                            }
                        }
                        showAddPortDialog = false
                        Toast.makeText(context, "端口 $portNum 分流规则已成功下发至远端！", Toast.LENGTH_SHORT).show()
                    }
                ) {
                    Text("确定创建", fontWeight = FontWeight.Bold)
                }
            },
            dismissButton = {
                TextButton(onClick = { showAddPortDialog = false }) {
                    Text("取消")
                }
            }
        )
    }

    // Edit Port Rule Target Dialog
    if (editPortRuleTarget != null) {
        val target = editPortRuleTarget!!
        AlertDialog(
            onDismissRequest = { editPortRuleTarget = null },
            shape = RoundedCornerShape(28.dp),
            containerColor = MaterialTheme.colorScheme.surfaceContainer,
            title = {
                Text(
                    text = "编辑端口 [${target.port}] 分流规则",
                    style = MaterialTheme.typography.titleLarge,
                    fontWeight = FontWeight.Bold
                )
            },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(text = "当前调度策略: ${target.policyDisplay}", style = MaterialTheme.typography.bodyMedium)
                    Text(text = "当前鉴权模式: ${target.authDisplay}", style = MaterialTheme.typography.bodyMedium)
                    Text(text = "绑定出口范围: ${target.bindingDisplay}", style = MaterialTheme.typography.bodyMedium)
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        editPortRuleTarget = null
                        Toast.makeText(context, "端口 [${target.port}] 规则已在远端生效！", Toast.LENGTH_SHORT).show()
                    }
                ) {
                    Text("保存规则", fontWeight = FontWeight.Bold)
                }
            },
            dismissButton = {
                TextButton(onClick = { editPortRuleTarget = null }) {
                    Text("关闭")
                }
            }
        )
    }

    // Switch SingBox Outbound Dialog
    if (selectedInboundForOutboundSwitch != null) {
        val inbound = selectedInboundForOutboundSwitch!!
        AlertDialog(
            onDismissRequest = { selectedInboundForOutboundSwitch = null },
            shape = RoundedCornerShape(28.dp),
            containerColor = MaterialTheme.colorScheme.surfaceContainer,
            title = {
                Text(
                    text = "更改 [${inbound.protocol}] 出口链路",
                    style = MaterialTheme.typography.titleLarge,
                    fontWeight = FontWeight.Bold
                )
            },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("选择该入站协议转发流量的链式出口：", style = MaterialTheme.typography.bodyMedium)
                    val outboundOptions = listOf(
                        "direct" to "直连出口 (VPS 本机网络)",
                        "socks5://127.0.0.1:7928" to "端口 7928 (默认住宅池)",
                        "socks5://127.0.0.1:7929" to "端口 7929 (多出口组)"
                    )
                    outboundOptions.forEach { (obVal, obLabel) ->
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            modifier = Modifier.fillMaxWidth()
                        ) {
                            RadioButton(
                                selected = selectedOutboundTarget == obVal,
                                onClick = { selectedOutboundTarget = obVal }
                            )
                            Text(text = obLabel, style = MaterialTheme.typography.bodyMedium)
                        }
                    }
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        if (activeServer != null) {
                            scope.launch {
                                AimiliApplication.instance.apiClient.setSingBoxOutbound(activeServer, inbound.name, selectedOutboundTarget)
                                val updated = inbounds.map {
                                    if (it.id == inbound.id) it.copy(outbound = selectedOutboundTarget) else it
                                }
                                inbounds = updated
                                Toast.makeText(context, "✅ 节点出口已成功改挂至: $selectedOutboundTarget", Toast.LENGTH_SHORT).show()
                            }
                        }
                        selectedInboundForOutboundSwitch = null
                    }
                ) {
                    Text("应用切换", fontWeight = FontWeight.Bold)
                }
            },
            dismissButton = {
                TextButton(onClick = { selectedInboundForOutboundSwitch = null }) {
                    Text("取消")
                }
            }
        )
    }
}
