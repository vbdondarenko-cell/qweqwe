package com.linkup.app.ui.social

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.core.network.PendingSlotRequest
import com.linkup.app.core.network.SlotModel
import com.linkup.app.core.network.SlotState
import com.linkup.app.core.network.SlotViewerState
import com.linkup.app.core.social.LoadState
import com.linkup.app.core.social.MutationState
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpSuccess
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning
import com.linkup.app.ui.theme.LinkUpZone

@Composable
fun SlotDetailScreen(
    state: LoadState<SlotModel>,
    pending: LoadState<List<PendingSlotRequest>>,
    mutation: MutationState,
    onBack: () -> Unit,
    onRefresh: (String) -> Unit,
    onRequest: (String) -> Unit,
    onLeave: (String) -> Unit,
    onRefreshPending: (String) -> Unit,
    onApprove: (String, String) -> Unit,
    onReject: (String, String) -> Unit,
    onStart: (String) -> Unit,
    onComplete: (String) -> Unit,
    onCancel: (String, Long) -> Unit,
    onEdit: (SlotModel) -> Unit,
    onOpenChat: (String) -> Unit,
) {
    Column(Modifier.fillMaxSize()) {
        Row(
            Modifier.fillMaxWidth().border(1.dp, LinkUpBorder).padding(horizontal = 14.dp, vertical = 10.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TextButton(onClick = onBack) { Text("‹", color = LinkUpTextDimmed, fontSize = 28.sp) }
            Text("LINK", color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, modifier = Modifier.weight(1f))
        }

        when (state) {
            LoadState.Idle, LoadState.Loading -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator(color = LinkUpRed) }
            LoadState.Empty -> Unit
            is LoadState.Failure -> CenterMessage("Couldn't load LINK", state.error.message, onBack)
            is LoadState.Content -> {
                val slot = state.value
                Column(
                    Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(horizontal = 20.dp, vertical = 14.dp),
                    verticalArrangement = Arrangement.spacedBy(14.dp),
                ) {
                    Row(verticalAlignment = Alignment.Top) {
                        Box(Modifier.size(58.dp).clip(RoundedCornerShape(16.dp)).background(LinkUpZone).border(1.dp, LinkUpBorder, RoundedCornerShape(16.dp)), contentAlignment = Alignment.Center) {
                            Text(activityEmojiForDetail(slot.activity), fontSize = 29.sp)
                        }
                        Spacer(Modifier.width(12.dp))
                        Column(modifier = Modifier.weight(1f)) {
                            Text(slot.state.name, color = stateColor(slot.state), fontSize = 10.sp, fontWeight = FontWeight.Bold)
                            Text(slot.title, color = LinkUpTextPrimary, fontSize = 20.sp, fontWeight = FontWeight.Bold)
                            Text(slot.placeText, color = LinkUpTextDimmed, fontSize = 13.sp)
                        }
                    }

                    slot.details?.takeIf { it.isNotBlank() }?.let { Text(it, color = LinkUpTextDimmed, fontSize = 14.sp) }

                    Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(14.dp)).background(LinkUpElevated).border(1.dp, LinkUpBorder, RoundedCornerShape(14.dp)).padding(14.dp)) {
                        Text("Host", color = LinkUpTextMuted, fontSize = 10.sp)
                        Text(slot.organizer.displayName, color = LinkUpTextPrimary, fontWeight = FontWeight.Bold)
                        Text("@${slot.organizer.username}", color = LinkUpTextMuted, fontSize = 12.sp)
                        Spacer(Modifier.height(8.dp))
                        Text("${slot.acceptedCount}/${slot.capacity} going", color = LinkUpTextDimmed, fontSize = 12.sp)
                        Text("Server version ${slot.version}", color = LinkUpTextMuted, fontSize = 10.sp)
                    }

                    MutationError(mutation)

                    when (slot.viewerState) {
                        SlotViewerState.NONE -> ActionButton("Request to join", LinkUpWarning, mutation !is MutationState.Running) { onRequest(slot.id) }
                        SlotViewerState.PENDING -> ActionButton("Cancel request", LinkUpWarning, mutation !is MutationState.Running) { onLeave(slot.id) }
                        SlotViewerState.ACCEPTED -> {
                            ActionButton("Open Chat", LinkUpRed, true) { onOpenChat(slot.id) }
                            ActionButton("Leave LINK", LinkUpWarning, mutation !is MutationState.Running) { onLeave(slot.id) }
                        }
                        SlotViewerState.HOST -> {
                            HostControls(
                                slot = slot,
                                pending = pending,
                                busy = mutation is MutationState.Running,
                                onRefreshPending = onRefreshPending,
                                onApprove = onApprove,
                                onReject = onReject,
                                onStart = onStart,
                                onComplete = onComplete,
                                onCancel = onCancel,
                                onEdit = onEdit,
                                onOpenChat = onOpenChat,
                            )
                        }
                    }

                    TextButton(onClick = { onRefresh(slot.id) }, modifier = Modifier.align(Alignment.CenterHorizontally)) {
                        Text("Refresh", color = LinkUpRed)
                    }
                }
            }
        }
    }
}

