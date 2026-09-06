package com.linkup.app.ui.social

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.core.network.SlotModel
import com.linkup.app.core.network.SlotState
import com.linkup.app.core.network.SlotViewerState
import com.linkup.app.core.social.LoadState
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpInfo
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpSuccess
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning
import com.linkup.app.ui.theme.LinkUpZone
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

@Composable
fun PulseScreen(
    state: LoadState<List<SlotModel>>,
    onRefresh: () -> Unit,
    onSlotClick: (SlotModel) -> Unit,
    onPrimaryAction: (SlotModel) -> Unit,
) {
    var search by remember { mutableStateOf("") }
    var selectedFilter by remember { mutableStateOf("All") }
    val filters = listOf("All", "Social", "Active", "Food")

    Column {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .background(Color(0xF2050506))
                .border(width = 1.dp, color = LinkUpBorder)
                .padding(start = 20.dp, end = 20.dp, top = 18.dp, bottom = 12.dp),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Column(modifier = Modifier.weight(1f)) {
                    Text("Pulse", color = LinkUpTextPrimary, fontSize = 20.sp, fontWeight = FontWeight.Bold)
                    Text("Real nearby LinkUps", color = LinkUpTextMuted, fontSize = 11.sp)
                }
                TextButton(onClick = onRefresh) { Text("Refresh", color = LinkUpRed, fontWeight = FontWeight.Bold) }
            }
            Spacer(Modifier.height(10.dp))
            OutlinedTextField(
                value = search,
                onValueChange = { search = it.take(120) },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                placeholder = { Text("Search activities, places...", color = LinkUpTextMuted, fontSize = 13.sp) },
                shape = RoundedCornerShape(12.dp),
            )
        }

        LazyRow(
            modifier = Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 10.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            items(filters) { filter ->
                FilterChip(filter, selectedFilter == filter) { selectedFilter = filter }
            }
        }

        when (state) {
            LoadState.Idle -> Box(Modifier.fillMaxWidth().padding(32.dp), contentAlignment = Alignment.Center) {
                TextButton(onClick = onRefresh) { Text("Load Pulse", color = LinkUpRed) }
            }
            LoadState.Loading -> Box(Modifier.fillMaxWidth().padding(40.dp), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = LinkUpRed)
            }
            LoadState.Empty -> EmptyState("Quiet around here", "No public LinkUps are available right now.", onRefresh)
            is LoadState.Failure -> EmptyState("Couldn't load Pulse", state.error.message, onRefresh)
            is LoadState.Content -> {
                val query = search.trim().lowercase()
                val filtered = state.value.filter { slot ->
                    val categoryMatch = selectedFilter == "All" || activityCategory(slot.activity) == selectedFilter
                    val searchMatch = query.isBlank() || slot.title.lowercase().contains(query) ||
                        slot.activity.lowercase().contains(query) || slot.placeText.lowercase().contains(query) ||
                        slot.details.orEmpty().lowercase().contains(query)
                    categoryMatch && searchMatch
                }
                if (filtered.isEmpty()) {
                    EmptyState("No matches", "Try a different filter or search term.", onRefresh)
                } else {
                    LazyColumn(
                        modifier = Modifier.fillMaxWidth(),
                        contentPadding = androidx.compose.foundation.layout.PaddingValues(start = 20.dp, end = 20.dp, bottom = 112.dp),
                        verticalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        items(filtered, key = { it.id }) { slot ->
                            SlotCard(slot, onSlotClick = { onSlotClick(slot) }, onPrimaryAction = { onPrimaryAction(slot) })
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun FilterChip(label: String, active: Boolean, onClick: () -> Unit) {
    val background = if (active) LinkUpRed.copy(alpha = 0.15f) else LinkUpElevated
    val border = if (active) LinkUpRed else LinkUpBorder
    val text = if (active) LinkUpRed else LinkUpTextDimmed
    Box(
        modifier = Modifier.clip(RoundedCornerShape(999.dp)).background(background).border(1.dp, border, RoundedCornerShape(999.dp)).clickable(onClick = onClick).padding(horizontal = 14.dp, vertical = 7.dp),
    ) { Text(label, color = text, fontSize = 12.sp, fontWeight = FontWeight.SemiBold) }
}

@Composable
fun SlotCard(slot: SlotModel, onSlotClick: () -> Unit, onPrimaryAction: () -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(16.dp))
            .background(LinkUpElevated)
            .border(BorderStroke(1.dp, LinkUpBorder), RoundedCornerShape(16.dp))
            .clickable(onClick = onSlotClick)
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(11.dp),
    ) {
        Row(verticalAlignment = Alignment.Top) {
            Box(
                modifier = Modifier.size(48.dp).clip(RoundedCornerShape(12.dp)).background(LinkUpZone).border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp)),
                contentAlignment = Alignment.Center,
            ) { Text(activityEmoji(slot.activity), fontSize = 23.sp) }
            Spacer(Modifier.width(12.dp))
            Column(modifier = Modifier.weight(1f)) {
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    StatusBadge(slot)
                    Text(timeLabel(slot.startAtEpochMillis), color = LinkUpTextMuted, fontSize = 11.sp)
                }
                Spacer(Modifier.height(4.dp))
                Text(slot.title, color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, fontSize = 16.sp, maxLines = 1, overflow = TextOverflow.Ellipsis)
                Text(slot.placeText, color = LinkUpTextDimmed, fontSize = 12.sp, maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
        }
        slot.details?.takeIf { it.isNotBlank() }?.let {
            Text(it, color = LinkUpTextDimmed, fontSize = 14.sp, maxLines = 2, overflow = TextOverflow.Ellipsis)
        }
        Row(verticalAlignment = Alignment.CenterVertically) {
            Box(Modifier.size(26.dp).clip(RoundedCornerShape(8.dp)).background(LinkUpZone), contentAlignment = Alignment.Center) {
                Text(slot.organizer.displayName.take(1).uppercase(), color = LinkUpTextPrimary, fontSize = 11.sp, fontWeight = FontWeight.Bold)
            }
            Spacer(Modifier.width(8.dp))
            Text(slot.organizer.displayName, color = LinkUpTextDimmed, fontSize = 12.sp, modifier = Modifier.weight(1f), maxLines = 1)
            Text("${slot.acceptedCount}/${slot.capacity}", color = LinkUpTextMuted, fontSize = 11.sp)
        }
        CapacityBar(slot.acceptedCount, slot.capacity)
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text("@${slot.organizer.username}", color = LinkUpTextMuted, fontSize = 10.sp, modifier = Modifier.weight(1f))
            PrimarySlotButton(slot, onPrimaryAction)
        }
    }
}

