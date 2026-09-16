package com.aimili.vpn.ui.screens

import android.widget.Toast
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
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
import androidx.compose.material.icons.automirrored.rounded.Sort
import androidx.compose.material.icons.rounded.Public
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
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
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
import androidx.compose.ui.Modifier
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
import com.aimili.vpn.ui.components.ConnectedListItem
import com.aimili.vpn.ui.components.GlobalServerSwitcherTitle
import kotlinx.coroutines.launch

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
    val scrollState = rememberScrollState()

    var searchQuery by remember { mutableStateOf("") }
    var isSearchActive by remember { mutableStateOf(false) }

    var selectedChipIndex by remember { mutableIntStateOf(0) }
    val chipList = listOf("全部", "日本 180", "美国 45", "住宅宽带", "智能全通")

    val sortOptions = listOf("延迟最低", "带宽最大", "信誉分最高")
    var selectedSortOption by remember { mutableStateOf(sortOptions[0]) }
    var dropdownExpanded by remember { mutableStateOf(false) }

    var isMainNodeStarred by remember { mutableStateOf(false) }

    var showBlacklistSheet by remember { mutableStateOf(false) }
    val blacklistSheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)

    // Full nodes list from server
    var allNodes by remember {
        mutableStateOf(
            listOf(
                NodeCandidate("jp-1", "220.92.176.236", 1194, "JP", "日本", 32, "中华电信骨干", 96, 48200000, "residential", false, "unlocked", "unlocked", "unlocked", "unlocked"),
                NodeCandidate("us-1", "142.250.80.45", 443, "US", "美国", 138, "Comcast", 82, 22500000, "hosting", false, "unlocked", "blocked", "unlocked", "unlocked"),
                NodeCandidate("sg-1", "47.238.2.197", 1194, "SG", "新加坡", 68, "Singtel", 91, 35100000, "residential", false, "unlocked", "unlocked", "unlocked", "unlocked")
            )
        )
    }

    var blacklistItems by remember {
        mutableStateOf(
            listOf(
                BlacklistRecord("218.146.192.89:1685", "218.146.192.89", "KR", "吞吐量趋零 (断流)"),
                BlacklistRecord("118.238.203.45:443", "118.238.203.45", "JP", "TCP 握手敲门未响应"),
                BlacklistRecord("49.213.12.18:1194", "49.213.12.18", "US", "物理出网验证超时")
            )
        )
    }

    // Pull real nodes from active server
    LaunchedEffect(activeServer?.id) {
        if (activeServer != null) {
            val nodesRes = AimiliApplication.instance.apiClient.fetchNodes(activeServer)
            if (nodesRes.isSuccess && !nodesRes.getOrNull().isNullOrEmpty()) {
                allNodes = nodesRes.getOrNull()!!
            }
            val blRes = AimiliApplication.instance.apiClient.fetchBlacklist(activeServer)
            if (blRes.isSuccess && !blRes.getOrNull().isNullOrEmpty()) {
                blacklistItems = blRes.getOrNull()!!
            }
        }
    }

    // Filter nodes
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

    val primaryNode = filteredNodes.firstOrNull() ?: allNodes.first()
    val secondaryNodes = if (filteredNodes.size > 1) filteredNodes.drop(1).take(2) else allNodes.drop(1).take(2)

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
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(horizontal = 16.dp)
                .verticalScroll(scrollState),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Spacer(Modifier.height(4.dp))

            if (isSearchActive) {
                OutlinedTextField(
                    value = searchQuery,
                    onValueChange = { searchQuery = it },
                    label = { Text("搜索 IP / 国家代码 / 运营商") },
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
                    supportingText = { Text("按${selectedSortOption}优先显示候选节点") },
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

            // 3. 填充卡片（高 174dp）: 日本节点展示
            Card(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(174.dp),
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
                        text = primaryNode.title,
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                    Spacer(Modifier.height(8.dp))
                    Text(
                        text = primaryNode.description,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        lineHeight = MaterialTheme.typography.bodyMedium.lineHeight * 1.35f
                    )
                }
            }

            // 4. 按钮组: “设为星标”(色调，可切换为“已星标”填充) “拉起隧道”(填充) “手动屏蔽”(描边)
            ConnectedButtonGroup(
                items = listOf(
                    ConnectedButtonItem(
                        text = if (isMainNodeStarred || primaryNode.isFavorite) "已星标" else "设为星标",
                        style = if (isMainNodeStarred || primaryNode.isFavorite) ConnectedButtonStyle.Filled else ConnectedButtonStyle.Tonal,
                        icon = if (isMainNodeStarred || primaryNode.isFavorite) Icons.Rounded.Star else Icons.Rounded.StarBorder,
                        onClick = {
                            isMainNodeStarred = !isMainNodeStarred
                            if (activeServer != null) {
                                scope.launch {
                                    AimiliApplication.instance.apiClient.toggleFavorite(activeServer, primaryNode.id)
                                }
                            }
                            val msg = if (isMainNodeStarred) "已将 [${primaryNode.ip}] 设为星标置顶！" else "已取消星标"
                            Toast.makeText(context, msg, Toast.LENGTH_SHORT).show()
                        }
                    ),
                    ConnectedButtonItem(
                        text = "拉起隧道",
                        style = ConnectedButtonStyle.Filled,
                        onClick = {
                            if (activeServer != null) {
                                scope.launch {
                                    AimiliApplication.instance.apiClient.startTunnel(activeServer, primaryNode.id)
                                    Toast.makeText(context, "已在 [${activeServer.name}] 为该节点拉起独立并发隧道！", Toast.LENGTH_SHORT).show()
                                }
                            }
                        }
                    ),
                    ConnectedButtonItem(
                        text = "手动屏蔽",
                        style = ConnectedButtonStyle.Outlined,
                        onClick = {
                            Toast.makeText(context, "已将节点 [${primaryNode.ip}] 加入隔离屏蔽库", Toast.LENGTH_SHORT).show()
                        }
                    )
                )
            )

            // 5. 2项列表: 次选候选节点
            val secondaryCount = secondaryNodes.size
            if (secondaryCount > 0) {
                Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
                    secondaryNodes.forEachIndexed { index, node ->
                        ConnectedListItem(
                            index = index,
                            total = secondaryCount,
                            headline = node.title,
                            supportingText = "速度 ${node.speedMbStr}，信誉评分 ${node.score}，${node.unlockDisplay}",
                            leadingIcon = Icons.Rounded.Public,
                            trailingContent = {
                                IconButton(onClick = {
                                    if (activeServer != null) {
                                        scope.launch {
                                            if (index == 0) {
                                                AimiliApplication.instance.apiClient.toggleFavorite(activeServer, node.id)
                                                Toast.makeText(context, "已将该节点添加至个人星标库", Toast.LENGTH_SHORT).show()
                                            } else {
                                                AimiliApplication.instance.apiClient.startTunnel(activeServer, node.id)
                                                Toast.makeText(context, "正在为节点 [${node.ip}] 分配虚拟网卡建立独立出口...", Toast.LENGTH_SHORT).show()
                                            }
                                        }
                                    }
                                }) {
                                    Icon(
                                        imageVector = if (index == 0) Icons.Rounded.StarBorder else Icons.Rounded.RocketLaunch,
                                        contentDescription = null,
                                        tint = if (index == 0) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.secondary
                                    )
                                }
                            },
                            onClick = {
                                Toast.makeText(context, "节点: ${node.ip}:${node.port} (${node.isp})", Toast.LENGTH_SHORT).show()
                            }
                        )
                    }
                }
            }

            // 6. 按钮组: “全量并发测速”(填充) “屏蔽库管理”(色调)
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
                        text = "屏蔽库管理",
                        style = ConnectedButtonStyle.Tonal,
                        onClick = { showBlacklistSheet = true }
                    )
                )
            )

            Spacer(Modifier.height(80.dp))
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
