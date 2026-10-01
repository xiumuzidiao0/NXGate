package com.nxgate.app.ui.screens.matrix

import android.widget.Toast
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.CloudDownload
import androidx.compose.material.icons.rounded.Delete
import androidx.compose.material.icons.rounded.QrCode2
import androidx.compose.material.icons.rounded.Security
import androidx.compose.material.icons.rounded.SwapHoriz
import androidx.compose.material.icons.rounded.VpnKey
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import com.nxgate.app.model.InboundProtocolItem
import com.nxgate.app.model.ServerProfile
import com.nxgate.app.ui.components.ConnectedButtonGroup
import com.nxgate.app.ui.components.ConnectedButtonItem
import com.nxgate.app.ui.components.ConnectedButtonStyle
import com.nxgate.app.ui.components.ConnectedListItem

@Composable
fun IngressProtocolsTab(
    inbounds: List<InboundProtocolItem>,
    activeServer: ServerProfile?,
    isEn: Boolean,
    onShowInboundQR: (InboundProtocolItem) -> Unit,
    onChangeOutbound: (InboundProtocolItem) -> Unit,
    onDeleteInbound: (InboundProtocolItem) -> Unit,
    onAddNodeClick: () -> Unit,
    onGetSubscriptionsClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    val context = LocalContext.current

    Column(
        modifier = modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        if (inbounds.isEmpty()) {
            Surface(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(16.dp),
                color = MaterialTheme.colorScheme.surfaceContainerLow
            ) {
                Text(
                    text = if (activeServer == null) {
                        if (isEn) "No servers managed yet. Go to Dashboard or Settings to add a VPS gateway." else "当前尚未纳管任何服务器，请前往「概览」或「设置」添加 VPS 网关。"
                    } else {
                        if (isEn) "No sing-box nodes configured. Tap \"Add Node\" below to create one." else "当前服务器尚未创建 sing-box 节点，请点击下方「添加节点」创建。"
                    },
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurface,
                    modifier = Modifier.padding(20.dp)
                )
            }
        } else {
            Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
                inbounds.forEachIndexed { index, inbound ->
                    ConnectedListItem(
                        index = index,
                        total = inbounds.size,
                        headline = inbound.title,
                        supportingText = inbound.subtitle,
                        leadingIcon = if (index % 2 == 0) Icons.Rounded.VpnKey else Icons.Rounded.Security,
                        trailingContent = {
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                IconButton(onClick = { onShowInboundQR(inbound) }) {
                                    Icon(Icons.Rounded.QrCode2, contentDescription = if (isEn) "Share" else "分享", tint = MaterialTheme.colorScheme.primary)
                                }
                                IconButton(onClick = { onChangeOutbound(inbound) }) {
                                    Icon(Icons.Rounded.SwapHoriz, contentDescription = if (isEn) "Change Egress" else "更改出口", tint = MaterialTheme.colorScheme.secondary)
                                }
                                IconButton(onClick = { onDeleteInbound(inbound) }) {
                                    Icon(Icons.Rounded.Delete, contentDescription = if (isEn) "Delete" else "删除入站", tint = MaterialTheme.colorScheme.error)
                                }
                            }
                        },
                        onClick = {
                            Toast.makeText(context, "${if (isEn) "Inbound: " else "入站: "}${inbound.name} (Port ${inbound.port})", Toast.LENGTH_SHORT).show()
                        }
                    )
                }
            }
        }

        // 入站协议操作组
        ConnectedButtonGroup(
            items = listOf(
                ConnectedButtonItem(
                    text = if (isEn) "Add Node" else "添加节点",
                    style = ConnectedButtonStyle.Filled,
                    icon = Icons.Rounded.Add,
                    onClick = onAddNodeClick
                ),
                ConnectedButtonItem(
                    text = if (isEn) "Get Subscriptions" else "获取订阅",
                    style = ConnectedButtonStyle.Tonal,
                    icon = Icons.Rounded.CloudDownload,
                    onClick = onGetSubscriptionsClick
                )
            )
        )
    }
}