@Composable
private fun StatusBadge(slot: SlotModel) {
    val label: String
    val color: Color
    when {
        slot.state == SlotState.ACTIVE -> { label = "LIVE"; color = LinkUpSuccess }
        slot.state == SlotState.FULL -> { label = "FULL"; color = LinkUpTextMuted }
        slot.viewerState == SlotViewerState.PENDING -> { label = "APPROVAL"; color = LinkUpWarning }
        else -> { label = "OPEN"; color = LinkUpInfo }
    }
    Box(Modifier.clip(RoundedCornerShape(6.dp)).background(color.copy(alpha = 0.15f)).border(1.dp, color.copy(alpha = 0.3f), RoundedCornerShape(6.dp)).padding(horizontal = 8.dp, vertical = 3.dp)) {
        Text(label, color = color, fontSize = 10.sp, fontWeight = FontWeight.Bold)
    }
}

@Composable
private fun PrimarySlotButton(slot: SlotModel, onClick: () -> Unit) {
    val (label, color) = when (slot.viewerState) {
        SlotViewerState.NONE -> "Request to join" to LinkUpWarning
        SlotViewerState.PENDING -> "Request sent" to LinkUpWarning
        SlotViewerState.ACCEPTED -> "Open" to LinkUpRed
        SlotViewerState.HOST -> "Manage" to LinkUpRed
    }
    Box(
        modifier = Modifier.clip(RoundedCornerShape(12.dp)).background(color.copy(alpha = if (slot.viewerState == SlotViewerState.NONE) 0.15f else 1f))
            .border(1.dp, color.copy(alpha = 0.35f), RoundedCornerShape(12.dp)).clickable(onClick = onClick).padding(horizontal = 14.dp, vertical = 9.dp),
    ) {
        Text(label, color = if (slot.viewerState == SlotViewerState.NONE) color else LinkUpTextPrimary, fontSize = 12.sp, fontWeight = FontWeight.Bold)
    }
}

@Composable
private fun CapacityBar(value: Int, max: Int) {
    val fraction = if (max <= 0) 0f else (value.toFloat() / max.toFloat()).coerceIn(0f, 1f)
    Box(Modifier.fillMaxWidth().height(5.dp).clip(RoundedCornerShape(99.dp)).background(LinkUpZone)) {
        Box(Modifier.fillMaxWidth(fraction).height(5.dp).background(LinkUpRed))
    }
}

@Composable
private fun EmptyState(title: String, subtitle: String, onRetry: () -> Unit) {
    Column(Modifier.fillMaxWidth().padding(horizontal = 32.dp, vertical = 48.dp), horizontalAlignment = Alignment.CenterHorizontally) {
        Text(title, color = LinkUpTextPrimary, fontSize = 17.sp, fontWeight = FontWeight.Bold)
        Spacer(Modifier.height(7.dp))
        Text(subtitle, color = LinkUpTextDimmed, fontSize = 13.sp)
        Spacer(Modifier.height(12.dp))
        TextButton(onClick = onRetry) { Text("Retry", color = LinkUpRed, fontWeight = FontWeight.Bold) }
    }
}

private fun activityEmoji(activity: String): String = when (activity.lowercase()) {
    "coffee" -> "☕"
    "running", "run" -> "🏃"
    "gym", "workout", "fitness" -> "🏋️"
    "food", "dinner", "lunch" -> "🍜"
    "walk" -> "🚶"
    "music" -> "🎵"
    "games", "gaming" -> "🎮"
    else -> "🎯"
}

private fun activityCategory(activity: String): String = when (activity.lowercase()) {
    "running", "run", "gym", "workout", "fitness", "walk" -> "Active"
    "food", "dinner", "lunch" -> "Food"
    else -> "Social"
}

private fun timeLabel(epochMillis: Long?): String {
    if (epochMillis == null) return "Now"
    return DateTimeFormatter.ofPattern("EEE HH:mm").withZone(ZoneId.systemDefault()).format(Instant.ofEpochMilli(epochMillis))
}
