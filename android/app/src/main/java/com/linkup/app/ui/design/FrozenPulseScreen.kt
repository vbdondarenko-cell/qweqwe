package com.linkup.app.ui.design

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.R
import com.linkup.app.core.network.SlotModel
import com.linkup.app.core.network.SlotState
import com.linkup.app.core.social.LoadState
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpSuccess
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning
import com.linkup.app.ui.theme.LinkUpZone
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId

private enum class PulseTimeFilter { NOW, TONIGHT, TOMORROW, ALL }
private enum class PulseCategoryFilter { ALL, SOCIAL, ACTIVE, FOOD }
private val timeFilters = PulseTimeFilter.values().toList()
private val categoryFilters = PulseCategoryFilter.values().toList()

@Composable
fun FrozenPulseScreen(
    state: LoadState<List<SlotModel>>,
    onRefresh: () -> Unit,
    onSlotClick: (SlotModel) -> Unit,
    onPrimaryAction: (SlotModel) -> Unit,
    waitlistEnabled: Boolean = false,
    modifier: Modifier = Modifier,
    onOpenNotifications: () -> Unit = onRefresh,
) {
    var query by remember { mutableStateOf("") }
    var time by remember { mutableStateOf(PulseTimeFilter.ALL) }
    var category by remember { mutableStateOf(PulseCategoryFilter.ALL) }
    val source = (state as? LoadState.Content)?.value.orEmpty()
    val filtered = source.filter { slot ->
        matchesQuery(slot, query) && matchesCategory(slot, category) && matchesTime(slot, time)
    }
    val happeningNow = source.firstOrNull { it.state == SlotState.ACTIVE }

    Column(modifier.fillMaxSize().background(Color(0xFF050506))) {
        FrozenPulseHeader(query, { query = it.take(120) }, onOpenNotifications)
        LazyRow(
            contentPadding = PaddingValues(horizontal = 20.dp, vertical = 10.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            items(timeFilters) { item -> LinkUpChip(timeFilterLabel(item), time == item, { time = item }) }
        }
        LazyRow(
            contentPadding = PaddingValues(horizontal = 20.dp, vertical = 0.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            items(categoryFilters) { item -> LinkUpChip(categoryFilterLabel(item), category == item, { category = item }) }
        }

        when (state) {
            LoadState.Idle -> LinkUpErrorState(stringResource(R.string.pulse_not_loaded), onRetry = onRefresh, title = stringResource(R.string.pulse_load))
            LoadState.Loading -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator(color = LinkUpRed) }
            LoadState.Empty -> LinkUpEmptyState(stringResource(R.string.pulse_quiet_title), stringResource(R.string.pulse_quiet_body))
            is LoadState.Failure -> LinkUpErrorState(state.error.message, onRetry = onRefresh)
            is LoadState.Content -> LazyColumn(
                modifier = Modifier.fillMaxSize(),
                contentPadding = PaddingValues(start = 20.dp, end = 20.dp, top = 10.dp, bottom = 112.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                if (happeningNow != null && (time == PulseTimeFilter.NOW || time == PulseTimeFilter.ALL)) {
                    item(key = "happening-${happeningNow.id}") {
                        FrozenHappeningNow(happeningNow.toFrozenSlot()) { onSlotClick(happeningNow) }
                    }
                }
                if (filtered.isEmpty()) {
                    item { LinkUpEmptyState(stringResource(R.string.pulse_no_matches_title), stringResource(R.string.pulse_no_matches_body)) }
                } else {
                    items(filtered, key = { it.id }) { slot ->
                        FrozenSlotCard(
                            slot = slot.toFrozenSlot(),
                            onClick = { onSlotClick(slot) },
                            actionLabel = primaryActionLabel(slot.primaryActionKind(waitlistEnabled)),
                            actionEnabled = slot.primaryActionEnabled(waitlistEnabled),
                            onPrimaryAction = { onPrimaryAction(slot) },
                        )
                    }
                }
            }
        }
    }
}

private fun matchesQuery(slot: SlotModel, query: String): Boolean {
    val q = query.trim()
    if (q.isBlank()) return true
    return listOf(slot.title, slot.activity, slot.placeText, slot.zoneText.orEmpty(), slot.details.orEmpty(), slot.organizer.displayName)
        .any { it.contains(q, ignoreCase = true) }
}

private fun matchesCategory(slot: SlotModel, category: PulseCategoryFilter): Boolean = when (category) {
    PulseCategoryFilter.ALL -> true
    PulseCategoryFilter.ACTIVE -> slot.activity.lowercase() in setOf("running", "run", "gym", "workout", "fitness", "walk")
    PulseCategoryFilter.FOOD -> slot.activity.lowercase() in setOf("food", "dinner", "lunch", "coffee")
    PulseCategoryFilter.SOCIAL -> slot.activity.lowercase() !in setOf("running", "run", "gym", "workout", "fitness", "walk", "food", "dinner", "lunch", "coffee")
}

private fun matchesTime(slot: SlotModel, filter: PulseTimeFilter): Boolean {
    if (filter == PulseTimeFilter.ALL) return true
    if (slot.state == SlotState.ACTIVE) return filter == PulseTimeFilter.NOW
    val epoch = slot.startAtEpochMillis ?: return filter == PulseTimeFilter.NOW
    val zone = ZoneId.systemDefault()
    val dateTime = Instant.ofEpochMilli(epoch).atZone(zone)
    val today = LocalDate.now(zone)
    return when (filter) {
        PulseTimeFilter.NOW -> dateTime.toLocalDate() == today
        PulseTimeFilter.TONIGHT -> dateTime.toLocalDate() == today && dateTime.hour >= 17
        PulseTimeFilter.TOMORROW -> dateTime.toLocalDate() == today.plusDays(1)
        PulseTimeFilter.ALL -> true
    }
}

@Composable
private fun timeFilterLabel(filter: PulseTimeFilter): String = stringResource(
    when (filter) {
        PulseTimeFilter.NOW -> R.string.common_now
        PulseTimeFilter.TONIGHT -> R.string.filter_tonight
        PulseTimeFilter.TOMORROW -> R.string.filter_tomorrow
        PulseTimeFilter.ALL -> R.string.filter_all
    },
)

@Composable
private fun categoryFilterLabel(filter: PulseCategoryFilter): String = stringResource(
    when (filter) {
        PulseCategoryFilter.ALL -> R.string.filter_all
        PulseCategoryFilter.SOCIAL -> R.string.filter_social
        PulseCategoryFilter.ACTIVE -> R.string.filter_active
        PulseCategoryFilter.FOOD -> R.string.filter_food
    },
)

@Composable
private fun primaryActionLabel(action: FrozenPrimaryAction): String = stringResource(
    when (action) {
        FrozenPrimaryAction.REQUEST -> R.string.slot_request_to_join
        FrozenPrimaryAction.JOIN -> R.string.slot_join_now
        FrozenPrimaryAction.WAITLIST -> R.string.slot_join_waitlist
        FrozenPrimaryAction.PENDING -> R.string.slot_request_sent
        FrozenPrimaryAction.WAITLISTED -> R.string.slot_waitlisted
        FrozenPrimaryAction.OPEN -> R.string.common_open
        FrozenPrimaryAction.MANAGE -> R.string.common_manage
        FrozenPrimaryAction.FULL -> R.string.slot_state_full
        FrozenPrimaryAction.ACTIVE -> R.string.slot_state_active
        FrozenPrimaryAction.CLOSED -> R.string.slot_closed
        FrozenPrimaryAction.UNAVAILABLE -> R.string.slot_waitlist_unavailable
    },
)

@Composable
private fun FrozenPulseHeader(query: String, onQueryChange: (String) -> Unit, onOpenNotifications: () -> Unit) {
    Column(
        Modifier.fillMaxWidth().background(Color(0xF20D0E10)).border(1.dp, LinkUpBorder)
            .padding(start = 20.dp, end = 20.dp, top = 56.dp, bottom = 12.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(7.dp)) {
                    FrozenLineIcon(FrozenIconKind.MAP, LinkUpRed, Modifier.size(14.dp))
                    Text(stringResource(R.string.pulse_subtitle), color = LinkUpTextPrimary, fontFamily = LinkUpDesign.bodyFont, fontWeight = FontWeight.SemiBold, fontSize = 14.sp)
                }
                Spacer(Modifier.height(5.dp))
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    LinkUpPulseDot(LinkUpRed, size = 8.dp)
                    Text(stringResource(R.string.pulse_live_data), color = LinkUpTextDimmed, fontFamily = LinkUpDesign.monoFont, fontSize = 12.sp)
                }
            }
            Box(
                Modifier.size(40.dp).clip(RoundedCornerShape(12.dp)).background(LinkUpElevated)
                    .border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp)).clickable(onClick = onOpenNotifications),
                contentAlignment = Alignment.Center,
            ) { FrozenLineIcon(FrozenIconKind.PULSE, LinkUpTextDimmed, Modifier.size(18.dp)) }
        }
        Spacer(Modifier.height(12.dp))
        Row(
            Modifier.fillMaxWidth().height(42.dp).clip(RoundedCornerShape(12.dp)).background(LinkUpElevated)
                .border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp)).padding(horizontal = 12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            FrozenLineIcon(FrozenIconKind.SEARCH, LinkUpTextMuted, Modifier.size(16.dp))
            Spacer(Modifier.width(8.dp))
            BasicTextField(
                value = query,
                onValueChange = onQueryChange,
                singleLine = true,
                modifier = Modifier.weight(1f),
                textStyle = TextStyle(color = LinkUpTextPrimary, fontFamily = LinkUpDesign.bodyFont, fontSize = 14.sp),
                decorationBox = { inner -> if (query.isBlank()) Text(stringResource(R.string.pulse_search_hint), color = LinkUpTextMuted, fontSize = 14.sp) else inner() },
            )
        }
    }
}

