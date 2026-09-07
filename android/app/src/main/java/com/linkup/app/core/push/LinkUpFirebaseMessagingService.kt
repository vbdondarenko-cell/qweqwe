package com.linkup.app.core.push

import android.Manifest
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import com.linkup.app.BuildConfig
import com.linkup.app.MainActivity
import com.linkup.app.R
import com.linkup.app.core.network.PushApiClient
import com.linkup.app.core.session.SecureSessionStore
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch

class LinkUpFirebaseMessagingService : FirebaseMessagingService() {
    private val serviceScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    override fun onNewToken(token: String) {
        val baseUrl = BuildConfig.LINKUP_API_BASE_URL.trim()
        if (baseUrl.isBlank() || token.isBlank()) return
        val sessions = SecureSessionStore(applicationContext)
        if (sessions.load() == null) return
        val store = PushInstallationStore(applicationContext)
        val api = PushApiClient(baseUrl, sessions)
        serviceScope.launch {
            try {
                api.registerAndroid(store.installationId(), token, BuildConfig.VERSION_NAME)
            } catch (error: CancellationException) {
                throw error
            } catch (_: Exception) {
                // Current token will be fetched and re-synced on the next signed-in app start.
            }
        }
    }

    override fun onMessageReceived(message: RemoteMessage) {
        val title = (message.notification?.title ?: message.data["title"] ?: getString(R.string.app_name)).take(120)
        val body = (message.notification?.body ?: message.data["body"] ?: "").take(500)
        if (body.isBlank()) return

        if (Build.VERSION.SDK_INT >= 33 && checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED) {
            return
        }
        val manager = getSystemService(NotificationManager::class.java)
        manager.createNotificationChannel(
            NotificationChannel(CHANNEL_ID, getString(R.string.push_channel_name), NotificationManager.IMPORTANCE_DEFAULT)
        )
        val openApp = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP or Intent.FLAG_ACTIVITY_SINGLE_TOP),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val notification = android.app.Notification.Builder(this, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_stat_linkup)
            .setContentTitle(title)
            .setContentText(body)
            .setStyle(android.app.Notification.BigTextStyle().bigText(body))
            .setAutoCancel(true)
            .setContentIntent(openApp)
            .build()
        manager.notify(message.messageId?.hashCode() ?: System.currentTimeMillis().toInt(), notification)
    }

    override fun onDestroy() {
        serviceScope.cancel()
        super.onDestroy()
    }

    private companion object {
        const val CHANNEL_ID = "linkup_social"
    }
}
