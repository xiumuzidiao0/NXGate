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
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material.icons.rounded.RocketLaunch
import androidx.compose.material.icons.rounded.Search
import androidx.compose.material.icons.rounded.Star
import androidx.compose.material.icons.rounded.StarBorder
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
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
import com.aimili.vpn.ui.components.ConnectedButtonItem
import com.aimili.vpn.ui.components.ConnectedButtonGroup
import com.aimili.vpn.ui.components.ConnectedButtonStyle
import com.aimili.vpn.ui.components.ConnectedChipGroup
import com.aimili.vpn.ui.components.GlobalServerSwitcherTitle
import com.aimili.vpn.ui.components.UnlockPill
import com.aimili.vpn.ui.components.countryFlag
import kotlinx.coroutines.launch

@Composable
fun FullNodeCard(
    node: NodeCandidate,
    isStarred: Boolean,
    onToggleStar: () -> Unit,
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
            // Row 1: Country + IP:Port + Latency Badge
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = "${countryFlag(node.countryShort)} ${node.countryLong.ifEmpty { node.countryShort }}",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                    Spacer(Modifier.width(6.dp))
                    Text(
                        text = "${node.ip}:${node.port}",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
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
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.weight(1f)
                )
                val isRes = node.ipType == "residential"
                Surface(
                    shape = RoundedCornerShape(6.dp),
                    color = if (isRes) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surfaceContainer
                ) {
                    Text(
                        text = if (isRes) "🏠 原生家宽" else "🏢 机房托管",
                        style = MaterialTheme.typography.labelSmall,
                        color = if (isRes) MaterialTheme.colorScheme.onPrimaryContainer else MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                    )
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
                Text("实测解锁:", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                UnlockPill("ChatGPT", node.openai)
                UnlockPill("Claude", node.claude)
                UnlockPill("Gemini", node.gemini)
                UnlockPill("Netflix", node.netflix)
            }

            Spacer(Modifier.height(2.dp))

            // Row 5: Action Button Group
            ConnectedButtonGroup(
                items = listOf(
                    ConnectedButtonItem(
                        text = if (isStarred) "已星标" else "设为星标",
                        style = if (isStarred) ConnectedButtonStyle.Filled else ConnectedButtonStyle.Tonal,
                        icon = if (isStarred) Icons.Rounded.Star else Icons.Rounded.StarBorder,
                        onClick = onToggleStar
                    ),
                    ConnectedButtonItem(
                        text = "拉起网卡",
                        style = ConnectedButtonStyle.Filled,
                        icon = Icons.Rounded.RocketLaunch,
                        onClick = onStartTunnel
                    ),
                    ConnectedButtonItem(
                        text = "屏蔽",
                        style = ConnectedButtonStyle.Outlined,
                        onClick = onBlacklist
                    )
                ),
                height = 46.dp
            )
        }
    }
}

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

    var selectedChipIndex by remember { mutableIntStateOf(0) }
    val chipList = listOf("全部", "日本", "美国", "原生家宽", "三大AI全通")

    val sortOptions = listOf("延迟最低", "带宽最大", "信誉分最高")
    var selectedSortOption by remember { mutableStateOf(sortOptions[0]) }
    var dropdownExpanded by remember { mutableStateOf(false) }

    var showBlacklistSheet by remember { mutableStateOf(false) }
    val blacklistSheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)

    // Full nodes list from server (dynamically loaded)
    var allNodes by remember { mutableStateOf<List<NodeCandidate>>(emptyList()) }
    var isLoadingNodes by remember { mutableStateOf(false) }
    var nodeLoadError by remember { mutableStateOf<String?>(null) }

    var blacklistItems by remember { mutableStateOf<List<BlacklistRecord>>(emptyList()) }

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
            }
        }
    }

    // Pull real nodes whenever active server changes
    LaunchedEffect(activeServer?.id) {
        refreshAllNodes()
    }

    // Dynamic filtering across all nodes
    val filteredNodes = remember(allNodes, selectedChipIndex, searchQuery, selectedSortOption) {
        var list = allNodes.filter { node ->
            val matchesQuery = searchQuery.isEmpty() ||
                    node.ip.contains(searchQuery, ignoreCase = true) ||
                    node.countryShort.contains(searchQuery, ignoreCase = true) ||
                    node.countryLong.contains(searchQuery, ignoreCase = true) ||
                    node.isp.contains(searchQuery, ignoreCase = true)

            val matchesChip = when (selectedChipIndex) {
                1 -> node.countryShort.equals("JP", ignoreCase = true)
                2 -> node.countryShort.equals("US", ignoreCase = true)
                3 -> node.ipType == "residential"
                4 -> node.openai == "unlocked" && node.claude == "unlocked" && node.gemini == "unlocked"
                else -> true
            }
            matchesQuery && matchesChip
        }

        list = when (selectedSortOption) {
            "延迟最低" -> list.sortedBy { if (it.latencyMs > 0) it.latencyMs else 9999 }
            "带宽最大" -> list.sortedByDescending { it.speedBps }
            "信誉分最高" -> list.sortedByDescending { it.score }
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

                // 1. 横向排列的标签片组
                Row(modifier = Modifier.horizontalScroll(rememberScrollState())) {
                    ConnectedChipGroup(
                        chips = chipList,
                        selectedIndex = selectedChipIndex,
                        onSelected = { selectedChipIndex = it }
                    )
                }

                // 2. 描边下拉菜单（Exposed Dropdown Menu）
                ExposedDropdownMenuBox(
                    expanded = dropdownExpanded,
                    onExpandedChange = { dropdownExpanded = !dropdownExpanded },
                    modifier = Modifier.fillMaxWidth()
                ) {
                    OutlinedTextField(
                        value = selectedSortOption,
                        onValueChange = {},
                        readOnly = true,
                        label = { Text("排序规则") },
                        supportingText = {
                            Text(
                                text = if (isLoadingNodes) "正在加载全量节点清单..." else "按${selectedSortOption}优先显示 (当前共展示 ${filteredNodes.size} 个可用节点)",
                                color = MaterialTheme.colorScheme.primary
                            )
                        },
                        leadingIcon = {
                            Icon(Icons.AutoMirrored.Rounded.Sort, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
                        },
                        trailingIcon = {
                            ExposedDropdownMenuDefaults.TrailingIcon(expanded = dropdownExpanded)
                        },
                        modifier = Modifier
                            .fillMaxWidth()
                            .menuAnchor(),
                        shape = RoundedCornerShape(16.dp),
                        colors = ExposedDropdownMenuDefaults.outlinedTextFieldColors()
                    )

                    ExposedDropdownMenu(
                        expanded = dropdownExpanded,
                        onDismissRequest = { dropdownExpanded = false }
                    ) {
                        sortOptions.forEach { option ->
                            DropdownMenuItem(
                                text = { Text(option, style = MaterialTheme.typography.bodyLarge) },
                                onClick = {
                                    selectedSortOption = option
                                    dropdownExpanded = false
                                }
                            )
                        }
                    }
                }

                // Action buttons bar at the top of list
                ConnectedButtonGroup(
                    items = listOf(
                        ConnectedButtonItem(
                            text = "全量并发测速",
                            style = ConnectedButtonStyle.Filled,
                            onClick = {
                                if (activeServer != null) {
                                    scope.launch {
                                        AimiliApplication.instance.apiClient.probeNodes(activeServer)
                                        Toast.makeText(context, "已触发远端对全量候选节点进行并发敲门与测速！", Toast.LENGTH_SHORT).show()
                                    }
                                }
                            }
                        ),
                        ConnectedButtonItem(
                            text = "屏蔽库管理 (${blacklistItems.size})",
                            style = ConnectedButtonStyle.Tonal,
                            onClick = { showBlacklistSheet = true }
                        )
                    ),
                    height = 46.dp
                )

                if (isLoadingNodes) {
                    LinearProgressIndicator(modifier = Modifier.fillMaxWidth(), color = MaterialTheme.colorScheme.primary)
                }

                // 3. 全量候选节点流 (所有节点全部展示，支持平板横屏双列网格)
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
                            text = "未找到符合当前筛选条件的节点，请尝试切换上方标签或点击全量测速。",
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
                            FullNodeCard(
                                node = node,
                                isStarred = node.isFavorite,
                                onToggleStar = {
                                    if (activeServer != null) {
                                        scope.launch {
                                            AimiliApplication.instance.apiClient.toggleFavorite(activeServer, node.id)
                                            allNodes = allNodes.map { if (it.id == node.id) it.copy(isFavorite = !it.isFavorite) else it }
                                            Toast.makeText(context, if (!node.isFavorite) "已星标 [${node.ip}]" else "已取消星标", Toast.LENGTH_SHORT).show()
                                        }
                                    }
                                },
                                onStartTunnel = {
                                    if (activeServer != null) {
                                        scope.launch {
                                            AimiliApplication.instance.apiClient.startTunnel(activeServer, node.id)
                                            Toast.makeText(context, "已在 [${activeServer.name}] 为该节点拉起独立并发隧道！", Toast.LENGTH_SHORT).show()
                                        }
                                    }
                                },
                                onBlacklist = {
                                    Toast.makeText(context, "已将节点 [${node.ip}] 加入隔离屏蔽库", Toast.LENGTH_SHORT).show()
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
                            FullNodeCard(
                                node = node,
                                isStarred = node.isFavorite,
                                onToggleStar = {
                                    if (activeServer != null) {
                                        scope.launch {
                                            AimiliApplication.instance.apiClient.toggleFavorite(activeServer, node.id)
                                            allNodes = allNodes.map { if (it.id == node.id) it.copy(isFavorite = !it.isFavorite) else it }
                                            Toast.makeText(context, if (!node.isFavorite) "已星标 [${node.ip}]" else "已取消星标", Toast.LENGTH_SHORT).show()
                                        }
                                    }
                                },
                                onStartTunnel = {
                                    if (activeServer != null) {
                                        scope.launch {
                                            AimiliApplication.instance.apiClient.startTunnel(activeServer, node.id)
                                            Toast.makeText(context, "已在 [${activeServer.name}] 为该节点拉起独立并发隧道！", Toast.LENGTH_SHORT).show()
                                        }
                                    }
                                },
                                onBlacklist = {
                                    Toast.makeText(context, "已将节点 [${node.ip}] 加入隔离屏蔽库", Toast.LENGTH_SHORT).show()
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
                                    .padding(10.dp),
                                horizontalArrangement = Arrangement.SpaceBetween
                            ) {
                                Text(
                                    text = "${item.ip} (${item.country}) - ${item.reason}",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.error
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}
