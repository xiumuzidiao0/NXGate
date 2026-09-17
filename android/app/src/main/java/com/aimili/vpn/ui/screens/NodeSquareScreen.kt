package com.aimili.vpn.ui.screens

import android.content.res.Configuration
import android.widget.Toast
import androidx.compose.foundation.BorderStroke
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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.Sort
import androidx.compose.material.icons.rounded.Block
import androidx.compose.material.icons.rounded.Bookmark
import androidx.compose.material.icons.rounded.BookmarkBorder
import androidx.compose.material.icons.rounded.Category
import androidx.compose.material.icons.rounded.CheckCircle
import androidx.compose.material.icons.rounded.Delete
import androidx.compose.material.icons.rounded.Domain
import androidx.compose.material.icons.rounded.Flag
import androidx.compose.material.icons.rounded.Home
import androidx.compose.material.icons.rounded.Lan
import androidx.compose.material.icons.rounded.Public
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material.icons.rounded.RocketLaunch
import androidx.compose.material.icons.rounded.Search
import androidx.compose.material.icons.rounded.Star
import androidx.compose.material.icons.rounded.StarBorder
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
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.material3.rememberModalBottomSheetState
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
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.aimili.vpn.AimiliApplication
import com.aimili.vpn.model.BlacklistRecord
import com.aimili.vpn.model.NodeCandidate
import com.aimili.vpn.model.ServerProfile
import com.aimili.vpn.ui.components.AppExposedDropdown
import com.aimili.vpn.ui.components.ConnectedButtonItem
import com.aimili.vpn.ui.components.ConnectedButtonGroup
import com.aimili.vpn.ui.components.ConnectedButtonStyle
import com.aimili.vpn.ui.components.ConnectedChipGroup
import com.aimili.vpn.ui.components.GlobalServerSwitcherTitle
import com.aimili.vpn.ui.components.UnlockPill
import com.aimili.vpn.ui.components.countryChineseName
import com.aimili.vpn.ui.components.countryFlag
import kotlinx.coroutines.launch

