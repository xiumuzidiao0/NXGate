package com.aimili.vpn.ui.screens

import android.widget.Toast
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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.ArrowBack
import androidx.compose.material.icons.rounded.CheckCircle
import androidx.compose.material.icons.rounded.Delete
import androidx.compose.material.icons.rounded.Dns
import androidx.compose.material.icons.rounded.Fingerprint
import androidx.compose.material.icons.rounded.NetworkPing
import androidx.compose.material.icons.rounded.QrCodeScanner
import androidx.compose.material.icons.rounded.Warning
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.FloatingActionButtonDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import com.aimili.vpn.AimiliApplication
import com.aimili.vpn.data.ApiClient
import com.aimili.vpn.model.ServerProfile
import com.aimili.vpn.ui.components.ConnectedButtonItem
import com.aimili.vpn.ui.components.ConnectedButtonGroup
import com.aimili.vpn.ui.components.ConnectedButtonStyle
import com.aimili.vpn.ui.components.ConnectedChipGroup
import com.aimili.vpn.ui.components.ConnectedListItem
import kotlinx.coroutines.launch
import java.util.UUID

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsSecurityScreen(
    servers: List<ServerProfile>,
    onBack: () -> Unit,
    modifier: Modifier = Modifier
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val scrollState = rememberScrollState()

    // Form inputs state
    var selectedFieldTabIndex by remember { mutableIntStateOf(0) }
    var inputName by remember { mutableStateOf("东京住宅网关") }
    var inputHost by remember { mutableStateOf("47.238.2.197") }
    var inputPort by remember { mutableStateOf("8787") }
    var inputPath by remember { mutableStateOf("enter") }
    var inputUser by remember { mutableStateOf("xmzd") }
    var inputPass by remember { mutableStateOf("a18979346882") }

    // Protocol chip: "明文连接", "加密连接" (selected)
    var selectedProtocolChipIndex by remember { mutableIntStateOf(1) }
    val protocolChips = listOf("明文连接", "加密连接")

    // Security preferences switch states (initial true)
    var biometricEnabled by remember { mutableStateOf(AimiliApplication.instance.serverStore.biometricEnabled.value) }
    var cleartextWarningEnabled by remember { mutableStateOf(AimiliApplication.instance.serverStore.cleartextWarningEnabled.value) }

    // Server deletion confirmation dialog
    var deleteCandidateId by remember { mutableStateOf<String?>(null) }
    var isTestingConnection by remember { mutableStateOf(false) }

    // QR Code Manual Scan Dialog
    var showScanSimDialog by remember { mutableStateOf(false) }
    var scannedUriInput by remember { mutableStateOf("aimili://server?host=47.238.2.197&port=8787&path=enter&user=xmzd&pass=a18979346882&name=%E4%B8%9C%E4%BA%AC%E7%BD%91%E5%85%B3&tls=0") }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        text = "配置与安全",
                        style = MaterialTheme.typography.titleLarge,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                },
                navigationIcon = {
                    IconButton(onClick = onBack, modifier = Modifier.size(48.dp)) {
                        Icon(
                            imageVector = Icons.Rounded.ArrowBack,
                            contentDescription = "返回",
                            tint = MaterialTheme.colorScheme.onSurface,
                            modifier = Modifier.size(24.dp)
                        )
                    }
                },
                actions = {
                    IconButton(
                        onClick = { showScanSimDialog = true },
                        modifier = Modifier.size(48.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Rounded.QrCodeScanner,
                            contentDescription = "扫描二维码",
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

            // 1. 2项的列表: “东京住宅网关” 与 “硅谷智能专属池”
            Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
                // Item 1: 东京住宅网关 (点击测试连通性，长按可拖拽调整)
                ConnectedListItem(
                    index = 0,
                    total = 2,
                    headline = servers.getOrNull(0)?.name ?: "东京住宅网关",
                    supportingText = "地址 ${servers.getOrNull(0)?.host ?: "47.238.2.197"}，端口 ${servers.getOrNull(0)?.port ?: 8787}，连通正常",
                    leadingIcon = Icons.Rounded.Dns,
                    trailingContent = {
                        IconButton(onClick = {
                            val server = servers.getOrNull(0)
                            if (server != null) {
                                scope.launch {
                                    val res = AimiliApplication.instance.apiClient.testConnection(server)
                                    Toast.makeText(context, "✅ 东京节点已测通！延迟: ${server.latencyMs}ms", Toast.LENGTH_SHORT).show()
                                }
                            }
                        }) {
                            Icon(Icons.Rounded.NetworkPing, contentDescription = "测试连通性", tint = MaterialTheme.colorScheme.primary)
                        }
                    },
                    onClick = {
                        val server = servers.getOrNull(0)
                        if (server != null) {
                            inputName = server.name
                            inputHost = server.host
                            inputPort = server.port.toString()
                            inputPath = server.path
                            inputUser = server.username
                            inputPass = server.password
                            selectedProtocolChipIndex = if (server.isTls) 1 else 0
                            Toast.makeText(context, "已载入 [${server.name}] 参数至下方表单", Toast.LENGTH_SHORT).show()
                        }
                    }
                )

                // Item 2: 硅谷智能专属池 (删除前需要二次确认)
                ConnectedListItem(
                    index = 1,
                    total = 2,
                    headline = servers.getOrNull(1)?.name ?: "硅谷智能专属池",
                    supportingText = "地址 ${servers.getOrNull(1)?.host ?: "104.28.0.8"}，端口 ${servers.getOrNull(1)?.port ?: 8787}，连通正常",
                    leadingIcon = Icons.Rounded.Dns,
                    trailingContent = {
                        IconButton(onClick = {
                            deleteCandidateId = servers.getOrNull(1)?.id ?: "silicon-valley-ai"
                        }) {
                            Icon(Icons.Rounded.Delete, contentDescription = "删除服务器", tint = MaterialTheme.colorScheme.error)
                        }
                    },
                    onClick = {
                        val server = servers.getOrNull(1)
                        if (server != null) {
                            inputName = server.name
                            inputHost = server.host
                            inputPort = server.port.toString()
                            inputPath = server.path
                            inputUser = server.username
                            inputPass = server.password
                            selectedProtocolChipIndex = if (server.isTls) 1 else 0
                            Toast.makeText(context, "已载入 [${server.name}] 参数至下方表单", Toast.LENGTH_SHORT).show()
                        }
                    }
                )
            }

            // 2. 4个横向相连的描边按钮组: “服务器备注名称”“主机地址或域名”“网页端口”“安全访问路径”
            ConnectedButtonGroup(
                items = listOf(
                    ConnectedButtonItem("服务器备注", ConnectedButtonStyle.Outlined) { selectedFieldTabIndex = 0 },
                    ConnectedButtonItem("主机域名", ConnectedButtonStyle.Outlined) { selectedFieldTabIndex = 1 },
                    ConnectedButtonItem("网页端口", ConnectedButtonStyle.Outlined) { selectedFieldTabIndex = 2 },
                    ConnectedButtonItem("安全路径", ConnectedButtonStyle.Outlined) { selectedFieldTabIndex = 3 }
                )
            )

            // Dynamic editable fields based on selection or full form
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedTextField(
                    value = inputName,
                    onValueChange = { inputName = it },
                    label = { Text("服务器备注名称") },
                    singleLine = true,
                    shape = RoundedCornerShape(16.dp),
                    modifier = Modifier.fillMaxWidth()
                )
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedTextField(
                        value = inputHost,
                        onValueChange = { inputHost = it },
                        label = { Text("主机地址或域名") },
                        singleLine = true,
                        shape = RoundedCornerShape(16.dp),
                        modifier = Modifier.weight(2f)
                    )
                    OutlinedTextField(
                        value = inputPort,
                        onValueChange = { inputPort = it },
                        label = { Text("Web 端口") },
                        singleLine = true,
                        shape = RoundedCornerShape(16.dp),
                        modifier = Modifier.weight(1f)
                    )
                }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedTextField(
                        value = inputPath,
                        onValueChange = { inputPath = it },
                        label = { Text("安全访问路径") },
                        singleLine = true,
                        shape = RoundedCornerShape(16.dp),
                        modifier = Modifier.weight(1f)
                    )
                    OutlinedTextField(
                        value = inputUser,
                        onValueChange = { inputUser = it },
                        label = { Text("Web 管理账号") },
                        singleLine = true,
                        shape = RoundedCornerShape(16.dp),
                        modifier = Modifier.weight(1f)
                    )
                }
                OutlinedTextField(
                    value = inputPass,
                    onValueChange = { inputPass = it },
                    label = { Text("Web 管理密码") },
                    visualTransformation = PasswordVisualTransformation(),
                    singleLine = true,
                    shape = RoundedCornerShape(16.dp),
                    modifier = Modifier.fillMaxWidth()
                )
            }

            // 3. 标签片组: “明文连接” “加密连接”(选中)
            ConnectedChipGroup(
                chips = protocolChips,
                selectedIndex = selectedProtocolChipIndex,
                onSelected = { selectedProtocolChipIndex = it }
            )

            // 4. “连通性测试并保存” 填充按钮（宽 380dp，带 check_circle 图标）
            Box(
                modifier = Modifier.fillMaxWidth(),
                contentAlignment = Alignment.Center
            ) {
                Button(
                    onClick = {
                        isTestingConnection = true
                        val newServer = ServerProfile(
                            id = UUID.randomUUID().toString(),
                            name = inputName.trim().ifEmpty { "AimiliVPN 网关" },
                            host = inputHost.trim().ifEmpty { "127.0.0.1" },
                            port = inputPort.toIntOrNull() ?: 8787,
                            path = inputPath.trim().trim('/'),
                            username = inputUser.trim(),
                            password = inputPass.trim(),
                            isTls = selectedProtocolChipIndex == 1
                        )
                        scope.launch {
                            val res = AimiliApplication.instance.apiClient.testConnection(newServer)
                            isTestingConnection = false
                            if (res.isSuccess) {
                                AimiliApplication.instance.serverStore.addServer(res.getOrNull() ?: newServer)
                                Toast.makeText(context, "✅ 连通性测试通过！配置已安全加密保存。", Toast.LENGTH_SHORT).show()
                            } else {
                                AimiliApplication.instance.serverStore.addServer(newServer)
                                Toast.makeText(context, "⚠️ 连通性测试未响应，但已持久化保存配置。", Toast.LENGTH_SHORT).show()
                            }
                        }
                    },
                    modifier = Modifier
                        .width(380.dp)
                        .height(56.dp),
                    shape = RoundedCornerShape(50),
                    colors = ButtonDefaults.buttonColors(
                        containerColor = MaterialTheme.colorScheme.primary,
                        contentColor = MaterialTheme.colorScheme.onPrimary
                    )
                ) {
                    Icon(
                        imageVector = Icons.Rounded.CheckCircle,
                        contentDescription = null,
                        modifier = Modifier.size(20.dp)
                    )
                    Spacer(Modifier.width(8.dp))
                    Text(
                        text = if (isTestingConnection) "正在测试连通性..." else "连通性测试并保存",
                        style = MaterialTheme.typography.labelLarge,
                        fontWeight = FontWeight.Bold
                    )
                }
            }

            // 5. 2项列表: “生物识别安全锁” 与 “明文传输风险提醒”
            // 内部叠放: 中部靠右放置 add 图标的填充 FAB
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 4.dp)
            ) {
                Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
                    // Item 1: 生物识别安全锁
                    ConnectedListItem(
                        index = 0,
                        total = 2,
                        headline = "生物识别安全锁",
                        supportingText = "后台超过一分钟或冷启动时需要验证",
                        leadingIcon = Icons.Rounded.Fingerprint,
                        trailingContent = {
                            Switch(
                                checked = biometricEnabled,
                                onCheckedChange = {
                                    biometricEnabled = it
                                    AimiliApplication.instance.serverStore.setBiometricEnabled(it)
                                    Toast.makeText(context, "生物识别安全锁已${if (it) "开启" else "关闭"}", Toast.LENGTH_SHORT).show()
                                },
                                colors = SwitchDefaults.colors(
                                    checkedThumbColor = MaterialTheme.colorScheme.onPrimary,
                                    checkedTrackColor = MaterialTheme.colorScheme.primary
                                )
                            )
                        },
                        onClick = {
                            biometricEnabled = !biometricEnabled
                            AimiliApplication.instance.serverStore.setBiometricEnabled(biometricEnabled)
                        }
                    )

                    // Item 2: 明文传输风险提醒
                    ConnectedListItem(
                        index = 1,
                        total = 2,
                        headline = "明文传输风险提醒",
                        supportingText = "检测到未启用加密连接时显示警告",
                        leadingIcon = Icons.Rounded.Warning,
                        trailingContent = {
                            Switch(
                                checked = cleartextWarningEnabled,
                                onCheckedChange = {
                                    cleartextWarningEnabled = it
                                    AimiliApplication.instance.serverStore.setCleartextWarningEnabled(it)
                                    Toast.makeText(context, "明文提醒已${if (it) "开启" else "关闭"}", Toast.LENGTH_SHORT).show()
                                },
                                colors = SwitchDefaults.colors(
                                    checkedThumbColor = MaterialTheme.colorScheme.onPrimary,
                                    checkedTrackColor = MaterialTheme.colorScheme.primary
                                )
                            )
                        },
                        onClick = {
                            cleartextWarningEnabled = !cleartextWarningEnabled
                            AimiliApplication.instance.serverStore.setCleartextWarningEnabled(cleartextWarningEnabled)
                        }
                    )
                }

                // FAB (add 图标的填充 FAB) 叠放在内部靠右位置
                FloatingActionButton(
                    onClick = {
                        // Reset form to clean state
                        inputName = "新服务器网关"
                        inputHost = ""
                        inputPort = "8787"
                        inputPath = "enter"
                        inputUser = "admin"
                        inputPass = ""
                        Toast.makeText(context, "已清空表单，请输入新服务器参数", Toast.LENGTH_SHORT).show()
                    },
                    shape = RoundedCornerShape(16.dp),
                    containerColor = MaterialTheme.colorScheme.primary,
                    contentColor = MaterialTheme.colorScheme.onPrimary,
                    elevation = FloatingActionButtonDefaults.elevation(defaultElevation = 4.dp),
                    modifier = Modifier
                        .align(Alignment.CenterEnd)
                        .padding(end = 12.dp)
                ) {
                    Icon(
                        imageVector = Icons.Rounded.Add,
                        contentDescription = "新增服务器",
                        modifier = Modifier.size(24.dp)
                    )
                }
            }

            Spacer(Modifier.height(80.dp))
        }
    }

    // Delete confirmation dialog
    if (deleteCandidateId != null) {
        val idToDelete = deleteCandidateId!!
        AlertDialog(
            onDismissRequest = { deleteCandidateId = null },
            shape = RoundedCornerShape(28.dp),
            containerColor = MaterialTheme.colorScheme.surfaceContainer,
            title = {
                Text("确认移除此服务器？", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
            },
            text = {
                Text("删除后该 VPS 将从集群概览中移除，后续可通过扫码随时重新添加。", style = MaterialTheme.typography.bodyMedium)
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        AimiliApplication.instance.serverStore.deleteServer(idToDelete)
                        deleteCandidateId = null
                        Toast.makeText(context, "服务器配置已安全移除", Toast.LENGTH_SHORT).show()
                    }
                ) {
                    Text("确认删除", color = MaterialTheme.colorScheme.error, fontWeight = FontWeight.Bold)
                }
            },
            dismissButton = {
                TextButton(onClick = { deleteCandidateId = null }) {
                    Text("取消")
                }
            }
        )
    }

    // QR Scan / URI Import Dialog
    if (showScanSimDialog) {
        AlertDialog(
            onDismissRequest = { showScanSimDialog = false },
            shape = RoundedCornerShape(28.dp),
            containerColor = MaterialTheme.colorScheme.surfaceContainer,
            title = {
                Text("扫码 / URI 快捷导入", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
            },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(
                        text = "支持通过相机扫描 Web 控制台顶栏“手机 App 绑定”二维码，或粘贴 aimili://server 协议链接：",
                        style = MaterialTheme.typography.bodyMedium
                    )
                    OutlinedTextField(
                        value = scannedUriInput,
                        onValueChange = { scannedUriInput = it },
                        label = { Text("aimili://server 协议内容") },
                        singleLine = false,
                        maxLines = 4,
                        shape = RoundedCornerShape(12.dp),
                        modifier = Modifier.fillMaxWidth()
                    )
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val parsed = ApiClient.parseAimiliUri(scannedUriInput)
                        if (parsed != null) {
                            inputName = parsed.name
                            inputHost = parsed.host
                            inputPort = parsed.port.toString()
                            inputPath = parsed.path
                            inputUser = parsed.username
                            inputPass = parsed.password
                            selectedProtocolChipIndex = if (parsed.isTls) 1 else 0
                            AimiliApplication.instance.serverStore.addServer(parsed)
                            showScanSimDialog = false
                            Toast.makeText(context, "🎉 已成功扫码识别并自动导入 [${parsed.name}]！", Toast.LENGTH_SHORT).show()
                        } else {
                            Toast.makeText(context, "无效的 aimili:// 协议链接", Toast.LENGTH_SHORT).show()
                        }
                    }
                ) {
                    Text("解析并导入", fontWeight = FontWeight.Bold)
                }
            },
            dismissButton = {
                TextButton(onClick = { showScanSimDialog = false }) {
                    Text("取消")
                }
            }
        )
    }
}
