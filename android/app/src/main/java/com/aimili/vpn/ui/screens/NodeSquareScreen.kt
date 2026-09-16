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
import androidx.compose.material.icons.rounded.CheckCircle
import androidx.compose.material.icons.rounded.Public
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material.icons.rounded.RocketLaunch
import androidx.compose.material.icons.rounded.Search
import androidx.compose.material.icons.rounded.Sort
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

    var selectedChipIndex by remember { mutableIntStateOf(0) }
    val chipList = listOf("全部", "日本 180", "美国 45", "住宅宽带", "智能全通")

    // Dropdown sort options
    val sortOptions = listOf("延迟最低", "带宽最大", "信誉分最高")
    var selectedSortOption by remember { mutableStateOf(sortOptions[0]) }
    var dropdownExpanded by remember { mutableStateOf(false) }

    // Star state for main card
    var isMainNodeStarred by remember { mutableStateOf(false) }

    // Show blacklist sheet
    var showBlacklistSheet by remember { mutableStateOf(false) }
    val blacklistSheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)

    val mainCandidate = remember {
        NodeCandidate(
            id = "jp-node-1",
            title = "日本节点 220.92.176.236 延迟32毫秒",
            latencyMs = 32,
            isp = "中华电信骨干",
            score = 96,
            ipType = "原生家宽",
            speedStr = "48.2 兆每秒",
            unlockStr = "智能服务与流媒体全通过",
            isStarred = false
        )
    }

    val otherNodes = remember {
        listOf(
            NodeCandidate("us-node-1", "美国节点 142.250.80.45 延迟138毫秒", 138, "Comcast", 82, "机房托管", "22.5 兆每秒", "克劳德未通过"),
            NodeCandidate("sg-node-1", "新加坡节点 47.238.2.197 延迟68毫秒", 68, "Singtel", 91, "原生家宽", "35.1 兆每秒", "智能服务全通过")
        )
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
                            Toast.makeText(context, "支持检索国家代码 (JP/US)、IP 或运营商关键词", Toast.LENGTH_SHORT).show()
                        },
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
                        Icon(Icons.Rounded.Sort, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
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

            // 3. 填充卡片（高 174dp）: 日本节点 220.92.176.236 延迟32毫秒
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
                        text = mainCandidate.title,
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                    Spacer(Modifier.height(8.dp))
                    Text(
                        text = mainCandidate.description,
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
                        text = if (isMainNodeStarred) "已星标" else "设为星标",
                        style = if (isMainNodeStarred) ConnectedButtonStyle.Filled else ConnectedButtonStyle.Tonal,
                        icon = if (isMainNodeStarred) Icons.Rounded.Star else Icons.Rounded.StarBorder,
                        onClick = {
                            isMainNodeStarred = !isMainNodeStarred
                            val msg = if (isMainNodeStarred) "已将该日本住宅节点设为星标置顶！" else "已取消星标"
                            Toast.makeText(context, msg, Toast.LENGTH_SHORT).show()
                        }
                    ),
                    ConnectedButtonItem(
                        text = "拉起隧道",
                        style = ConnectedButtonStyle.Filled,
                        onClick = {
                            Toast.makeText(context, "正在为该节点独立拉起虚拟网卡并配置策略路由...", Toast.LENGTH_SHORT).show()
                        }
                    ),
                    ConnectedButtonItem(
                        text = "手动屏蔽",
                        style = ConnectedButtonStyle.Outlined,
                        onClick = {
                            Toast.makeText(context, "已将节点加入隔离屏蔽库", Toast.LENGTH_SHORT).show()
                        }
                    )
                )
            )

            // 5. 2项列表: 美国节点、新加坡节点
            Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
                // Item 1: 美国节点 142.250.80.45
                ConnectedListItem(
                    index = 0,
                    total = 2,
                    headline = otherNodes[0].title,
                    supportingText = "速度 22.5 兆每秒，信誉评分 82，克劳德未通过",
                    leadingIcon = Icons.Rounded.Public,
                    trailingContent = {
                        IconButton(onClick = {
                            Toast.makeText(context, "已将该节点添加至个人星标库", Toast.LENGTH_SHORT).show()
                        }) {
                            Icon(Icons.Rounded.StarBorder, contentDescription = "星标", tint = MaterialTheme.colorScheme.primary)
                        }
                    },
                    onClick = {
                        Toast.makeText(context, "查看该节点多维信息详情与测速历史", Toast.LENGTH_SHORT).show()
                    }
                )

                // Item 2: 新加坡节点 47.238.2.197
                ConnectedListItem(
                    index = 1,
                    total = 2,
                    headline = otherNodes[1].title,
                    supportingText = "速度 35.1 兆每秒，信誉评分 91，智能服务全通过",
                    leadingIcon = Icons.Rounded.Public,
                    trailingContent = {
                        IconButton(onClick = {
                            Toast.makeText(context, "正在为新加坡节点分配虚拟网卡建立独立出口...", Toast.LENGTH_SHORT).show()
                        }) {
                            Icon(Icons.Rounded.RocketLaunch, contentDescription = "拉起出口", tint = MaterialTheme.colorScheme.secondary)
                        }
                    },
                    onClick = {
                        Toast.makeText(context, "新加坡节点全AI通过，可随时用于出海调度", Toast.LENGTH_SHORT).show()
                    }
                )
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
                                    Toast.makeText(context, "已触发远端服务器对全量候选节点进行并发预检和测速！", Toast.LENGTH_SHORT).show()
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
                    text = "包含因吞吐量低于 70KB/s、握手超时或离线而被系统自动隔离的死节点",
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
                                        Toast.makeText(context, "已在后台启动探活探测，恢复连通的节点将自动放回候选池！", Toast.LENGTH_SHORT).show()
                                    }
                                }
                            }
                        )
                    )
                )

                Spacer(Modifier.height(14.dp))

                val mockBlacklisted = listOf(
                    "218.146.192.89:1685 - 隔离原因: 吞吐量趋零(断流)",
                    "118.238.203.45:443 - 隔离原因: TCP 握手敲门未响应",
                    "49.213.12.18:1194 - 隔离原因: 物理出网验证超时"
                )

                LazyColumn(
                    modifier = Modifier.height(160.dp),
                    verticalArrangement = Arrangement.spacedBy(6.dp)
                ) {
                    items(mockBlacklisted) { item ->
                        Surface(
                            modifier = Modifier.fillMaxWidth(),
                            shape = RoundedCornerShape(8.dp),
                            color = MaterialTheme.colorScheme.surfaceContainerHighest
                        ) {
                            Text(
                                text = item,
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.error,
                                modifier = Modifier.padding(10.dp)
                            )
                        }
                    }
                }
            }
        }
    }
}
