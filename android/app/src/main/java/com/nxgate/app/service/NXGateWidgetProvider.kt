package com.nxgate.app.service

import android.app.PendingIntent
import android.appwidget.AppWidgetManager
import android.appwidget.AppWidgetProvider
import android.content.ComponentName
import android.content.Context
import android.content.Intent
import android.widget.RemoteViews
import android.widget.Toast
import com.nxgate.app.MainActivity
import com.nxgate.app.NXGateApplication
import com.nxgate.app.R
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

class NXGateWidgetProvider : AppWidgetProvider() {

    companion object {
        const val ACTION_WIDGET_ROTATE = "com.nxgate.app.ACTION_WIDGET_ROTATE"
        const val ACTION_WIDGET_REFRESH = "com.nxgate.app.ACTION_WIDGET_REFRESH"

        fun updateAllWidgets(context: Context) {
            val appWidgetManager = AppWidgetManager.getInstance(context)
            val thisWidget = ComponentName(context, NXGateWidgetProvider::class.java)
            val allWidgetIds = appWidgetManager.getAppWidgetIds(thisWidget)
            val intent = Intent(context, NXGateWidgetProvider::class.java).apply {
                action = AppWidgetManager.ACTION_APPWIDGET_UPDATE
                putExtra(AppWidgetManager.EXTRA_APPWIDGET_IDS, allWidgetIds)
            }
            context.sendBroadcast(intent)
        }
    }

    override fun onUpdate(context: Context, appWidgetManager: AppWidgetManager, appWidgetIds: IntArray) {
        val server = NXGateApplication.instance.serverStore.activeServer.value
            ?: NXGateApplication.instance.serverStore.servers.value.firstOrNull()

        for (appWidgetId in appWidgetIds) {
            val views = RemoteViews(context.packageName, R.layout.widget_nxgate)

            // Intent to open Main App
            val openIntent = Intent(context, MainActivity::class.java).apply {
                flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            }
            val openPendingIntent = PendingIntent.getActivity(
                context,
                0,
                openIntent,
                PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
            )
            views.setOnClickPendingIntent(R.id.widget_btn_open, openPendingIntent)
            views.setOnClickPendingIntent(R.id.widget_root, openPendingIntent)

            // Intent for Rotate button
            val rotateIntent = Intent(context, NXGateWidgetProvider::class.java).apply {
                action = ACTION_WIDGET_ROTATE
            }
            val rotatePendingIntent = PendingIntent.getBroadcast(
                context,
                1,
                rotateIntent,
                PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
            )
            views.setOnClickPendingIntent(R.id.widget_btn_rotate, rotatePendingIntent)

            // Intent for Refresh button
            val refreshIntent = Intent(context, NXGateWidgetProvider::class.java).apply {
                action = ACTION_WIDGET_REFRESH
            }
            val refreshPendingIntent = PendingIntent.getBroadcast(
                context,
                2,
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
                appWidgetManager.updateAppWidget(appWidgetId, views)
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
                appWidgetManager.updateAppWidget(appWidgetId, views)

                // Asynchronously query live status
                CoroutineScope(Dispatchers.IO).launch {
                    try {
                        val res = NXGateApplication.instance.apiClient.fetchServerStatus(server)
                        if (res.isSuccess) {
                            val data = res.getOrNull()
                            if (data != null) {
                                val master = data.masterGateway
                                val tf = data.traffic
                                val updatedViews = RemoteViews(context.packageName, R.layout.widget_nxgate)
                                updatedViews.setOnClickPendingIntent(R.id.widget_btn_open, openPendingIntent)
                                updatedViews.setOnClickPendingIntent(R.id.widget_root, openPendingIntent)
                                updatedViews.setOnClickPendingIntent(R.id.widget_btn_rotate, rotatePendingIntent)
                                updatedViews.setOnClickPendingIntent(R.id.widget_btn_refresh, refreshPendingIntent)

                                updatedViews.setTextViewText(R.id.widget_title, server.name)
                                updatedViews.setTextViewText(R.id.widget_exit_ip, "出口: ${master.nodeName}")
                                updatedViews.setTextViewText(R.id.widget_speed, "下行: ${tf.downSpeedMbStr} · 上行: ${tf.upSpeedMbStr}")
                                updatedViews.setTextViewText(R.id.widget_status_text, if (master.isConnected) "在线" else "离线")
                                updatedViews.setImageViewResource(
                                    R.id.widget_status_dot,
                                    if (master.isConnected) R.drawable.widget_dot_online else R.drawable.widget_dot_offline
                                )
                                appWidgetManager.updateAppWidget(appWidgetId, updatedViews)

                                NXGateApplication.instance.serverStore.updateServerTraffic(
                                    server.id,
                                    downSpeedStr = tf.downSpeedMbStr,
                                    upSpeedStr = tf.upSpeedMbStr,
                                    totalTrafficStr = tf.totalTrafficGbStr,
                                    activeConns = tf.activeConnections
                                )
                            }
                        }
                    } catch (_: Exception) {}
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
                } else {
                    Toast.makeText(context, "正在为 [${server.name}] 换线...", Toast.LENGTH_SHORT).show()
                    CoroutineScope(Dispatchers.IO).launch {
                        try {
                            val res = NXGateApplication.instance.apiClient.triggerRotate(server)
                            val msg = res.getOrDefault("已触发换线评估")
                            CoroutineScope(Dispatchers.Main).launch {
                                Toast.makeText(context, msg, Toast.LENGTH_SHORT).show()
                                updateAllWidgets(context)
                            }
                        } catch (e: Exception) {
                            CoroutineScope(Dispatchers.Main).launch {
                                Toast.makeText(context, "换线失败: ${e.message}", Toast.LENGTH_SHORT).show()
                            }
                        }
                    }
                }
            }
            ACTION_WIDGET_REFRESH -> {
                updateAllWidgets(context)
                Toast.makeText(context, "网关状态已刷新", Toast.LENGTH_SHORT).show()
            }
        }
    }
}
