package com.linkup.app

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.repeatOnLifecycle
import com.linkup.app.core.capability.CapabilityCoordinator
import com.linkup.app.core.hosting.V11HostingCoordinator
import com.linkup.app.core.mutation.DurableMutationHttpTransport
import com.linkup.app.core.mutation.DurableMutationRunner
import com.linkup.app.core.mutation.DurableSocialApi
import com.linkup.app.core.mutation.SecureMutationOutbox
import com.linkup.app.core.network.CapabilityKey
import com.linkup.app.core.network.LinkUpApiClient
import com.linkup.app.core.network.PushApiClient
import com.linkup.app.core.network.RealtimeApiClient
import com.linkup.app.core.network.SlotViewerState
import com.linkup.app.core.network.passwordResetToken
import com.linkup.app.core.push.PushCoordinator
import com.linkup.app.core.realtime.RealtimeCoordinator
import com.linkup.app.core.realtime.RealtimePull
import com.linkup.app.core.realtime.SharedPreferencesRealtimeCursorStore
import com.linkup.app.core.session.SecureSessionStore
import com.linkup.app.core.session.SessionCoordinator
import com.linkup.app.core.session.SessionState
import com.linkup.app.core.social.ChatRefreshResult
import com.linkup.app.core.social.LoadState
import com.linkup.app.core.social.SocialCoordinator
import com.linkup.app.ui.LinkUpApp
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpTheme
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

class MainActivity : ComponentActivity() {
    // Password reset credentials are intentionally process-memory only.
    private var pendingResetToken by mutableStateOf<String?>(null)
    private var notificationPermissionRequested = false
    private val activityScope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        captureResetToken(intent)

        val apiBaseUrl = BuildConfig.LINKUP_API_BASE_URL.trim()
        if (apiBaseUrl.isBlank()) {
            setContent {
                LinkUpTheme {
                    Column(
                        Modifier.fillMaxSize().padding(28.dp),
                        verticalArrangement = Arrangement.Center,
                        horizontalAlignment = Alignment.CenterHorizontally,
                    ) {
                        Text(stringResource(R.string.config_required_title), color = LinkUpTextPrimary, fontSize = 20.sp, fontWeight = FontWeight.Bold)
                        Text(
                            stringResource(R.string.config_required_body),
                            color = LinkUpTextDimmed,
                            modifier = Modifier.padding(top = 10.dp),
                        )
                        Text(stringResource(R.string.config_fail_closed), color = LinkUpRed, modifier = Modifier.padding(top = 10.dp), fontWeight = FontWeight.Bold)
                    }
                }
            }
            return
        }

        val sessionStore = SecureSessionStore(applicationContext)
        val api = LinkUpApiClient(apiBaseUrl, sessionStore)
        val sessionCoordinator = SessionCoordinator(api, sessionStore)
        val mutationOutbox = SecureMutationOutbox(applicationContext)
        val durableSocialApi = DurableSocialApi(
            delegate = api,
            currentBearerToken = { sessionStore.load()?.token },
            runner = DurableMutationRunner(mutationOutbox),
            transport = DurableMutationHttpTransport(apiBaseUrl, sessionStore),
            onUnauthorized = { sessionCoordinator.clearLocalSession() },
        )
        val hostingCoordinator = V11HostingCoordinator(durableSocialApi)
        val socialCoordinator = SocialCoordinator(durableSocialApi)
        val capabilityCoordinator = CapabilityCoordinator(api)
        val realtimeCoordinator = RealtimeCoordinator(
            RealtimeApiClient(apiBaseUrl, sessionStore),
            SharedPreferencesRealtimeCursorStore(applicationContext),
        )
        val pushCoordinator = PushCoordinator(applicationContext, PushApiClient(apiBaseUrl, sessionStore))
        val pushConfigured = pushCoordinator.configure()

        activityScope.launch {
            lifecycle.repeatOnLifecycle(Lifecycle.State.STARTED) {
                sessionCoordinator.state.collectLatest { state ->
                    if (state is SessionState.SignedIn) {
                        capabilityCoordinator.refresh()
                    } else {
                        capabilityCoordinator.reset()
                    }
                }
            }
        }

