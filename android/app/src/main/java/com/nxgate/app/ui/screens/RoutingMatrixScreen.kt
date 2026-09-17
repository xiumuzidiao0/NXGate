package com.nxgate.app.ui.screens

import android.content.res.Configuration
import android.widget.Toast
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.BorderStroke
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
import com.nxgate.app.NXGateApplication
import com.nxgate.app.model.AvailableOutbound
import com.nxgate.app.model.DynamicGroupCard
import com.nxgate.app.model.InboundProtocolItem
import com.nxgate.app.model.PortRuleItem
import com.nxgate.app.model.ServerProfile
import com.nxgate.app.model.TunnelItem
import com.nxgate.app.ui.components.AppExposedDropdown
import com.nxgate.app.ui.components.ConnectedButtonItem
import com.nxgate.app.ui.components.ConnectedButtonGroup
import com.nxgate.app.ui.components.ConnectedButtonStyle
import com.nxgate.app.ui.components.ConnectedListItem
import com.nxgate.app.ui.components.GlobalServerSwitcherTitle
import com.nxgate.app.ui.components.countryChineseName
import com.nxgate.app.ui.components.countryFlag
import kotlinx.coroutines.launch

// Dropdown option data models
data class PortPolicyOption(val key: String, val label: String)
data class PortIntervalOption(val seconds: Int, val label: String)
data class PortAuthOption(val key: String, val label: String)