@Composable
fun FullNodeCard(
    node: NodeCandidate,
    isMaster: Boolean = false,
    onSetMaster: () -> Unit,
    isFavorite: Boolean,
    onToggleFavorite: () -> Unit,
    onStartTunnel: () -> Unit,
    onBlacklist: () -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(20.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainerHighest
        )
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            // Row 1: Country Flag + Country Name + IP:Port + (主连 Badge) + Latency Badge
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    val displayName = node.countryLong.ifBlank { countryChineseName(node.countryShort) }
                    Text(
                        text = "${countryFlag(node.countryShort)} $displayName",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                    Spacer(Modifier.width(6.dp))
                    Text(
                        text = "${node.ip}:${node.port}",
                        style = MaterialTheme.typography.bodySmall,
                        fontWeight = FontWeight.Medium,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                    if (isMaster) {
                        Surface(
                            shape = RoundedCornerShape(6.dp),
                            color = MaterialTheme.colorScheme.primaryContainer,
                            modifier = Modifier.padding(start = 6.dp)
                        ) {
                            Text(
                                text = "主连",
                                style = MaterialTheme.typography.labelSmall,
                                fontWeight = FontWeight.Bold,
                                color = MaterialTheme.colorScheme.onPrimaryContainer,
                                modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                            )
                        }
                    }
                }

                val latencyColor = when {
                    node.latencyMs in 1..80 -> MaterialTheme.colorScheme.primary
                    node.latencyMs in 81..180 -> MaterialTheme.colorScheme.tertiary
                    else -> MaterialTheme.colorScheme.error
                }
                Surface(
                    shape = RoundedCornerShape(8.dp),
                    color = latencyColor.copy(alpha = 0.15f)
                ) {
                    Text(
                        text = if (node.latencyMs > 0) "${node.latencyMs} ms" else "待测",
                        style = MaterialTheme.typography.labelSmall,
                        fontWeight = FontWeight.Bold,
                        color = latencyColor,
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                    )
                }
            }

            // Row 2: ISP + IP Type Badge
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "运营商: ${node.isp.ifEmpty { "全球骨干网络" }}",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurface,
                    modifier = Modifier.weight(1f)
                )
                val isRes = node.ipType == "residential"
                Surface(
                    shape = RoundedCornerShape(6.dp),
                    color = if (isRes) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surfaceContainer
                ) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                    ) {
                        Icon(
                            imageVector = if (isRes) Icons.Rounded.Home else Icons.Rounded.Domain,
                            contentDescription = null,
                            tint = if (isRes) MaterialTheme.colorScheme.onPrimaryContainer else MaterialTheme.colorScheme.onSurface,
                            modifier = Modifier.size(12.dp)
                        )
                        Spacer(Modifier.width(3.dp))
                        Text(
                            text = if (isRes) "原生家宽" else "机房托管",
                            style = MaterialTheme.typography.labelSmall,
                            fontWeight = FontWeight.Medium,
                            color = if (isRes) MaterialTheme.colorScheme.onPrimaryContainer else MaterialTheme.colorScheme.onSurface
                        )
                    }
                }
            }

            // Row 3: Speed & Score
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "实测速度: ${node.speedMbStr}",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurface
                )
                Text(
                    text = "信誉评分: ${node.score} 分",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.primary,
                    fontWeight = FontWeight.Bold
                )
            }

            // Row 4: AI & Streaming Unlock Badges Row
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(4.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text("实测解锁:", style = MaterialTheme.typography.labelSmall, fontWeight = FontWeight.Medium, color = MaterialTheme.colorScheme.onSurface)
                UnlockPill("ChatGPT", node.openai)
                UnlockPill("Claude", node.claude)
                UnlockPill("Gemini", node.gemini)
                UnlockPill("Netflix", node.netflix)
            }

            Spacer(Modifier.height(2.dp))

            // Row 5: Action Button Group (设为主出口, 拉起网卡, 收藏, 屏蔽)
            ConnectedButtonGroup(
                items = listOf(
                    ConnectedButtonItem(
                        text = if (isMaster) "当前主连" else "设为主出口",
                        style = if (isMaster) ConnectedButtonStyle.Tonal else ConnectedButtonStyle.Filled,
                        icon = if (isMaster) Icons.Rounded.CheckCircle else Icons.Rounded.Lan,
                        onClick = onSetMaster
                    ),
                    ConnectedButtonItem(
                        text = "拉起网卡",
                        style = ConnectedButtonStyle.Tonal,
                        icon = Icons.Rounded.RocketLaunch,
                        onClick = onStartTunnel
                    ),
                    ConnectedButtonItem(
                        text = if (isFavorite) "已收藏" else "收藏",
                        style = if (isFavorite) ConnectedButtonStyle.Filled else ConnectedButtonStyle.Tonal,
                        icon = if (isFavorite) Icons.Rounded.Bookmark else Icons.Rounded.BookmarkBorder,
                        onClick = onToggleFavorite
                    ),
                    ConnectedButtonItem(
                        text = "屏蔽",
                        style = ConnectedButtonStyle.Outlined,
                        icon = Icons.Rounded.Block,
                        onClick = onBlacklist
                    )
                ),
                height = 42.dp
            )
        }
    }
}