        if (pushConfigured) {
            activityScope.launch {
                combine(
                    sessionCoordinator.state.map { it is SessionState.SignedIn },
                    capabilityCoordinator.snapshot.map { it.enabled(CapabilityKey.NOTIFICATIONS) },
                ) { signedIn, notificationsEnabled -> signedIn && notificationsEnabled }
                    .distinctUntilChanged()
                    .collectLatest { enabled ->
                        if (!enabled) return@collectLatest
                        if (Build.VERSION.SDK_INT >= 33 &&
                            !notificationPermissionRequested &&
                            checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED
                        ) {
                            notificationPermissionRequested = true
                            requestPermissions(arrayOf(Manifest.permission.POST_NOTIFICATIONS), REQUEST_NOTIFICATIONS)
                        }
                        try {
                            pushCoordinator.sync()
                        } catch (error: CancellationException) {
                            throw error
                        } catch (_: Exception) {
                            // Non-critical. The next signed-in foreground refresh re-evaluates the server gate and retries.
                        }
                    }
            }
        }

        activityScope.launch {
            sessionCoordinator.state.collectLatest { state ->
                if (state is SessionState.SignedOut) {
                    mutationOutbox.clearAll()
                    hostingCoordinator.clear()
                    capabilityCoordinator.reset()
                }
            }
        }

        activityScope.launch {
            lifecycle.repeatOnLifecycle(Lifecycle.State.STARTED) {
                combine(
                    sessionCoordinator.state.map { (it as? SessionState.SignedIn)?.user?.id },
                    capabilityCoordinator.snapshot.map { it.enabled(CapabilityKey.REALTIME) },
                ) { userId, realtimeEnabled -> if (realtimeEnabled) userId else null }
                    .distinctUntilChanged()
                    .collectLatest { userId ->
                        if (userId != null) {
                            runRealtimeLoop(
                                userId = userId,
                                realtime = realtimeCoordinator,
                                sessions = sessionCoordinator,
                                social = socialCoordinator,
                                durableSocial = durableSocialApi,
                            )
                        }
                    }
            }
        }

