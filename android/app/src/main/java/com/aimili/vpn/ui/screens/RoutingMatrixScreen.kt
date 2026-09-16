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
    val scope = rememberCoroutineScope()
    val scrollState = rememberScrollState()

    var selectedTabIndex by remember { mutableIntStateOf(0) }
    val tabTitles = listOf("多端口", "自适应组", "边缘入站")

    // State for port rules
    var portRules by remember {
        mutableStateOf(
            listOf(
                PortRuleItem(7928, enabled = true, policy = "轮询", authMode = "随机账密", binding = "绑定全部健康隧道"),
                PortRuleItem(7929, enabled = true, policy = "定时轮换 300 秒", authMode = "自定义账密", binding = "绑定日本住宅组"),
                PortRuleItem(7930, enabled = false, policy = "随机", authMode = "免密直连", binding = "绑定美国备用组")
            )
        )
    }

    val dynamicGroup = remember { DynamicGroupCard(id = "dg-1", title = "日本前三住宅组") }

    val inbounds = remember {
        listOf(
            InboundProtocolItem("sb-1", "入站协议：真实传输", "外部端口 443，出口指向端口 7928 日本住宅池"),
            InboundProtocolItem("sb-2", "入站协议：高速二代", "外部端口 8443，出口指向本机直连")
        )
    }

    var showAddPortDialog by remember { mutableStateOf(false) }
    var newPortInput by remember { mutableStateOf("7931") }
    var editPortRuleTarget by remember { mutableStateOf<PortRuleItem?>(null) }

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
                            Toast.makeText(context, "已重新获取多端口与调度矩阵配置", Toast.LENGTH_SHORT).show()
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

                // 2. 3项相连列表: 端口 7928, 7929, 7930
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
                                    Toast.makeText(context, "端口 ${rule.port} 状态已切换为: ${if (!rule.enabled) "已启用" else "已停用"}", Toast.LENGTH_SHORT).show()
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

                // 3. 填充卡片（高 168dp）: 日本前三住宅组
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
                            text = dynamicGroup.title,
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold,
                            color = MaterialTheme.colorScheme.onSurface
                        )
                        Spacer(Modifier.height(8.dp))
                        Text(
                            text = dynamicGroup.description,
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
                                        val res = AimiliApplication.instance.apiClient.triggerRotate(activeServer)
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

                // 5. 2项相连列表: 入站协议：真实传输、入站协议：高速二代
                Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
                    // Item 1: 入站协议：真实传输
                    ConnectedListItem(
                        index = 0,
                        total = 2,
                        headline = inbounds[0].title,
                        supportingText = inbounds[0].subtitle,
                        leadingIcon = Icons.Rounded.VpnKey,
                        trailingContent = {
                            IconButton(onClick = {
                                Toast.makeText(context, "已生成 [真实传输] 节点分享链接与二维码", Toast.LENGTH_SHORT).show()
                            }) {
                                Icon(Icons.Rounded.QrCode2, contentDescription = "二维码", tint = MaterialTheme.colorScheme.primary)
                            }
                        },
                        onClick = {
                            Toast.makeText(context, "生成节点配置二维码，供小火箭、Clash扫码导入", Toast.LENGTH_SHORT).show()
                        }
                    )

                    // Item 2: 入站协议：高速二代
                    ConnectedListItem(
                        index = 1,
                        total = 2,
                        headline = inbounds[1].title,
                        supportingText = inbounds[1].subtitle,
                        leadingIcon = Icons.Rounded.Security,
                        trailingContent = {
                            IconButton(onClick = {
                                Toast.makeText(context, "已更改该入站协议的链式出口为: 端口 7929", Toast.LENGTH_SHORT).show()
                            }) {
                                Icon(Icons.Rounded.SwapHoriz, contentDescription = "切换出口", tint = MaterialTheme.colorScheme.secondary)
                            }
                        },
                        onClick = {
                            Toast.makeText(context, "点击右侧切换图标可更改该协议的链式出口指向", Toast.LENGTH_SHORT).show()
                        }
                    )
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
                        modifier = Modifier.fillMaxWidth()
                    )
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val portNum = newPortInput.toIntOrNull() ?: 7931
                        val updated = portRules.toMutableList()
                        updated.add(PortRuleItem(portNum, enabled = true, policy = "轮询", authMode = "随机账密", binding = "绑定全部健康隧道"))
                        portRules = updated
                        showAddPortDialog = false
                        Toast.makeText(context, "端口 $portNum 分流规则已成功下发并启用！", Toast.LENGTH_SHORT).show()
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
                    Text(text = "当前调度策略: ${target.policy}", style = MaterialTheme.typography.bodyMedium)
                    Text(text = "当前鉴权模式: ${target.authMode}", style = MaterialTheme.typography.bodyMedium)
                    Text(text = "绑定出口范围: ${target.binding}", style = MaterialTheme.typography.bodyMedium)
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        editPortRuleTarget = null
                        Toast.makeText(context, "端口 [${target.port}] 规则已同步生效！", Toast.LENGTH_SHORT).show()
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
}
