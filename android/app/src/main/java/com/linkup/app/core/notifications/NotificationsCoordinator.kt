package com.linkup.app.core.notifications

import com.linkup.app.core.network.ApiException
import com.linkup.app.core.network.NotificationInboxModel
import com.linkup.app.core.network.NotificationsApiClient
import com.linkup.app.core.social.LoadState
import com.linkup.app.core.social.SocialError
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * Backs the frozen design reference's NotificationsPanel with the real,
 * already-persisted notification_deliveries history (see backend
 * internal/notification.InboxService) -- previously the bell icon had no
 * real data source at all, since GET /v1/me/notifications did not exist
 * before this coordinator was written alongside it.
 */
class NotificationsCoordinator(private val api: NotificationsApiClient) {
    private val mutableInbox = MutableStateFlow<LoadState<NotificationInboxModel>>(LoadState.Idle)
    val inbox: StateFlow<LoadState<NotificationInboxModel>> = mutableInbox.asStateFlow()

    val unreadCount: Int
        get() = (mutableInbox.value as? LoadState.Content)?.value?.unreadCount ?: 0

    suspend fun refresh() {
        val previous = mutableInbox.value
        mutableInbox.value = LoadState.Loading
        try {
            val snapshot = api.list()
            mutableInbox.value = if (snapshot.items.isEmpty()) LoadState.Empty else LoadState.Content(snapshot)
        } catch (error: CancellationException) {
            throw error
        } catch (error: Exception) {
            // A failed refresh should not erase content the user already
            // saw -- only fall back to a bare Failure when there was
            // nothing on screen to preserve.
            mutableInbox.value = if (previous is LoadState.Content) {
                previous.copy(refreshError = error.toSocialError())
            } else {
                LoadState.Failure(error.toSocialError())
            }
        }
    }

    suspend fun markAllRead() {
        val current = (mutableInbox.value as? LoadState.Content)?.value ?: return
        if (current.unreadCount == 0) return
        try {
            api.markAllRead()
            val updated = current.copy(items = current.items.map { it.copy(read = true) }, unreadCount = 0)
            mutableInbox.value = LoadState.Content(updated)
        } catch (error: CancellationException) {
            throw error
        } catch (_: Exception) {
            // Best-effort: the next refresh reconciles with the server's
            // own state either way, matching this codebase's other
            // optimistic-update-with-server-reconciliation coordinators.
        }
    }

    fun clear() {
        mutableInbox.value = LoadState.Idle
    }

    private fun Exception.toSocialError(): SocialError = when (this) {
        is ApiException -> SocialError(code, message, requestId, status)
        else -> SocialError("network_error", message ?: "request failed")
    }
}
