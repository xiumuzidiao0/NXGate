package com.nxgate.app.service

import android.app.PendingIntent
import android.appwidget.AppWidgetManager
import android.appwidget.AppWidgetProvider
import android.content.ComponentName
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.widget.RemoteViews
import android.widget.Toast
import com.nxgate.app.MainActivity
import com.nxgate.app.NXGateApplication
import com.nxgate.app.R
import com.nxgate.app.model.LiveTrafficInfo
import com.nxgate.app.model.MasterGatewayInfo
import com.nxgate.app.model.ServerProfile
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

class NXGateWidgetProvider : AppWidgetProvider() {

    companion object {
        const val ACTION_WIDGET_ROTATE = "com.nxgate.app.ACTION_WIDGET_ROTATE"
        const val ACTION_WIDGET_REFRESH = "com.nxgate.app.ACTION_WIDGET_REFRESH"

        fun buildRemoteViews(
            context: Context,
            appWidgetId: Int,
            server: ServerProfile?,
            liveStatus: Pair<MasterGatewayInfo, LiveTrafficInfo>? = null
        ): RemoteViews {
            val views = RemoteViews(context.packageName, R.layout.widget_nxgate)

            // Intent to open Main App
            val openIntent = Intent(context, MainActivity::class.java).apply {
                flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
                data = Uri.parse("nxgate://widget/open/$appWidgetId")
            }
            val openPendingIntent = PendingIntent.getActivity(
                context,
                appWidgetId * 10,
                openIntent,
                PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
            )
            views.setOnClickPendingIntent(R.id.widget_btn_open, openPendingIntent)
            views.setOnClickPendingIntent(R.id.widget_root, openPendingIntent)

            // Intent for Rotate button
            val rotateIntent = Intent(context, NXGateWidgetProvider::class.java).apply {
                action = ACTION_WIDGET_ROTATE
                `package` = context.packageName
                data = Uri.parse("nxgate://widget/rotate/$appWidgetId")
            }
            val rotatePendingIntent = PendingIntent.getBroadcast(
                context,
                appWidgetId * 10 + 1,
                rotateIntent,
                PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
            )
            views.setOnClickPendingIntent(R.id.widget_btn_rotate, rotatePendingIntent)

            // Intent for Refresh button
            val refreshIntent = Intent(context, NXGateWidgetProvider::class.java).apply {
                action = ACTION_WIDGET_REFRESH
                `package` = context.packageName
                data = Uri.parse("nxgate://widget/refresh/$appWidgetId")
            }
            val refreshPendingIntent = PendingIntent.getBroadcast(
                context,
                appWidgetId * 10 + 2,
                refreshIntent,
                PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
            )
            views.setOnClickPendingIntent(R.id.widget_btn_refresh, refreshPendingIntent)

            if (server == null) {
                views.setTextViewText(R.id.widget_title, "NXGate (未配置)")
                views.setTextViewText(R.id.widget_exit_ip, "请先在应用内添加 VPS 网关")
                views.setTextViewText(R.id.widget_speed, "下行: 0.0 Mb/s · 总计: 0.0 Mb")
                views.setTextViewText(R.id.widget_status_text, "未连接")
                views.setImageViewResource(R.id.widget_status_dot, R.drawable.widget_dot_offline)
            } else if (liveStatus != null) {
                val (master, traffic) = liveStatus
                views.setTextViewText(R.id.widget_title, server.name)
                val nodeName = if (master.nodeName.isNotBlank()) master.nodeName else "主出口就绪"
                views.setTextViewText(R.id.widget_exit_ip, "出口: $nodeName")
                views.setTextViewText(R.id.widget_speed, "下行: ${traffic.downSpeedMbStr} · 上行: ${traffic.upSpeedMbStr}")
                views.setTextViewText(R.id.widget_status_text, if (master.isConnected) "在线" else "离线")
                views.setImageViewResource(
                    R.id.widget_status_dot,
                    if (master.isConnected) R.drawable.widget_dot_online else R.drawable.widget_dot_offline
                )
            } else {
                views.setTextViewText(R.id.widget_title, server.name)
                val exitDesc = if (server.exitIp.isNotEmpty()) "出口: ${server.exitIp} ${server.ipType}" else "出口: 待测"
                views.setTextViewText(R.id.widget_exit_ip, exitDesc)
                views.setTextViewText(R.id.widget_speed, "下行: ${server.downSpeedStr} · 总计: ${server.totalTrafficStr}")
                views.setTextViewText(R.id.widget_status_text, if (server.isOnline) "在线" else "离线")
                views.setImageViewResource(
                    R.id.widget_status_dot,
                    if (server.isOnline) R.drawable.widget_dot_online else R.drawable.widget_dot_offline
                )
            }

            return views
        }

        fun updateAllWidgets(context: Context) {
            try {
                val appWidgetManager = AppWidgetManager.getInstance(context)
                val thisWidget = ComponentName(context, NXGateWidgetProvider::class.java)
                val allWidgetIds = appWidgetManager.getAppWidgetIds(thisWidget)
                if (allWidgetIds.isEmpty()) return

                val server = NXGateApplication.instance.serverStore.activeServer.value
                    ?: NXGateApplication.instance.serverStore.servers.value.firstOrNull()

                for (id in allWidgetIds) {
                    val views = buildRemoteViews(context, id, server)
                    appWidgetManager.updateAppWidget(id, views)
                }
            } catch (_: Exception) {}
        }
    }

