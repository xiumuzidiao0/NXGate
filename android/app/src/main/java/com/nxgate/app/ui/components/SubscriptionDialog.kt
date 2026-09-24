package com.nxgate.app.ui.components

import android.content.Intent
import android.net.Uri
import android.widget.Toast
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.AltRoute
import androidx.compose.material.icons.automirrored.rounded.OpenInNew
import androidx.compose.material.icons.rounded.CloudDownload
import androidx.compose.material.icons.rounded.ContentCopy
import androidx.compose.material.icons.rounded.Link
import androidx.compose.material.icons.rounded.Lock
import androidx.compose.material.icons.rounded.Share
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.nxgate.app.NXGateApplication
import com.nxgate.app.model.ServerProfile
import com.nxgate.app.util.AppStringsEn
import com.nxgate.app.util.LocalAppStrings

@Composable
fun SubscriptionDialog(
    server: ServerProfile,
    onDismissRequest: () -> Unit
) {
    val context = LocalContext.current
    val strings = LocalAppStrings.current
    val clipboardManager = LocalClipboardManager.current
    var isLoading by remember { mutableStateOf(true) }
    var genericUrl by remember { mutableStateOf("${server.baseUrl}/api/singbox/subscription") }
    var clashUrl by remember { mutableStateOf("${server.baseUrl}/api/singbox/subscription/clash") }
    var nodeCount by remember { mutableIntStateOf(0) }
    var isAgeEnabled by remember { mutableStateOf(false) }
    var agePublicKey by remember { mutableStateOf("") }

    LaunchedEffect(server) {
        isLoading = true
        val result = NXGateApplication.instance.apiClient.fetchSubscriptionInfo(server)
        result.onSuccess { info ->
            genericUrl = info.genericSubUrl
            clashUrl = info.clashSubUrl
            nodeCount = info.nodeCount
            isAgeEnabled = info.ageEncryptEnabled
            agePublicKey = info.agePublicKey
        }
        isLoading = false
    }

    AlertDialog(
        onDismissRequest = onDismissRequest,
        shape = RoundedCornerShape(28.dp),
        title = {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(10.dp)
            ) {
                Surface(
                    shape = RoundedCornerShape(12.dp),
                    color = MaterialTheme.colorScheme.primary.copy(alpha = 0.12f),
                    modifier = Modifier.size(40.dp)
                ) {
                    Box(contentAlignment = Alignment.Center) {
                        Icon(
                            Icons.Rounded.CloudDownload,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.primary,
                            modifier = Modifier.size(24.dp)
                        )
                    }
                }
                Column {
                    Text(
                        text = strings.subDialogTitle,
                        style = MaterialTheme.typography.titleLarge,
                        fontWeight = FontWeight.Bold
                    )
                    Text(
                        text = if (nodeCount > 0) (if (strings == AppStringsEn) "Loaded $nodeCount online nodes" else "已载入 ${nodeCount} 个在线可用节点") else strings.subDialogSubtitle,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }
        },
        text = {
            Column(
                modifier = Modifier.fillMaxWidth(),
                verticalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                Text(
                    text = "网关目标: ${server.name} (${server.host}:${server.port})",
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )

                if (isAgeEnabled) {
                    Surface(
                        shape = RoundedCornerShape(8.dp),
                        color = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.65f),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Row(
                            modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp),
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(6.dp)
                        ) {
                            Icon(
                                Icons.Rounded.Lock,
                                contentDescription = null,
                                tint = MaterialTheme.colorScheme.primary,
                                modifier = Modifier.size(14.dp)
                            )
                            Text(
                                text = strings.ageEncryptedBadge,
                                style = MaterialTheme.typography.labelMedium,
                                fontWeight = FontWeight.SemiBold,
                                color = MaterialTheme.colorScheme.onPrimaryContainer
                            )
                        }
                    }
                }

                if (isLoading) {
                    Box(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 24.dp),
                        contentAlignment = Alignment.Center
                    ) {
                        CircularProgressIndicator(modifier = Modifier.size(32.dp))
                    }
                } else {
                    // 1. 通用全量订阅卡片
                    Surface(
                        shape = RoundedCornerShape(18.dp),
                        color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.55f),
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable {
                                clipboardManager.setText(AnnotatedString(genericUrl))
                                val tip = if (strings == AppStringsEn) {
                                    if (isAgeEnabled) "Copied age-encrypted universal subscription!" else "Copied universal subscription link!"
                                } else {
                                    if (isAgeEnabled) "已复制已通过 age 加密的通用订阅 (Base64/Raw)！" else "已复制通用全量订阅链接 (Base64/Raw)！"
                                }
                                Toast.makeText(context, tip, Toast.LENGTH_SHORT).show()
                                onDismissRequest()
                            }
                    ) {
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(14.dp),
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(12.dp)
                        ) {
                            Surface(
                                shape = RoundedCornerShape(12.dp),
                                color = MaterialTheme.colorScheme.primary.copy(alpha = 0.15f),
                                modifier = Modifier.size(42.dp)
                            ) {
                                Box(contentAlignment = Alignment.Center) {
                                    Icon(
                                        Icons.Rounded.Link,
                                        contentDescription = null,
                                        tint = MaterialTheme.colorScheme.primary,
                                        modifier = Modifier.size(22.dp)
                                    )
                                }
                            }
                            Column(modifier = Modifier.weight(1f)) {
                                Text(
                                    text = strings.genericSubTitle,
                                    style = MaterialTheme.typography.titleSmall,
                                    fontWeight = FontWeight.SemiBold
                                )
                                Spacer(Modifier.height(2.dp))
                                Text(
                                    text = strings.genericSubDesc,
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    fontSize = 11.sp
                                )
                            }
                            IconButton(
                                onClick = {
                                    val encodedUrl = Uri.encode(genericUrl)
                                    val profileName = Uri.encode("NXGate-${server.name}")
                                    val singBoxUri = "sing-box://import-remote-profile?url=$encodedUrl#$profileName"
                                    val v2rayUri = "v2rayng://install-sub?url=$encodedUrl&name=$profileName"
                                    var launched = false
                                    for (targetUri in listOf(singBoxUri, v2rayUri)) {
                                        try {
                                            val intent = Intent(Intent.ACTION_VIEW, Uri.parse(targetUri)).apply {
                                                flags = Intent.FLAG_ACTIVITY_NEW_TASK
                                            }
                                            context.startActivity(intent)
                                            launched = true
                                            Toast.makeText(context, "已唤起客户端导入订阅", Toast.LENGTH_SHORT).show()
                                            onDismissRequest()
                                            break
                                        } catch (_: Exception) {}
                                    }
                                    if (!launched) {
                                        clipboardManager.setText(AnnotatedString(genericUrl))
                                        Toast.makeText(context, "未检测到已安装的通用代理客户端，已复制链接", Toast.LENGTH_SHORT).show()
                                    }
                                },
                                modifier = Modifier.size(36.dp)
                            ) {
                                Icon(
                                    Icons.AutoMirrored.Rounded.OpenInNew,
                                    contentDescription = strings.importToClient,
                                    tint = MaterialTheme.colorScheme.primary,
                                    modifier = Modifier.size(18.dp)
                                )
                            }
                            IconButton(
                                onClick = {
                                    val sendIntent = Intent().apply {
                                        action = Intent.ACTION_SEND
                                        putExtra(Intent.EXTRA_TEXT, genericUrl)
                                        type = "text/plain"
                                    }
                                    context.startActivity(Intent.createChooser(sendIntent, if (strings == AppStringsEn) "Share Universal Subscription" else "分享通用订阅链接"))
                                },
                                modifier = Modifier.size(36.dp)
                            ) {
                                Icon(
                                    Icons.Rounded.Share,
                                    contentDescription = strings.share,
                                    tint = MaterialTheme.colorScheme.primary,
                                    modifier = Modifier.size(18.dp)
                                )
                            }
                            Icon(
                                Icons.Rounded.ContentCopy,
                                contentDescription = strings.copy,
                                tint = MaterialTheme.colorScheme.primary,
                                modifier = Modifier.size(20.dp)
                            )
                        }
                    }

                    // 2. Clash Meta / Mihomo 订阅卡片
                    Surface(
                        shape = RoundedCornerShape(18.dp),
                        color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.55f),
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable {
                                clipboardManager.setText(AnnotatedString(clashUrl))
                                val tip = if (strings == AppStringsEn) {
                                    if (isAgeEnabled) "Copied age-encrypted Clash subscription!" else "Copied Clash Meta / Mihomo subscription!"
                                } else {
                                    if (isAgeEnabled) "已复制已通过 age 加密的 Clash Meta 订阅！" else "已复制 Clash Meta / Mihomo 分流订阅！"
                                }
                                Toast.makeText(context, tip, Toast.LENGTH_SHORT).show()
                                onDismissRequest()
                            }
                    ) {
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(14.dp),
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(12.dp)
                        ) {
                            Surface(
                                shape = RoundedCornerShape(12.dp),
                                color = MaterialTheme.colorScheme.secondary.copy(alpha = 0.15f),
                                modifier = Modifier.size(42.dp)
                            ) {
                                Box(contentAlignment = Alignment.Center) {
                                    Icon(
                                        Icons.AutoMirrored.Rounded.AltRoute,
                                        contentDescription = null,
                                        tint = MaterialTheme.colorScheme.secondary,
                                        modifier = Modifier.size(22.dp)
                                    )
                                }
                            }
                            Column(modifier = Modifier.weight(1f)) {
                                Text(
                                    text = strings.clashSubTitle,
                                    style = MaterialTheme.typography.titleSmall,
                                    fontWeight = FontWeight.SemiBold
                                )
                                Spacer(Modifier.height(2.dp))
                                Text(
                                    text = strings.clashSubDesc,
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    fontSize = 11.sp
                                )
                            }
                            IconButton(
                                onClick = {
                                    val encodedUrl = Uri.encode(clashUrl)
                                    val profileName = Uri.encode("NXGate-${server.name}")
                                    val clashUri = "clash://install-config?url=$encodedUrl&name=$profileName"
                                    try {
                                        val intent = Intent(Intent.ACTION_VIEW, Uri.parse(clashUri)).apply {
                                            flags = Intent.FLAG_ACTIVITY_NEW_TASK
                                        }
                                        context.startActivity(intent)
                                        Toast.makeText(context, "已唤起 Clash 客户端导入配置", Toast.LENGTH_SHORT).show()
                                        onDismissRequest()
                                    } catch (e: Exception) {
                                        clipboardManager.setText(AnnotatedString(clashUrl))
                                        Toast.makeText(context, "未检测到已安装的 Clash 客户端，已复制链接", Toast.LENGTH_SHORT).show()
                                    }
                                },
                                modifier = Modifier.size(36.dp)
                            ) {
                                Icon(
                                    Icons.AutoMirrored.Rounded.OpenInNew,
                                    contentDescription = strings.importToClash,
                                    tint = MaterialTheme.colorScheme.secondary,
                                    modifier = Modifier.size(18.dp)
                                )
                            }
                            IconButton(
                                onClick = {
                                    val sendIntent = Intent().apply {
                                        action = Intent.ACTION_SEND
                                        putExtra(Intent.EXTRA_TEXT, clashUrl)
                                        type = "text/plain"
                                    }
                                    context.startActivity(Intent.createChooser(sendIntent, if (strings == AppStringsEn) "Share Clash Subscription" else "分享 Clash 订阅链接"))
                                },
                                modifier = Modifier.size(36.dp)
                            ) {
                                Icon(
                                    Icons.Rounded.Share,
                                    contentDescription = strings.share,
                                    tint = MaterialTheme.colorScheme.secondary,
                                    modifier = Modifier.size(18.dp)
                                )
                            }
                            Icon(
                                Icons.Rounded.ContentCopy,
                                contentDescription = strings.copy,
                                tint = MaterialTheme.colorScheme.secondary,
                                modifier = Modifier.size(20.dp)
                            )
                        }
                    }
                }
            }
        },
        confirmButton = {
            TextButton(onClick = onDismissRequest) {
                Text(strings.close)
            }
        }
    )
}