data class CountryFilterOption(val code: String, val label: String)
data class IpTypeFilterOption(val key: String, val label: String)
data class SortFilterOption(val key: String, val label: String)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun NodeSquareScreen(
    activeServer: ServerProfile?,
    allServers: List<ServerProfile>,
    onSelectServer: (String) -> Unit,
    onPreviousServer: () -> Unit,
    onNextServer: () -> Unit,
    modifier: Modifier = Modifier
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()

    val configuration = LocalConfiguration.current
    val isLandscape = configuration.orientation == Configuration.ORIENTATION_LANDSCAPE
    val isTablet = configuration.screenWidthDp >= 600
    val isTabletLandscape = isTablet && isLandscape

    var searchQuery by remember { mutableStateOf("") }
    var isSearchActive by remember { mutableStateOf(false) }

    // Quick filter chips (包含我的收藏)
    var selectedChipIndex by remember { mutableIntStateOf(0) }
    val chipList = listOf("全部", "我的收藏", "日本", "美国", "原生家宽", "三大AI全通")

    // Full nodes list from server (dynamically loaded)
    var allNodes by remember { mutableStateOf<List<NodeCandidate>>(emptyList()) }
    var isLoadingNodes by remember { mutableStateOf(false) }
    var nodeLoadError by remember { mutableStateOf<String?>(null) }

    var blacklistItems by remember { mutableStateOf<List<BlacklistRecord>>(emptyList()) }
    var showBlacklistSheet by remember { mutableStateOf(false) }
    var activeMasterIp by remember { mutableStateOf(activeServer?.exitIp ?: "") }
    val blacklistSheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)

    // Dynamic country dropdown options calculated from allNodes (matching Web)
    val countryOptions = remember(allNodes) {
        val countMap = allNodes.groupingBy { it.countryShort.uppercase() }.eachCount()
        val list = mutableListOf(CountryFilterOption("", "全部国家/地区 (共 ${allNodes.size} 节点)"))
        countMap.entries.sortedByDescending { it.value }.forEach { (code, count) ->
            val flag = countryFlag(code)
            val rawName = allNodes.find { it.countryShort.equals(code, ignoreCase = true) }?.countryLong
            val name = if (!rawName.isNullOrBlank()) rawName else countryChineseName(code)
            list.add(CountryFilterOption(code, "$flag $name ($code · $count)"))
        }
        list
    }
    var selectedCountryOption by remember { mutableStateOf(countryOptions.first()) }

    // IP Type dropdown options (matching Web)
    val ipTypeOptions = remember {
        listOf(
            IpTypeFilterOption("all", "全部网络类型"),
            IpTypeFilterOption("residential", "原生住宅宽带 (家宽)"),
            IpTypeFilterOption("hosting", "机房/数据中心 IP")
        )
    }
    var selectedIpTypeOption by remember { mutableStateOf(ipTypeOptions.first()) }

    // Sort dropdown options (matching Web)
    val sortOptions = remember {
        listOf(
            SortFilterOption("latency_asc", "按测速延迟 (从低到高)"),
            SortFilterOption("speed_desc", "按节点带宽 (从大到小)"),
            SortFilterOption("score_desc", "按综合评分 (从高到低)")
        )
    }
    var selectedSortOption by remember { mutableStateOf(sortOptions.first()) }

    fun refreshAllNodes() {
        if (activeServer != null) {
            isLoadingNodes = true
            nodeLoadError = null
            scope.launch {
                val nodesRes = AimiliApplication.instance.apiClient.fetchNodes(activeServer)
                isLoadingNodes = false
                if (nodesRes.isSuccess) {
                    allNodes = nodesRes.getOrNull() ?: emptyList()
                } else {
                    nodeLoadError = nodesRes.exceptionOrNull()?.message ?: "连接超时"
                }

                val blRes = AimiliApplication.instance.apiClient.fetchBlacklist(activeServer)
                if (blRes.isSuccess) {
                    blacklistItems = blRes.getOrNull() ?: emptyList()
                }

                val statusRes = AimiliApplication.instance.apiClient.fetchStatus(activeServer)
                if (statusRes.isSuccess) {
                    val info = statusRes.getOrNull()
                    if (info != null && info.nodeIp.isNotEmpty()) {
                        activeMasterIp = info.nodeIp
                    }
                }
            }
        }
    }

    // Pull real nodes whenever active server changes
    LaunchedEffect(activeServer?.id) {
        refreshAllNodes()
    }

    // Update country options selection if options change
    LaunchedEffect(countryOptions) {
        if (selectedCountryOption.code.isNotEmpty()) {
            selectedCountryOption = countryOptions.find { it.code == selectedCountryOption.code } ?: countryOptions.first()
        } else {
            selectedCountryOption = countryOptions.first()
        }
    }

    // Dynamic filtering across all nodes
    val filteredNodes = remember(allNodes, selectedChipIndex, selectedCountryOption, selectedIpTypeOption, selectedSortOption, searchQuery) {
        var list = allNodes.filter { node ->
            val matchesQuery = searchQuery.isEmpty() ||
                    node.ip.contains(searchQuery, ignoreCase = true) ||
                    node.countryShort.contains(searchQuery, ignoreCase = true) ||
                    node.countryLong.contains(searchQuery, ignoreCase = true) ||
                    node.isp.contains(searchQuery, ignoreCase = true)

            val matchesCountry = selectedCountryOption.code.isEmpty() ||
                    node.countryShort.equals(selectedCountryOption.code, ignoreCase = true)

            val matchesIpType = when (selectedIpTypeOption.key) {
                "residential" -> node.ipType == "residential"
                "hosting" -> node.ipType == "hosting"
                else -> true
            }

            val matchesChip = when (selectedChipIndex) {
                1 -> node.isFavorite
                2 -> node.countryShort.equals("JP", ignoreCase = true)
                3 -> node.countryShort.equals("US", ignoreCase = true)
                4 -> node.ipType == "residential"
                5 -> node.openai == "unlocked" && node.claude == "unlocked" && node.gemini == "unlocked"
                else -> true
            }
            matchesQuery && matchesCountry && matchesIpType && matchesChip
        }

        list = when (selectedSortOption.key) {
            "latency_asc" -> list.sortedBy { if (it.latencyMs > 0) it.latencyMs else 9999 }
            "speed_desc" -> list.sortedByDescending { it.speedBps }
            "score_desc" -> list.sortedByDescending { it.score }
            else -> list
        }
        list
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
                        onClick = { isSearchActive = !isSearchActive },
                        modifier = Modifier.size(48.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Rounded.Search,
                            contentDescription = "检索候选节点",
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
                    .align(Alignment.TopCenter),
                verticalArrangement = Arrangement.spacedBy(10.dp)
            ) {
                Spacer(Modifier.height(2.dp))

                if (isSearchActive) {
                    OutlinedTextField(
                        value = searchQuery,
                        onValueChange = { searchQuery = it },
                        label = { Text("搜索 IP / 国家代码 / 运营商关键词") },
                        singleLine = true,
                        shape = RoundedCornerShape(16.dp),
                        modifier = Modifier.fillMaxWidth()
                    )
                }

                // 1. 横向快捷标签片
                Row(modifier = Modifier.horizontalScroll(rememberScrollState())) {
                    ConnectedChipGroup(
                        chips = chipList,
                        selectedIndex = selectedChipIndex,
                        onSelected = { selectedChipIndex = it }
                    )
                }

                // 2. 下拉框筛选控制区域 (平板横屏三列并排，手机自适应堆叠)
                if (isTabletLandscape) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(10.dp)
                    ) {
                        AppExposedDropdown(
                            label = "国家/地区分类",
                            options = countryOptions,
                            selectedOption = selectedCountryOption,
                            onOptionSelected = { selectedCountryOption = it },
                            optionLabel = { it.label },
                            leadingIcon = Icons.Rounded.Flag,
                            modifier = Modifier.weight(1.2f)
                        )
                        AppExposedDropdown(
                            label = "网络类型",
                            options = ipTypeOptions,
                            selectedOption = selectedIpTypeOption,
                            onOptionSelected = { selectedIpTypeOption = it },
                            optionLabel = { it.label },
                            leadingIcon = Icons.Rounded.Category,
                            modifier = Modifier.weight(1f)
                        )
                        AppExposedDropdown(
                            label = "排序规则",
                            options = sortOptions,
                            selectedOption = selectedSortOption,
                            onOptionSelected = { selectedSortOption = it },
                            optionLabel = { it.label },
                            leadingIcon = Icons.AutoMirrored.Rounded.Sort,
                            modifier = Modifier.weight(1f)
                        )
                    }
                } else {
                    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        AppExposedDropdown(
                            label = "按国家或地区分类选择",
                            options = countryOptions,
                            selectedOption = selectedCountryOption,
                            onOptionSelected = { selectedCountryOption = it },
                            optionLabel = { it.label },
                            leadingIcon = Icons.Rounded.Flag,
                            modifier = Modifier.fillMaxWidth()
                        )
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.spacedBy(8.dp)
                        ) {
                            AppExposedDropdown(
                                label = "网络类型",
                                options = ipTypeOptions,
                                selectedOption = selectedIpTypeOption,
                                onOptionSelected = { selectedIpTypeOption = it },
                                optionLabel = { it.label },
                                leadingIcon = Icons.Rounded.Category,
                                modifier = Modifier.weight(1f)
                            )
                            AppExposedDropdown(
                                label = "排序方式",
                                options = sortOptions,
                                selectedOption = selectedSortOption,
                                onOptionSelected = { selectedSortOption = it },
                                optionLabel = { it.label },
                                leadingIcon = Icons.AutoMirrored.Rounded.Sort,
                                modifier = Modifier.weight(1f)
                            )
                        }
                    }
                }

                // 3. 操作按钮与计数条
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = if (isLoadingNodes) "正在加载全量节点..." else "当前展示 ${filteredNodes.size} / 全库 ${allNodes.size} 个节点",
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.primary,
                        fontWeight = FontWeight.Bold
                    )
                }

                ConnectedButtonGroup(
                    items = listOf(
                        ConnectedButtonItem(
                            text = "全量并发测速",
                            style = ConnectedButtonStyle.Filled,
                            onClick = {
                                if (activeServer != null) {
                                    scope.launch {
                                        AimiliApplication.instance.apiClient.probeNodes(activeServer)
                                        Toast.makeText(context, "已触发远端对候选节点进行并发测速！", Toast.LENGTH_SHORT).show()
                                    }
                                }
                            }
                        ),
                        ConnectedButtonItem(
                            text = "屏蔽库 (${blacklistItems.size})",
                            style = ConnectedButtonStyle.Tonal,
                            onClick = { showBlacklistSheet = true }
                        )
                    ),
                    height = 44.dp
                )

                if (isLoadingNodes) {
                    LinearProgressIndicator(modifier = Modifier.fillMaxWidth(), color = MaterialTheme.colorScheme.primary)
                }

                // 4. 全量候选节点流 (所有节点全部展示，平板横屏 2 列响应式网格，手机单列平滑可滚动流)
                if (allNodes.isEmpty() && !isLoadingNodes) {
                    Surface(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(top = 16.dp),
                        shape = RoundedCornerShape(16.dp),
                        color = MaterialTheme.colorScheme.surfaceContainerLow
                    ) {
                        Column(
                            modifier = Modifier.padding(20.dp),
                            horizontalAlignment = Alignment.CenterHorizontally
                        ) {
                            Text(
                                text = if (nodeLoadError != null) "拉取节点失败: $nodeLoadError" else "当前服务器暂未拉取到候选节点，请检查网络或点击重试",
                                style = MaterialTheme.typography.bodyMedium,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                            Spacer(Modifier.height(10.dp))
                            OutlinedButton(onClick = { refreshAllNodes() }) {
                                Icon(Icons.Rounded.Refresh, contentDescription = null, modifier = Modifier.size(16.dp))
                                Spacer(Modifier.width(6.dp))
                                Text("立即重新拉取全部节点")
                            }
                        }
                    }
                } else if (filteredNodes.isEmpty()) {
                    Surface(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(top = 16.dp),
                        shape = RoundedCornerShape(16.dp),
                        color = MaterialTheme.colorScheme.surfaceContainerLow
                    ) {
                        Text(
                            text = "未找到符合当前筛选条件的节点，请尝试切换上方国家下拉框或点击全量测速。",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(20.dp)
                        )
                    }
                } else if (isTabletLandscape) {
                    // 平板横屏：2列响应式大网格
                    LazyVerticalGrid(
                        columns = GridCells.Fixed(2),
                        verticalArrangement = Arrangement.spacedBy(10.dp),
                        horizontalArrangement = Arrangement.spacedBy(10.dp),
                        modifier = Modifier
                            .fillMaxWidth()
                            .weight(1f)
                    ) {
                        items(filteredNodes, key = { it.id }) { node ->
                            val isMaster = node.ip == activeMasterIp || (activeServer != null && activeServer.exitIp == node.ip)
                            FullNodeCard(
                                node = node,
                                isMaster = isMaster,
                                onSetMaster = {
                                    if (activeServer != null) {
                                        scope.launch {
                                            Toast.makeText(context, "正在请求将 [${node.countryLong.ifEmpty { node.countryShort }} ${node.ip}] 设为系统主出口...", Toast.LENGTH_SHORT).show()
                                            val res = AimiliApplication.instance.apiClient.connectMaster(activeServer, node.id)
                                            if (res.isSuccess) {
                                                activeMasterIp = node.ip
                                                Toast.makeText(context, "已将 [${node.countryLong.ifEmpty { node.countryShort }} ${node.ip}] 设为主网关出口", Toast.LENGTH_SHORT).show()
                                                refreshAllNodes()
                                            } else {
                                                Toast.makeText(context, "切换主出口失败: ${res.exceptionOrNull()?.message}", Toast.LENGTH_LONG).show()
                                            }
                                        }
                                    }
                                },
                                isFavorite = node.isFavorite,
                                onToggleFavorite = {
                                    if (activeServer != null) {
                                        scope.launch {
                                            AimiliApplication.instance.apiClient.toggleFavorite(activeServer, node.id)
                                            allNodes = allNodes.map { if (it.id == node.id) it.copy(isFavorite = !it.isFavorite) else it }
                                            Toast.makeText(context, if (!node.isFavorite) "已收藏 [${node.ip}]" else "已取消收藏", Toast.LENGTH_SHORT).show()
                                        }
                                    }
                                },
                                onStartTunnel = {
                                    if (activeServer != null) {
                                        scope.launch {
                                            AimiliApplication.instance.apiClient.startTunnel(activeServer, node.id)
                                            Toast.makeText(context, "已在 [${activeServer.name}] 为该节点拉起独立并发网卡！", Toast.LENGTH_SHORT).show()
                                        }
                                    }
                                },
                                onBlacklist = {
                                    if (activeServer != null) {
                                        scope.launch {
                                            val res = AimiliApplication.instance.apiClient.addBlacklist(activeServer, node.id, node.ip, node.countryShort)
                                            if (res.isSuccess) {
                                                allNodes = allNodes.filter { it.id != node.id }
                                                Toast.makeText(context, "已将节点 [${node.ip}] 移入 24 小时隔离屏蔽库", Toast.LENGTH_SHORT).show()
                                                val blRes = AimiliApplication.instance.apiClient.fetchBlacklist(activeServer)
                                                if (blRes.isSuccess) {
                                                    blacklistItems = blRes.getOrNull() ?: emptyList()
                                                }
                                            } else {
                                                Toast.makeText(context, "屏蔽失败: ${res.exceptionOrNull()?.message}", Toast.LENGTH_SHORT).show()
                                            }
                                        }
                                    }
                                }
                            )
                        }
                    }
                } else {
                    // 竖屏/手机：单列平滑可滚动流 (全量展示所有节点)
                    LazyColumn(
                        verticalArrangement = Arrangement.spacedBy(10.dp),
                        modifier = Modifier
                            .fillMaxWidth()
                            .weight(1f)
                    ) {
                        items(filteredNodes, key = { it.id }) { node ->
                            val isMaster = node.ip == activeMasterIp || (activeServer != null && activeServer.exitIp == node.ip)
                            FullNodeCard(
                                node = node,
                                isMaster = isMaster,
                                onSetMaster = {
                                    if (activeServer != null) {
                                        scope.launch {
                                            Toast.makeText(context, "正在请求将 [${node.countryLong.ifEmpty { node.countryShort }} ${node.ip}] 设为系统主出口...", Toast.LENGTH_SHORT).show()
                                            val res = AimiliApplication.instance.apiClient.connectMaster(activeServer, node.id)
                                            if (res.isSuccess) {
                                                activeMasterIp = node.ip
                                                Toast.makeText(context, "已将 [${node.countryLong.ifEmpty { node.countryShort }} ${node.ip}] 设为主网关出口", Toast.LENGTH_SHORT).show()
                                                refreshAllNodes()
                                            } else {
                                                Toast.makeText(context, "切换主出口失败: ${res.exceptionOrNull()?.message}", Toast.LENGTH_LONG).show()
                                            }
                                        }
                                    }
                                },
                                isFavorite = node.isFavorite,
                                onToggleFavorite = {
                                    if (activeServer != null) {
                                        scope.launch {
                                            AimiliApplication.instance.apiClient.toggleFavorite(activeServer, node.id)
                                            allNodes = allNodes.map { if (it.id == node.id) it.copy(isFavorite = !it.isFavorite) else it }
                                            Toast.makeText(context, if (!node.isFavorite) "已收藏 [${node.ip}]" else "已取消收藏", Toast.LENGTH_SHORT).show()
                                        }
                                    }
                                },
                                onStartTunnel = {
                                    if (activeServer != null) {
                                        scope.launch {
                                            AimiliApplication.instance.apiClient.startTunnel(activeServer, node.id)
                                            Toast.makeText(context, "已在 [${activeServer.name}] 为该节点拉起独立并发网卡！", Toast.LENGTH_SHORT).show()
                                        }
                                    }
                                },
                                onBlacklist = {
                                    if (activeServer != null) {
                                        scope.launch {
                                            val res = AimiliApplication.instance.apiClient.addBlacklist(activeServer, node.id, node.ip, node.countryShort)
                                            if (res.isSuccess) {
                                                allNodes = allNodes.filter { it.id != node.id }
                                                Toast.makeText(context, "已将节点 [${node.ip}] 移入 24 小时隔离屏蔽库", Toast.LENGTH_SHORT).show()
                                                val blRes = AimiliApplication.instance.apiClient.fetchBlacklist(activeServer)
                                                if (blRes.isSuccess) {
                                                    blacklistItems = blRes.getOrNull() ?: emptyList()
                                                }
                                            } else {
                                                Toast.makeText(context, "屏蔽失败: ${res.exceptionOrNull()?.message}", Toast.LENGTH_SHORT).show()
                                            }
                                        }
                                    }
                                }
                            )
                        }
                    }
                }
            }
        }
    }

    // Blacklist Modal Bottom Sheet
    if (showBlacklistSheet) {
        ModalBottomSheet(
            onDismissRequest = { showBlacklistSheet = false },
            sheetState = blacklistSheetState,
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
                    text = "故障隔离屏蔽库 (Blacklist)",
                    style = MaterialTheme.typography.titleLarge,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.onSurface
                )
                Text(
                    text = "包含因吞吐量低于 70KB/s、握手超时或离线而被系统自动隔离的死节点 (${blacklistItems.size} 个)",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )

                Spacer(Modifier.height(14.dp))
                HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
                Spacer(Modifier.height(12.dp))

                // Action button: 一键批量探活复活
                ConnectedButtonGroup(
                    items = listOf(
                        ConnectedButtonItem(
                            text = "一键批量探活复活",
                            style = ConnectedButtonStyle.Filled,
                            icon = Icons.Rounded.Refresh,
                            onClick = {
                                if (activeServer != null) {
                                    scope.launch {
                                        AimiliApplication.instance.apiClient.resurrectBlacklist(activeServer)
                                        Toast.makeText(context, "已在后台启动探活探测，恢复连通的节点将自动解封放回候选池！", Toast.LENGTH_SHORT).show()
                                    }
                                }
                            }
                        )
                    )
                )

                Spacer(Modifier.height(14.dp))

                LazyColumn(
                    modifier = Modifier.height(180.dp),
                    verticalArrangement = Arrangement.spacedBy(6.dp)
                ) {
                    items(blacklistItems) { item ->
                        Surface(
                            modifier = Modifier.fillMaxWidth(),
                            shape = RoundedCornerShape(8.dp),
                            color = MaterialTheme.colorScheme.surfaceContainerHighest
                        ) {
                            Row(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(horizontal = 12.dp, vertical = 6.dp),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Text(
                                    text = "${item.ip} (${item.country}) - ${item.reason}",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.error,
                                    modifier = Modifier.weight(1f)
                                )
                                IconButton(
                                    onClick = {
                                        if (activeServer != null) {
                                            scope.launch {
                                                AimiliApplication.instance.apiClient.removeBlacklist(activeServer, item.nodeId)
                                                blacklistItems = blacklistItems.filter { it.nodeId != item.nodeId }
                                                Toast.makeText(context, "已解除对 [${item.ip}] 的屏蔽！", Toast.LENGTH_SHORT).show()
                                                refreshAllNodes()
                                            }
                                        }
                                    },
                                    modifier = Modifier.size(28.dp)
                                ) {
                                    Icon(
                                        imageVector = Icons.Rounded.Delete,
                                        contentDescription = "解除屏蔽",
                                        tint = MaterialTheme.colorScheme.error,
                                        modifier = Modifier.size(16.dp)
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