    override fun onUpdate(context: Context, appWidgetManager: AppWidgetManager, appWidgetIds: IntArray) {
        val server = NXGateApplication.instance.serverStore.activeServer.value
            ?: NXGateApplication.instance.serverStore.servers.value.firstOrNull()

        // 1. 同步即时渲染，保障桌面微件瞬时刷新不白屏
        for (appWidgetId in appWidgetIds) {
            val views = buildRemoteViews(context, appWidgetId, server)
            appWidgetManager.updateAppWidget(appWidgetId, views)
        }

        // 2. 异步向远端拉取最新状态（利用 goAsync 保障安卓 14/15/16 后台协程生命周期）
        if (server != null) {
            val pendingResult = goAsync()
            CoroutineScope(Dispatchers.IO).launch {
                try {
                    val res = NXGateApplication.instance.apiClient.fetchServerStatus(server)
                    if (res.isSuccess) {
                        val data = res.getOrNull()
                        if (data != null) {
                            val live = Pair(data.masterGateway, data.traffic)
                            for (appWidgetId in appWidgetIds) {
                                val views = buildRemoteViews(context, appWidgetId, server, live)
                                appWidgetManager.updateAppWidget(appWidgetId, views)
                            }
                            NXGateApplication.instance.serverStore.updateServerTraffic(
                                server.id,
                                downSpeedStr = data.traffic.downSpeedMbStr,
                                upSpeedStr = data.traffic.upSpeedMbStr,
                                totalTrafficStr = data.traffic.totalTrafficGbStr,
                                activeConns = data.traffic.activeConnections
                            )
                        }
                    }
                } catch (_: Exception) {
                } finally {
                    pendingResult.finish()
                }
            }
        }
    }

    override fun onReceive(context: Context, intent: Intent) {
        super.onReceive(context, intent)
        when (intent.action) {
            ACTION_WIDGET_ROTATE -> {
                val server = NXGateApplication.instance.serverStore.activeServer.value
                    ?: NXGateApplication.instance.serverStore.servers.value.firstOrNull()
                if (server == null) {
                    Toast.makeText(context, "未配置任何网关服务器", Toast.LENGTH_SHORT).show()
                    return
                }

                Toast.makeText(context, "正在为 [${server.name}] 换线...", Toast.LENGTH_SHORT).show()
                val pendingResult = goAsync()
                CoroutineScope(Dispatchers.IO).launch {
                    try {
                        val res = NXGateApplication.instance.apiClient.triggerRotate(server)
                        val msg = res.getOrDefault("已触发换线评估")
                        withContext(Dispatchers.Main) {
                            Toast.makeText(context, msg, Toast.LENGTH_SHORT).show()
                        }
                        // 换线后重新拉取并渲染
                        val statusRes = NXGateApplication.instance.apiClient.fetchServerStatus(server)
                        if (statusRes.isSuccess) {
                            val data = statusRes.getOrNull()
                            if (data != null) {
                                val appWidgetManager = AppWidgetManager.getInstance(context)
                                val thisWidget = ComponentName(context, NXGateWidgetProvider::class.java)
                                val allWidgetIds = appWidgetManager.getAppWidgetIds(thisWidget)
                                val live = Pair(data.masterGateway, data.traffic)
                                for (id in allWidgetIds) {
                                    val views = buildRemoteViews(context, id, server, live)
                                    appWidgetManager.updateAppWidget(id, views)
                                }
                            }
                        }
                    } catch (e: Exception) {
                        withContext(Dispatchers.Main) {
                            Toast.makeText(context, "换线失败: ${e.message}", Toast.LENGTH_SHORT).show()
                        }
                    } finally {
                        pendingResult.finish()
                    }
                }
            }
            ACTION_WIDGET_REFRESH -> {
                val server = NXGateApplication.instance.serverStore.activeServer.value
                    ?: NXGateApplication.instance.serverStore.servers.value.firstOrNull()
                updateAllWidgets(context)

                if (server != null) {
                    val pendingResult = goAsync()
                    CoroutineScope(Dispatchers.IO).launch {
                        try {
                            val statusRes = NXGateApplication.instance.apiClient.fetchServerStatus(server)
                            if (statusRes.isSuccess) {
                                val data = statusRes.getOrNull()
                                if (data != null) {
                                    val appWidgetManager = AppWidgetManager.getInstance(context)
                                    val thisWidget = ComponentName(context, NXGateWidgetProvider::class.java)
                                    val allWidgetIds = appWidgetManager.getAppWidgetIds(thisWidget)
                                    val live = Pair(data.masterGateway, data.traffic)
                                    for (id in allWidgetIds) {
                                        val views = buildRemoteViews(context, id, server, live)
                                        appWidgetManager.updateAppWidget(id, views)
                                    }
                                    withContext(Dispatchers.Main) {
                                        Toast.makeText(context, "网关状态已刷新", Toast.LENGTH_SHORT).show()
                                    }
                                }
                            }
                        } catch (_: Exception) {
                        } finally {
                            pendingResult.finish()
                        }
                    }
                } else {
                    Toast.makeText(context, "网关状态已刷新", Toast.LENGTH_SHORT).show()
                }
            }
        }
    }
}