data class GroupCountryOption(val code: String, val label: String)
data class GroupIpTypeOption(val key: String, val label: String)
data class GroupUnlockOption(val key: String, val label: String)
data class GroupSortOption(val key: String, val label: String)
data class GroupTargetCountOption(val count: Int, val label: String)
data class GroupIntervalOption(val minutes: Int, val label: String)

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

    // Real server states
    var portRules by remember { mutableStateOf<List<PortRuleItem>>(emptyList()) }
    var dynamicGroups by remember { mutableStateOf<List<DynamicGroupCard>>(emptyList()) }
    var inbounds by remember { mutableStateOf<List<InboundProtocolItem>>(emptyList()) }
    var availableOutbounds by remember {
        mutableStateOf(
            listOf(
                AvailableOutbound(7928, "socks5://127.0.0.1:7928", "NXGate 默认出口 (PORT 7928)", true),
                AvailableOutbound(0, "direct", "直连出口 (VPS 本机原生网络)", false)
            )
        )
    }

    // Modal dialog states: Port Rules (Full CRUD with Dropdowns)
    var showPortDialog by remember { mutableStateOf(false) }
    var editPortRuleTarget by remember { mutableStateOf<PortRuleItem?>(null) }
    var inputPortNum by remember { mutableStateOf("7931") }

    val portPolicyOptions = remember {
        listOf(
            PortPolicyOption("round_robin", "轮询负载均衡 (Round-Robin)"),
            PortPolicyOption("interval", "定时自动轮换 (Interval)"),
            PortPolicyOption("random", "动态随机分流 (Random)")
        )
    }
    var selectedPortPolicyOption by remember { mutableStateOf(portPolicyOptions[0]) }

    val portIntervalOptions = remember {
        listOf(
            PortIntervalOption(60, "1 分钟 (60 秒)"),
            PortIntervalOption(300, "5 分钟 (300 秒 - 默认)"),
            PortIntervalOption(900, "15 分钟 (900 秒)"),
            PortIntervalOption(3600, "1 小时 (3600 秒)")
        )
    }
    var selectedPortIntervalOption by remember { mutableStateOf(portIntervalOptions[1]) }

    val portAuthOptions = remember {
        listOf(
            PortAuthOption("random", "系统随机账密 (安全解耦)"),
            PortAuthOption("none", "免密直接连接 (本地首选)"),
            PortAuthOption("custom", "自定义独立账密 (指定账号密码)")
        )
    }
    var selectedPortAuthOption by remember { mutableStateOf(portAuthOptions[0]) }

    var inputAuthUser by remember { mutableStateOf("") }
    var inputAuthPass by remember { mutableStateOf("") }
    var bindAllTunnels by remember { mutableStateOf(true) }
    var selectedBoundGroups by remember { mutableStateOf(setOf<String>()) }
    var deletePortRuleCandidate by remember { mutableStateOf<PortRuleItem?>(null) }

    // Modal dialog states: Dynamic Groups (Full CRUD with Dropdowns)
    var showGroupDialog by remember { mutableStateOf(false) }
    var editGroupTarget by remember { mutableStateOf<DynamicGroupCard?>(null) }
    var deleteGroupCandidate by remember { mutableStateOf<DynamicGroupCard?>(null) }
    var groupNameInput by remember { mutableStateOf("日本Top3住宅组") }

    val groupCountryOptions = remember {
        val list = mutableListOf(
            GroupCountryOption("ALL", "全部国家/地区 (不限)")
        )
        val majorCodes = listOf(
            "JP", "US", "KR", "TW", "HK", "SG", "GB", "DE", "FR", "CA", "AU",
            "NL", "VN", "TH", "MY", "IN", "RU", "BR", "PH", "ID", "IT", "ES",
            "SE", "CH", "NZ", "PL", "UA", "TR", "ZA", "AR", "CL", "CO", "MX",
            "NO", "FI", "DK", "IE", "AT", "BE", "CZ", "RO", "IL", "AE", "SA", "CN"
        )
        majorCodes.forEach { code ->
            val flag = countryFlag(code)
            val name = countryChineseName(code)
            list.add(GroupCountryOption(code, "$flag $name ($code)"))
        }
        list
    }
    var selectedGroupCountryOption by remember { mutableStateOf(groupCountryOptions[1]) }

    val groupIpTypeOptions = remember {
        listOf(
            GroupIpTypeOption("all", "全部网络类型 (不限)"),
            GroupIpTypeOption("residential", "住宅宽带 IP (家宽原生)"),
            GroupIpTypeOption("hosting", "机房/数据中心 IP")
        )
    }
    var selectedGroupIpTypeOption by remember { mutableStateOf(groupIpTypeOptions[1]) }

    val groupUnlockOptions = remember {
        listOf(
            GroupUnlockOption("none", "不限解锁能力 (全量候选)"),
            GroupUnlockOption("ai", "必须支持三大 AI (ChatGPT+Claude+Gemini)"),
            GroupUnlockOption("streaming", "必须支持主流流媒体 (Netflix/Google)"),
            GroupUnlockOption("all", "全解锁 (三大 AI + 流媒体)")
        )
    }
    var selectedGroupUnlockOption by remember { mutableStateOf(groupUnlockOptions[1]) }

    val groupSortOptions = remember {
        listOf(
            GroupSortOption("latency", "最低延迟优先 (TCP 测速)"),
            GroupSortOption("speed", "最大带宽优先 (Mbps)"),
            GroupSortOption("score", "综合评分最高优先")
        )
    }
    var selectedGroupSortOption by remember { mutableStateOf(groupSortOptions[0]) }

    val groupTargetCountOptions = remember {
        listOf(
            GroupTargetCountOption(1, "维持 1 个并发出口"),
            GroupTargetCountOption(2, "维持 2 个并发出口"),
            GroupTargetCountOption(3, "维持 3 个并发出口 (推荐)"),
            GroupTargetCountOption(5, "维持 5 个并发出口"),
            GroupTargetCountOption(8, "维持 8 个并发出口")
        )
    }
    var selectedGroupTargetCountOption by remember { mutableStateOf(groupTargetCountOptions[2]) }

    val groupIntervalOptions = remember {
        listOf(
            GroupIntervalOption(0, "关闭定时轮换 (保持固定连接)"),
            GroupIntervalOption(5, "每 5 分钟重评轮换"),
            GroupIntervalOption(15, "每 15 分钟重评轮换 (推荐)"),
            GroupIntervalOption(30, "每 30 分钟重评轮换"),
            GroupIntervalOption(60, "每 60 分钟 (1 小时) 重评轮换")
        )
    }
    var selectedGroupIntervalOption by remember { mutableStateOf(groupIntervalOptions[2]) }

    // Modal dialog states: SingBox Inbounds (Full 22 Protocols + Dropdown Outbounds)
    var showAddInboundDialog by remember { mutableStateOf(false) }
    var selectedProtoCat by remember { mutableIntStateOf(0) }
    var newInboundProtocol by remember { mutableStateOf("reality") }
    var newInboundPort by remember { mutableStateOf("auto") }

    var selectedInboundOutboundOption by remember { mutableStateOf(availableOutbounds.first()) }
    var selectedInboundForOutboundSwitch by remember { mutableStateOf<InboundProtocolItem?>(null) }
    var switchOutboundTargetOption by remember { mutableStateOf(availableOutbounds.first()) }
    var deleteInboundCandidate by remember { mutableStateOf<InboundProtocolItem?>(null) }
    var showInboundQRDialog by remember { mutableStateOf<InboundProtocolItem?>(null) }

    // Pull real data when active server changes
    LaunchedEffect(activeServer?.id) {
        if (activeServer != null) {
            val portsRes = NXGateApplication.instance.apiClient.fetchPortRules(activeServer)
            if (portsRes.isSuccess) {
                portRules = portsRes.getOrNull() ?: emptyList()
            }
            val groupsRes = NXGateApplication.instance.apiClient.fetchDynamicGroups(activeServer)
            if (groupsRes.isSuccess) {
                dynamicGroups = groupsRes.getOrNull() ?: emptyList()
            }
            val sbOverviewRes = NXGateApplication.instance.apiClient.fetchSingBoxOverview(activeServer)
            if (sbOverviewRes.isSuccess) {
                val data = sbOverviewRes.getOrNull()
                if (data != null) {
                    inbounds = data.nodes
                    if (data.availableOutbounds.isNotEmpty()) {
                        availableOutbounds = data.availableOutbounds
                        selectedInboundOutboundOption = data.availableOutbounds.first()
                        switchOutboundTargetOption = data.availableOutbounds.first()
                    }
                }
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
                                    val portsRes = NXGateApplication.instance.apiClient.fetchPortRules(activeServer)
                                    if (portsRes.isSuccess && !portsRes.getOrNull().isNullOrEmpty()) portRules = portsRes.getOrNull()!!
                                    val groupsRes = NXGateApplication.instance.apiClient.fetchDynamicGroups(activeServer)
                                    if (groupsRes.isSuccess && !groupsRes.getOrNull().isNullOrEmpty()) dynamicGroups = groupsRes.getOrNull()!!
                                    val sbOverviewRes = NXGateApplication.instance.apiClient.fetchSingBoxOverview(activeServer)
                                    if (sbOverviewRes.isSuccess) {
                                        val data = sbOverviewRes.getOrNull()
                                        if (data != null) {
                                            if (data.nodes.isNotEmpty()) inbounds = data.nodes
                                            if (data.availableOutbounds.isNotEmpty()) availableOutbounds = data.availableOutbounds
                                        }
                                    }
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
                                fontWeight = if (selectedTabIndex == index) FontWeight.Bold else FontWeight.Medium,
                                color = if (selectedTabIndex == index) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface
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
                                    if (portRules.isEmpty()) {
                                        Surface(
                                            modifier = Modifier.fillMaxWidth(),
                                            shape = RoundedCornerShape(16.dp),
                                            color = MaterialTheme.colorScheme.surfaceContainerLow
                                        ) {
                                            Text(
                                                text = if (activeServer == null) "当前尚未纳管任何服务器，请前往「概览」或「设置」添加 VPS 网关。" else "当前服务器尚未配置独立代理端口规则，请点击下方「新建代理端口」添加。",
                                                style = MaterialTheme.typography.bodyMedium,
                                                color = MaterialTheme.colorScheme.onSurface,
                                                modifier = Modifier.padding(20.dp)
                                            )
                                        }
                                    } else {
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
                                                                    NXGateApplication.instance.apiClient.savePortRules(activeServer, updated)
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
                                                    selectedPortPolicyOption = portPolicyOptions.find { it.key == rule.policy } ?: portPolicyOptions[0]
                                                    selectedPortIntervalOption = portIntervalOptions.find { it.seconds == rule.intervalSeconds } ?: portIntervalOptions[1]
                                                    selectedPortAuthOption = portAuthOptions.find { it.key == rule.authMode } ?: portAuthOptions[0]
                                                    inputAuthUser = rule.authUser
                                                    inputAuthPass = rule.authPass
                                                    bindAllTunnels = rule.boundGroupIds.isEmpty() && rule.boundTunnelIds.isEmpty()
                                                    selectedBoundGroups = rule.boundGroupIds.toSet()
                                                    showPortDialog = true
                                                }
                                            )
                                        }
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
                                                    selectedPortPolicyOption = portPolicyOptions[0]
                                                    selectedPortIntervalOption = portIntervalOptions[1]
                                                    selectedPortAuthOption = portAuthOptions[0]
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
                                                            val res = NXGateApplication.instance.apiClient.fetchPortRules(activeServer)
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
                                    if (dynamicGroups.isEmpty()) {
                                        Surface(
                                            modifier = Modifier.fillMaxWidth(),
                                            shape = RoundedCornerShape(16.dp),
                                            color = MaterialTheme.colorScheme.surfaceContainerLow
                                        ) {
                                            Text(
                                                text = if (activeServer == null) "当前尚未纳管任何服务器，请前往「概览」或「设置」添加 VPS 网关。" else "当前服务器尚未配置动态自适应组，请点击下方「新建自适应组」添加。",
                                                style = MaterialTheme.typography.bodyMedium,
                                                color = MaterialTheme.colorScheme.onSurface,
                                                modifier = Modifier.padding(20.dp)
                                            )
                                        }
                                    } else {
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
                                                    color = MaterialTheme.colorScheme.onSurface,
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
                                                            selectedGroupCountryOption = groupCountryOptions.find { it.code.equals(group.country, true) } ?: groupCountryOptions[1]
                                                            selectedGroupIpTypeOption = groupIpTypeOptions.find { it.key.equals(group.ipType, true) } ?: groupIpTypeOptions[1]
                                                            selectedGroupUnlockOption = groupUnlockOptions.find { it.key.equals(group.unlockFilter, true) } ?: groupUnlockOptions[1]
                                                            selectedGroupSortOption = groupSortOptions.find { it.key.equals(group.sortBy, true) } ?: groupSortOptions[0]
                                                            selectedGroupTargetCountOption = groupTargetCountOptions.find { it.count == group.targetCount } ?: groupTargetCountOptions[2]
                                                            selectedGroupIntervalOption = groupIntervalOptions.find { it.minutes == group.intervalMinutes } ?: groupIntervalOptions[2]
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
                                                            val res = NXGateApplication.instance.apiClient.triggerRotate(activeServer)
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
                                                    selectedGroupCountryOption = groupCountryOptions[1]
                                                    selectedGroupIpTypeOption = groupIpTypeOptions[1]
                                                    selectedGroupUnlockOption = groupUnlockOptions[1]
                                                    selectedGroupSortOption = groupSortOptions[0]
                                                    selectedGroupTargetCountOption = groupTargetCountOptions[2]
                                                    selectedGroupIntervalOption = groupIntervalOptions[2]
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
                                    if (inbounds.isEmpty()) {
                                        Surface(
                                            modifier = Modifier.fillMaxWidth(),
                                            shape = RoundedCornerShape(16.dp),
                                            color = MaterialTheme.colorScheme.surfaceContainerLow
                                        ) {
                                            Text(
                                                text = if (activeServer == null) "当前尚未纳管任何服务器，请前往「概览」或「设置」添加 VPS 网关。" else "当前服务器尚未创建边缘抗封锁入站节点，请点击下方「新建入站节点」创建。",
                                                style = MaterialTheme.typography.bodyMedium,
                                                color = MaterialTheme.colorScheme.onSurface,
                                                modifier = Modifier.padding(20.dp)
                                            )
                                        }
                                    } else {
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
                                                            switchOutboundTargetOption = availableOutbounds.find { it.addr == inbound.outbound } ?: availableOutbounds.first()
                                                        }) {
                                                            Icon(Icons.Rounded.SwapHoriz, contentDescription = "更改出口", tint = MaterialTheme.colorScheme.secondary)
                                                        }
                                                        IconButton(onClick = { deleteInboundCandidate = inbound }) {
                                                            Icon(Icons.Rounded.Delete, contentDescription = "删除入站", tint = MaterialTheme.colorScheme.error)
                                                        }
                                                    }
                                                },
                                                onClick = {
                                                    Toast.makeText(context, "入站: ${inbound.name} (外部端口 ${inbound.port})", Toast.LENGTH_SHORT).show()
                                                }
                                            )
                                        }
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
                                                    selectedInboundOutboundOption = availableOutbounds.first()
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

    // 1. Add / Edit Port Dialog (包含策略下拉框、周期下拉框、鉴权下拉框、自适应端口组多选)
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

                    // 调度策略下拉框
                    AppExposedDropdown(
                        label = "调度策略 (分流算法)",
                        options = portPolicyOptions,
                        selectedOption = selectedPortPolicyOption,
                        onOptionSelected = { selectedPortPolicyOption = it },
                        optionLabel = { it.label }
                    )

                    // 定时轮换周期下拉框
                    if (selectedPortPolicyOption.key == "interval") {
                        AppExposedDropdown(
                            label = "自动轮换周期",
                            options = portIntervalOptions,
                            selectedOption = selectedPortIntervalOption,
                            onOptionSelected = { selectedPortIntervalOption = it },
                            optionLabel = { it.label }
                        )
                    }

                    // 鉴权模式下拉框
                    AppExposedDropdown(
                        label = "代理鉴权模式",
                        options = portAuthOptions,
                        selectedOption = selectedPortAuthOption,
                        onOptionSelected = { selectedPortAuthOption = it },
                        optionLabel = { it.label }
                    )

                    if (selectedPortAuthOption.key == "custom") {
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

                    // 绑定自适应组与出口部分 (对齐 Web 控制台)
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

                    // Checkbox 2: 绑定自适应动态出口组
                    if (dynamicGroups.isNotEmpty()) {
                        Text("自适应动态出口组 (自动维持Top N并定期轮换):", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.primary)
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
                        val intervalSec = selectedPortIntervalOption.seconds
                        val updated = portRules.toMutableList()
                        val boundGroupList = if (bindAllTunnels) emptyList() else selectedBoundGroups.toList()

                        val rule = PortRuleItem(
                            port = portNum,
                            enabled = true,
                            policy = selectedPortPolicyOption.key,
                            intervalSeconds = intervalSec,
                            authMode = selectedPortAuthOption.key,
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
                                NXGateApplication.instance.apiClient.savePortRules(activeServer, updated)
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
                                NXGateApplication.instance.apiClient.savePortRules(activeServer, updated)
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

    // 3. Add / Edit Dynamic Group Dialog (完整下拉框支持: 国家、网络类型、解锁要求、排序指标、维持数量、周期)
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
                    verticalArrangement = Arrangement.spacedBy(10.dp)
                ) {
                    OutlinedTextField(
                        value = groupNameInput,
                        onValueChange = { groupNameInput = it },
                        label = { Text("组名称") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth()
                    )

                    // 国家下拉框
                    AppExposedDropdown(
                        label = "目标国家/地区",
                        options = groupCountryOptions,
                        selectedOption = selectedGroupCountryOption,
                        onOptionSelected = { selectedGroupCountryOption = it },
                        optionLabel = { it.label }
                    )

                    // 网络类型下拉框
                    AppExposedDropdown(
                        label = "目标网络类型",
                        options = groupIpTypeOptions,
                        selectedOption = selectedGroupIpTypeOption,
                        onOptionSelected = { selectedGroupIpTypeOption = it },
                        optionLabel = { it.label }
                    )

                    // 解锁要求下拉框
                    AppExposedDropdown(
                        label = "节点解锁要求",
                        options = groupUnlockOptions,
                        selectedOption = selectedGroupUnlockOption,
                        onOptionSelected = { selectedGroupUnlockOption = it },
                        optionLabel = { it.label }
                    )

                    // 择优排序指标下拉框
                    AppExposedDropdown(
                        label = "择优筛选指标",
                        options = groupSortOptions,
                        selectedOption = selectedGroupSortOption,
                        onOptionSelected = { selectedGroupSortOption = it },
                        optionLabel = { it.label }
                    )

                    // 维持并发数与周期下拉框
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        AppExposedDropdown(
                            label = "维持并发数",
                            options = groupTargetCountOptions,
                            selectedOption = selectedGroupTargetCountOption,
                            onOptionSelected = { selectedGroupTargetCountOption = it },
                            optionLabel = { it.label },
                            modifier = Modifier.weight(1f)
                        )
                        AppExposedDropdown(
                            label = "重评轮换周期",
                            options = groupIntervalOptions,
                            selectedOption = selectedGroupIntervalOption,
                            onOptionSelected = { selectedGroupIntervalOption = it },
                            optionLabel = { it.label },
                            modifier = Modifier.weight(1f)
                        )
                    }
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val targetId = editGroupTarget?.id ?: "dg-${System.currentTimeMillis() % 100000}"
                        val newG = DynamicGroupCard(
                            id = targetId,
                            name = groupNameInput.trim().ifEmpty { "自适应组" },
                            country = selectedGroupCountryOption.code,
                            targetCount = selectedGroupTargetCountOption.count,
                            intervalMinutes = selectedGroupIntervalOption.minutes,
                            ipType = selectedGroupIpTypeOption.key,
                            unlockFilter = selectedGroupUnlockOption.key,
                            sortBy = selectedGroupSortOption.key,
                            isSystem = editGroupTarget?.isSystem ?: false
                        )
                        val updated = dynamicGroups.toMutableList()
                        val existIdx = updated.indexOfFirst { it.id == targetId }
                        if (existIdx >= 0) updated[existIdx] = newG else updated.add(newG)
                        dynamicGroups = updated

                        if (activeServer != null) {
                            scope.launch {
                                NXGateApplication.instance.apiClient.saveDynamicGroup(activeServer, newG)
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
                                NXGateApplication.instance.apiClient.deleteDynamicGroup(activeServer, target.id)
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

    // 5. Add SingBox Inbound Protocol Dialog (完整 22 种协议分类与动态出口下拉框)
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
                    // Category selector chips
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

                    Text("选择具体协议:", style = MaterialTheme.typography.labelMedium)
                    val currentProtos = protoCategories[selectedProtoCat].second
                    currentProtos.forEach { (pKey, pLabel) ->
                        Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.fillMaxWidth()) {
                            androidx.compose.material3.RadioButton(selected = newInboundProtocol == pKey, onClick = { newInboundProtocol = pKey })
                            Text(pLabel, style = MaterialTheme.typography.bodySmall)
                        }
                    }

                    OutlinedTextField(
                        value = newInboundPort,
                        onValueChange = { newInboundPort = it },
                        label = { Text("外部端口 (填 auto 自动分配高位端口)") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth()
                    )

                    // 动态出口下拉框
                    AppExposedDropdown(
                        label = "出口链式分流指向",
                        options = availableOutbounds,
                        selectedOption = selectedInboundOutboundOption,
                        onOptionSelected = { selectedInboundOutboundOption = it },
                        optionLabel = { it.label }
                    )
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        if (activeServer != null) {
                            scope.launch {
                                val res = NXGateApplication.instance.apiClient.addSingBoxNode(activeServer, newInboundProtocol, newInboundPort, selectedInboundOutboundOption.addr)
                                if (res.isSuccess) {
                                    val refreshRes = NXGateApplication.instance.apiClient.fetchSingBoxOverview(activeServer)
                                    if (refreshRes.isSuccess) {
                                        val data = refreshRes.getOrNull()
                                        if (data != null && data.nodes.isNotEmpty()) inbounds = data.nodes
                                    }
                                    Toast.makeText(context, "边缘入站节点已成功创建并启动", Toast.LENGTH_SHORT).show()
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

    // 6. Switch SingBox Outbound Dialog (动态出口下拉选择)
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
                Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    Text("选择该入站协议转发流量的目标出口代理：", style = MaterialTheme.typography.bodyMedium)

                    AppExposedDropdown(
                        label = "目标出口链路",
                        options = availableOutbounds,
                        selectedOption = switchOutboundTargetOption,
                        onOptionSelected = { switchOutboundTargetOption = it },
                        optionLabel = { it.label }
                    )
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        if (activeServer != null) {
                            scope.launch {
                                NXGateApplication.instance.apiClient.setSingBoxOutbound(activeServer, inbound.name, switchOutboundTargetOption.addr)
                                val updated = inbounds.map {
                                    if (it.id == inbound.id) it.copy(outbound = switchOutboundTargetOption.addr, outboundLabel = switchOutboundTargetOption.label) else it
                                }
                                inbounds = updated
                                Toast.makeText(context, "[${inbound.name}] 出口已改挂至: ${switchOutboundTargetOption.label}", Toast.LENGTH_SHORT).show()
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
                                NXGateApplication.instance.apiClient.deleteSingBoxNode(activeServer, target.name)
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