@Composable
private fun FrozenHappeningNow(slot: FrozenSlot, onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().clip(RoundedCornerShape(16.dp)).background(LinkUpSuccess.copy(alpha = .10f))
            .border(1.dp, LinkUpSuccess.copy(alpha = .25f), RoundedCornerShape(16.dp))
            .clickable(onClick = onClick).padding(16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(Modifier.size(40.dp).clip(RoundedCornerShape(12.dp)).background(LinkUpSuccess.copy(alpha = .20f)), contentAlignment = Alignment.Center) {
            FrozenLineIcon(FrozenIconKind.PULSE, LinkUpSuccess, Modifier.size(18.dp))
        }
        Spacer(Modifier.width(12.dp))
        Column(Modifier.weight(1f)) {
            Text(stringResource(R.string.pulse_happening_now), color = LinkUpSuccess, fontFamily = LinkUpDesign.monoFont, fontWeight = FontWeight.SemiBold, fontSize = 10.sp)
            Text(slot.title, color = LinkUpTextPrimary, fontFamily = LinkUpDesign.displayFont, fontWeight = FontWeight.Bold, fontSize = 14.sp, maxLines = 1, overflow = TextOverflow.Ellipsis)
            Text("${slot.location} · ${stringResource(R.string.slot_going_format, slot.joined, slot.capacity)}", color = LinkUpTextDimmed, fontSize = 12.sp, maxLines = 1, overflow = TextOverflow.Ellipsis)
        }
    }
}

