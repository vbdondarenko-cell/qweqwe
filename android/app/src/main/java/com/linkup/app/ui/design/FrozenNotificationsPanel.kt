package com.linkup.app.ui.design

import androidx.compose.foundation.background
import androidx.compose.foundation.border
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
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.R
import com.linkup.app.core.network.NotificationDeliveryModel
import com.linkup.app.core.network.NotificationInboxModel
import com.linkup.app.core.social.LoadState
import com.linkup.app.ui.theme.LinkUpInfo
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpSuccess
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

/**
 * Ports the frozen design reference's NotificationsPanel.tsx onto real,
 * server-backed data (backend internal/notification.InboxService) instead
 * of the reference's fixed demo array -- see IMPLEMENTATION_STATUS.md for
 * the block that added the underlying GET/POST endpoints.
 */
@Composable
fun FrozenNotificationsPanel(
    open: Boolean,
    onClose: () -> Unit,
    state: LoadState<NotificationInboxModel>,
    onMarkAllRead: () -> Unit,
    onRetry: () -> Unit,
) {
    val unreadCount = (state as? LoadState.Content)?.value?.unreadCount ?: 0
    LinkUpSheet(
        open = open,
        onClose = onClose,
        title = stringResource(R.string.notifications_title),
        action = if (unreadCount > 0) {
            { LinkUpButton(stringResource(R.string.notifications_mark_all_read), onMarkAllRead, variant = LinkUpButtonVariant.GHOST, size = LinkUpButtonSize.SM) }
        } else {
            null
        },
    ) {
        when (state) {
            LoadState.Idle, LoadState.Loading -> Box(Modifier.fillMaxWidth().padding(vertical = 32.dp), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = LinkUpRed)
            }
            LoadState.Empty -> FrozenNotificationsEmptyState()
            is LoadState.Failure -> LinkUpErrorState(state.error.message, onRetry = onRetry)
            is LoadState.Content -> Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                if (state.value.items.isEmpty()) {
                    FrozenNotificationsEmptyState()
                } else {
                    state.value.items.forEach { item -> FrozenNotificationRow(item) }
                }
                state.refreshError?.let { Text(it.message, color = LinkUpWarning, fontSize = 11.sp) }
                Spacer(Modifier.height(4.dp))
                Text(
                    stringResource(R.string.notifications_footer),
                    color = LinkUpTextMuted,
                    fontFamily = LinkUpDesign.monoFont,
                    fontSize = 10.sp,
                    modifier = Modifier.fillMaxWidth().padding(vertical = 8.dp),
                )
            }
        }
    }
}

@Composable
private fun FrozenNotificationsEmptyState() {
    LinkUpEmptyState(
        title = stringResource(R.string.notifications_empty_title),
        subtitle = stringResource(R.string.notifications_empty_body),
    )
}

internal data class NotificationVisual(val icon: FrozenIconKind, val color: Color)

internal fun visualFor(type: String): NotificationVisual = when (type) {
    "FRIEND_REQUEST", "FRIEND_ACCEPTED" -> NotificationVisual(FrozenIconKind.ME, LinkUpSuccess)
    "EVENT_RECOMMENDATION" -> NotificationVisual(FrozenIconKind.PULSE, LinkUpInfo)
    "PROMO", "ADVERTISEMENT" -> NotificationVisual(FrozenIconKind.FLY, LinkUpWarning)
    "SYSTEM", "SECURITY", "ACCOUNT" -> NotificationVisual(FrozenIconKind.BELL, LinkUpTextMuted)
    else -> NotificationVisual(FrozenIconKind.PULSE, LinkUpRed) // MESSAGE, EVENT, EVENT_REMINDER
}

@Composable
private fun FrozenNotificationRow(item: NotificationDeliveryModel) {
    val visual = visualFor(item.type)
    Row(
        Modifier.fillMaxWidth()
            .clip(RoundedCornerShape(16.dp))
            .background(if (item.read) Color.Transparent else LinkUpRed.copy(alpha = .05f))
            .then(if (item.read) Modifier else Modifier.border(1.dp, LinkUpRed.copy(alpha = .15f), RoundedCornerShape(16.dp)))
            .padding(12.dp),
        verticalAlignment = Alignment.Top,
    ) {
        Box(
            Modifier.size(40.dp).clip(CircleShape).background(visual.color.copy(alpha = .15f))
                .border(1.dp, visual.color.copy(alpha = .30f), CircleShape),
            contentAlignment = Alignment.Center,
        ) { FrozenLineIcon(visual.icon, visual.color, Modifier.size(16.dp)) }
        Spacer(Modifier.width(12.dp))
        Column(Modifier.weight(1f)) {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                Text(
                    item.title,
                    color = LinkUpTextPrimary,
                    fontWeight = FontWeight.SemiBold,
                    fontSize = 14.sp,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f, fill = false),
                )
                if (!item.read) Box(Modifier.size(6.dp).clip(CircleShape).background(LinkUpRed))
            }
            Text(item.body, color = LinkUpTextDimmed, fontSize = 12.sp, maxLines = 2, overflow = TextOverflow.Ellipsis)
            Text(notificationTime(item.createdAtEpochMillis), color = LinkUpTextMuted, fontFamily = LinkUpDesign.monoFont, fontSize = 10.sp)
        }
    }
}

private fun notificationTime(epochMillis: Long): String = DateTimeFormatter.ofPattern("dd.MM HH:mm")
    .withZone(ZoneId.systemDefault())
    .format(Instant.ofEpochMilli(epochMillis))
