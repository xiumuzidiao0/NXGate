package com.nxgate.app.ui.screens

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.widget.Toast
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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.Article
import androidx.compose.material.icons.automirrored.rounded.Send
import androidx.compose.material.icons.rounded.CheckCircle
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material.icons.rounded.ContentCopy
import androidx.compose.material.icons.rounded.DeleteSweep
import androidx.compose.material.icons.rounded.Dns
import androidx.compose.material.icons.rounded.Key
import androidx.compose.material.icons.rounded.LinearScale
import androidx.compose.material.icons.rounded.Lock
import androidx.compose.material.icons.rounded.LockOpen
import androidx.compose.material.icons.rounded.Notifications
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material.icons.rounded.Security
import androidx.compose.material.icons.rounded.Sync
import androidx.compose.material.icons.rounded.SystemUpdate
import androidx.compose.material.icons.rounded.Terminal
import androidx.compose.material.icons.rounded.Tune
import androidx.compose.material.icons.rounded.VpnKey
import androidx.compose.material.icons.rounded.Warning
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
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
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.nxgate.app.NXGateApplication
import com.nxgate.app.model.AgeKeyGenerationResult
import com.nxgate.app.model.ServerProfile
import com.nxgate.app.model.ServerSettingsDTO
import com.nxgate.app.model.ServerUpdateCheck
import com.nxgate.app.model.ServerUpdateStatus
import com.nxgate.app.model.SystemLogEntry
import com.nxgate.app.ui.components.ConnectedButtonItem
import com.nxgate.app.ui.components.ConnectedButtonGroup
import com.nxgate.app.ui.components.ConnectedButtonStyle
import com.nxgate.app.ui.components.GlobalServerSwitcherTitle
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import java.security.SecureRandom

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ServerSystemScreen(
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

    val configuration = LocalConfiguration.current
    val isTablet = configuration.screenWidthDp >= 600

    // Update state
    var updateInfo by remember { mutableStateOf<ServerUpdateCheck?>(null) }
    var isCheckingUpdate by remember { mutableStateOf(false) }
    var updateStatus by remember { mutableStateOf<ServerUpdateStatus?>(null) }
    var isUpdating by remember { mutableStateOf(false) }
    var showUpdateConfirmDialog by remember { mutableStateOf(false) }
    var showReleaseNotesDialog by remember { mutableStateOf(false) }

    // Server Settings state
    var serverSettings by remember { mutableStateOf<ServerSettingsDTO?>(null) }
    var isLoadingSettings by remember { mutableStateOf(false) }
    var isSavingSettings by remember { mutableStateOf(false) }

    // Age Encryption local draft
    var ageEnabled by remember { mutableStateOf(false) }
    var agePublicKey by remember { mutableStateOf("") }
    var showAgeGenDialog by remember { mutableStateOf(false) }
    var showAgeDeriveDialog by remember { mutableStateOf(false) }
    var ageGenType by remember { mutableStateOf("x25519") }
    var generatedAgeResult by remember { mutableStateOf<AgeKeyGenerationResult?>(null) }
    var isGeneratingAge by remember { mutableStateOf(false) }
    var deriveSecretKeyInput by remember { mutableStateOf("") }
    var isDerivingAge by remember { mutableStateOf(false) }

    // SubToken draft
    var subToken by remember { mutableStateOf("") }

    // Telegram draft
    var telegramToken by remember { mutableStateOf("") }
    var telegramChatId by remember { mutableStateOf("") }
    var isTestingTelegram by remember { mutableStateOf(false) }

    // Maintenance toolkit actions
    var isRefreshingPool by remember { mutableStateOf(false) }
    var isResurrecting by remember { mutableStateOf(false) }
    var isClearingBlacklist by remember { mutableStateOf(false) }
    var showClearBlacklistDialog by remember { mutableStateOf(false) }

    // System logs bottom sheet
    var showLogsSheet by remember { mutableStateOf(false) }
    var systemLogs by remember { mutableStateOf<List<SystemLogEntry>>(emptyList()) }
    var isLoadingLogs by remember { mutableStateOf(false) }
    val logsSheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)

    fun copyToClipboard(label: String, text: String) {
        val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
        clipboard.setPrimaryClip(ClipData.newPlainText(label, text))
        Toast.makeText(context, "已复制 $label 到剪贴板", Toast.LENGTH_SHORT).show()
    }

    fun loadAllServerData() {
        if (activeServer == null) return
        scope.launch {
            isLoadingSettings = true
            isCheckingUpdate = true
            val setRes = NXGateApplication.instance.apiClient.fetchServerSettings(activeServer)
            isLoadingSettings = false
            if (setRes.isSuccess) {
                val s = setRes.getOrNull()
                if (s != null) {
                    serverSettings = s
                    ageEnabled = s.ageEncryptEnabled
                    agePublicKey = s.agePublicKey
                    subToken = s.subToken
                    telegramToken = s.telegramBotToken
                    telegramChatId = s.telegramChatID
                }
            }

            val updRes = NXGateApplication.instance.apiClient.checkServerUpdate(activeServer)
            isCheckingUpdate = false
            if (updRes.isSuccess) {
                updateInfo = updRes.getOrNull()
            }
        }
    }

    LaunchedEffect(activeServer?.id) {
        loadAllServerData()
    }

    // Polling for update status while update is in progress
    LaunchedEffect(isUpdating) {
        if (!isUpdating || activeServer == null) return@LaunchedEffect
        while (isActive && isUpdating) {
            delay(1500)
            val res = NXGateApplication.instance.apiClient.fetchServerUpdateStatus(activeServer)
            if (res.isSuccess) {
                val status = res.getOrNull()
                updateStatus = status
                if (status != null && !status.inProgress) {
                    isUpdating = false
                    if (status.error.isNotEmpty()) {
                        Toast.makeText(context, "更新失败: ${status.error}", Toast.LENGTH_LONG).show()
                    } else {
                        Toast.makeText(context, "网关自愈更新成功，已重启服务！", Toast.LENGTH_LONG).show()
                        loadAllServerData()
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
                        onClick = { loadAllServerData() },
                        modifier = Modifier.size(48.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Rounded.Refresh,
                            contentDescription = "刷新系统配置",
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
                    .align(Alignment.TopCenter)
                    .verticalScroll(scrollState),
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                Spacer(Modifier.height(4.dp))

                if (activeServer == null) {
                    Surface(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(16.dp),
                        color = MaterialTheme.colorScheme.surfaceContainerLow
                    ) {
                        Text(
                            text = "请先在首页或设置中添加并选择一台在线的网关服务器。",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(16.dp)
                        )
                    }
                } else {
                    // ==========================================
                    // 1. 服务端版本与自愈更新 (Server Update Card)
                    // ==========================================
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(20.dp),
                        colors = CardDefaults.cardColors(
                            containerColor = MaterialTheme.colorScheme.surfaceContainer
                        ),
                        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant)
                    ) {
                        Column(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(20.dp),
                            verticalArrangement = Arrangement.spacedBy(12.dp)
                        ) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Row(
                                    verticalAlignment = Alignment.CenterVertically,
                                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                                ) {
                                    Icon(
                                        imageVector = Icons.Rounded.SystemUpdate,
                                        contentDescription = null,
                                        tint = MaterialTheme.colorScheme.primary,
                                        modifier = Modifier.size(22.dp)
                                    )
                                    Text(
                                        text = "版本与自愈更新",
                                        style = MaterialTheme.typography.titleMedium,
                                        fontWeight = FontWeight.Bold,
                                        color = MaterialTheme.colorScheme.onSurface
                                    )
                                }

                                Surface(
                                    shape = RoundedCornerShape(8.dp),
                                    color = MaterialTheme.colorScheme.primaryContainer
                                ) {
                                    Text(
                                        text = "v${updateInfo?.currentVersion ?: "2.5.9"}",
                                        style = MaterialTheme.typography.labelSmall,
                                        fontWeight = FontWeight.Bold,
                                        color = MaterialTheme.colorScheme.onPrimaryContainer,
                                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                                    )
                                }
                            }

                            Text(
                                text = "目标服务器: ${activeServer.name} (${activeServer.host}:${activeServer.port})",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )

                            // Status Banner
                            if (isUpdating) {
                                Surface(
                                    shape = RoundedCornerShape(12.dp),
                                    color = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.5f),
                                    modifier = Modifier.fillMaxWidth()
                                ) {
                                    Column(
                                        modifier = Modifier.padding(14.dp),
                                        verticalArrangement = Arrangement.spacedBy(8.dp)
                                    ) {
                                        Row(
                                            verticalAlignment = Alignment.CenterVertically,
                                            horizontalArrangement = Arrangement.spacedBy(8.dp)
                                        ) {
                                            CircularProgressIndicator(
                                                modifier = Modifier.size(16.dp),
                                                strokeWidth = 2.dp,
                                                color = MaterialTheme.colorScheme.primary
                                            )
                                            Text(
                                                text = "正在执行服务端自愈更新: ${updateStatus?.step ?: "准备中..."}",
                                                style = MaterialTheme.typography.bodyMedium,
                                                fontWeight = FontWeight.SemiBold,
                                                color = MaterialTheme.colorScheme.onPrimaryContainer
                                            )
                                        }
                                        LinearProgressIndicator(
                                            modifier = Modifier.fillMaxWidth(),
                                            color = MaterialTheme.colorScheme.primary
                                        )
                                    }
                                }
                            } else if (updateInfo != null) {
                                if (updateInfo!!.hasUpdate) {
                                    Surface(
                                        shape = RoundedCornerShape(12.dp),
                                        color = MaterialTheme.colorScheme.tertiaryContainer.copy(alpha = 0.6f),
                                        border = BorderStroke(1.dp, MaterialTheme.colorScheme.tertiary.copy(alpha = 0.4f)),
                                        modifier = Modifier.fillMaxWidth()
                                    ) {
                                        Column(
                                            modifier = Modifier.padding(14.dp),
                                            verticalArrangement = Arrangement.spacedBy(6.dp)
                                        ) {
                                            Row(
                                                verticalAlignment = Alignment.CenterVertically,
                                                horizontalArrangement = Arrangement.spacedBy(8.dp)
                                            ) {
                                                Icon(
                                                    imageVector = Icons.Rounded.Warning,
                                                    contentDescription = null,
                                                    tint = MaterialTheme.colorScheme.tertiary,
                                                    modifier = Modifier.size(18.dp)
                                                )
                                                Text(
                                                    text = "发现新版本可用: v${updateInfo!!.latestVersion}",
                                                    style = MaterialTheme.typography.titleSmall,
                                                    fontWeight = FontWeight.Bold,
                                                    color = MaterialTheme.colorScheme.onTertiaryContainer
                                                )
                                            }
                                            if (updateInfo!!.releaseName.isNotEmpty()) {
                                                Text(
                                                    text = updateInfo!!.releaseName,
                                                    style = MaterialTheme.typography.bodySmall,
                                                    color = MaterialTheme.colorScheme.onTertiaryContainer
                                                )
                                            }
                                            if (updateInfo!!.publishedAt.isNotEmpty()) {
                                                Text(
                                                    text = "发布时间: ${updateInfo!!.publishedAt.take(10)}",
                                                    style = MaterialTheme.typography.labelSmall,
                                                    color = MaterialTheme.colorScheme.onTertiaryContainer.copy(alpha = 0.8f)
                                                )
                                            }
                                        }
                                    }
                                } else {
                                    Surface(
                                        shape = RoundedCornerShape(12.dp),
                                        color = MaterialTheme.colorScheme.surfaceContainerHighest,
                                        modifier = Modifier.fillMaxWidth()
                                    ) {
                                        Row(
                                            modifier = Modifier.padding(12.dp),
                                            verticalAlignment = Alignment.CenterVertically,
                                            horizontalArrangement = Arrangement.spacedBy(8.dp)
                                        ) {
                                            Icon(
                                                imageVector = Icons.Rounded.CheckCircle,
                                                contentDescription = null,
                                                tint = MaterialTheme.colorScheme.primary,
                                                modifier = Modifier.size(18.dp)
                                            )
                                            Text(
                                                text = "当前已是最新构建 (v${updateInfo!!.currentVersion})",
                                                style = MaterialTheme.typography.bodyMedium,
                                                color = MaterialTheme.colorScheme.onSurface
                                            )
                                        }
                                    }
                                }
                            }

                            // Actions
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.spacedBy(10.dp)
                            ) {
                                OutlinedButton(
                                    onClick = {
                                        scope.launch {
                                            isCheckingUpdate = true
                                            val res = NXGateApplication.instance.apiClient.checkServerUpdate(activeServer)
                                            isCheckingUpdate = false
                                            if (res.isSuccess) {
                                                updateInfo = res.getOrNull()
                                                Toast.makeText(context, "版本检测完成", Toast.LENGTH_SHORT).show()
                                            } else {
                                                Toast.makeText(context, "检查更新失败: ${res.exceptionOrNull()?.message}", Toast.LENGTH_SHORT).show()
                                            }
                                        }
                                    },
                                    enabled = !isCheckingUpdate && !isUpdating,
                                    shape = RoundedCornerShape(12.dp),
                                    modifier = Modifier.weight(1f)
                                ) {
                                    if (isCheckingUpdate) {
                                        CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp)
                                    } else {
                                        Icon(Icons.Rounded.Sync, contentDescription = null, modifier = Modifier.size(16.dp))
                                        Spacer(Modifier.width(6.dp))
                                        Text("检查更新")
                                    }
                                }

                                if (updateInfo?.hasUpdate == true) {
                                    Button(
                                        onClick = { showUpdateConfirmDialog = true },
                                        enabled = !isUpdating,
                                        shape = RoundedCornerShape(12.dp),
                                        colors = ButtonDefaults.buttonColors(
                                            containerColor = MaterialTheme.colorScheme.primary
                                        ),
                                        modifier = Modifier.weight(1f)
                                    ) {
                                        Icon(Icons.Rounded.SystemUpdate, contentDescription = null, modifier = Modifier.size(16.dp))
                                        Spacer(Modifier.width(6.dp))
                                        Text("升级服务器")
                                    }
                                } else if (updateInfo?.releaseNotes?.isNotBlank() == true) {
                                    OutlinedButton(
                                        onClick = { showReleaseNotesDialog = true },
                                        shape = RoundedCornerShape(12.dp),
                                        modifier = Modifier.weight(1f)
                                    ) {
                                        Icon(Icons.AutoMirrored.Rounded.Article, contentDescription = null, modifier = Modifier.size(16.dp))
                                        Spacer(Modifier.width(6.dp))
                                        Text("更新日志")
                                    }
                                }
                            }
                        }
                    }

                    // ==========================================
                    // 2. age 端到端前向安全加密 (Age Encryption Card)
                    // ==========================================
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(20.dp),
                        colors = CardDefaults.cardColors(
                            containerColor = MaterialTheme.colorScheme.surfaceContainer
                        ),
                        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant)
                    ) {
                        Column(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(20.dp),
                            verticalArrangement = Arrangement.spacedBy(14.dp)
                        ) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Row(
                                    verticalAlignment = Alignment.CenterVertically,
                                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                                ) {
                                    Icon(
                                        imageVector = Icons.Rounded.Security,
                                        contentDescription = null,
                                        tint = MaterialTheme.colorScheme.primary,
                                        modifier = Modifier.size(22.dp)
                                    )
                                    Text(
                                        text = "age 端到端前向安全加密",
                                        style = MaterialTheme.typography.titleMedium,
                                        fontWeight = FontWeight.Bold,
                                        color = MaterialTheme.colorScheme.onSurface
                                    )
                                }
                            }

                            Text(
                                text = "开启后，服务端下发的通用订阅与分流配置将自动转换为标准 ASCII Armored age 密文。仅持有私钥的受信任端方可解密，杜绝中转节点嗅探与缓存劫持。",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                lineHeight = MaterialTheme.typography.bodySmall.lineHeight * 1.35f
                            )

                            // Toggle Switch Row
                            Surface(
                                shape = RoundedCornerShape(14.dp),
                                color = MaterialTheme.colorScheme.surfaceContainerHigh,
                                modifier = Modifier.fillMaxWidth()
                            ) {
                                Row(
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .padding(horizontal = 14.dp, vertical = 10.dp),
                                    horizontalArrangement = Arrangement.SpaceBetween,
                                    verticalAlignment = Alignment.CenterVertically
                                ) {
                                    Column(modifier = Modifier.weight(1f)) {
                                        Text(
                                            text = "开启订阅加密保护",
                                            style = MaterialTheme.typography.bodyMedium,
                                            fontWeight = FontWeight.SemiBold,
                                            color = MaterialTheme.colorScheme.onSurface
                                        )
                                        Text(
                                            text = if (ageEnabled) "已启用: 未授权访问仅能拉取密文" else "已停用: 下发普通 Base64/YAML",
                                            style = MaterialTheme.typography.bodySmall,
                                            color = if (ageEnabled) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant
                                        )
                                    }
                                    Switch(
                                        checked = ageEnabled,
                                        onCheckedChange = { ageEnabled = it }
                                    )
                                }
                            }

                            // Public Key Input
                            OutlinedTextField(
                                value = agePublicKey,
                                onValueChange = { agePublicKey = it.trim() },
                                label = { Text("age Recipient 公钥 (age1... / age1pq...)") },
                                placeholder = { Text("例如 age1ql3z7hjy...") },
                                singleLine = false,
                                maxLines = 3,
                                shape = RoundedCornerShape(12.dp),
                                textStyle = MaterialTheme.typography.bodySmall.copy(fontFamily = FontFamily.Monospace),
                                modifier = Modifier.fillMaxWidth()
                            )

                            // Key Pair Generation & Derivation Buttons
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.spacedBy(8.dp)
                            ) {
                                OutlinedButton(
                                    onClick = {
                                        generatedAgeResult = null
                                        showAgeGenDialog = true
                                    },
                                    shape = RoundedCornerShape(12.dp),
                                    modifier = Modifier.weight(1f)
                                ) {
                                    Icon(Icons.Rounded.VpnKey, contentDescription = null, modifier = Modifier.size(16.dp))
                                    Spacer(Modifier.width(6.dp))
                                    Text("生成新密钥对", fontSize = 12.sp)
                                }

                                OutlinedButton(
                                    onClick = {
                                        deriveSecretKeyInput = ""
                                        showAgeDeriveDialog = true
                                    },
                                    shape = RoundedCornerShape(12.dp),
                                    modifier = Modifier.weight(1f)
                                ) {
                                    Icon(Icons.Rounded.Key, contentDescription = null, modifier = Modifier.size(16.dp))
                                    Spacer(Modifier.width(6.dp))
                                    Text("从私钥推导", fontSize = 12.sp)
                                }
                            }

                            // Save Age Settings Button
                            Button(
                                onClick = {
                                    if (serverSettings == null) return@Button
                                    scope.launch {
                                        isSavingSettings = true
                                        val updated = serverSettings!!.copy(
                                            ageEncryptEnabled = ageEnabled,
                                            agePublicKey = agePublicKey
                                        )
                                        val res = NXGateApplication.instance.apiClient.saveServerSettings(activeServer, updated)
                                        isSavingSettings = false
                                        if (res.isSuccess) {
                                            serverSettings = updated
                                            Toast.makeText(context, "age 加密策略配置已持久化保存！", Toast.LENGTH_SHORT).show()
                                        } else {
                                            Toast.makeText(context, "保存失败: ${res.exceptionOrNull()?.message}", Toast.LENGTH_LONG).show()
                                        }
                                    }
                                },
                                enabled = !isSavingSettings,
                                shape = RoundedCornerShape(12.dp),
                                colors = ButtonDefaults.buttonColors(
                                    containerColor = MaterialTheme.colorScheme.primary
                                ),
                                modifier = Modifier.fillMaxWidth()
                            ) {
                                if (isSavingSettings) {
                                    CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp, color = MaterialTheme.colorScheme.onPrimary)
                                } else {
                                    Icon(Icons.Rounded.Lock, contentDescription = null, modifier = Modifier.size(16.dp))
                                    Spacer(Modifier.width(6.dp))
                                    Text("保存 age 加密策略")
                                }
                            }
                        }
                    }

                    // ==========================================
                    // 3. 免密拉取令牌管理 (SubToken Card)
                    // ==========================================
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(20.dp),
                        colors = CardDefaults.cardColors(
                            containerColor = MaterialTheme.colorScheme.surfaceContainer
                        ),
                        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant)
                    ) {
                        Column(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(20.dp),
                            verticalArrangement = Arrangement.spacedBy(12.dp)
                        ) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Row(
                                    verticalAlignment = Alignment.CenterVertically,
                                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                                ) {
                                    Icon(
                                        imageVector = Icons.Rounded.Key,
                                        contentDescription = null,
                                        tint = MaterialTheme.colorScheme.primary,
                                        modifier = Modifier.size(22.dp)
                                    )
                                    Text(
                                        text = "免密订阅令牌 (SubToken)",
                                        style = MaterialTheme.typography.titleMedium,
                                        fontWeight = FontWeight.Bold,
                                        color = MaterialTheme.colorScheme.onSurface
                                    )
                                }
                            }

                            Text(
                                text = "专属 16 位安全令牌可隔离管理面板路径与密码，供 Clash、sing-box 等外部客户端免认证静默拉取订阅，探测扫描直接返回 404。",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )

                            OutlinedTextField(
                                value = subToken,
                                onValueChange = { subToken = it.trim() },
                                label = { Text("SubToken 安全令牌") },
                                trailingIcon = {
                                    IconButton(onClick = { copyToClipboard("SubToken", subToken) }) {
                                        Icon(Icons.Rounded.ContentCopy, contentDescription = "复制", modifier = Modifier.size(18.dp))
                                    }
                                },
                                singleLine = true,
                                shape = RoundedCornerShape(12.dp),
                                textStyle = MaterialTheme.typography.bodyMedium.copy(fontFamily = FontFamily.Monospace),
                                modifier = Modifier.fillMaxWidth()
                            )

                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.spacedBy(10.dp)
                            ) {
                                OutlinedButton(
                                    onClick = {
                                        val chars = "abcdefghijklmnopqrstuvwxyz0123456789"
                                        val rnd = SecureRandom()
                                        val sb = StringBuilder(16)
                                        for (i in 0 until 16) {
                                            sb.append(chars[rnd.nextInt(chars.length)])
                                        }
                                        subToken = sb.toString()
                                        Toast.makeText(context, "已生成随机新 Token，请点击保存", Toast.LENGTH_SHORT).show()
                                    },
                                    shape = RoundedCornerShape(12.dp),
                                    modifier = Modifier.weight(1f)
                                ) {
                                    Icon(Icons.Rounded.Refresh, contentDescription = null, modifier = Modifier.size(16.dp))
                                    Spacer(Modifier.width(6.dp))
                                    Text("重置令牌")
                                }

                                Button(
                                    onClick = {
                                        if (serverSettings == null) return@Button
                                        scope.launch {
                                            isSavingSettings = true
                                            val updated = serverSettings!!.copy(subToken = subToken)
                                            val res = NXGateApplication.instance.apiClient.saveServerSettings(activeServer, updated)
                                            isSavingSettings = false
                                            if (res.isSuccess) {
                                                serverSettings = updated
                                                Toast.makeText(context, "SubToken 已更新并生效！", Toast.LENGTH_SHORT).show()
                                            } else {
                                                Toast.makeText(context, "保存失败: ${res.exceptionOrNull()?.message}", Toast.LENGTH_LONG).show()
                                            }
                                        }
                                    },
                                    shape = RoundedCornerShape(12.dp),
                                    modifier = Modifier.weight(1f)
                                ) {
                                    Text("保存令牌")
                                }
                            }
                        }
                    }

                    // ==========================================
                    // 4. Telegram 告警推送与联动 (Telegram Card)
                    // ==========================================
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(20.dp),
                        colors = CardDefaults.cardColors(
                            containerColor = MaterialTheme.colorScheme.surfaceContainer
                        ),
                        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant)
                    ) {
                        Column(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(20.dp),
                            verticalArrangement = Arrangement.spacedBy(12.dp)
                        ) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Row(
                                    verticalAlignment = Alignment.CenterVertically,
                                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                                ) {
                                    Icon(
                                        imageVector = Icons.Rounded.Notifications,
                                        contentDescription = null,
                                        tint = MaterialTheme.colorScheme.primary,
                                        modifier = Modifier.size(22.dp)
                                    )
                                    Text(
                                        text = "Telegram 异常通报与告警",
                                        style = MaterialTheme.typography.titleMedium,
                                        fontWeight = FontWeight.Bold,
                                        color = MaterialTheme.colorScheme.onSurface
                                    )
                                }
                            }

                            Text(
                                text = "当 VPS 网关出现物理断流、假死触发 Failover 故障转移换线、或节点池全量拉取异常时，远端通过 Telegram 机器人实时推流通报。",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )

                            OutlinedTextField(
                                value = telegramToken,
                                onValueChange = { telegramToken = it.trim() },
                                label = { Text("Bot Token") },
                                placeholder = { Text("123456789:ABCdefGhIJKlmNoP...") },
                                singleLine = true,
                                shape = RoundedCornerShape(12.dp),
                                textStyle = MaterialTheme.typography.bodySmall.copy(fontFamily = FontFamily.Monospace),
                                modifier = Modifier.fillMaxWidth()
                            )

                            OutlinedTextField(
                                value = telegramChatId,
                                onValueChange = { telegramChatId = it.trim() },
                                label = { Text("Chat ID") },
                                placeholder = { Text("例如 987654321") },
                                singleLine = true,
                                shape = RoundedCornerShape(12.dp),
                                textStyle = MaterialTheme.typography.bodySmall.copy(fontFamily = FontFamily.Monospace),
                                modifier = Modifier.fillMaxWidth()
                            )

                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.spacedBy(10.dp)
                            ) {
                                OutlinedButton(
                                    onClick = {
                                        scope.launch {
                                            isTestingTelegram = true
                                            val res = NXGateApplication.instance.apiClient.testTelegram(activeServer)
                                            isTestingTelegram = false
                                            if (res.isSuccess) {
                                                Toast.makeText(context, res.getOrNull() ?: "测试消息发送成功", Toast.LENGTH_SHORT).show()
                                            } else {
                                                Toast.makeText(context, "测试失败: ${res.exceptionOrNull()?.message}", Toast.LENGTH_LONG).show()
                                            }
                                        }
                                    },
                                    enabled = !isTestingTelegram,
                                    shape = RoundedCornerShape(12.dp),
                                    modifier = Modifier.weight(1f)
                                ) {
                                    if (isTestingTelegram) {
                                        CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp)
                                    } else {
                                        Icon(Icons.AutoMirrored.Rounded.Send, contentDescription = null, modifier = Modifier.size(16.dp))
                                        Spacer(Modifier.width(6.dp))
                                        Text("测试推送")
                                    }
                                }

                                Button(
                                    onClick = {
                                        if (serverSettings == null) return@Button
                                        scope.launch {
                                            isSavingSettings = true
                                            val updated = serverSettings!!.copy(
                                                telegramBotToken = telegramToken,
                                                telegramChatID = telegramChatId
                                            )
                                            val res = NXGateApplication.instance.apiClient.saveServerSettings(activeServer, updated)
                                            isSavingSettings = false
                                            if (res.isSuccess) {
                                                serverSettings = updated
                                                Toast.makeText(context, "Telegram 配置已保存", Toast.LENGTH_SHORT).show()
                                            } else {
                                                Toast.makeText(context, "保存失败: ${res.exceptionOrNull()?.message}", Toast.LENGTH_LONG).show()
                                            }
                                        }
                                    },
                                    shape = RoundedCornerShape(12.dp),
                                    modifier = Modifier.weight(1f)
                                ) {
                                    Text("保存通知配置")
                                }
                            }
                        }
                    }

                    // ==========================================
                    // 5. 运维与自愈工具箱 (Maintenance Toolkit Card)
                    // ==========================================
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(20.dp),
                        colors = CardDefaults.cardColors(
                            containerColor = MaterialTheme.colorScheme.surfaceContainer
                        ),
                        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant)
                    ) {
                        Column(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(20.dp),
                            verticalArrangement = Arrangement.spacedBy(14.dp)
                        ) {
                            Row(
                                verticalAlignment = Alignment.CenterVertically,
                                horizontalArrangement = Arrangement.spacedBy(8.dp)
                            ) {
                                Icon(
                                    imageVector = Icons.Rounded.Tune,
                                    contentDescription = null,
                                    tint = MaterialTheme.colorScheme.primary,
                                    modifier = Modifier.size(22.dp)
                                )
                                Text(
                                    text = "网关运维与自愈工具箱",
                                    style = MaterialTheme.typography.titleMedium,
                                    fontWeight = FontWeight.Bold,
                                    color = MaterialTheme.colorScheme.onSurface
                                )
                            }

                            ConnectedButtonGroup(
                                items = listOf(
                                    ConnectedButtonItem(
                                        text = if (isRefreshingPool) "刷新中..." else "刷新节点源",
                                        style = ConnectedButtonStyle.Tonal,
                                        icon = Icons.Rounded.Refresh,
                                        onClick = {
                                            scope.launch {
                                                isRefreshingPool = true
                                                val res = NXGateApplication.instance.apiClient.refreshPool(activeServer)
                                                isRefreshingPool = false
                                                if (res.isSuccess) {
                                                    Toast.makeText(context, "已触发服务端全量镜像拉取任务！", Toast.LENGTH_SHORT).show()
                                                } else {
                                                    Toast.makeText(context, "触发失败: ${res.exceptionOrNull()?.message}", Toast.LENGTH_SHORT).show()
                                                }
                                            }
                                        }
                                    ),
                                    ConnectedButtonItem(
                                        text = if (isResurrecting) "探活中..." else "影子协议探活",
                                        style = ConnectedButtonStyle.Tonal,
                                        icon = Icons.Rounded.Sync,
                                        onClick = {
                                            scope.launch {
                                                isResurrecting = true
                                                val res = NXGateApplication.instance.apiClient.resurrectBlacklist(activeServer)
                                                isResurrecting = false
                                                if (res.isSuccess) {
                                                    Toast.makeText(context, "已触发对屏蔽隔离节点的深度 OpenVPN/TLS 影子探活！", Toast.LENGTH_SHORT).show()
                                                } else {
                                                    Toast.makeText(context, "探活失败: ${res.exceptionOrNull()?.message}", Toast.LENGTH_SHORT).show()
                                                }
                                            }
                                        }
                                    ),
                                    ConnectedButtonItem(
                                        text = "清空硬屏蔽",
                                        style = ConnectedButtonStyle.Outlined,
                                        icon = Icons.Rounded.DeleteSweep,
                                        onClick = { showClearBlacklistDialog = true }
                                    ),
                                    ConnectedButtonItem(
                                        text = "实时日志",
                                        style = ConnectedButtonStyle.Filled,
                                        icon = Icons.AutoMirrored.Rounded.Article,
                                        onClick = {
                                            showLogsSheet = true
                                            scope.launch {
                                                isLoadingLogs = true
                                                val res = NXGateApplication.instance.apiClient.fetchLogs(activeServer)
                                                isLoadingLogs = false
                                                if (res.isSuccess) {
                                                    systemLogs = res.getOrNull() ?: emptyList()
                                                }
                                            }
                                        }
                                    )
                                ),
                                height = 44.dp
                            )
                        }
                    }

                    Spacer(Modifier.height(80.dp))
                }
            }
        }
    }

    // ==========================================
    // Update Confirmation Dialog
    // ==========================================
    if (showUpdateConfirmDialog && activeServer != null && updateInfo != null) {
        AlertDialog(
            onDismissRequest = { showUpdateConfirmDialog = false },
            shape = RoundedCornerShape(24.dp),
            title = {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Icon(
                        imageVector = Icons.Rounded.SystemUpdate,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.size(24.dp)
                    )
                    Text("确认升级网关核心", fontWeight = FontWeight.Bold)
                }
            },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(
                        text = "即将为服务器 [${activeServer.name}] 执行极速热升级：",
                        style = MaterialTheme.typography.bodyMedium
                    )
                    Surface(
                        shape = RoundedCornerShape(10.dp),
                        color = MaterialTheme.colorScheme.surfaceContainerHighest,
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Column(modifier = Modifier.padding(10.dp)) {
                            Text("当前版本: v${updateInfo!!.currentVersion}", style = MaterialTheme.typography.bodySmall)
                            Text("目标版本: v${updateInfo!!.latestVersion}", style = MaterialTheme.typography.bodySmall, fontWeight = FontWeight.Bold, color = MaterialTheme.colorScheme.primary)
                            Text("发布说明: ${updateInfo!!.releaseName}", style = MaterialTheme.typography.bodySmall)
                        }
                    }
                    Text(
                        text = "服务端将通过 GitHub Release 下载最新预编译包，比对可信 SHA-256 哈希清单，备份当前可执行程序，并通过 systemd 平滑自愈重启网关服务。",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            },
            confirmButton = {
                Button(
                    onClick = {
                        showUpdateConfirmDialog = false
                        isUpdating = true
                        scope.launch {
                            val res = NXGateApplication.instance.apiClient.triggerServerUpdate(activeServer)
                            if (res.isSuccess) {
                                Toast.makeText(context, "更新任务已启动，正在执行自动部署...", Toast.LENGTH_SHORT).show()
                            } else {
                                isUpdating = false
                                Toast.makeText(context, "触发更新失败: ${res.exceptionOrNull()?.message}", Toast.LENGTH_LONG).show()
                            }
                        }
                    }
                ) {
                    Text("立即升级")
                }
            },
            dismissButton = {
                TextButton(onClick = { showUpdateConfirmDialog = false }) {
                    Text("取消")
                }
            }
        )
    }

    // ==========================================
    // Release Notes Dialog
    // ==========================================
    if (showReleaseNotesDialog && updateInfo != null) {
        AlertDialog(
            onDismissRequest = { showReleaseNotesDialog = false },
            shape = RoundedCornerShape(24.dp),
            title = {
                Text("版本说明: v${updateInfo!!.latestVersion.ifEmpty { updateInfo!!.currentVersion }}", fontWeight = FontWeight.Bold)
            },
            text = {
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(300.dp)
                        .verticalScroll(rememberScrollState())
                ) {
                    Text(
                        text = updateInfo!!.releaseNotes.ifEmpty { "暂无详细发布日志" },
                        style = MaterialTheme.typography.bodySmall,
                        fontFamily = FontFamily.Monospace
                    )
                }
            },
            confirmButton = {
                TextButton(onClick = { showReleaseNotesDialog = false }) {
                    Text("关闭")
                }
            }
        )
    }

    // ==========================================
    // Generate age Key Pair Modal Dialog
    // ==========================================
    if (showAgeGenDialog && activeServer != null) {
        AlertDialog(
            onDismissRequest = { showAgeGenDialog = false },
            shape = RoundedCornerShape(24.dp),
            title = {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Icon(Icons.Rounded.VpnKey, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
                    Text("生成全新 age 密钥对", fontWeight = FontWeight.Bold)
                }
            },
            text = {
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .verticalScroll(rememberScrollState()),
                    verticalArrangement = Arrangement.spacedBy(10.dp)
                ) {
                    Text(
                        text = "选择密钥算法规范：",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )

                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        FilterChip(
                            selected = ageGenType == "x25519",
                            onClick = { ageGenType = "x25519" },
                            label = { Text("X25519 (标准)") },
                            modifier = Modifier.weight(1f)
                        )
                        FilterChip(
                            selected = ageGenType == "mlkem768x25519",
                            onClick = { ageGenType = "mlkem768x25519" },
                            label = { Text("MLKEM768 (抗量子)") },
                            modifier = Modifier.weight(1f)
                        )
                    }

                    if (generatedAgeResult == null) {
                        Button(
                            onClick = {
                                scope.launch {
                                    isGeneratingAge = true
                                    val res = NXGateApplication.instance.apiClient.generateAgeKey(activeServer, ageGenType)
                                    isGeneratingAge = false
                                    if (res.isSuccess) {
                                        generatedAgeResult = res.getOrNull()
                                    } else {
                                        Toast.makeText(context, "生成失败: ${res.exceptionOrNull()?.message}", Toast.LENGTH_LONG).show()
                                    }
                                }
                            },
                            enabled = !isGeneratingAge,
                            shape = RoundedCornerShape(12.dp),
                            modifier = Modifier.fillMaxWidth()
                        ) {
                            if (isGeneratingAge) {
                                CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp, color = MaterialTheme.colorScheme.onPrimary)
                            } else {
                                Text("立即随机生成")
                            }
                        }
                    } else {
                        val res = generatedAgeResult!!
                        Surface(
                            shape = RoundedCornerShape(12.dp),
                            color = MaterialTheme.colorScheme.surfaceContainerHighest,
                            modifier = Modifier.fillMaxWidth()
                        ) {
                            Column(
                                modifier = Modifier.padding(12.dp),
                                verticalArrangement = Arrangement.spacedBy(8.dp)
                            ) {
                                Text("Recipient 公钥:", style = MaterialTheme.typography.labelSmall, fontWeight = FontWeight.Bold)
                                Text(res.publicKey, style = MaterialTheme.typography.bodySmall, fontFamily = FontFamily.Monospace)
                                OutlinedButton(
                                    onClick = { copyToClipboard("公钥", res.publicKey) },
                                    modifier = Modifier.fillMaxWidth()
                                ) {
                                    Icon(Icons.Rounded.ContentCopy, contentDescription = null, modifier = Modifier.size(14.dp))
                                    Spacer(Modifier.width(6.dp))
                                    Text("复制公钥")
                                }

                                HorizontalDivider(modifier = Modifier.padding(vertical = 4.dp))

                                Text("Identity 私钥 (请妥善离线保存!):", style = MaterialTheme.typography.labelSmall, fontWeight = FontWeight.Bold, color = MaterialTheme.colorScheme.error)
                                Text(res.secretKey, style = MaterialTheme.typography.bodySmall, fontFamily = FontFamily.Monospace)
                                OutlinedButton(
                                    onClick = { copyToClipboard("私钥", res.secretKey) },
                                    modifier = Modifier.fillMaxWidth()
                                ) {
                                    Icon(Icons.Rounded.ContentCopy, contentDescription = null, modifier = Modifier.size(14.dp))
                                    Spacer(Modifier.width(6.dp))
                                    Text("复制私钥")
                                }
                            }
                        }

                        Button(
                            onClick = {
                                agePublicKey = res.publicKey
                                showAgeGenDialog = false
                                Toast.makeText(context, "已将新生成的公钥填入配置，记得点击「保存」！", Toast.LENGTH_SHORT).show()
                            },
                            shape = RoundedCornerShape(12.dp),
                            modifier = Modifier.fillMaxWidth()
                        ) {
                            Text("应用为此网关公钥")
                        }
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = { showAgeGenDialog = false }) {
                    Text("关闭")
                }
            }
        )
    }

    // ==========================================
    // Derive age Key Dialog
    // ==========================================
    if (showAgeDeriveDialog && activeServer != null) {
        AlertDialog(
            onDismissRequest = { showAgeDeriveDialog = false },
            shape = RoundedCornerShape(24.dp),
            title = {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Icon(Icons.Rounded.Key, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
                    Text("从已有私钥推导公钥", fontWeight = FontWeight.Bold)
                }
            },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text(
                        text = "输入已有的 age Identity 私钥 (AGE-SECRET-KEY-1...)：",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    OutlinedTextField(
                        value = deriveSecretKeyInput,
                        onValueChange = { deriveSecretKeyInput = it.trim() },
                        label = { Text("Secret Key") },
                        singleLine = false,
                        maxLines = 3,
                        textStyle = MaterialTheme.typography.bodySmall.copy(fontFamily = FontFamily.Monospace),
                        shape = RoundedCornerShape(12.dp),
                        modifier = Modifier.fillMaxWidth()
                    )
                }
            },
            confirmButton = {
                Button(
                    onClick = {
                        if (deriveSecretKeyInput.isBlank()) {
                            Toast.makeText(context, "请输入私钥文本", Toast.LENGTH_SHORT).show()
                            return@Button
                        }
                        scope.launch {
                            isDerivingAge = true
                            val res = NXGateApplication.instance.apiClient.deriveAgeKey(activeServer, deriveSecretKeyInput)
                            isDerivingAge = false
                            if (res.isSuccess) {
                                val pub = res.getOrNull()?.publicKey ?: ""
                                if (pub.isNotEmpty()) {
                                    agePublicKey = pub
                                    showAgeDeriveDialog = false
                                    Toast.makeText(context, "推导成功，已自动填入 Recipient 公钥！", Toast.LENGTH_SHORT).show()
                                }
                            } else {
                                Toast.makeText(context, "推导失败: ${res.exceptionOrNull()?.message}", Toast.LENGTH_LONG).show()
                            }
                        }
                    },
                    enabled = !isDerivingAge
                ) {
                    if (isDerivingAge) {
                        CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp, color = MaterialTheme.colorScheme.onPrimary)
                    } else {
                        Text("推导并填入")
                    }
                }
            },
            dismissButton = {
                TextButton(onClick = { showAgeDeriveDialog = false }) {
                    Text("取消")
                }
            }
        )
    }

    // ==========================================
    // Clear Blacklist Dialog
    // ==========================================
    if (showClearBlacklistDialog && activeServer != null) {
        AlertDialog(
            onDismissRequest = { showClearBlacklistDialog = false },
            shape = RoundedCornerShape(24.dp),
            title = { Text("清空全部屏蔽库", fontWeight = FontWeight.Bold) },
            text = {
                Text("确定要清空该服务器上的全部硬屏蔽节点与隔离记录吗？清空后，所有被拉黑的节点将在下一周期恢复探测。")
            },
            confirmButton = {
                Button(
                    onClick = {
                        showClearBlacklistDialog = false
                        scope.launch {
                            isClearingBlacklist = true
                            val res = NXGateApplication.instance.apiClient.clearBlacklist(activeServer)
                            isClearingBlacklist = false
                            if (res.isSuccess) {
                                Toast.makeText(context, "已成功清空服务端屏蔽库！", Toast.LENGTH_SHORT).show()
                            } else {
                                Toast.makeText(context, "清空失败: ${res.exceptionOrNull()?.message}", Toast.LENGTH_SHORT).show()
                            }
                        }
                    },
                    colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error)
                ) {
                    Text("确认清空")
                }
            },
            dismissButton = {
                TextButton(onClick = { showClearBlacklistDialog = false }) {
                    Text("取消")
                }
            }
        )
    }

    // ==========================================
    // System Logs Modal Bottom Sheet
    // ==========================================
    if (showLogsSheet) {
        ModalBottomSheet(
            onDismissRequest = { showLogsSheet = false },
            sheetState = logsSheetState,
            containerColor = MaterialTheme.colorScheme.surface
        ) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(500.dp)
                    .padding(horizontal = 20.dp, vertical = 10.dp)
            ) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "服务端实时运行日志",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold
                    )
                    IconButton(onClick = { showLogsSheet = false }) {
                        Icon(Icons.Rounded.Close, contentDescription = "关闭")
                    }
                }

                HorizontalDivider(modifier = Modifier.padding(vertical = 8.dp))

                if (isLoadingLogs) {
                    Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                        CircularProgressIndicator()
                    }
                } else if (systemLogs.isEmpty()) {
                    Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                        Text("暂无日志记录", style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                } else {
                    LazyColumn(
                        modifier = Modifier.fillMaxSize(),
                        verticalArrangement = Arrangement.spacedBy(6.dp)
                    ) {
                        items(systemLogs) { log ->
                            Surface(
                                shape = RoundedCornerShape(8.dp),
                                color = MaterialTheme.colorScheme.surfaceContainerHighest,
                                modifier = Modifier.fillMaxWidth()
                            ) {
                                Text(
                                    text = log.formatted,
                                    style = MaterialTheme.typography.bodySmall,
                                    fontFamily = FontFamily.Monospace,
                                    fontSize = 11.sp,
                                    color = MaterialTheme.colorScheme.onSurface,
                                    modifier = Modifier.padding(8.dp)
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}
