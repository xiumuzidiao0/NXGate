package com.nxgate.app.ui.screens

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
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
import androidx.compose.material.icons.rounded.CloudSync
import androidx.compose.material.icons.rounded.ColorLens
import androidx.compose.material.icons.rounded.ContentCopy
import androidx.compose.material.icons.rounded.ContentPaste
import androidx.compose.material.icons.rounded.Delete
import androidx.compose.material.icons.rounded.Dns
import androidx.compose.material.icons.rounded.FileDownload
import androidx.compose.material.icons.rounded.FileUpload
import androidx.compose.material.icons.rounded.Fingerprint
import androidx.compose.material.icons.rounded.NetworkPing
import androidx.compose.material.icons.rounded.Palette
import androidx.compose.material.icons.rounded.QrCodeScanner
import androidx.compose.material.icons.rounded.Share
import androidx.compose.material.icons.rounded.Timer
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
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalHapticFeedback
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
    val haptic = LocalHapticFeedback.current
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
    var inputName by remember { mutableStateOf(initialServer?.name ?: "") }
    var inputHost by remember { mutableStateOf(initialServer?.host ?: "") }
    var inputPort by remember { mutableStateOf(initialServer?.port?.toString() ?: "8787") }
    var inputPath by remember { mutableStateOf(initialServer?.path ?: "enter") }
    var inputUser by remember { mutableStateOf(initialServer?.username ?: "admin") }
    var inputPass by remember { mutableStateOf(initialServer?.password ?: "") }

    // Protocol chip: "HTTP", "HTTPS" (selected)
    var selectedProtocolChipIndex by remember { mutableIntStateOf(if (initialServer?.isTls == true) 1 else 0) }
    var allowInsecureTls by remember { mutableStateOf(initialServer?.allowInsecureTls == true) }
    val protocolChips = listOf("HTTP", "HTTPS")

    // Security preferences switch states
    var biometricEnabled by remember { mutableStateOf(NXGateApplication.instance.serverStore.biometricEnabled.value) }
    val biometricTimeout by NXGateApplication.instance.serverStore.biometricTimeoutSeconds.collectAsState()
    var showBiometricTimeoutDialog by remember { mutableStateOf(false) }
    var cleartextWarningEnabled by remember { mutableStateOf(NXGateApplication.instance.serverStore.cleartextWarningEnabled.value) }

    // Server deletion confirmation target
    var deleteCandidate by remember { mutableStateOf<ServerProfile?>(null) }
    var isTestingConnection by remember { mutableStateOf(false) }

    // QR Code Manual Scan Dialog
    var showScanSimDialog by remember { mutableStateOf(false) }
    var scannedUriInput by remember { mutableStateOf("") }

    // Cluster Backup & Restore Dialog
    var showClusterBackupDialog by remember { mutableStateOf(false) }
    var clusterBackupTab by remember { mutableIntStateOf(0) }
    var importJsonInput by remember { mutableStateOf("") }
    var exportIncludePassword by remember { mutableStateOf(true) }

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
                        onClick = { showClusterBackupDialog = true },
                        modifier = Modifier.size(48.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Rounded.CloudSync,
                            contentDescription = "集群备份与还原",
                            tint = MaterialTheme.colorScheme.onSurface,
                            modifier = Modifier.size(24.dp)
                        )
                    }
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
                    allowInsecureTls = false
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
                                                                val updated = res.getOrNull() ?: server
                                                                NXGateApplication.instance.serverStore.updateServer(updated)
                                                                haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                                                                Toast.makeText(context, "[${updated.name}] 测活通过！延迟: ${updated.latencyMs}ms", Toast.LENGTH_SHORT).show()
                                                            } else {
                                                                haptic.performHapticFeedback(HapticFeedbackType.LongPress)
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
                                                allowInsecureTls = server.allowInsecureTls
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

                            if (selectedProtocolChipIndex == 1) {
                                Surface(
                                    modifier = Modifier.fillMaxWidth(),
                                    shape = RoundedCornerShape(12.dp),
                                    color = MaterialTheme.colorScheme.surfaceContainerHigh
                                ) {
                                    Row(
                                        modifier = Modifier
                                            .fillMaxWidth()
                                            .padding(horizontal = 14.dp, vertical = 8.dp),
                                        horizontalArrangement = Arrangement.SpaceBetween,
                                        verticalAlignment = Alignment.CenterVertically
                                    ) {
                                        Column(modifier = Modifier.weight(1f)) {
                                            Text(
                                                text = "跳过自签证书与域名校验",
                                                style = MaterialTheme.typography.bodyMedium,
                                                fontWeight = FontWeight.SemiBold,
                                                color = MaterialTheme.colorScheme.onSurface
                                            )
                                            Text(
                                                text = "允许 IP 直连私有自签证书或内网 HTTPS 网关",
                                                style = MaterialTheme.typography.bodySmall,
                                                color = MaterialTheme.colorScheme.onSurfaceVariant
                                            )
                                        }
                                        Switch(
                                            checked = allowInsecureTls,
                                            onCheckedChange = { allowInsecureTls = it }
                                        )
                                    }
                                }
                            }

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
                                            text = "HTTP 连接提示：通信未经 HTTPS 加密，请仅在受信任私网中使用。",
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
                                        isTls = selectedProtocolChipIndex == 1,
                                        allowInsecureTls = if (selectedProtocolChipIndex == 1) allowInsecureTls else false
                                    )
                                    scope.launch {
                                        val res = NXGateApplication.instance.apiClient.testConnection(newServer)
                                        isTestingConnection = false
                                        if (res.isSuccess) {
                                            val profileToSave = res.getOrNull() ?: newServer
                                            if (editingServerId != null && servers.any { it.id == editingServerId }) {
                                                NXGateApplication.instance.serverStore.updateServer(profileToSave)
                                                Toast.makeText(context, "[${profileToSave.name}] 测活通过 (${profileToSave.latencyMs}ms)，配置已更新", Toast.LENGTH_SHORT).show()
                                            } else {
                                                NXGateApplication.instance.serverStore.addServer(profileToSave)
                                                editingServerId = profileToSave.id
                                                Toast.makeText(context, "[${profileToSave.name}] 连通测试通过 (${profileToSave.latencyMs}ms)，新网关已添加", Toast.LENGTH_SHORT).show()
                                            }
                                        } else {
                                            val offlineServer = newServer.copy(isOnline = false, latencyMs = 0)
                                            val errMsg = res.exceptionOrNull()?.message ?: "连接超时"
                                            if (editingServerId != null && servers.any { it.id == editingServerId }) {
                                                NXGateApplication.instance.serverStore.updateServer(offlineServer)
                                                Toast.makeText(context, "警告: 连通失败 ($errMsg)，已离线保存配置", Toast.LENGTH_LONG).show()
                                            } else {
                                                NXGateApplication.instance.serverStore.addServer(offlineServer)
                                                editingServerId = offlineServer.id
                                                Toast.makeText(context, "警告: 连通失败 ($errMsg)，已离线添加网关", Toast.LENGTH_LONG).show()
                                            }
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
                            val landscapePrefCount = if (biometricEnabled) 3 else 2
                            Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
                                ConnectedListItem(
                                    index = 0,
                                    total = landscapePrefCount,
                                    headline = "生物识别安全锁",
                                    supportingText = if (biometricEnabled) "后台等待 ${formatBiometricTimeoutShort(biometricTimeout)} 或冷启动时需要验证" else "冷启动与切回前台时无需验证",
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

                                if (biometricEnabled) {
                                    ConnectedListItem(
                                        index = 1,
                                        total = landscapePrefCount,
                                        headline = "后台锁定等待时间",
                                        supportingText = formatBiometricTimeout(biometricTimeout),
                                        leadingIcon = Icons.Rounded.Timer,
                                        trailingContent = {
                                            Text(
                                                text = formatBiometricTimeoutShort(biometricTimeout),
                                                style = MaterialTheme.typography.labelLarge,
                                                color = MaterialTheme.colorScheme.primary,
                                                fontWeight = FontWeight.Bold
                                            )
                                        },
                                        onClick = { showBiometricTimeoutDialog = true }
                                    )
                                }

                                ConnectedListItem(
                                    index = if (biometricEnabled) 2 else 1,
                                    total = landscapePrefCount,
                                    headline = "HTTP 传输风险提醒",
                                    supportingText = "检测到使用 HTTP 未加密连接时显示警告",
                                    leadingIcon = Icons.Rounded.Warning,
                                    trailingContent = {
                                        Switch(
                                            checked = cleartextWarningEnabled,
                                            onCheckedChange = {
                                                cleartextWarningEnabled = it
                                                NXGateApplication.instance.serverStore.setCleartextWarningEnabled(it)
                                                Toast.makeText(context, "HTTP 传输提醒已${if (it) "开启" else "关闭"}", Toast.LENGTH_SHORT).show()
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
                                                        val updated = res.getOrNull() ?: server
                                                        NXGateApplication.instance.serverStore.updateServer(updated)
                                                        haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                                                        Toast.makeText(context, "[${updated.name}] 测活通过！延迟: ${updated.latencyMs}ms", Toast.LENGTH_SHORT).show()
                                                    } else {
                                                        haptic.performHapticFeedback(HapticFeedbackType.LongPress)
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
                                        allowInsecureTls = server.allowInsecureTls
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

                    if (selectedProtocolChipIndex == 1) {
                        Surface(
                            modifier = Modifier.fillMaxWidth(),
                            shape = RoundedCornerShape(12.dp),
                            color = MaterialTheme.colorScheme.surfaceContainerHigh
                        ) {
                            Row(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(horizontal = 14.dp, vertical = 8.dp),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Column(modifier = Modifier.weight(1f)) {
                                    Text(
                                        text = "跳过自签证书与域名校验",
                                        style = MaterialTheme.typography.bodyMedium,
                                        fontWeight = FontWeight.SemiBold,
                                        color = MaterialTheme.colorScheme.onSurface
                                    )
                                    Text(
                                        text = "允许 IP 直连私有自签证书或内网 HTTPS 网关",
                                        style = MaterialTheme.typography.bodySmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant
                                    )
                                }
                                Switch(
                                    checked = allowInsecureTls,
                                    onCheckedChange = { allowInsecureTls = it }
                                )
                            }
                        }
                    }

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
                                    name = inputName.trim().ifEmpty { "NXGate 网关" },
                                    host = inputHost.trim().ifEmpty { "127.0.0.1" },
                                    port = inputPort.toIntOrNull() ?: 8787,
                                    path = inputPath.trim().trim('/'),
                                    username = inputUser.trim(),
                                    password = inputPass.trim(),
                                    isTls = selectedProtocolChipIndex == 1,
                                    allowInsecureTls = if (selectedProtocolChipIndex == 1) allowInsecureTls else false
                                )
                                scope.launch {
                                    val res = NXGateApplication.instance.apiClient.testConnection(newServer)
                                    isTestingConnection = false
                                    if (res.isSuccess) {
                                        val profileToSave = res.getOrNull() ?: newServer
                                        if (editingServerId != null && servers.any { it.id == editingServerId }) {
                                            NXGateApplication.instance.serverStore.updateServer(profileToSave)
                                            Toast.makeText(context, "[${profileToSave.name}] 测活通过 (${profileToSave.latencyMs}ms)，配置已更新", Toast.LENGTH_SHORT).show()
                                        } else {
                                            NXGateApplication.instance.serverStore.addServer(profileToSave)
                                            editingServerId = profileToSave.id
                                            Toast.makeText(context, "[${profileToSave.name}] 连通测试通过 (${profileToSave.latencyMs}ms)，新网关已添加", Toast.LENGTH_SHORT).show()
                                        }
                                    } else {
                                        val offlineServer = newServer.copy(isOnline = false, latencyMs = 0)
                                        val errMsg = res.exceptionOrNull()?.message ?: "连接超时"
                                        if (editingServerId != null && servers.any { it.id == editingServerId }) {
                                            NXGateApplication.instance.serverStore.updateServer(offlineServer)
                                            Toast.makeText(context, "警告: 连通失败 ($errMsg)，已离线保存配置", Toast.LENGTH_LONG).show()
                                        } else {
                                            NXGateApplication.instance.serverStore.addServer(offlineServer)
                                            editingServerId = offlineServer.id
                                            Toast.makeText(context, "警告: 连通失败 ($errMsg)，已离线添加网关", Toast.LENGTH_LONG).show()
                                        }
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

                    // 5. 安全偏好列表
                    val portraitPrefCount = if (biometricEnabled) 3 else 2
                    Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
                        ConnectedListItem(
                            index = 0,
                            total = portraitPrefCount,
                            headline = "生物识别安全锁",
                            supportingText = if (biometricEnabled) "后台等待 ${formatBiometricTimeoutShort(biometricTimeout)} 或冷启动时需要验证" else "冷启动与切回前台时无需验证",
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

                        if (biometricEnabled) {
                            ConnectedListItem(
                                index = 1,
                                total = portraitPrefCount,
                                headline = "后台锁定等待时间",
                                supportingText = formatBiometricTimeout(biometricTimeout),
                                leadingIcon = Icons.Rounded.Timer,
                                trailingContent = {
                                    Text(
                                        text = formatBiometricTimeoutShort(biometricTimeout),
                                        style = MaterialTheme.typography.labelLarge,
                                        color = MaterialTheme.colorScheme.primary,
                                        fontWeight = FontWeight.Bold
                                    )
                                },
                                onClick = { showBiometricTimeoutDialog = true }
                            )
                        }

                        ConnectedListItem(
                            index = if (biometricEnabled) 2 else 1,
                            total = portraitPrefCount,
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
                        haptic.performHapticFeedback(HapticFeedbackType.LongPress)
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
                allowInsecureTls = parsed.allowInsecureTls
                editingServerId = parsed.id
                scope.launch {
                    val res = NXGateApplication.instance.apiClient.testConnection(parsed)
                    val profileToSave = res.getOrNull() ?: parsed.copy(isOnline = false, latencyMs = 0)
                    NXGateApplication.instance.serverStore.addServer(profileToSave)
                    if (res.isSuccess) {
                        Toast.makeText(context, "已成功扫码录入并测活网关 [${profileToSave.name}] (${profileToSave.latencyMs}ms)", Toast.LENGTH_SHORT).show()
                    } else {
                        Toast.makeText(context, "已扫码录入网关 [${profileToSave.name}]，但连通测试未通过: ${res.exceptionOrNull()?.message}", Toast.LENGTH_LONG).show()
                    }
                }
            }
        )
    }

    // Cluster Backup & Restore Modal Dialog
    if (showBiometricTimeoutDialog) {
        val timeoutOptions = listOf(0, 15, 30, 60, 120, 300, 600)
        AlertDialog(
            onDismissRequest = { showBiometricTimeoutDialog = false },
            shape = RoundedCornerShape(28.dp),
            title = {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(10.dp)
                ) {
                    Icon(
                        Icons.Rounded.Timer,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.size(24.dp)
                    )
                    Text(
                        text = "后台锁定等待时间",
                        style = MaterialTheme.typography.titleLarge,
                        fontWeight = FontWeight.Bold
                    )
                }
            },
            text = {
                Column(
                    modifier = Modifier.fillMaxWidth(),
                    verticalArrangement = Arrangement.spacedBy(6.dp)
                ) {
                    Text(
                        text = "应用切入后台后，等待多久重新打开时触发生物识别验证：",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Spacer(Modifier.height(4.dp))
                    timeoutOptions.forEach { sec ->
                        val isSelected = biometricTimeout == sec
                        Surface(
                            shape = RoundedCornerShape(14.dp),
                            color = if (isSelected) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.45f),
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable {
                                    NXGateApplication.instance.serverStore.setBiometricTimeoutSeconds(sec)
                                    Toast.makeText(context, "已设置后台锁定时间: ${formatBiometricTimeout(sec)}", Toast.LENGTH_SHORT).show()
                                    showBiometricTimeoutDialog = false
                                }
                        ) {
                            Row(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(horizontal = 14.dp, vertical = 12.dp),
                                verticalAlignment = Alignment.CenterVertically,
                                horizontalArrangement = Arrangement.SpaceBetween
                            ) {
                                Text(
                                    text = formatBiometricTimeout(sec),
                                    style = MaterialTheme.typography.bodyMedium,
                                    fontWeight = if (isSelected) FontWeight.Bold else FontWeight.Normal,
                                    color = if (isSelected) MaterialTheme.colorScheme.onPrimaryContainer else MaterialTheme.colorScheme.onSurface
                                )
                                if (isSelected) {
                                    Icon(
                                        Icons.Rounded.Check,
                                        contentDescription = null,
                                        tint = MaterialTheme.colorScheme.primary,
                                        modifier = Modifier.size(20.dp)
                                    )
                                }
                            }
                        }
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = { showBiometricTimeoutDialog = false }) {
                    Text("取消")
                }
            }
        )
    }

    if (showClusterBackupDialog) {
        AlertDialog(
            onDismissRequest = { showClusterBackupDialog = false },
            shape = RoundedCornerShape(28.dp),
            containerColor = MaterialTheme.colorScheme.surfaceContainer,
            title = {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Icon(
                        imageVector = Icons.Rounded.CloudSync,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.size(24.dp)
                    )
                    Text("集群备份与还原", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
                }
            },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(14.dp)) {
                    // Tab 切换: 导出备份 / 导入还原
                    ConnectedChipGroup(
                        chips = listOf("导出集群备份", "导入集群配置"),
                        selectedIndex = clusterBackupTab,
                        onSelected = { clusterBackupTab = it }
                    )

                    if (clusterBackupTab == 0) {
                        // ================= 导出模块 =================
                        Text(
                            text = "当前已纳管 ${servers.size} 台网关服务器配置，点击下方按钮将生成全量标准 JSON 备份。",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )

                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Text("备份中包含连接密码", style = MaterialTheme.typography.bodyMedium)
                            Switch(
                                checked = exportIncludePassword,
                                onCheckedChange = { exportIncludePassword = it }
                            )
                        }

                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.spacedBy(10.dp)
                        ) {
                            Button(
                                onClick = {
                                    val json = NXGateApplication.instance.serverStore.exportClusterJson(includePasswords = exportIncludePassword)
                                    val cm = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
                                    cm.setPrimaryClip(ClipData.newPlainText("NXGate Cluster Backup", json))
                                    haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                                    Toast.makeText(context, "全量集群备份 JSON 已成功复制到剪贴板！", Toast.LENGTH_SHORT).show()
                                },
                                modifier = Modifier.weight(1f),
                                shape = RoundedCornerShape(12.dp)
                            ) {
                                Icon(Icons.Rounded.ContentCopy, contentDescription = null, modifier = Modifier.size(18.dp))
                                Spacer(Modifier.width(6.dp))
                                Text("复制 JSON")
                            }

                            Button(
                                onClick = {
                                    val json = NXGateApplication.instance.serverStore.exportClusterJson(includePasswords = exportIncludePassword)
                                    val sendIntent = Intent().apply {
                                        action = Intent.ACTION_SEND
                                        putExtra(Intent.EXTRA_TEXT, json)
                                        type = "text/plain"
                                    }
                                    val shareIntent = Intent.createChooser(sendIntent, "分享或导出 NXGate 集群备份")
                                    context.startActivity(shareIntent)
                                },
                                modifier = Modifier.weight(1f),
                                shape = RoundedCornerShape(12.dp),
                                colors = ButtonDefaults.buttonColors(
                                    containerColor = MaterialTheme.colorScheme.secondary
                                )
                            ) {
                                Icon(Icons.Rounded.Share, contentDescription = null, modifier = Modifier.size(18.dp))
                                Spacer(Modifier.width(6.dp))
                                Text("系统分享")
                            }
                        }
                    } else {
                        // ================= 导入模块 =================
                        Text(
                            text = "请粘贴 NXGate 备份 JSON 文本，系统将自动识别并合并导入至当前网关列表：",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )

                        OutlinedTextField(
                            value = importJsonInput,
                            onValueChange = { importJsonInput = it },
                            placeholder = { Text("粘贴备份 JSON 文本，包含 { \"servers\": [...] } 或 [...]") },
                            modifier = Modifier
                                .fillMaxWidth()
                                .height(120.dp),
                            shape = RoundedCornerShape(12.dp),
                            maxLines = 6
                        )

                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            TextButton(
                                onClick = {
                                    val cm = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
                                    val clipText = cm.primaryClip?.getItemAt(0)?.text?.toString() ?: ""
                                    if (clipText.isNotEmpty()) {
                                        importJsonInput = clipText
                                        Toast.makeText(context, "已从剪贴板粘贴文本", Toast.LENGTH_SHORT).show()
                                    } else {
                                        Toast.makeText(context, "剪贴板为空", Toast.LENGTH_SHORT).show()
                                    }
                                }
                            ) {
                                Icon(Icons.Rounded.ContentPaste, contentDescription = null, modifier = Modifier.size(16.dp))
                                Spacer(Modifier.width(4.dp))
                                Text("从剪贴板粘贴")
                            }

                            Button(
                                onClick = {
                                    if (importJsonInput.isBlank()) {
                                        Toast.makeText(context, "请先输入或粘贴备份内容", Toast.LENGTH_SHORT).show()
                                        return@Button
                                    }
                                    val res = NXGateApplication.instance.serverStore.importClusterJson(importJsonInput)
                                    if (res.isSuccess) {
                                        haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                                        val count = res.getOrDefault(0)
                                        Toast.makeText(context, "成功导入并合并 $count 台网关节点！", Toast.LENGTH_SHORT).show()
                                        showClusterBackupDialog = false
                                        importJsonInput = ""
                                    } else {
                                        Toast.makeText(context, "导入失败: ${res.exceptionOrNull()?.message}", Toast.LENGTH_LONG).show()
                                    }
                                },
                                shape = RoundedCornerShape(12.dp)
                            ) {
                                Icon(Icons.Rounded.FileUpload, contentDescription = null, modifier = Modifier.size(18.dp))
                                Spacer(Modifier.width(4.dp))
                                Text("确认导入")
                            }
                        }
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = { showClusterBackupDialog = false }) {
                    Text("关闭")
                }
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

private fun formatBiometricTimeout(seconds: Int): String {
    return when (seconds) {
        0 -> "立即锁定 (离开即锁)"
        15 -> "15 秒"
        30 -> "30 秒"
        60 -> "1 分钟 (默认推荐)"
        120 -> "2 分钟"
        300 -> "5 分钟"
        600 -> "10 分钟"
        else -> "${seconds} 秒"
    }
}

private fun formatBiometricTimeoutShort(seconds: Int): String {
    return when (seconds) {
        0 -> "立即"
        15 -> "15s"
        30 -> "30s"
        60 -> "1m"
        120 -> "2m"
        300 -> "5m"
        600 -> "10m"
        else -> "${seconds}s"
    }
}


