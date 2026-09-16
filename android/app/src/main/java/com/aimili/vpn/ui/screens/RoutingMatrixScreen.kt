package com.aimili.vpn.ui.screens

import android.content.res.Configuration
import android.widget.Toast
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.Delete
import androidx.compose.material.icons.rounded.Edit
import androidx.compose.material.icons.rounded.Lan
import androidx.compose.material.icons.rounded.QrCode2
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material.icons.rounded.Security
import androidx.compose.material.icons.rounded.SwapHoriz
import androidx.compose.material.icons.rounded.ToggleOff
import androidx.compose.material.icons.rounded.ToggleOn
import androidx.compose.material.icons.rounded.VpnKey
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Checkbox
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
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
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.aimili.vpn.AimiliApplication
import com.aimili.vpn.model.DynamicGroupCard
import com.aimili.vpn.model.InboundProtocolItem
import com.aimili.vpn.model.PortRuleItem
import com.aimili.vpn.model.ServerProfile
import com.aimili.vpn.model.TunnelItem
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

    val configuration = LocalConfiguration.current
    val isLandscape = configuration.orientation == Configuration.ORIENTATION_LANDSCAPE
    val isTablet = configuration.screenWidthDp >= 600
    val isTabletLandscape = isTablet && isLandscape

    // Tab state: 0 = 多端口, 1 = 自适应组, 2 = 边缘入站
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
                InboundProtocolItem("sb-1", "Hysteria2-62799", "VLESS-REALITY", 443, "socks5://127.0.0.1:7928", 7928),
                InboundProtocolItem("sb-2", "AnyTLS-12974", "Hysteria2", 8443, "direct", 0),
                InboundProtocolItem("sb-3", "TUIC-19078", "TUIC v5", 19078, "socks5://127.0.0.1:7929", 7929)
            )
        )
    }

    var activeTunnels by remember {
        mutableStateOf<List<TunnelItem>>(emptyList())
    }

    // Modal dialog states: Port Rules (Full CRUD)
    var showPortDialog by remember { mutableStateOf(false) }
    var editPortRuleTarget by remember { mutableStateOf<PortRuleItem?>(null) }
    var inputPortNum by remember { mutableStateOf("7931") }
    var inputPolicy by remember { mutableStateOf("round_robin") }
    var inputIntervalSec by remember { mutableStateOf("300") }
    var inputAuthMode by remember { mutableStateOf("random") }
    var inputAuthUser by remember { mutableStateOf("") }
    var inputAuthPass by remember { mutableStateOf("") }
    var bindAllTunnels by remember { mutableStateOf(true) }
    var selectedBoundGroups by remember { mutableStateOf(setOf<String>()) }
    var deletePortRuleCandidate by remember { mutableStateOf<PortRuleItem?>(null) }

    // Modal dialog states: Dynamic Groups (Full CRUD)
    var showGroupDialog by remember { mutableStateOf(false) }
    var editGroupTarget by remember { mutableStateOf<DynamicGroupCard?>(null) }
    var deleteGroupCandidate by remember { mutableStateOf<DynamicGroupCard?>(null) }
    var groupNameInput by remember { mutableStateOf("日本Top3住宅组") }
    var groupCountryInput by remember { mutableStateOf("JP") }
    var groupIpTypeInput by remember { mutableStateOf("residential") }
    var groupUnlockInput by remember { mutableStateOf("ai") }
    var groupSortInput by remember { mutableStateOf("latency") }
    var groupTargetCountInput by remember { mutableStateOf("3") }
    var groupIntervalInput by remember { mutableStateOf("15") }

    // Modal dialog states: SingBox Inbounds (Full CRUD with 22 Protocols)
    var showAddInboundDialog by remember { mutableStateOf(false) }
    var selectedProtoCat by remember { mutableIntStateOf(0) }
    var newInboundProtocol by remember { mutableStateOf("reality") }
    var newInboundPort by remember { mutableStateOf("auto") }
    var newInboundOutbound by remember { mutableStateOf("socks5://127.0.0.1:7928") }
    var selectedInboundForOutboundSwitch by remember { mutableStateOf<InboundProtocolItem?>(null) }
    var selectedOutboundTarget by remember { mutableStateOf("direct") }
    var deleteInboundCandidate by remember { mutableStateOf<InboundProtocolItem?>(null) }
    var showInboundQRDialog by remember { mutableStateOf<InboundProtocolItem?>(null) }

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
            val tunnelsRes = AimiliApplication.instance.apiClient.fetchTunnels(activeServer)
            if (tunnelsRes.isSuccess && !tunnelsRes.getOrNull().isNullOrEmpty()) {
                activeTunnels = tunnelsRes.getOrNull()!!
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
                                    Toast.makeText(context, "已从 [${activeServer.name}] 同步最新调度分流数据", Toast.LENGTH_SHORT).show()
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
            // 1. M3 Primary Tabs (多端口 / 自适应组 / 边缘入站)
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

            // Tab Content Router with Tablet Centering
            Box(
                modifier = Modifier
                    .fillMaxSize()
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

                    AnimatedContent(
                        targetState = selectedTabIndex,
                        transitionSpec = { fadeIn() togetherWith fadeOut() },
                        label = "RoutingTabContent"
                    ) { tabIndex ->
                        when (tabIndex) {
                            // ==================== TAB 0: 多端口分流矩阵 (全增删改查) ====================
                            0 -> {
                                Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
                                    // 端口列表项 (带启停切换、编辑抽屉、删除操作)
                                    Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
                                        portRules.forEachIndexed { index, rule ->
                                            ConnectedListItem(
                                                index = index,
                                                total = portRules.size,
                                                headline = rule.title,
                                                supportingText = rule.subtitle,
                                                leadingIcon = Icons.Rounded.Lan,
                                                trailingContent = {
                                                    Row(verticalAlignment = Alignment.CenterVertically) {
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
                                                        IconButton(onClick = { deletePortRuleCandidate = rule }) {
                                                            Icon(Icons.Rounded.Delete, contentDescription = "删除端口规则", tint = MaterialTheme.colorScheme.error)
                                                        }
                                                    }
                                                },
                                                onClick = {
                                                    editPortRuleTarget = rule
                                                    inputPortNum = rule.port.toString()
                                                    inputPolicy = rule.policy
                                                    inputIntervalSec = rule.intervalSeconds.toString()
                                                    inputAuthMode = rule.authMode
                                                    inputAuthUser = rule.authUser
                                                    inputAuthPass = rule.authPass
                                                    bindAllTunnels = rule.boundGroupIds.isEmpty() && rule.boundTunnelIds.isEmpty()
                                                    selectedBoundGroups = rule.boundGroupIds.toSet()
                                                    showPortDialog = true
                                                }
                                            )
                                        }
                                    }

                                    // 新建端口操作组
                                    ConnectedButtonGroup(
                                        items = listOf(
                                            ConnectedButtonItem(
                                                text = "新建代理端口",
                                                style = ConnectedButtonStyle.Filled,
                                                icon = Icons.Rounded.Add,
                                                onClick = {
                                                    editPortRuleTarget = null
                                                    inputPortNum = (portRules.maxOfOrNull { it.port }?.plus(1) ?: 7931).toString()
                                                    inputPolicy = "round_robin"
                                                    inputIntervalSec = "300"
                                                    inputAuthMode = "random"
                                                    inputAuthUser = ""
                                                    inputAuthPass = ""
                                                    bindAllTunnels = true
                                                    selectedBoundGroups = emptySet()
                                                    showPortDialog = true
                                                }
                                            ),
                                            ConnectedButtonItem(
                                                text = "刷新端口状态",
                                                style = ConnectedButtonStyle.Tonal,
                                                icon = Icons.Rounded.Refresh,
                                                onClick = {
                                                    if (activeServer != null) {
                                                        scope.launch {
                                                            val res = AimiliApplication.instance.apiClient.fetchPortRules(activeServer)
                                                            if (res.isSuccess) portRules = res.getOrNull() ?: portRules
                                                            Toast.makeText(context, "已从服务器获取最新端口分流规则！", Toast.LENGTH_SHORT).show()
                                                        }
                                                    }
                                                }
                                            )
                                        )
                                    )
                                }
                            }

                            // ==================== TAB 1: 动态自适应组 (全增删改查) ====================
                            1 -> {
                                Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
                                    dynamicGroups.forEach { group ->
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
                                                        Text(
                                                            text = if (group.isSystem) "系统主出口(tun0)" else "自适应池 (${group.targetCount}网卡)",
                                                            style = MaterialTheme.typography.labelSmall,
                                                            color = MaterialTheme.colorScheme.primary
                                                        )
                                                        if (!group.isSystem) {
                                                            IconButton(
                                                                onClick = { deleteGroupCandidate = group },
                                                                modifier = Modifier.size(32.dp)
                                                            ) {
                                                                Icon(Icons.Rounded.Delete, contentDescription = "删除组", tint = MaterialTheme.colorScheme.error, modifier = Modifier.size(18.dp))
                                                            }
                                                        }
                                                    }
                                                }
                                                Spacer(Modifier.height(8.dp))
                                                Text(
                                                    text = group.description,
                                                    style = MaterialTheme.typography.bodyMedium,
                                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                                    lineHeight = MaterialTheme.typography.bodyMedium.lineHeight * 1.35f
                                                )
                                                Spacer(Modifier.height(12.dp))
                                                Row(
                                                    modifier = Modifier.fillMaxWidth(),
                                                    horizontalArrangement = Arrangement.End
                                                ) {
                                                    OutlinedButton(
                                                        onClick = {
                                                            editGroupTarget = group
                                                            groupNameInput = group.name
                                                            groupCountryInput = group.country
                                                            groupIpTypeInput = group.ipType
                                                            groupUnlockInput = group.unlockFilter
                                                            groupSortInput = group.sortBy
                                                            groupTargetCountInput = group.targetCount.toString()
                                                            groupIntervalInput = group.intervalMinutes.toString()
                                                            showGroupDialog = true
                                                        },
                                                        modifier = Modifier.height(36.dp),
                                                        shape = RoundedCornerShape(18.dp)
                                                    ) {
                                                        Icon(Icons.Rounded.Edit, contentDescription = null, modifier = Modifier.size(16.dp))
                                                        Spacer(Modifier.width(4.dp))
                                                        Text("编辑规则", style = MaterialTheme.typography.labelMedium)
                                                    }
                                                }
                                            }
                                        }
                                    }

                                    // 自适应组操作组
                                    ConnectedButtonGroup(
                                        items = listOf(
                                            ConnectedButtonItem(
                                                text = "立即重评换线",
                                                style = ConnectedButtonStyle.Filled,
                                                icon = Icons.Rounded.Refresh,
                                                onClick = {
                                                    if (activeServer != null) {
                                                        scope.launch {
                                                            val res = AimiliApplication.instance.apiClient.triggerRotate(activeServer)
                                                            Toast.makeText(context, res.getOrDefault("已触发自适应组重新测速探测并替换失效节点！"), Toast.LENGTH_SHORT).show()
                                                        }
                                                    }
                                                }
                                            ),
                                            ConnectedButtonItem(
                                                text = "新建自适应组",
                                                style = ConnectedButtonStyle.Tonal,
                                                icon = Icons.Rounded.Add,
                                                onClick = {
                                                    editGroupTarget = null
                                                    groupNameInput = "新自适应住宅池"
                                                    groupCountryInput = "JP"
                                                    groupIpTypeInput = "residential"
                                                    groupUnlockInput = "ai"
                                                    groupSortInput = "latency"
                                                    groupTargetCountInput = "3"
                                                    groupIntervalInput = "15"
                                                    showGroupDialog = true
                                                }
                                            )
                                        )
                                    )
                                }
                            }

                            // ==================== TAB 2: 边缘抗封锁入站 (22 种协议全管理) ====================
                            2 -> {
                                Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
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
                                                        IconButton(onClick = { showInboundQRDialog = inbound }) {
                                                            Icon(Icons.Rounded.QrCode2, contentDescription = "分享", tint = MaterialTheme.colorScheme.primary)
                                                        }
                                                        IconButton(onClick = {
                                                            selectedInboundForOutboundSwitch = inbound
                                                            selectedOutboundTarget = inbound.outbound
                                                        }) {
                                                            Icon(Icons.Rounded.SwapHoriz, contentDescription = "更改出口", tint = MaterialTheme.colorScheme.secondary)
                                                        }
                                                        IconButton(onClick = { deleteInboundCandidate = inbound }) {
                                                            Icon(Icons.Rounded.Delete, contentDescription = "删除入站", tint = MaterialTheme.colorScheme.error)
                                                        }
                                                    }
                                                },
                                                onClick = {
                                                    Toast.makeText(context, "入站: ${inbound.name} (端口 ${inbound.port})", Toast.LENGTH_SHORT).show()
                                                }
                                            )
                                        }
                                    }

                                    // 入站协议操作组
                                    ConnectedButtonGroup(
                                        items = listOf(
                                            ConnectedButtonItem(
                                                text = "新建入站节点",
                                                style = ConnectedButtonStyle.Filled,
                                                icon = Icons.Rounded.Add,
                                                onClick = {
                                                    newInboundProtocol = "reality"
                                                    newInboundPort = "auto"
                                                    newInboundOutbound = "socks5://127.0.0.1:7928"
                                                    showAddInboundDialog = true
                                                }
                                            ),
                                            ConnectedButtonItem(
                                                text = "复制 Clash 订阅",
                                                style = ConnectedButtonStyle.Tonal,
                                                onClick = {
                                                    if (activeServer != null) {
                                                        scope.launch {
                                                            val clashUrl = "${activeServer.baseUrl}/api/singbox/subscription/clash"
                                                            clipboardManager.setText(AnnotatedString(clashUrl))
                                                            Toast.makeText(context, "已复制 Clash Meta 完整分流订阅！", Toast.LENGTH_SHORT).show()
                                                        }
                                                    }
                                                }
                                            )
                                        )
                                    )
                                }
                            }
                        }
                    }

                    Spacer(Modifier.height(80.dp))
                }
            }
        }
    }

    // ======================== MODAL DIALOGS ========================

    // 1. Add / Edit Port Dialog (包含自适应端口组与指定隧道绑定多选)
    if (showPortDialog) {
        val isEditing = editPortRuleTarget != null
        AlertDialog(
            onDismissRequest = {
                showPortDialog = false
                editPortRuleTarget = null
            },
            shape = RoundedCornerShape(28.dp),
            containerColor = MaterialTheme.colorScheme.surfaceContainer,
            title = {
                Text(
                    text = if (isEditing) "编辑端口 [${editPortRuleTarget?.port}] 规则" else "新建代理分流端口",
                    style = MaterialTheme.typography.titleLarge,
                    fontWeight = FontWeight.Bold
                )
            },
            text = {
                Column(
                    modifier = Modifier.verticalScroll(rememberScrollState()),
                    verticalArrangement = Arrangement.spacedBy(10.dp)
                ) {
                    OutlinedTextField(
                        value = inputPortNum,
                        onValueChange = { inputPortNum = it },
                        label = { Text("监听端口 [1-65535]") },
                        singleLine = true,
                        enabled = !isEditing,
                        modifier = Modifier.fillMaxWidth()
                    )

                    Text("调度策略 (分流算法):", style = MaterialTheme.typography.labelMedium)
                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                        listOf("round_robin" to "轮询", "interval" to "定时", "random" to "随机").forEach { (pol, label) ->
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                RadioButton(selected = inputPolicy == pol, onClick = { inputPolicy = pol })
                                Text(label, style = MaterialTheme.typography.bodySmall)
                            }
                        }
                    }

                    if (inputPolicy == "interval") {
                        OutlinedTextField(
                            value = inputIntervalSec,
                            onValueChange = { inputIntervalSec = it },
                            label = { Text("轮换周期 (秒)") },
                            singleLine = true,
                            modifier = Modifier.fillMaxWidth()
                        )
                    }

                    Text("鉴权模式:", style = MaterialTheme.typography.labelMedium)
                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                        listOf("random" to "系统随机", "none" to "免密", "custom" to "自定义").forEach { (auth, label) ->
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                RadioButton(selected = inputAuthMode == auth, onClick = { inputAuthMode = auth })
                                Text(label, style = MaterialTheme.typography.bodySmall)
                            }
                        }
                    }

                    if (inputAuthMode == "custom") {
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            OutlinedTextField(
                                value = inputAuthUser,
                                onValueChange = { inputAuthUser = it },
                                label = { Text("用户名") },
                                singleLine = true,
                                modifier = Modifier.weight(1f)
                            )
                            OutlinedTextField(
                                value = inputAuthPass,
                                onValueChange = { inputAuthPass = it },
                                label = { Text("密码") },
                                singleLine = true,
                                modifier = Modifier.weight(1f)
                            )
                        }
                    }

                    HorizontalDivider(modifier = Modifier.padding(vertical = 4.dp))

                    // 绑定自适应组与出口部分 (参照 Web 控制台实现)
                    Text("绑定出网出口范围:", style = MaterialTheme.typography.labelMedium, fontWeight = FontWeight.Bold)

                    // Checkbox 1: 全部在线隧道
                    Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.fillMaxWidth()) {
                        Checkbox(
                            checked = bindAllTunnels,
                            onCheckedChange = {
                                bindAllTunnels = it
                                if (it) selectedBoundGroups = emptySet()
                            }
                        )
                        Text("全部在线隧道 (默认全池负载均衡)", style = MaterialTheme.typography.bodyMedium)
                    }

                    // Checkbox 2: 绑定自适应端口组
                    if (dynamicGroups.isNotEmpty()) {
                        Text("自适应动态出口组:", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.primary)
                        dynamicGroups.forEach { group ->
                            val isChecked = !bindAllTunnels && selectedBoundGroups.contains(group.id)
                            Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.fillMaxWidth()) {
                                Checkbox(
                                    checked = isChecked,
                                    onCheckedChange = { checked ->
                                        bindAllTunnels = false
                                        selectedBoundGroups = if (checked) selectedBoundGroups + group.id else selectedBoundGroups - group.id
                                    }
                                )
                                Text("${group.name} (${group.country} · ${group.targetCount}出口)", style = MaterialTheme.typography.bodySmall)
                            }
                        }
                    }
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val portNum = inputPortNum.toIntOrNull() ?: 7931
                        val intervalSec = inputIntervalSec.toIntOrNull() ?: 300
                        val updated = portRules.toMutableList()
                        val boundGroupList = if (bindAllTunnels) emptyList() else selectedBoundGroups.toList()

                        val rule = PortRuleItem(
                            port = portNum,
                            enabled = true,
                            policy = inputPolicy,
                            intervalSeconds = intervalSec,
                            authMode = inputAuthMode,
                            authUser = inputAuthUser,
                            authPass = inputAuthPass,
                            boundGroupIds = boundGroupList
                        )
                        val existIdx = updated.indexOfFirst { it.port == portNum }
                        if (existIdx >= 0) {
                            updated[existIdx] = rule
                        } else {
                            updated.add(rule)
                        }
                        portRules = updated
                        if (activeServer != null) {
                            scope.launch {
                                AimiliApplication.instance.apiClient.savePortRules(activeServer, updated)
                            }
                        }
                        showPortDialog = false
                        editPortRuleTarget = null
                        Toast.makeText(context, "端口 [$portNum] 分流规则与出口绑定已下发！", Toast.LENGTH_SHORT).show()
                    }
                ) {
                    Text("保存生效", fontWeight = FontWeight.Bold)
                }
            },
            dismissButton = {
                TextButton(onClick = {
                    showPortDialog = false
                    editPortRuleTarget = null
                }) {
                    Text("取消")
                }
            }
        )
    }

    // 2. Delete Port Rule Dialog
    if (deletePortRuleCandidate != null) {
        val target = deletePortRuleCandidate!!
        AlertDialog(
            onDismissRequest = { deletePortRuleCandidate = null },
            shape = RoundedCornerShape(28.dp),
            containerColor = MaterialTheme.colorScheme.surfaceContainer,
            title = { Text("确认删除端口 [${target.port}] 分流规则？", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold) },
            text = { Text("删除后将停止该端口的监听与分流转发，确定删除吗？", style = MaterialTheme.typography.bodyMedium) },
            confirmButton = {
                TextButton(
                    onClick = {
                        val updated = portRules.filter { it.port != target.port }
                        portRules = updated
                        if (activeServer != null) {
                            scope.launch {
                                AimiliApplication.instance.apiClient.savePortRules(activeServer, updated)
                            }
                        }
                        deletePortRuleCandidate = null
                        Toast.makeText(context, "已删除端口 [${target.port}] 规则！", Toast.LENGTH_SHORT).show()
                    }
                ) {
                    Text("确认删除", color = MaterialTheme.colorScheme.error, fontWeight = FontWeight.Bold)
                }
            },
            dismissButton = {
                TextButton(onClick = { deletePortRuleCandidate = null }) { Text("取消") }
            }
        )
    }

    // 3. Add / Edit Dynamic Group Dialog
    if (showGroupDialog) {
        val isEditing = editGroupTarget != null
        AlertDialog(
            onDismissRequest = {
                showGroupDialog = false
                editGroupTarget = null
            },
            shape = RoundedCornerShape(28.dp),
            containerColor = MaterialTheme.colorScheme.surfaceContainer,
            title = {
                Text(
                    text = if (isEditing) "编辑自适应组 [${editGroupTarget?.name}]" else "新建自适应动态组",
                    style = MaterialTheme.typography.titleLarge,
                    fontWeight = FontWeight.Bold
                )
            },
            text = {
                Column(
                    modifier = Modifier.verticalScroll(rememberScrollState()),
                    verticalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    OutlinedTextField(
                        value = groupNameInput,
                        onValueChange = { groupNameInput = it },
                        label = { Text("组名称") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth()
                    )
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        OutlinedTextField(
                            value = groupCountryInput,
                            onValueChange = { groupCountryInput = it },
                            label = { Text("国家 (如 JP, US, ALL)") },
                            singleLine = true,
                            modifier = Modifier.weight(1f)
                        )
                        OutlinedTextField(
                            value = groupTargetCountInput,
                            onValueChange = { groupTargetCountInput = it },
                            label = { Text("维持并发数") },
                            singleLine = true,
                            modifier = Modifier.weight(1f)
                        )
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        OutlinedTextField(
                            value = groupIntervalInput,
                            onValueChange = { groupIntervalInput = it },
                            label = { Text("重评周期 (分钟)") },
                            singleLine = true,
                            modifier = Modifier.weight(1f)
                        )
                        OutlinedTextField(
                            value = groupUnlockInput,
                            onValueChange = { groupUnlockInput = it },
                            label = { Text("解锁过滤 (ai/all/none)") },
                            singleLine = true,
                            modifier = Modifier.weight(1f)
                        )
                    }
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val count = groupTargetCountInput.toIntOrNull() ?: 3
                        val interval = groupIntervalInput.toIntOrNull() ?: 15
                        val targetId = editGroupTarget?.id ?: "dg-${System.currentTimeMillis() % 100000}"
                        val newG = DynamicGroupCard(
                            id = targetId,
                            name = groupNameInput.trim().ifEmpty { "自适应组" },
                            country = groupCountryInput.trim().uppercase(),
                            targetCount = count,
                            intervalMinutes = interval,
                            ipType = groupIpTypeInput,
                            unlockFilter = groupUnlockInput,
                            sortBy = groupSortInput,
                            isSystem = editGroupTarget?.isSystem ?: false
                        )
                        val updated = dynamicGroups.toMutableList()
                        val existIdx = updated.indexOfFirst { it.id == targetId }
                        if (existIdx >= 0) updated[existIdx] = newG else updated.add(newG)
                        dynamicGroups = updated

                        if (activeServer != null) {
                            scope.launch {
                                AimiliApplication.instance.apiClient.saveDynamicGroup(activeServer, newG)
                            }
                        }
                        showGroupDialog = false
                        editGroupTarget = null
                        Toast.makeText(context, "自适应组 [${newG.name}] 已成功保存下发生效！", Toast.LENGTH_SHORT).show()
                    }
                ) {
                    Text("保存生效", fontWeight = FontWeight.Bold)
                }
            },
            dismissButton = {
                TextButton(onClick = {
                    showGroupDialog = false
                    editGroupTarget = null
                }) {
                    Text("取消")
                }
            }
        )
    }

    // 4. Delete Dynamic Group Dialog
    if (deleteGroupCandidate != null) {
        val target = deleteGroupCandidate!!
        AlertDialog(
            onDismissRequest = { deleteGroupCandidate = null },
            shape = RoundedCornerShape(28.dp),
            containerColor = MaterialTheme.colorScheme.surfaceContainer,
            title = { Text("确认删除自适应组？", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold) },
            text = { Text("删除组 [${target.name}] 将同步释放其维护的所有并发隧道，确定删除吗？", style = MaterialTheme.typography.bodyMedium) },
            confirmButton = {
                TextButton(
                    onClick = {
                        dynamicGroups = dynamicGroups.filter { it.id != target.id }
                        if (activeServer != null) {
                            scope.launch {
                                AimiliApplication.instance.apiClient.deleteDynamicGroup(activeServer, target.id)
                            }
                        }
                        deleteGroupCandidate = null
                        Toast.makeText(context, "已成功删除自适应组 [${target.name}]！", Toast.LENGTH_SHORT).show()
                    }
                ) {
                    Text("确认删除", color = MaterialTheme.colorScheme.error, fontWeight = FontWeight.Bold)
                }
            },
            dismissButton = {
                TextButton(onClick = { deleteGroupCandidate = null }) { Text("取消") }
            }
        )
    }

    // 5. Add SingBox Inbound Protocol Dialog (完整 22 种协议支持)
    if (showAddInboundDialog) {
        val protoCategories = listOf(
            "推荐免域名" to listOf(
                "reality" to "VLESS-REALITY (免域名)",
                "rh2" to "VLESS-H2-REALITY (多路复用)",
                "hy2" to "Hysteria2 (QUIC丢包克星)",
                "tuic" to "TUIC v5 (BBR拥塞控制)",
                "ss" to "Shadowsocks 2022"
            ),
            "经典穿透" to listOf(
                "trojan" to "Trojan (经典HTTPS伪装)",
                "anytls" to "AnyTLS (自适应特征)",
                "socks" to "Socks5 (标准局域网代理)",
                "direct" to "Direct (端口转发中继)"
            ),
            "CDN/域名TLS" to listOf(
                "vws" to "VLESS-WS-TLS (CDN优选)",
                "wss" to "VMess-WS-TLS (救砖方案)",
                "tws" to "Trojan-WS-TLS",
                "vhu" to "VLESS-HTTPUpgrade-TLS",
                "hu" to "VMess-HTTPUpgrade-TLS",
                "thu" to "Trojan-HTTPUpgrade-TLS",
                "vh2" to "VLESS-H2-TLS",
                "h2" to "VMess-H2-TLS",
                "th2" to "Trojan-H2-TLS"
            ),
            "原生传输" to listOf(
                "ws" to "VMess-WS (明文反代)",
                "tcp" to "VMess-TCP",
                "http" to "HTTP 代理",
                "quic" to "VMess-QUIC"
            )
        )

        AlertDialog(
            onDismissRequest = { showAddInboundDialog = false },
            shape = RoundedCornerShape(28.dp),
            containerColor = MaterialTheme.colorScheme.surfaceContainer,
            title = { Text("新建边缘抗封锁入站节点", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold) },
            text = {
                Column(
                    modifier = Modifier.verticalScroll(rememberScrollState()),
                    verticalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    // Category selector
                    Row(
                        modifier = Modifier.horizontalScroll(rememberScrollState()),
                        horizontalArrangement = Arrangement.spacedBy(4.dp)
                    ) {
                        protoCategories.forEachIndexed { idx, (catName, _) ->
                            Surface(
                                shape = RoundedCornerShape(8.dp),
                                color = if (selectedProtoCat == idx) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surfaceContainerHigh,
                                modifier = Modifier.clickable { selectedProtoCat = idx }
                            ) {
                                Text(
                                    text = catName,
                                    style = MaterialTheme.typography.labelMedium,
                                    color = if (selectedProtoCat == idx) MaterialTheme.colorScheme.onPrimaryContainer else MaterialTheme.colorScheme.onSurfaceVariant,
                                    modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp)
                                )
                            }
                        }
                    }

                    Text("选择协议类型:", style = MaterialTheme.typography.labelMedium)
                    val currentProtos = protoCategories[selectedProtoCat].second
                    currentProtos.forEach { (pKey, pLabel) ->
                        Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.fillMaxWidth()) {
                            RadioButton(selected = newInboundProtocol == pKey, onClick = { newInboundProtocol = pKey })
                            Text(pLabel, style = MaterialTheme.typography.bodySmall)
                        }
                    }

                    OutlinedTextField(
                        value = newInboundPort,
                        onValueChange = { newInboundPort = it },
                        label = { Text("外部端口 (填 auto 自动挑选可用高位端口)") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth()
                    )

                    Text("出口链式分流指向:", style = MaterialTheme.typography.labelMedium)
                    val outboundChoices = listOf(
                        "socks5://127.0.0.1:7928" to "端口 7928 (默认住宅池)",
                        "direct" to "直连出口 (VPS 本机原生网络)"
                    )
                    outboundChoices.forEach { (obVal, obLabel) ->
                        Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.fillMaxWidth()) {
                            RadioButton(selected = newInboundOutbound == obVal, onClick = { newInboundOutbound = obVal })
                            Text(obLabel, style = MaterialTheme.typography.bodySmall)
                        }
                    }
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        if (activeServer != null) {
                            scope.launch {
                                val res = AimiliApplication.instance.apiClient.addSingBoxNode(activeServer, newInboundProtocol, newInboundPort, newInboundOutbound)
                                if (res.isSuccess) {
                                    val refreshRes = AimiliApplication.instance.apiClient.fetchSingBoxNodes(activeServer)
                                    if (refreshRes.isSuccess) inbounds = refreshRes.getOrNull() ?: inbounds
                                    Toast.makeText(context, "✅ 边缘入站节点已成功在服务器创建并启动！", Toast.LENGTH_SHORT).show()
                                } else {
                                    Toast.makeText(context, "创建失败: ${res.exceptionOrNull()?.message}", Toast.LENGTH_SHORT).show()
                                }
                            }
                        }
                        showAddInboundDialog = false
                    }
                ) {
                    Text("确定创建", fontWeight = FontWeight.Bold)
                }
            },
            dismissButton = {
                TextButton(onClick = { showAddInboundDialog = false }) { Text("取消") }
            }
        )
    }

    // 6. Switch SingBox Outbound Dialog
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

    // 7. Delete SingBox Node Dialog
    if (deleteInboundCandidate != null) {
        val target = deleteInboundCandidate!!
        AlertDialog(
            onDismissRequest = { deleteInboundCandidate = null },
            shape = RoundedCornerShape(28.dp),
            containerColor = MaterialTheme.colorScheme.surfaceContainer,
            title = { Text("确认删除入站协议？", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold) },
            text = { Text("确认彻底删除协议节点 [${target.name}] (端口 ${target.port}) 吗？", style = MaterialTheme.typography.bodyMedium) },
            confirmButton = {
                TextButton(
                    onClick = {
                        inbounds = inbounds.filter { it.id != target.id }
                        if (activeServer != null) {
                            scope.launch {
                                AimiliApplication.instance.apiClient.deleteSingBoxNode(activeServer, target.name)
                            }
                        }
                        deleteInboundCandidate = null
                        Toast.makeText(context, "已删除入站节点 [${target.name}]！", Toast.LENGTH_SHORT).show()
                    }
                ) {
                    Text("确认删除", color = MaterialTheme.colorScheme.error, fontWeight = FontWeight.Bold)
                }
            },
            dismissButton = {
                TextButton(onClick = { deleteInboundCandidate = null }) { Text("取消") }
            }
        )
    }

    // 8. SingBox Inbound QR Code & Share Link Dialog
    if (showInboundQRDialog != null) {
        val inbound = showInboundQRDialog!!
        val shareUrl = inbound.shareUrl.ifEmpty { "vless://${inbound.uuid}@${activeServer?.host ?: "127.0.0.1"}:${inbound.port}?security=reality#${inbound.name}" }
        AlertDialog(
            onDismissRequest = { showInboundQRDialog = null },
            shape = RoundedCornerShape(28.dp),
            containerColor = MaterialTheme.colorScheme.surfaceContainer,
            title = { Text("客户端配置分享与导入", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text("协议: ${inbound.protocol} (外部监听端口: ${inbound.port})", style = MaterialTheme.typography.bodyMedium)
                    OutlinedTextField(
                        value = shareUrl,
                        onValueChange = {},
                        readOnly = true,
                        label = { Text("完整分享链接 (URL)") },
                        maxLines = 4,
                        modifier = Modifier.fillMaxWidth()
                    )
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        clipboardManager.setText(AnnotatedString(shareUrl))
                        Toast.makeText(context, "已复制节点配置链接！", Toast.LENGTH_SHORT).show()
                        showInboundQRDialog = null
                    }
                ) {
                    Text("复制链接", fontWeight = FontWeight.Bold)
                }
            },
            dismissButton = {
                TextButton(onClick = { showInboundQRDialog = null }) { Text("关闭") }
            }
        )
    }
}
