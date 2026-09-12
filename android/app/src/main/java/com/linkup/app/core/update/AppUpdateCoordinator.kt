package com.linkup.app.core.update

import android.app.DownloadManager
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.net.Uri
import android.provider.Settings
import androidx.core.content.ContextCompat
import com.linkup.app.BuildConfig
import com.linkup.app.R
import com.linkup.app.core.network.AppUpdateApiClient
import com.linkup.app.core.network.AppUpdateInfo
import java.security.MessageDigest
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/** One step in the over-the-air update flow -- see [AppUpdateCoordinator]. */
sealed interface AppUpdateState {
    data object Idle : AppUpdateState
    data class Available(val info: AppUpdateInfo) : AppUpdateState
    data class Downloading(val info: AppUpdateInfo) : AppUpdateState
    data class ReadyToInstall(val info: AppUpdateInfo, val apkUri: Uri) : AppUpdateState
    data class Failed(val info: AppUpdateInfo?, val message: String) : AppUpdateState
}

/**
 * Drives LinkUp's over-the-air update flow: ask the server whether a newer
 * build exists, hand the download to [DownloadManager] (so the OS -- not
 * this process -- keeps it going across the app being backgrounded or
 * killed), verify the finished file's SHA-256 against the server-signed
 * manifest, then offer it to the platform installer.
 *
 * Deliberately not silent and not a Play Store replacement: Android
 * requires explicit "install unknown apps" consent per source app (see
 * [canInstallUnknownApps]), and this always launches the normal system
 * install confirmation screen. There is no background/auto-install path.
 *
 * Uses [DownloadManager.getUriForDownloadedFile] rather than a custom
 * destination + FileProvider: leaving the destination unset means
 * DownloadManager keeps the file in its own storage and hands back a
 * `content://` URI that both this app and the system installer can read.
 */
class AppUpdateCoordinator(
    context: Context,
    private val api: AppUpdateApiClient,
) {
    private val appContext = context.applicationContext
    private val downloadManager =
        appContext.getSystemService(Context.DOWNLOAD_SERVICE) as DownloadManager

    private val mutableState = MutableStateFlow<AppUpdateState>(AppUpdateState.Idle)
    val state: StateFlow<AppUpdateState> = mutableState.asStateFlow()

    @Volatile private var pendingDownloadId: Long? = null
    @Volatile private var pendingInfo: AppUpdateInfo? = null
    private var registered = false

    private val receiver = object : BroadcastReceiver() {
        override fun onReceive(receivedContext: Context, intent: Intent) {
            val completedId = intent.getLongExtra(DownloadManager.EXTRA_DOWNLOAD_ID, -1L)
            val expectedId = pendingDownloadId ?: return
            if (completedId == expectedId) onDownloadComplete(expectedId)
        }
    }

    /** Call once (e.g. from onCreate) before a download can ever complete. */
    fun register() {
        if (registered) return
        registered = true
        val filter = IntentFilter(DownloadManager.ACTION_DOWNLOAD_COMPLETE)
        // DownloadManager's own system process sends this broadcast, not
        // this app, so it must be registered as exported to be delivered.
        ContextCompat.registerReceiver(appContext, receiver, filter, ContextCompat.RECEIVER_EXPORTED)
    }

    fun unregister() {
        if (!registered) return
        registered = false
        runCatching { appContext.unregisterReceiver(receiver) }
    }

    /** Fire-and-forget: a failed check just means "nothing to report yet". */
    suspend fun check() {
        val info = try {
            api.latest()
        } catch (error: CancellationException) {
            throw error
        } catch (_: Exception) {
            return
        }
        if (info.versionCode > BuildConfig.VERSION_CODE) {
            mutableState.value = AppUpdateState.Available(info)
        }
    }

    fun canInstallUnknownApps(): Boolean = appContext.packageManager.canRequestPackageInstalls()

    fun requestInstallUnknownAppsSettings(): Intent =
        Intent(
            Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES,
            Uri.parse("package:${appContext.packageName}"),
        )

    fun startDownload(info: AppUpdateInfo) {
        val request = DownloadManager.Request(Uri.parse(info.apkUrl))
            .setTitle(appContext.getString(R.string.app_update_notification_title))
            .setDescription(info.versionName)
            .setNotificationVisibility(DownloadManager.Request.VISIBILITY_VISIBLE_NOTIFY_COMPLETED)
            .setMimeType("application/vnd.android.package-archive")
        pendingInfo = info
        pendingDownloadId = downloadManager.enqueue(request)
        mutableState.value = AppUpdateState.Downloading(info)
    }

    private fun onDownloadComplete(downloadId: Long) {
        val info = pendingInfo
        pendingDownloadId = null
        val statusOk = downloadManager.query(DownloadManager.Query().setFilterById(downloadId))?.use { cursor ->
            if (!cursor.moveToFirst()) return@use false
            val statusIndex = cursor.getColumnIndex(DownloadManager.COLUMN_STATUS)
            statusIndex >= 0 && cursor.getInt(statusIndex) == DownloadManager.STATUS_SUCCESSFUL
        } ?: false
        if (!statusOk) {
            mutableState.value = AppUpdateState.Failed(info, "download did not complete successfully")
            return
        }
        val uri = downloadManager.getUriForDownloadedFile(downloadId)
        if (uri == null || info == null) {
            mutableState.value = AppUpdateState.Failed(info, "downloaded file unavailable")
            return
        }
        if (!verifySha256(uri, info.sha256)) {
            mutableState.value = AppUpdateState.Failed(info, "checksum mismatch -- discarding download")
            return
        }
        mutableState.value = AppUpdateState.ReadyToInstall(info, uri)
    }

    private fun verifySha256(uri: Uri, expectedHex: String): Boolean {
        val digest = MessageDigest.getInstance("SHA-256")
        val stream = runCatching { appContext.contentResolver.openInputStream(uri) }.getOrNull() ?: return false
        stream.use { input ->
            val buffer = ByteArray(64 * 1024)
            while (true) {
                val read = input.read(buffer)
                if (read < 0) break
                digest.update(buffer, 0, read)
            }
        }
        val actualHex = digest.digest().joinToString("") { "%02x".format(it) }
        return actualHex.equals(expectedHex.trim(), ignoreCase = true)
    }

    fun installIntent(apkUri: Uri): Intent =
        Intent(Intent.ACTION_VIEW).apply {
            setDataAndType(apkUri, "application/vnd.android.package-archive")
            addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }

    fun dismiss() {
        mutableState.value = AppUpdateState.Idle
    }
}
