package com.nxgate.app.ui.screens

import android.content.res.Configuration
import android.widget.Toast
import androidx.biometric.BiometricManager
import androidx.compose.foundation.BorderStroke
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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.CheckCircle
import androidx.compose.material.icons.rounded.ColorLens
import androidx.compose.material.icons.rounded.Delete
import androidx.compose.material.icons.rounded.Dns
import androidx.compose.material.icons.rounded.Fingerprint
import androidx.compose.material.icons.rounded.NetworkPing
import androidx.compose.material.icons.rounded.Palette
import androidx.compose.material.icons.rounded.QrCodeScanner
import androidx.compose.material.icons.rounded.Warning
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.FloatingActionButtonDefaults
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import com.nxgate.app.NXGateApplication
import com.nxgate.app.MainActivity
import com.nxgate.app.data.ApiClient
import com.nxgate.app.model.ServerProfile
import com.nxgate.app.theme.ACCENT_OPTIONS
import com.nxgate.app.theme.AVAILABLE_PALETTES
import com.nxgate.app.theme.BASE_TONE_OPTIONS
import com.nxgate.app.ui.components.AppExposedDropdown
import com.nxgate.app.ui.components.AppPermissionRationaleDialog
import com.nxgate.app.ui.components.CameraQrScannerDialog
import com.nxgate.app.ui.components.ConnectedButtonItem
import com.nxgate.app.ui.components.ConnectedButtonGroup
import com.nxgate.app.ui.components.ConnectedButtonStyle
import com.nxgate.app.ui.components.ConnectedChipGroup
import com.nxgate.app.ui.components.ConnectedListItem
import com.nxgate.app.ui.components.openSecuritySettings
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

    val configuration = LocalConfiguration.current
    val isLandscape = configuration.orientation == Configuration.ORIENTATION_LANDSCAPE
    val isTablet = configuration.screenWidthDp >= 600
    val isTabletLandscape = isTablet && isLandscape

    // Tracking which server is being edited (null = adding a new server)
    var editingServerId by remember { mutableStateOf<String?>(servers.firstOrNull()?.id) }

    // Form inputs state initialized from first server or default
    val initialServer = servers.firstOrNull()
    var inputName by remember { mutableStateOf(initialServer?.name ?: "东京住宅网关") }
    var inputHost by remember { mutableStateOf(initialServer?.host ?: "47.238.2.197") }
    var inputPort by remember { mutableStateOf(initialServer?.port?.toString() ?: "8787") }
    var inputPath by remember { mutableStateOf(initialServer?.path ?: "enter") }
    var inputUser by remember { mutableStateOf(initialServer?.username ?: "xmzd") }
    var inputPass by remember { mutableStateOf(initialServer?.password ?: "a18979346882") }

    // Protocol chip: "明文连接", "加密连接" (selected)
    var selectedProtocolChipIndex by remember { mutableIntStateOf(if (initialServer?.isTls == true) 1 else 0) }
    val protocolChips = listOf("明文连接", "加密连接")

    // Security preferences switch states
    var biometricEnabled by remember { mutableStateOf(NXGateApplication.instance.serverStore.biometricEnabled.value) }
    var cleartextWarningEnabled by remember { mutableStateOf(NXGateApplication.instance.serverStore.cleartextWarningEnabled.value) }

    // Server deletion confirmation target
    var deleteCandidate by remember { mutableStateOf<ServerProfile?>(null) }
    var isTestingConnection by remember { mutableStateOf(false) }

    // QR Code Manual Scan Dialog
    var showScanSimDialog by remember { mutableStateOf(false) }
    var scannedUriInput by remember { mutableStateOf("") }

    val onToggleBiometric: (Boolean) -> Unit = { targetChecked ->
        if (targetChecked) {
            val bm = BiometricManager.from(context)
            val canAuth = bm.canAuthenticate(
                BiometricManager.Authenticators.BIOMETRIC_STRONG or
                        BiometricManager.Authenticators.BIOMETRIC_WEAK or
                        BiometricManager.Authenticators.DEVICE_CREDENTIAL
            )
            when (canAuth) {
                BiometricManager.BIOMETRIC_ERROR_NONE_ENROLLED -> {
                    Toast.makeText(context, "系统尚未录入指纹或锁屏密码，正在跳转系统设置...", Toast.LENGTH_LONG).show()
                    openSecuritySettings(context)
                }
                BiometricManager.BIOMETRIC_ERROR_NO_HARDWARE -> {
                    Toast.makeText(context, "当前设备未配备指纹或面容硬件，无法开启安全锁", Toast.LENGTH_SHORT).show()
                }
                else -> {
                    MainActivity.instance?.showBiometricPrompt(
                        onSuccess = {
                            biometricEnabled = true
                            NXGateApplication.instance.serverStore.setBiometricEnabled(true)
                            Toast.makeText(context, "生物识别安全锁已开启，切出后台与冷启动时将验证", Toast.LENGTH_SHORT).show()
                        },
                        onError = { err ->
                            Toast.makeText(context, "指纹/面容验证未通过: $err", Toast.LENGTH_SHORT).show()
                        }
                    ) ?: run {
                        biometricEnabled = true
                        NXGateApplication.instance.serverStore.setBiometricEnabled(true)
                        Toast.makeText(context, "生物识别安全锁已开启", Toast.LENGTH_SHORT).show()
                    }
                }
            }
        } else {
            MainActivity.instance?.showBiometricPrompt(
                onSuccess = {
                    biometricEnabled = false
                    NXGateApplication.instance.serverStore.setBiometricEnabled(false)
                    Toast.makeText(context, "生物识别安全锁已解除", Toast.LENGTH_SHORT).show()
                },
                onError = { err ->
                    Toast.makeText(context, "验证未通过，未能关闭安全锁: $err", Toast.LENGTH_SHORT).show()
                }
            ) ?: run {
                biometricEnabled = false
                NXGateApplication.instance.serverStore.setBiometricEnabled(false)
                Toast.makeText(context, "生物识别安全锁已关闭", Toast.LENGTH_SHORT).show()
            }
        }
    }

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
                            imageVector = Icons.AutoMirrored.Rounded.ArrowBack,
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
        floatingActionButton = {
            FloatingActionButton(
                onClick = {
                    editingServerId = null
                    inputName = "新服务器网关"
                    inputHost = ""
                    inputPort = "8787"
                    inputPath = "enter"
                    inputUser = "admin"
                    inputPass = ""
                    Toast.makeText(context, "已切换为新增模式，可在右/下方表单填写或扫码导入！", Toast.LENGTH_SHORT).show()
                },
                shape = RoundedCornerShape(16.dp),
                containerColor = MaterialTheme.colorScheme.primary,
                contentColor = MaterialTheme.colorScheme.onPrimary,
                elevation = FloatingActionButtonDefaults.elevation(defaultElevation = 6.dp),
                modifier = Modifier.padding(bottom = 16.dp, end = 16.dp)
            ) {
                Icon(
                    imageVector = Icons.Rounded.Add,
                    contentDescription = "新增服务器",
                    modifier = Modifier.size(24.dp)
                )
            }
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
                    .widthIn(max = if (isTabletLandscape) 1100.dp else if (isTablet) 840.dp else 500.dp)
                    .align(Alignment.TopCenter)
                    .verticalScroll(scrollState),
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                Spacer(Modifier.height(4.dp))

                if (isTabletLandscape) {
                    // ==================== 平板横屏：左右双列工作台布局 ====================
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(20.dp)
                    ) {
                        // 左侧：服务器管理列表
                        Column(
                            modifier = Modifier.weight(1f),
                            verticalArrangement = Arrangement.spacedBy(12.dp)
                        ) {
                            Text(
                                text = "已纳管网关列表 (${servers.size} 台)",
                                style = MaterialTheme.typography.titleMedium,
                                fontWeight = FontWeight.Bold,
                                color = MaterialTheme.colorScheme.onSurface
                            )

                            if (servers.isEmpty()) {
                                Surface(
                                    modifier = Modifier.fillMaxWidth(),
                                    shape = RoundedCornerShape(16.dp),
                                    color = MaterialTheme.colorScheme.surfaceContainerLow
                                ) {
                                    Text(
                                        text = "暂无已纳管的 VPS，请在右侧表单填写或右上角扫码添加。",
                                        style = MaterialTheme.typography.bodyMedium,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                                        modifier = Modifier.padding(16.dp)
                                    )
                                }
                            } else {
                                Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
                                    servers.forEachIndexed { index, server ->
                                        ConnectedListItem(
                                            index = index,
                                            total = servers.size,
                                            headline = server.name,
                                            supportingText = "地址 ${server.host}:${server.port}，${if (server.isOnline) "连通 (${server.latencyMs}ms)" else "待测"}",
                                            leadingIcon = Icons.Rounded.Dns,
                                            trailingContent = {
                                                Row(verticalAlignment = Alignment.CenterVertically) {
                                                    IconButton(onClick = {
                                                        scope.launch {
                                                            val res = NXGateApplication.instance.apiClient.testConnection(server)
                                                            if (res.isSuccess) {
                                                                NXGateApplication.instance.serverStore.updateServer(res.getOrNull() ?: server)
                                                                Toast.makeText(context, "[${server.name}] 测活通过！延迟: ${server.latencyMs}ms", Toast.LENGTH_SHORT).show()
                                                            } else {
                                                                Toast.makeText(context, "[${server.name}] 测活失败: ${res.exceptionOrNull()?.message}", Toast.LENGTH_SHORT).show()
                                                            }
                                                        }
                                                    }) {
                                                        Icon(Icons.Rounded.NetworkPing, contentDescription = "测试连通性", tint = MaterialTheme.colorScheme.primary)
                                                    }
                                                    IconButton(onClick = {
                                                        deleteCandidate = server
                                                    }) {
                                                        Icon(Icons.Rounded.Delete, contentDescription = "删除服务器", tint = MaterialTheme.colorScheme.error)
                                                    }
                                                }
                                            },
                                            onClick = {
                                                editingServerId = server.id
                                                inputName = server.name
                                                inputHost = server.host
                                                inputPort = server.port.toString()
                                                inputPath = server.path
                                                inputUser = server.username
                                                inputPass = server.password
                                                selectedProtocolChipIndex = if (server.isTls) 1 else 0
                                                Toast.makeText(context, "已载入 [${server.name}] 至右侧表单", Toast.LENGTH_SHORT).show()
                                            }
                                        )
                                    }
                                }
                            }
                        }

                        // 右侧：服务器配置表单与安全偏好
                        Column(
                            modifier = Modifier.weight(1f),
                            verticalArrangement = Arrangement.spacedBy(14.dp)
                        ) {
                            Text(
                                text = if (editingServerId != null) "编辑服务器参数" else "新增服务器参数",
                                style = MaterialTheme.typography.titleMedium,
                                fontWeight = FontWeight.Bold,
                                color = MaterialTheme.colorScheme.onSurface
                            )

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

                            ConnectedChipGroup(
                                chips = protocolChips,
                                selectedIndex = selectedProtocolChipIndex,
                                onSelected = { selectedProtocolChipIndex = it }
                            )

                            if (selectedProtocolChipIndex == 0 && cleartextWarningEnabled) {
                                Surface(
                                    shape = RoundedCornerShape(12.dp),
                                    color = MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.5f),
                                    border = BorderStroke(0.5.dp, MaterialTheme.colorScheme.error.copy(alpha = 0.5f)),
                                    modifier = Modifier.fillMaxWidth()
                                ) {
                                    Row(
                                        modifier = Modifier.padding(horizontal = 12.dp, vertical = 8.dp),
                                        verticalAlignment = Alignment.CenterVertically
                                    ) {
                                        Icon(
                                            imageVector = Icons.Rounded.Warning,
                                            contentDescription = null,
                                            tint = MaterialTheme.colorScheme.error,
                                            modifier = Modifier.size(16.dp)
                                        )
                                        Spacer(Modifier.width(6.dp))
                                        Text(
                                            text = "明文传输警告：HTTP 通信未经 TLS 加密，请仅在受信任私网中使用。",
                                            style = MaterialTheme.typography.labelSmall,
                                            color = MaterialTheme.colorScheme.onErrorContainer
                                        )
                                    }
                                }
                            }

                            Button(
                                onClick = {
                                    isTestingConnection = true
                                    val targetId = editingServerId ?: UUID.randomUUID().toString()
                                    val newServer = ServerProfile(
                                        id = targetId,
                                        name = inputName.trim().ifEmpty { "NXGate 网关" },
                                        host = inputHost.trim().ifEmpty { "127.0.0.1" },
                                        port = inputPort.toIntOrNull() ?: 8787,
                                        path = inputPath.trim().trim('/'),
                                        username = inputUser.trim(),
                                        password = inputPass.trim(),
                                        isTls = selectedProtocolChipIndex == 1
                                    )
                                    scope.launch {
                                        val res = NXGateApplication.instance.apiClient.testConnection(newServer)
                                        isTestingConnection = false
                                        val profileToSave = res.getOrNull() ?: newServer

                                        if (editingServerId != null && servers.any { it.id == editingServerId }) {
                                            NXGateApplication.instance.serverStore.updateServer(profileToSave)
                                            Toast.makeText(context, "[${profileToSave.name}] 配置已更新并安全保存", Toast.LENGTH_SHORT).show()
                                        } else {
                                            NXGateApplication.instance.serverStore.addServer(profileToSave)
                                            editingServerId = profileToSave.id
                                            Toast.makeText(context, "新服务器 [${profileToSave.name}] 已成功添加", Toast.LENGTH_SHORT).show()
                                        }
                                    }
                                },
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .height(52.dp),
                                shape = RoundedCornerShape(50),
                                colors = ButtonDefaults.buttonColors(
                                    containerColor = MaterialTheme.colorScheme.primary,
                                    contentColor = MaterialTheme.colorScheme.onPrimary
                                )
                            ) {
                                Icon(Icons.Rounded.CheckCircle, contentDescription = null, modifier = Modifier.size(20.dp))
                                Spacer(Modifier.width(8.dp))
                                Text(
                                    text = if (isTestingConnection) "正在测试连通性..." else "连通性测试并保存",
                                    style = MaterialTheme.typography.labelLarge,
                                    fontWeight = FontWeight.Bold
                                )
                            }

                            HorizontalDivider(modifier = Modifier.padding(vertical = 4.dp))

                            // 安全偏好
                            Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
                                ConnectedListItem(
                                    index = 0,
                                    total = 2,
                                    headline = "生物识别安全锁",
                                    supportingText = "后台超过一分钟或冷启动时需要验证",
                                    leadingIcon = Icons.Rounded.Fingerprint,
                                    trailingContent = {
                                        Switch(
                                            checked = biometricEnabled,
                                            onCheckedChange = { onToggleBiometric(it) },
                                            colors = SwitchDefaults.colors(
                                                checkedThumbColor = MaterialTheme.colorScheme.onPrimary,
                                                checkedTrackColor = MaterialTheme.colorScheme.primary
                                            )
                                        )
                                    },
                                    onClick = { onToggleBiometric(!biometricEnabled) }
                                )

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
                                                NXGateApplication.instance.serverStore.setCleartextWarningEnabled(it)
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
                                        NXGateApplication.instance.serverStore.setCleartextWarningEnabled(cleartextWarningEnabled)
                                    }
                                )
                            }

                            HorizontalDivider(modifier = Modifier.padding(vertical = 4.dp))

                            // 外观与主题调色板 (包含莫奈动态取色)
                            ThemeAppearanceSettingsSection()
                        }
                    }
                } else {
                    // ==================== 手机 / 竖屏：垂直流式布局 ====================
                    // 1. 动态已纳管服务器列表
                    if (servers.isEmpty()) {
                        Surface(
                            modifier = Modifier.fillMaxWidth(),
                            shape = RoundedCornerShape(16.dp),
                            color = MaterialTheme.colorScheme.surfaceContainerLow
                        ) {
                            Text(
                                text = "暂无已纳管的 VPS，请点击下方表单或右上角二维码扫码添加服务器",
                                style = MaterialTheme.typography.bodyMedium,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.padding(16.dp)
                            )
                        }
                    } else {
                        Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
                            servers.forEachIndexed { index, server ->
                                ConnectedListItem(
                                    index = index,
                                    total = servers.size,
                                    headline = server.name,
                                    supportingText = "地址 ${server.host}，端口 ${server.port}，${if (server.isOnline) "连通正常 (${server.latencyMs}ms)" else "离线/待测"}",
                                    leadingIcon = Icons.Rounded.Dns,
                                    trailingContent = {
                                        Row(verticalAlignment = Alignment.CenterVertically) {
                                            IconButton(onClick = {
                                                scope.launch {
                                                    val res = NXGateApplication.instance.apiClient.testConnection(server)
                                                    if (res.isSuccess) {
                                                        NXGateApplication.instance.serverStore.updateServer(res.getOrNull() ?: server)
                                                        Toast.makeText(context, "[${server.name}] 测活通过！延迟: ${server.latencyMs}ms", Toast.LENGTH_SHORT).show()
                                                    } else {
                                                        Toast.makeText(context, "[${server.name}] 测活失败: ${res.exceptionOrNull()?.message}", Toast.LENGTH_SHORT).show()
                                                    }
                                                }
                                            }) {
                                                Icon(Icons.Rounded.NetworkPing, contentDescription = "测试连通性", tint = MaterialTheme.colorScheme.primary)
                                            }
                                            IconButton(onClick = {
                                                deleteCandidate = server
                                            }) {
                                                Icon(Icons.Rounded.Delete, contentDescription = "删除服务器", tint = MaterialTheme.colorScheme.error)
                                            }
                                        }
                                    },
                                    onClick = {
                                        editingServerId = server.id
                                        inputName = server.name
                                        inputHost = server.host
                                        inputPort = server.port.toString()
                                        inputPath = server.path
                                        inputUser = server.username
                                        inputPass = server.password
                                        selectedProtocolChipIndex = if (server.isTls) 1 else 0
                                        Toast.makeText(context, "已载入 [${server.name}] 参数至下方表单，可直接修改或保存", Toast.LENGTH_SHORT).show()
                                    }
                                )
                            }
                        }
                    }

                    // 2. 4个横向相连的描边按钮组
                    ConnectedButtonGroup(
                        items = listOf(
                            ConnectedButtonItem("服务器备注", ConnectedButtonStyle.Outlined) {},
                            ConnectedButtonItem("主机地址", ConnectedButtonStyle.Outlined) {},
                            ConnectedButtonItem("网页端口", ConnectedButtonStyle.Outlined) {},
                            ConnectedButtonItem("安全路径", ConnectedButtonStyle.Outlined) {}
                        )
                    )

                    // 表单输入项
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

                    if (selectedProtocolChipIndex == 0 && cleartextWarningEnabled) {
                        Surface(
                            shape = RoundedCornerShape(12.dp),
                            color = MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.5f),
                            border = BorderStroke(0.5.dp, MaterialTheme.colorScheme.error.copy(alpha = 0.5f)),
                            modifier = Modifier.fillMaxWidth()
                        ) {
                            Row(
                                modifier = Modifier.padding(horizontal = 12.dp, vertical = 8.dp),
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Icon(
                                    imageVector = Icons.Rounded.Warning,
                                    contentDescription = null,
                                    tint = MaterialTheme.colorScheme.error,
                                    modifier = Modifier.size(16.dp)
                                )
                                Spacer(Modifier.width(6.dp))
                                Text(
                                    text = "明文传输警告：HTTP 通信未经 TLS 加密，请仅在受信任私网中使用。",
                                    style = MaterialTheme.typography.labelSmall,
                                    color = MaterialTheme.colorScheme.onErrorContainer
                                )
                            }
                        }
                    }

                    // 4. “连通性测试并保存” 填充按钮（宽 380dp，带 check_circle 图标）
                    Box(
                        modifier = Modifier.fillMaxWidth(),
                        contentAlignment = Alignment.Center
                    ) {
                        Button(
                            onClick = {
                                isTestingConnection = true
                                val targetId = editingServerId ?: UUID.randomUUID().toString()
                                val newServer = ServerProfile(
                                    id = targetId,
                                    name = inputName.trim().ifEmpty { "AimiliVPN 网关" },
                                    host = inputHost.trim().ifEmpty { "127.0.0.1" },
                                    port = inputPort.toIntOrNull() ?: 8787,
                                    path = inputPath.trim().trim('/'),
                                    username = inputUser.trim(),
                                    password = inputPass.trim(),
                                    isTls = selectedProtocolChipIndex == 1
                                )
                                scope.launch {
                                    val res = NXGateApplication.instance.apiClient.testConnection(newServer)
                                    isTestingConnection = false
                                    val profileToSave = res.getOrNull() ?: newServer

                                    if (editingServerId != null && servers.any { it.id == editingServerId }) {
                                        NXGateApplication.instance.serverStore.updateServer(profileToSave)
                                        Toast.makeText(context, "[${profileToSave.name}] 配置已更新并安全保存", Toast.LENGTH_SHORT).show()
                                    } else {
                                        NXGateApplication.instance.serverStore.addServer(profileToSave)
                                        editingServerId = profileToSave.id
                                        Toast.makeText(context, "新服务器 [${profileToSave.name}] 已成功添加", Toast.LENGTH_SHORT).show()
                                    }
                                }
                            },
                            modifier = Modifier
                                .widthIn(max = 380.dp)
                                .fillMaxWidth()
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

                    // 5. 2项安全偏好列表 (不再与 FAB 重合遮挡)
                    Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
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
                                        NXGateApplication.instance.serverStore.setBiometricEnabled(it)
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
                                NXGateApplication.instance.serverStore.setBiometricEnabled(biometricEnabled)
                            }
                        )

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
                                        NXGateApplication.instance.serverStore.setCleartextWarningEnabled(it)
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
                                NXGateApplication.instance.serverStore.setCleartextWarningEnabled(cleartextWarningEnabled)
                            }
                        )
                    }

                    // 6. 外观与主题个性化 (支持莫奈动态取色)
                    ThemeAppearanceSettingsSection()
                }

                Spacer(Modifier.height(80.dp))
            }
        }
    }

    // 真正可靠的删除二次确认弹窗
    if (deleteCandidate != null) {
        val server = deleteCandidate!!
        AlertDialog(
            onDismissRequest = { deleteCandidate = null },
            shape = RoundedCornerShape(28.dp),
            containerColor = MaterialTheme.colorScheme.surfaceContainer,
            title = {
                Text("确认移除服务器？", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
            },
            text = {
                Text(
                    "服务器 [${server.name}] (${server.host}:${server.port}) 将从应用纳管列表中彻底删除，确定要移除吗？",
                    style = MaterialTheme.typography.bodyMedium
                )
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        NXGateApplication.instance.serverStore.deleteServer(server.id)
                        if (editingServerId == server.id) {
                            editingServerId = null
                        }
                        deleteCandidate = null
                        Toast.makeText(context, "已成功移除服务器 [${server.name}]", Toast.LENGTH_SHORT).show()
                    }
                ) {
                    Text("确认删除", color = MaterialTheme.colorScheme.error, fontWeight = FontWeight.Bold)
                }
            },
            dismissButton = {
                TextButton(onClick = { deleteCandidate = null }) {
                    Text("取消")
                }
            }
        )
    }

    // Camera QR Scan & URI Import Dialog
    if (showScanSimDialog) {
        CameraQrScannerDialog(
            onDismiss = { showScanSimDialog = false },
            onServerScanned = { parsed ->
                inputName = parsed.name
                inputHost = parsed.host
                inputPort = parsed.port.toString()
                inputPath = parsed.path
                inputUser = parsed.username
                inputPass = parsed.password
                selectedProtocolChipIndex = if (parsed.isTls) 1 else 0
                editingServerId = parsed.id
                NXGateApplication.instance.serverStore.addServer(parsed)
                showScanSimDialog = false
                Toast.makeText(context, "已成功扫码识别并自动导入 [${parsed.name}]", Toast.LENGTH_SHORT).show()
            }
        )
    }
}