@Composable
private fun HostControls(
    slot: SlotModel,
    pending: LoadState<List<PendingSlotRequest>>,
    busy: Boolean,
    onRefreshPending: (String) -> Unit,
    onApprove: (String, String) -> Unit,
    onReject: (String, String) -> Unit,
    onStart: (String) -> Unit,
    onComplete: (String) -> Unit,
    onCancel: (String, Long) -> Unit,
    onEdit: (SlotModel) -> Unit,
    onOpenChat: (String) -> Unit,
) {
    Text("Host controls", color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, fontSize = 15.sp)
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        SmallAction("Edit", Modifier.weight(1f), !busy) { onEdit(slot) }
        SmallAction("Requests", Modifier.weight(1f), !busy) { onRefreshPending(slot.id) }
    }

    when (pending) {
        LoadState.Idle -> Unit
        LoadState.Loading -> CircularProgressIndicator(color = LinkUpRed, modifier = Modifier.size(24.dp))
        LoadState.Empty -> Text("No pending requests", color = LinkUpTextMuted, fontSize = 12.sp)
        is LoadState.Failure -> Text(pending.error.message, color = LinkUpWarning, fontSize = 12.sp)
        is LoadState.Content -> pending.value.forEach { request ->
            Row(
                Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp)).background(LinkUpElevated).border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp)).padding(12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(request.user.displayName, color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, fontSize = 13.sp, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    Text("@${request.user.username}", color = LinkUpTextMuted, fontSize = 11.sp)
                }
                TextButton(onClick = { onReject(slot.id, request.user.id) }, enabled = !busy) { Text("Decline", color = LinkUpTextMuted) }
                TextButton(onClick = { onApprove(slot.id, request.user.id) }, enabled = !busy) { Text("Accept", color = LinkUpSuccess, fontWeight = FontWeight.Bold) }
            }
        }
    }

    if (slot.acceptedCount > 0 && slot.state != SlotState.COMPLETED && slot.state != SlotState.CANCELLED) {
        ActionButton("Open Chat", LinkUpRed, true) { onOpenChat(slot.id) }
    }
    when (slot.state) {
        SlotState.FILLING, SlotState.FULL -> if (slot.acceptedCount > 0) ActionButton("START", LinkUpSuccess, !busy) { onStart(slot.id) }
        SlotState.ACTIVE -> ActionButton("COMPLETE", LinkUpSuccess, !busy) { onComplete(slot.id) }
        else -> Unit
    }
    if (slot.state !in setOf(SlotState.COMPLETED, SlotState.CANCELLED, SlotState.EXPIRED, SlotState.MODERATED)) {
        ActionButton("Delete / Cancel LINK", LinkUpWarning, !busy) { onCancel(slot.id, slot.version) }
    }
}

@Composable
private fun MutationError(state: MutationState) {
    if (state is MutationState.Failed) Text(state.error.message, color = LinkUpWarning, fontSize = 12.sp)
}

@Composable
private fun ActionButton(label: String, color: androidx.compose.ui.graphics.Color, enabled: Boolean, onClick: () -> Unit) {
    Box(
        Modifier.fillMaxWidth().height(48.dp).clip(RoundedCornerShape(12.dp)).background(color.copy(alpha = if (enabled) 1f else .35f))
            .clickable(enabled = enabled, onClick = onClick),
        contentAlignment = Alignment.Center,
    ) { Text(label, color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, fontSize = 13.sp) }
}

@Composable
private fun SmallAction(label: String, modifier: Modifier, enabled: Boolean, onClick: () -> Unit) {
    Box(
        modifier.height(44.dp).clip(RoundedCornerShape(12.dp)).background(LinkUpElevated).border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp))
            .clickable(enabled = enabled, onClick = onClick),
        contentAlignment = Alignment.Center,
    ) { Text(label, color = LinkUpTextDimmed, fontWeight = FontWeight.Bold, fontSize = 12.sp) }
}

@Composable
private fun CenterMessage(title: String, message: String, onBack: () -> Unit) {
    Column(Modifier.fillMaxSize().padding(32.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) {
        Text(title, color = LinkUpTextPrimary, fontWeight = FontWeight.Bold)
        Text(message, color = LinkUpTextDimmed)
        TextButton(onClick = onBack) { Text("Back", color = LinkUpRed) }
    }
}

private fun activityEmojiForDetail(activity: String) = when (activity.lowercase()) {
    "coffee" -> "☕"; "running", "run" -> "🏃"; "gym", "fitness" -> "🏋️"; "food" -> "🍜"; "walk" -> "🚶"; "music" -> "🎵"; "games" -> "🎮"; else -> "🎯"
}

private fun stateColor(state: SlotState) = when (state) {
    SlotState.ACTIVE -> LinkUpSuccess
    SlotState.CANCELLED, SlotState.MODERATED -> LinkUpWarning
    else -> LinkUpRed
}
