package com.nxgate.app.util

import android.Manifest
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import com.nxgate.app.MainActivity
import com.nxgate.app.R

object NotificationHelper {
    const val CHANNEL_ALERTS = "nxgate_alerts"
    const val CHANNEL_STATUS = "nxgate_status"

    private const val NOTIFICATION_ID_ALERT = 1001
    private const val NOTIFICATION_ID_STATUS = 1002

    fun initChannels(context: Context) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val notificationManager = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager

            val alertChannel = NotificationChannel(
                CHANNEL_ALERTS,
                "NXGate 故障告警与换线",
                NotificationManager.IMPORTANCE_HIGH
            ).apply {
                description = "当远端网关发生断流、自动故障转移或节点异常时发出紧急通报"
                enableVibration(true)
            }

            val statusChannel = NotificationChannel(
                CHANNEL_STATUS,
                "NXGate 网关状态",
                NotificationManager.IMPORTANCE_DEFAULT
            ).apply {
                description = "常规状态通报与快捷换线操作反馈"
            }

            notificationManager.createNotificationChannel(alertChannel)
            notificationManager.createNotificationChannel(statusChannel)
        }
    }

    fun sendAlertNotification(context: Context, title: String, message: String) {
        sendNotification(
            context = context,
            channelId = CHANNEL_ALERTS,
            notificationId = NOTIFICATION_ID_ALERT,
            title = title,
            message = message,
            priority = NotificationCompat.PRIORITY_HIGH
        )
    }

    fun sendStatusNotification(context: Context, title: String, message: String) {
        sendNotification(
            context = context,
            channelId = CHANNEL_STATUS,
            notificationId = NOTIFICATION_ID_STATUS,
            title = title,
            message = message,
            priority = NotificationCompat.PRIORITY_DEFAULT
        )
    }

    private fun sendNotification(
        context: Context,
        channelId: String,
        notificationId: Int,
        title: String,
        message: String,
        priority: Int
    ) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            if (ContextCompat.checkSelfPermission(context, Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED) {
                return
            }
        }

        val intent = Intent(context, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
        }
        val pendingIntent = PendingIntent.getActivity(
            context,
            0,
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val notification = NotificationCompat.Builder(context, channelId)
            .setSmallIcon(R.drawable.ic_launcher_foreground)
            .setContentTitle(title)
            .setContentText(message)
            .setStyle(NotificationCompat.BigTextStyle().bigText(message))
            .setPriority(priority)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .build()

        val notificationManager = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        notificationManager.notify(notificationId, notification)
    }
}