@Composable
fun ThemeAppearanceSettingsSection(modifier: Modifier = Modifier) {
    val context = LocalContext.current
    val serverStore = NXGateApplication.instance.serverStore
    val themeMode by serverStore.themeMode.collectAsState()
    val themePalette by serverStore.themePalette.collectAsState()
    val themeAccent by serverStore.themeAccent.collectAsState()
    val themeBase by serverStore.themeBase.collectAsState()

    Card(
        modifier = modifier.fillMaxWidth(),
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
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp)
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Surface(
                    shape = CircleShape,
                    color = MaterialTheme.colorScheme.primaryContainer,
                    modifier = Modifier.size(40.dp)
                ) {
                    Box(contentAlignment = Alignment.Center) {
                        Icon(
                            imageVector = Icons.Rounded.Palette,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.primary,
                            modifier = Modifier.size(22.dp)
                        )
                    }
                }
                Spacer(Modifier.width(12.dp))
                Column {
                    Text(
                        text = "外观与调色板",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                    Text(
                        text = "参考 Google Pixel 桌面标准：强调色与基准底色解耦混色",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                }
            }

            // 1. 深浅明暗模式
            ConnectedChipGroup(
                chips = listOf("跟随系统", "浅色模式", "深色模式"),
                selectedIndex = when (themeMode) {
                    "light" -> 1
                    "dark" -> 2
                    else -> 0
                },
                onSelected = { idx ->
                    val newMode = when (idx) {
                        1 -> "light"
                        2 -> "dark"
                        else -> "system"
                    }
                    serverStore.setThemeMode(newMode)
                }
            )

            // 2. 主题调色架构模式切换
            val themeModeChips = listOf("Pixel 自由混色", "壁纸莫奈动态", "小米澎湃")
            val selectedPaletteChipIdx = when (themePalette) {
                "monet" -> 1
                "miuix" -> 2
                else -> 0
            }
            ConnectedChipGroup(
                chips = themeModeChips,
                selectedIndex = selectedPaletteChipIdx,
                onSelected = { idx ->
                    when (idx) {
                        1 -> serverStore.setThemePalette("monet")
                        2 -> serverStore.setThemePalette("miuix")
                        else -> serverStore.setThemePalette("pixel")
                    }
                }
            )

            if (themePalette == "pixel" || (themePalette != "monet" && themePalette != "miuix")) {
                val currentAccent = ACCENT_OPTIONS.find { it.id == themeAccent } ?: ACCENT_OPTIONS.first()
                val currentBase = BASE_TONE_OPTIONS.find { it.id == themeBase } ?: BASE_TONE_OPTIONS.first()

                // 2.1 强调色区域 (用于主按钮、开关与高光交互)
                Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Text(
                        text = "强调色 (用于主按钮、开关与交互高光)",
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.primary,
                        fontWeight = FontWeight.Bold
                    )
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        ACCENT_OPTIONS.forEach { acc ->
                            val isSelected = acc.id == themeAccent
                            Surface(
                                modifier = Modifier
                                    .size(36.dp)
                                    .clip(CircleShape)
                                    .clickable {
                                        serverStore.setThemeAccent(acc.id)
                                        serverStore.setThemePalette("pixel")
                                        Toast.makeText(context, "强调色已设为「${acc.name}」", Toast.LENGTH_SHORT).show()
                                    },
                                shape = CircleShape,
                                color = acc.lightPrimary,
                                border = if (isSelected) BorderStroke(3.dp, MaterialTheme.colorScheme.onSurface) else BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant)
                            ) {
                                if (isSelected) {
                                    Box(contentAlignment = Alignment.Center) {
                                        Icon(
                                            imageVector = Icons.Rounded.Check,
                                            contentDescription = null,
                                            tint = Color.White,
                                            modifier = Modifier.size(18.dp)
                                        )
                                    }
                                }
                            }
                        }
                    }
                }

                // 2.2 基准色区域 (用于卡片底色、页面背景与边框灰阶)
                Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Text(
                        text = "基准底色 (用于卡片底色、页面背景与边框灰阶)",
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onSurface,
                        fontWeight = FontWeight.Bold
                    )
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        BASE_TONE_OPTIONS.forEach { baseOpt ->
                            val isSelected = baseOpt.id == themeBase
                            Surface(
                                modifier = Modifier
                                    .weight(1f)
                                    .height(42.dp)
                                    .clip(RoundedCornerShape(12.dp))
                                    .clickable {
                                        serverStore.setThemeBase(baseOpt.id)
                                        serverStore.setThemePalette("pixel")
                                        Toast.makeText(context, "基准色已设为「${baseOpt.name}」", Toast.LENGTH_SHORT).show()
                                    },
                                shape = RoundedCornerShape(12.dp),
                                color = if (isSelected) MaterialTheme.colorScheme.secondaryContainer else MaterialTheme.colorScheme.surfaceContainer,
                                border = BorderStroke(if (isSelected) 2.dp else 1.dp, if (isSelected) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outlineVariant)
                            ) {
                                Box(
                                    modifier = Modifier.fillMaxSize(),
                                    contentAlignment = Alignment.Center
                                ) {
                                    Text(
                                        text = baseOpt.name,
                                        style = MaterialTheme.typography.labelMedium,
                                        fontWeight = if (isSelected) FontWeight.Bold else FontWeight.Normal,
                                        color = if (isSelected) MaterialTheme.colorScheme.onSecondaryContainer else MaterialTheme.colorScheme.onSurface
                                    )
                                }
                            }
                        }
                    }
                }

                // 2.3 Pixel 风格双色对比预览胶囊
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(14.dp),
                    color = MaterialTheme.colorScheme.surfaceContainerLow,
                    border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant)
                ) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = 14.dp, vertical = 10.dp),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Surface(
                                shape = CircleShape,
                                color = currentAccent.lightPrimary,
                                modifier = Modifier.size(16.dp)
                            ) {}
                            Spacer(Modifier.width(6.dp))
                            Text(
                                text = "强调色: ${currentAccent.name}",
                                style = MaterialTheme.typography.bodySmall,
                                fontWeight = FontWeight.SemiBold,
                                color = MaterialTheme.colorScheme.onSurface
                            )
                        }

                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Surface(
                                shape = RoundedCornerShape(4.dp),
                                color = currentBase.darkSurface,
                                modifier = Modifier.size(16.dp)
                            ) {}
                            Spacer(Modifier.width(6.dp))
                            Text(
                                text = "基准底色: ${currentBase.name}",
                                style = MaterialTheme.typography.bodySmall,
                                fontWeight = FontWeight.SemiBold,
                                color = MaterialTheme.colorScheme.onSurface
                            )
                        }
                    }
                }
            } else if (themePalette == "monet") {
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(14.dp),
                    color = MaterialTheme.colorScheme.surfaceContainerLow
                ) {
                    Text(
                        text = "正在从当前系统壁纸提取动态色彩 (Monet)。基准底色与强调色均由系统壁纸自动生成。",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface,
                        modifier = Modifier.padding(14.dp)
                    )
                }
            } else {
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(14.dp),
                    color = MaterialTheme.colorScheme.surfaceContainerLow
                ) {
                    Text(
                        text = "正在使用小米澎湃 (HyperOS · MIUIX) 主题。基准色为纯黑/纯白双层悬浮卡片，强调色为经典超凡蔚蓝。",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface,
                        modifier = Modifier.padding(14.dp)
                    )
                }
            }
        }
    }
}