@Composable
fun FrozenSlotCard(
    slot: FrozenSlot,
    onClick: () -> Unit,
    actionLabel: String,
    actionEnabled: Boolean = true,
    onPrimaryAction: () -> Unit = onClick,
) {
    LinkUpCard(modifier = Modifier.fillMaxWidth(), onClick = onClick) {
        Row(verticalAlignment = Alignment.Top) {
            Box(
                Modifier.size(48.dp).clip(RoundedCornerShape(12.dp)).background(LinkUpZone)
                    .border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp)),
                contentAlignment = Alignment.Center,
            ) { Text(slot.emoji, fontSize = 24.sp) }
            Spacer(Modifier.width(12.dp))
            Column(Modifier.weight(1f)) {
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    LinkUpStatusBadge(slot.status)
                    Text(slot.time, color = LinkUpTextMuted, fontFamily = LinkUpDesign.monoFont, fontSize = 11.sp)
                }
                Spacer(Modifier.height(4.dp))
                Text(slot.title, color = LinkUpTextPrimary, fontFamily = LinkUpDesign.displayFont, fontWeight = FontWeight.Bold, fontSize = 16.sp, maxLines = 1, overflow = TextOverflow.Ellipsis)
                Text(slot.location + slot.distance.takeIf { it.isNotBlank() }?.let { " · $it" }.orEmpty(), color = LinkUpTextDimmed, fontSize = 12.sp, maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
        }
        Spacer(Modifier.height(12.dp))
        Text(slot.description, color = LinkUpTextDimmed, fontSize = 14.sp, maxLines = 2, overflow = TextOverflow.Ellipsis)
        Spacer(Modifier.height(12.dp))
        Row(verticalAlignment = Alignment.CenterVertically) {
            LinkUpAvatar(slot.organizer.initials, Color(slot.organizer.color), LinkUpAvatarSize.SM)
            Spacer(Modifier.width(8.dp))
            Text(slot.organizer.name, color = LinkUpTextDimmed, fontSize = 12.sp)
            Spacer(Modifier.weight(1f))
        }
        Spacer(Modifier.height(12.dp))
        LinkUpProgressBar(slot.joined, slot.capacity, showLabel = true)
        Spacer(Modifier.height(12.dp))
        Row(verticalAlignment = Alignment.CenterVertically) {
            Row(Modifier.weight(1f), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                slot.tags.take(3).forEach { tag ->
                    Box(
                        Modifier.clip(RoundedCornerShape(6.dp)).background(LinkUpZone)
                            .border(1.dp, LinkUpBorder, RoundedCornerShape(6.dp)).padding(horizontal = 8.dp, vertical = 2.dp),
                    ) { Text(tag, color = LinkUpTextMuted, fontSize = 10.sp) }
                }
            }
            Box(
                Modifier.clip(RoundedCornerShape(12.dp))
                    .background(if (slot.approval) LinkUpWarning.copy(alpha = .15f) else LinkUpRed)
                    .then(if (slot.approval) Modifier.border(1.dp, LinkUpWarning.copy(alpha = .30f), RoundedCornerShape(12.dp)) else Modifier)
                    .clickable(enabled = actionEnabled, onClick = onPrimaryAction)
                    .padding(horizontal = 14.dp, vertical = 9.dp),
            ) {
                Text(actionLabel, color = if (!actionEnabled) LinkUpTextMuted else if (slot.approval) LinkUpWarning else LinkUpTextPrimary, fontSize = 12.sp, fontWeight = FontWeight.Bold)
            }
        }
    }
}
