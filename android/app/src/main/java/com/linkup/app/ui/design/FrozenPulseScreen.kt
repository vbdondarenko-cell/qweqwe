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
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
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

private val timeFilters = listOf("Now", "Tonight", "Tomorrow", "All")
private val categoryFilters = listOf("All", "Social", "Active", "Food")

@Composable
fun FrozenPulseScreen(
    modifier: Modifier = Modifier,
    onOpenNotifications: () -> Unit = {},
    onSlotClick: (FrozenSlot) -> Unit = {},
) {
    var query by remember { mutableStateOf("") }
    var time by remember { mutableStateOf("Now") }
    var category by remember { mutableStateOf("All") }
    val filtered = FrozenPulseSlots.filter { slot ->
        query.isBlank() || listOf(slot.title, slot.location, slot.description).any { it.contains(query, ignoreCase = true) } ||
            slot.tags.any { it.contains(query, ignoreCase = true) }
    }

    Column(modifier.fillMaxSize().background(Color(0xFF050506))) {
        FrozenPulseHeader(query, { query = it }, onOpenNotifications)
        LazyRow(
            contentPadding = PaddingValues(horizontal = 20.dp, vertical = 10.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            items(timeFilters) { item -> LinkUpChip(item, time == item, { time = item }) }
        }
        LazyRow(
            contentPadding = PaddingValues(horizontal = 20.dp, vertical = 0.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            items(categoryFilters) { item -> LinkUpChip(item, category == item, { category = item }) }
        }
        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(start = 20.dp, end = 20.dp, top = 10.dp, bottom = 112.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            if (time == "Now" && FrozenPulseSlots.isNotEmpty()) {
                item { FrozenHappeningNow(FrozenPulseSlots.first(), onSlotClick) }
            }
            if (filtered.isEmpty()) {
                item { LinkUpEmptyState("No matches", "Try a different filter or search term.") }
            } else {
                items(filtered) { slot -> FrozenSlotCard(slot = slot, onClick = { onSlotClick(slot) }) }
            }
        }
    }
}

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
                    Text("Kyiv · Podil", color = LinkUpTextPrimary, fontFamily = LinkUpDesign.bodyFont, fontWeight = FontWeight.SemiBold, fontSize = 14.sp)
                }
                Spacer(Modifier.height(5.dp))
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Box(Modifier.size(8.dp).clip(CircleShape).background(LinkUpRed))
                    Text("City BPM ", color = LinkUpTextDimmed, fontFamily = LinkUpDesign.monoFont, fontSize = 12.sp)
                    Text("87", color = LinkUpRed, fontFamily = LinkUpDesign.monoFont, fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
                    Text("· High activity", color = LinkUpTextMuted, fontSize = 10.sp)
                }
            }
            Box(
                Modifier.size(40.dp).clip(RoundedCornerShape(12.dp)).background(LinkUpElevated)
                    .border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp)).clickable(onClick = onOpenNotifications),
                contentAlignment = Alignment.Center,
            ) {
                FrozenLineIcon(FrozenIconKind.BELL, LinkUpTextDimmed, Modifier.size(18.dp))
                Box(
                    Modifier.align(Alignment.TopEnd).size(20.dp).clip(CircleShape).background(LinkUpRed),
                    contentAlignment = Alignment.Center,
                ) { Text("3", color = LinkUpTextPrimary, fontFamily = LinkUpDesign.monoFont, fontWeight = FontWeight.Bold, fontSize = 10.sp) }
            }
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
                decorationBox = { inner -> if (query.isBlank()) Text("Search activities, places...", color = LinkUpTextMuted, fontSize = 14.sp) else inner() },
            )
        }
    }
}

@Composable
private fun FrozenHappeningNow(slot: FrozenSlot, onSlotClick: (FrozenSlot) -> Unit) {
    Row(
        Modifier.fillMaxWidth().clip(RoundedCornerShape(16.dp)).background(LinkUpSuccess.copy(alpha = .10f))
            .border(1.dp, LinkUpSuccess.copy(alpha = .25f), RoundedCornerShape(16.dp))
            .clickable { onSlotClick(slot) }.padding(16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(Modifier.size(40.dp).clip(RoundedCornerShape(12.dp)).background(LinkUpSuccess.copy(alpha = .20f)), contentAlignment = Alignment.Center) {
            FrozenLineIcon(FrozenIconKind.PULSE, LinkUpSuccess, Modifier.size(18.dp))
        }
        Spacer(Modifier.width(12.dp))
        Column(Modifier.weight(1f)) {
            Text("HAPPENING NOW", color = LinkUpSuccess, fontFamily = LinkUpDesign.monoFont, fontWeight = FontWeight.SemiBold, fontSize = 10.sp)
            Text(slot.title, color = LinkUpTextPrimary, fontFamily = LinkUpDesign.displayFont, fontWeight = FontWeight.Bold, fontSize = 14.sp, maxLines = 1, overflow = TextOverflow.Ellipsis)
            Text("${slot.location} · ${slot.joined}/${slot.capacity} going", color = LinkUpTextDimmed, fontSize = 12.sp, maxLines = 1, overflow = TextOverflow.Ellipsis)
        }
    }
}

@Composable
fun FrozenSlotCard(slot: FrozenSlot, onClick: () -> Unit = {}) {
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
                Text("${slot.location} · ${slot.distance}", color = LinkUpTextDimmed, fontSize = 12.sp, maxLines = 1, overflow = TextOverflow.Ellipsis)
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
            Text("${slot.organizer.reliability}% reliable", color = LinkUpTextMuted, fontFamily = LinkUpDesign.monoFont, fontSize = 10.sp)
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
                    .padding(horizontal = 14.dp, vertical = 9.dp),
            ) {
                Text(if (slot.approval) "Request to join" else "Join now", color = if (slot.approval) LinkUpWarning else LinkUpTextPrimary, fontSize = 12.sp, fontWeight = FontWeight.Bold)
            }
        }
    }
}
