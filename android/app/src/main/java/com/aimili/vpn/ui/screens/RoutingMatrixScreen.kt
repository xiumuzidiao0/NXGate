package com.aimili.vpn.ui.screens

import android.content.res.Configuration
import android.widget.Toast
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
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
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.itemsIndexed
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
                InboundProtocolItem("sb-1", "Hysteria2-62799", "真实传输 (REALITY)", 443, "socks5://127.0.0.1:7928", 7928),
                InboundProtocolItem("sb-2", "AnyTLS-12974", "高速二代 (Hysteria2)", 8443, "direct", 0)
            )
        )
    }

    // Modal dialog states: Port Rules
    var showAddPortDialog by remember { mutableStateOf(false) }
    var newPortInput by remember { mutableStateOf("7931") }
    var newPolicyInput by remember { mutableStateOf("round_robin") }
    var newIntervalInput by remember { mutableStateOf("300") }
    var newAuthInput by remember { mutableStateOf("random") }
    var newAuthUserInput by remember { mutableStateOf("") }
    var newAuthPassInput by remember { mutableStateOf("") }
    var editPortRuleTarget by remember { mutableStateOf<PortRuleItem?>(null) }
    var deletePortRuleCandidate by remember { mutableStateOf<PortRuleItem?>(null) }

    // Modal dialog states: Dynamic Groups
    var showAddGroupDialog by remember { mutableStateOf(false) }
    var editGroupTarget by remember { mutableStateOf<DynamicGroupCard?>(null) }
    var deleteGroupCandidate by remember { mutableStateOf<DynamicGroupCard?>(null) }
    var groupNameInput by remember { mutableStateOf("新自适应住宅池") }
    var groupCountryInput by remember { mutableStateOf("JP") }
    var groupIpTypeInput by remember { mutableStateOf("residential") }
    var groupUnlockInput by remember { mutableStateOf("ai") }
    var groupSortInput by remember { mutableStateOf("latency") }
    var groupTargetCountInput by remember { mutableStateOf("3") }
    var groupIntervalInput by remember { mutableStateOf("15") }

    // Modal dialog states: SingBox Inbounds
    var showAddInboundDialog by remember { mutableStateOf(false) }
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
                                    Toast.makeText(context, "已从 [${activeServer.name}] 刷新调度与入站数据", Toast.LENGTH_SHORT).show()
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

            // Tab Content Router
            Box(
                modifier = Modifier
                    .fillMaxSize()
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

                    AnimatedContent(
                        targetState = selectedTabIndex,
                        transitionSpec = { fadeIn() togetherWith fadeOut() },
                        label = "RoutingTabContent"
                    ) { tabIndex ->
                        when (tabIndex) {
                            // ==================== TAB 0: 多端口分流矩阵 (全增删改查) ====================
                            0 -> {
                                Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
                                    // 端口列表项
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
                                                            Toast.makeText(context, "端口 ${rule.port} 状态已切换为: ${if (!rule.enabled) "已启用" else "已停用"}", Toast.LENGTH_SHORT).show()
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
                                                    newPortInput = rule.port.toString()
                                                    newPolicyInput = rule.policy
                                                    newIntervalInput = rule.intervalSeconds.toString()
                                                    newAuthInput = rule.authMode
                                                    newAuthUserInput = rule.authUser
                                                    newAuthPassInput = rule.authPass
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
                                                    newPortInput = (portRules.maxOfOrNull { it.port }?.plus(1) ?: 7931).toString()
                                                    newPolicyInput = "round_robin"
                                                    newIntervalInput = "300"
                                                    newAuthInput = "random"
                                                    newAuthUserInput = ""
                                                    newAuthPassInput = ""
                                                    showAddPortDialog = true
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
                                                            text = if (group.isSystem) "系统主出口(tun0)" else "动态池 (${group.targetCount}网卡)",
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
                                                    groupNameInput = "新自适应住宅池"
                                                    groupCountryInput = "JP"
                                                    groupIpTypeInput = "residential"
                                                    groupUnlockInput = "ai"
                                                    groupSortInput = "latency"
                                                    groupTargetCountInput = "3"
                                                    groupIntervalInput = "15"
                                                    showAddGroupDialog = true
                                                }
                                            )
                                        )
                                    )
                                }
                            }

                            // ==================== TAB 2: 边缘抗封锁入站 (全增删改查) ====================
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

    // 1. Add / Edit Port Dialog
    if (showAddPortDialog || editPortRuleTarget != null) {
        val isEditing = editPortRuleTarget != null
        AlertDialog(
            onDismissRequest = {
                showAddPortDialog = false
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
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    OutlinedTextField(
                        value = newPortInput,
                        onValueChange = { newPortInput = it },
                        label = { Text("监听端口 [1-65535]") },
                        singleLine = true,
                        enabled = !isEditing,
                        modifier = Modifier.fillMaxWidth()
                    )

                    Text("调度策略 (分流算法):", style = MaterialTheme.typography.labelMedium)
                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                        listOf("round_robin" to "轮询", "interval" to "定时", "random" to "随机").forEach { (pol, label) ->
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                RadioButton(selected = newPolicyInput == pol, onClick = { newPolicyInput = pol })
                                Text(label, style = MaterialTheme.typography.bodySmall)
                            }
                        }
                    }

                    if (newPolicyInput == "interval") {
                        OutlinedTextField(
                            value = newIntervalInput,
                            onValueChange = { newIntervalInput = it },
                            label = { Text("轮换周期 (秒)") },
                            singleLine = true,
                            modifier = Modifier.fillMaxWidth()
                        )
                    }

                    Text("鉴权模式:", style = MaterialTheme.typography.labelMedium)
                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                        listOf("random" to "系统随机", "none" to "免密", "custom" to "自定义").forEach { (auth, label) ->
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                RadioButton(selected = newAuthInput == auth, onClick = { newAuthInput = auth })
                                Text(label, style = MaterialTheme.typography.bodySmall)
                            }
                        }
                    }

                    if (newAuthInput == "custom") {
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            OutlinedTextField(
                                value = newAuthUserInput,
                                onValueChange = { newAuthUserInput = it },
                                label = { Text("用户名") },
                                singleLine = true,
                                modifier = Modifier.weight(1f)
                            )
                            OutlinedTextField(
                                value = newAuthPassInput,
                                onValueChange = { newAuthPassInput = it },
                                label = { Text("密码") },
                                singleLine = true,
                                modifier = Modifier.weight(1f)
                            )
                        }
                    }
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val portNum = newPortInput.toIntOrNull() ?: 7931
                        val intervalSec = newIntervalInput.toIntOrNull() ?: 300
                        val updated = portRules.toMutableList()
                        val rule = PortRuleItem(
                            port = portNum,
                            enabled = true,
                            policy = newPolicyInput,
                            intervalSeconds = intervalSec,
                            authMode = newAuthInput,
                            authUser = newAuthUserInput,
                            authPass = newAuthPassInput
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
                        showAddPortDialog = false
                        editPortRuleTarget = null
                        Toast.makeText(context, "端口 [$portNum] 分流规则已成功下发并生效！", Toast.LENGTH_SHORT).show()
                    }
                ) {
                    Text("保存生效", fontWeight = FontWeight.Bold)
                }
            },
            dismissButton = {
                TextButton(onClick = {
                    showAddPortDialog = false
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
    if (showAddGroupDialog || editGroupTarget != null) {
        val isEditing = editGroupTarget != null
        AlertDialog(
            onDismissRequest = {
                showAddGroupDialog = false
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
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
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
                            label = { Text("目标国家 (如 JP, US)") },
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
                            label = { Text("评估周期 (分钟)") },
                            singleLine = true,
                            modifier = Modifier.weight(1f)
                        )
                        OutlinedTextField(
                            value = groupUnlockInput,
                            onValueChange = { groupUnlockInput = it },
                            label = { Text("解锁要求 (ai, all, none)") },
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
                        showAddGroupDialog = false
                        editGroupTarget = null
                        Toast.makeText(context, "自适应组 [${newG.name}] 已成功保存并下发！", Toast.LENGTH_SHORT).show()
                    }
                ) {
                    Text("保存生效", fontWeight = FontWeight.Bold)
                }
            },
            dismissButton = {
                TextButton(onClick = {
                    showAddGroupDialog = false
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

    // 5. Add SingBox Inbound Protocol Dialog
    if (showAddInboundDialog) {
        AlertDialog(
            onDismissRequest = { showAddInboundDialog = false },
            shape = RoundedCornerShape(28.dp),
            containerColor = MaterialTheme.colorScheme.surfaceContainer,
            title = { Text("新建边缘抗封锁入站节点", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("选择抗审查协议：", style = MaterialTheme.typography.labelMedium)
                    val protos = listOf("reality" to "VLESS-REALITY", "hy2" to "Hysteria2", "tuic" to "TUIC v5", "ss" to "Shadowsocks")
                    protos.forEach { (pKey, pName) ->
                        Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.fillMaxWidth()) {
                            RadioButton(selected = newInboundProtocol == pKey, onClick = { newInboundProtocol = pKey })
                            Text(pName, style = MaterialTheme.typography.bodyMedium)
                        }
                    }
                    OutlinedTextField(
                        value = newInboundPort,
                        onValueChange = { newInboundPort = it },
                        label = { Text("外部监听端口 (或填 auto)") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth()
                    )
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
            text = { Text("确认删除协议节点 [${target.name}] (端口 ${target.port}) 吗？", style = MaterialTheme.typography.bodyMedium) },
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
            title = { Text("客户端配置分享", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text("节点类型: ${inbound.protocol} (外部端口 ${inbound.port})", style = MaterialTheme.typography.bodyMedium)
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