        setContent {
            LinkUpTheme {
                LinkUpApp(
                    lifecycle = lifecycle,
                    api = api,
                    sessions = sessionCoordinator,
                    social = socialCoordinator,
                    hosting = hostingCoordinator,
                    capabilities = capabilityCoordinator,
                    resetToken = pendingResetToken,
                    onResetTokenConsumed = { pendingResetToken = null },
                )
            }
        }
    }

    private suspend fun runRealtimeLoop(
        userId: String,
        realtime: RealtimeCoordinator,
        sessions: SessionCoordinator,
        social: SocialCoordinator,
        durableSocial: DurableSocialApi,
    ) {
        var needsAuthoritativeSnapshot = true
        while (currentCoroutineContext().isActive) {
            val nextDelay = try {
                if (needsAuthoritativeSnapshot) {
                    if (!reconcileAuthoritativeSnapshot(sessions, social)) {
                        REALTIME_RETRY_MS
                    } else {
                        needsAuthoritativeSnapshot = false
                        REALTIME_DRAIN_MS
                    }
                } else {
                    val replay = durableSocial.replayPending()
                    if (replay != null && (replay.acknowledgedKeys.isNotEmpty() || replay.definitiveFailureKey != null)) {
                        if (!reconcileAfterDurableReplay(social)) {
                            REALTIME_RETRY_MS
                        } else {
                            REALTIME_DRAIN_MS
                        }
                    } else {
                        val pull = realtime.pull(userId, REALTIME_BATCH_SIZE)
                        if (pull.nextCursor > pull.fromCursor) {
                            val reconciled = reconcileRealtime(pull, sessions, social)
                            if (reconciled) {
                                realtime.acknowledge(userId, pull.nextCursor)
                            }
                            when {
                                !reconciled -> REALTIME_RETRY_MS
                                pull.events.size >= REALTIME_BATCH_SIZE -> REALTIME_DRAIN_MS
                                else -> REALTIME_POLL_MS
                            }
                        } else {
                            REALTIME_POLL_MS
                        }
                    }
                }
            } catch (error: CancellationException) {
                throw error
            } catch (_: Exception) {
                REALTIME_RETRY_MS
            }
            delay(nextDelay)
        }
    }

    private suspend fun reconcileAuthoritativeSnapshot(
        sessions: SessionCoordinator,
        social: SocialCoordinator,
    ): Boolean {
        if (!sessions.refreshSignedInProfile()) return false
        return reconcileAfterDurableReplay(social)
    }

    private suspend fun reconcileAfterDurableReplay(social: SocialCoordinator): Boolean {
        social.refreshPulse()
        if (!social.pulse.value.refreshSucceeded()) return false

        val before = (social.selectedSlot.value as? LoadState.Content)?.value ?: return true
        social.openSlot(before.id)
        val selectedState = social.selectedSlot.value
        if (selectedState is LoadState.Failure && selectedState.error.isDefinitiveAccessLoss()) {
            social.clearSelected()
            return true
        }
        if (!selectedState.refreshSucceeded()) return false

        val selected = (social.selectedSlot.value as? LoadState.Content)?.value ?: return true
        if (selected.viewerState == SlotViewerState.HOST) {
            social.refreshPending(selected.id)
            if (!social.pending.value.refreshSucceeded()) return false
            social.refreshAccepted(selected.id)
            if (!social.accepted.value.refreshSucceeded()) return false
        }
        if (social.chat.value !is LoadState.Idle) {
            when (social.refreshChat(selected.id)) {
                ChatRefreshResult.SUCCESS -> Unit
                ChatRefreshResult.RETRY -> return false
                ChatRefreshResult.STOP -> social.clearChat()
            }
        }
        return true
    }

    private suspend fun reconcileRealtime(
        pull: RealtimePull,
        sessions: SessionCoordinator,
        social: SocialCoordinator,
    ): Boolean {
        val hints = pull.hints

        if (hints.refreshProfile && !sessions.refreshSignedInProfile()) return false

        if (hints.refreshPulse) {
            social.refreshPulse()
            if (!social.pulse.value.refreshSucceeded()) return false
        }

        val before = (social.selectedSlot.value as? LoadState.Content)?.value
        if (before != null && before.id in hints.slotIds) {
            social.openSlot(before.id)
            val selectedState = social.selectedSlot.value
            if (selectedState is LoadState.Failure && selectedState.error.isDefinitiveAccessLoss()) {
                social.clearSelected()
            } else if (!selectedState.refreshSucceeded()) {
                return false
            }
        }

        val selected = (social.selectedSlot.value as? LoadState.Content)?.value
        if (selected != null && hints.refreshRelationships && selected.id in hints.slotIds && selected.viewerState == SlotViewerState.HOST) {
            social.refreshPending(selected.id)
            if (!social.pending.value.refreshSucceeded()) return false
            social.refreshAccepted(selected.id)
            if (!social.accepted.value.refreshSucceeded()) return false
        }

        if (selected != null && selected.id in hints.chatSlotIds && social.chat.value !is LoadState.Idle) {
            when (social.refreshChat(selected.id)) {
                ChatRefreshResult.SUCCESS -> Unit
                ChatRefreshResult.RETRY -> return false
                ChatRefreshResult.STOP -> social.clearChat()
            }
        }

        return true
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        captureResetToken(intent)
    }

    override fun onDestroy() {
        activityScope.cancel()
        super.onDestroy()
    }

    private fun captureResetToken(intent: Intent?) {
        if (intent?.action != Intent.ACTION_VIEW) return
        val token = passwordResetToken(intent.dataString.orEmpty()) ?: return
        pendingResetToken = token
        // Drop the URI reference after extracting the credential so it is not retained by Activity intent state.
        intent.data = null
    }

    private fun LoadState<*>.refreshSucceeded(): Boolean = when (this) {
        is LoadState.Failure -> false
        is LoadState.Loading -> false
        is LoadState.Content<*> -> refreshError == null
        LoadState.Idle, LoadState.Empty -> true
    }

    private fun com.linkup.app.core.social.SocialError.isDefinitiveAccessLoss(): Boolean =
        code in setOf("unauthorized", "forbidden", "not_found")

    private companion object {
        const val REQUEST_NOTIFICATIONS = 1001
        const val REALTIME_BATCH_SIZE = 100
        const val REALTIME_DRAIN_MS = 250L
        const val REALTIME_POLL_MS = 4_000L
        const val REALTIME_RETRY_MS = 5_000L
    }
}
