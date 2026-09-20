package com.nxgate.app.service

import android.content.Intent
import android.os.Build
import android.service.quicksettings.Tile
import android.service.quicksettings.TileService
import com.nxgate.app.MainActivity
import com.nxgate.app.NXGateApplication
import com.nxgate.app.util.NotificationHelper
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

class NXGateTileService : TileService() {
    private var serviceScope: CoroutineScope? = null

    override fun onStartListening() {
        super.onStartListening()
        serviceScope?.cancel()
        serviceScope = CoroutineScope(SupervisorJob() + Dispatchers.Main)

        updateTileState()
    }

    override fun onStopListening() {
        super.onStopListening()
        serviceScope?.cancel()
        serviceScope = null
    }

    private fun updateTileState() {
        val tile = qsTile ?: return
        val activeServer = NXGateApplication.instance.serverStore.activeServer.value

        if (activeServer == null) {
            tile.state = Tile.STATE_INACTIVE
            tile.label = "NXGate 网关"
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
                tile.subtitle = "未配置网关"
            }
            tile.updateTile()
            return
        }

        // 显示已有状态
        tile.state = if (activeServer.isOnline) Tile.STATE_ACTIVE else Tile.STATE_INACTIVE
        tile.label = activeServer.name
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            val latencyDesc = if (activeServer.latencyMs > 0) "${activeServer.latencyMs}ms" else "就绪"
            tile.subtitle = if (activeServer.isOnline) "在线 • $latencyDesc" else "离线"
        }
        tile.updateTile()

        // 异步向远端查询最新状态
        serviceScope?.launch {
            val res = withContext(Dispatchers.IO) {
                NXGateApplication.instance.apiClient.fetchServerStatus(activeServer)
            }
            if (res.isSuccess) {
                val data = res.getOrNull()
                if (data != null && qsTile != null) {
                    val isConn = data.masterGateway.isConnected
                    qsTile.state = if (isConn) Tile.STATE_ACTIVE else Tile.STATE_INACTIVE
                    qsTile.label = activeServer.name
                    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
                        val nodeDesc = data.masterGateway.nodeName.ifEmpty { "主出口就绪" }
                        val speedDesc = data.traffic.downSpeedMbStr
                        qsTile.subtitle = if (isConn) "$nodeDesc • $speedDesc" else "未连接"
                    }
                    qsTile.updateTile()
                }
            }
        }
    }

    override fun onClick() {
        super.onClick()
        val activeServer = NXGateApplication.instance.serverStore.activeServer.value

        if (activeServer == null) {
            // 未配置，直接打开主界面
            val intent = Intent(this, MainActivity::class.java).apply {
                flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            }
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.UPSIDE_DOWN_CAKE) {
                val pendingIntent = android.app.PendingIntent.getActivity(
                    this, 0, intent,
                    android.app.PendingIntent.FLAG_IMMUTABLE
                )
                startActivityAndCollapse(pendingIntent)
            } else {
                @Suppress("DEPRECATION")
                startActivityAndCollapse(intent)
            }
            return
        }

        // 已配置，点击触发“快速换线”
        val tile = qsTile ?: return
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            tile.subtitle = "正在切换最优出口..."
        }
        tile.updateTile()

        serviceScope?.launch {
            val res = withContext(Dispatchers.IO) {
                NXGateApplication.instance.apiClient.refreshPool(activeServer)
            }
            if (res.isSuccess) {
                NotificationHelper.sendStatusNotification(
                    this@NXGateTileService,
                    "NXGate 快速换线",
                    "[${activeServer.name}] 已触发远端节点刷新并重新挑选最优主出口！"
                )
            }
            updateTileState()
        }
    }
}
