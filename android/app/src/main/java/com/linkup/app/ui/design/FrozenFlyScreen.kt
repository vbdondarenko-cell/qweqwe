package com.linkup.app.ui.design

import androidx.compose.foundation.background
import androidx.compose.foundation.border
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
import androidx.compose.foundation.shape.RoundedCornerShape
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpCritical
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpSuccess
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpZone

private val flyTabs = listOf("Now", "Travel", "Motion")

@Composable
fun FrozenFlyScreen(modifier: Modifier = Modifier) {
    var tab by remember { mutableStateOf("Now") }
    Column(modifier.fillMaxSize().background(Color(0xFF050506))) {
        Column(
            Modifier.fillMaxWidth().background(LinkUpElevated.copy(alpha = .72f)).border(1.dp, LinkUpBorder)
                .padding(start = 20.dp, end = 20.dp, top = 56.dp, bottom = 12.dp),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                FrozenLineIcon(FrozenIconKind.FLY, LinkUpRed, Modifier.size(20.dp))
                Spacer(Modifier.width(8.dp))
                Text("Fly Now", color = LinkUpTextPrimary, fontFamily = LinkUpDesign.displayFont, fontWeight = FontWeight.Black, fontSize = 24.sp)
                Spacer(Modifier.weight(1f))
                Box(
                    Modifier.clip(RoundedCornerShape(8.dp)).background(LinkUpRed.copy(alpha = .15f))
                        .border(1.dp, LinkUpRed.copy(alpha = .30f), RoundedCornerShape(8.dp)).padding(horizontal = 10.dp, vertical = 4.dp),
                ) { Text("3 LIVE", color = LinkUpRed, fontFamily = LinkUpDesign.monoFont, fontWeight = FontWeight.Bold, fontSize = 10.sp) }
            }
            Spacer(Modifier.height(8.dp))
            Text("Flash drops · Kyiv · right now", color = LinkUpTextDimmed, fontSize = 14.sp)
            Spacer(Modifier.height(12.dp))
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                FrozenCounter("Joined", "0", LinkUpSuccess, Modifier.weight(1f))
                FrozenCounter("Passed", "0", LinkUpTextMuted, Modifier.weight(1f))
                FrozenCounter("Available", "3", LinkUpRed, Modifier.weight(1f))
            }
        }
        LazyRow(
            contentPadding = PaddingValues(horizontal = 20.dp, vertical = 12.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            items(flyTabs) { item -> LinkUpChip(item, tab == item, { tab = item }) }
        }
        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(start = 20.dp, end = 20.dp, bottom = 112.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            items(FrozenFlashDrops) { drop -> FrozenFlashCard(drop) }
        }
    }
}

@Composable
private fun FrozenCounter(label: String, value: String, color: Color, modifier: Modifier = Modifier) {
    Column(
        modifier.clip(RoundedCornerShape(12.dp)).background(LinkUpElevated)
            .border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp)).padding(horizontal = 12.dp, vertical = 8.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(value, color = color, fontFamily = LinkUpDesign.monoFont, fontWeight = FontWeight.Bold, fontSize = 20.sp)
        Text(label, color = LinkUpTextMuted, fontSize = 10.sp)
    }
}

@Composable
private fun FrozenFlashCard(drop: FrozenFlashDrop) {
    Column(
        Modifier.fillMaxWidth().clip(RoundedCornerShape(16.dp)).background(LinkUpElevated)
            .border(1.dp, LinkUpBorder, RoundedCornerShape(16.dp)).padding(16.dp),
    ) {
        Row(verticalAlignment = Alignment.Top) {
            Box(
                Modifier.size(48.dp).clip(RoundedCornerShape(12.dp)).background(LinkUpZone)
                    .border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp)),
                contentAlignment = Alignment.Center,
            ) { Text(drop.emoji, fontSize = 24.sp) }
            Spacer(Modifier.width(12.dp))
            Column(Modifier.weight(1f)) {
                Text(drop.title, color = LinkUpTextPrimary, fontFamily = LinkUpDesign.displayFont, fontWeight = FontWeight.Bold, fontSize = 16.sp, maxLines = 1, overflow = TextOverflow.Ellipsis)
                Text("${drop.location} · ${drop.distance}", color = LinkUpTextDimmed, fontSize = 12.sp, maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
            Box(
                Modifier.clip(RoundedCornerShape(12.dp)).background(if (drop.urgent) LinkUpCritical.copy(alpha = .15f) else LinkUpZone)
                    .border(1.dp, if (drop.urgent) LinkUpCritical.copy(alpha = .30f) else LinkUpBorder, RoundedCornerShape(12.dp))
                    .padding(horizontal = 10.dp, vertical = 6.dp),
            ) {
                Text(drop.time, color = if (drop.urgent) LinkUpCritical else LinkUpTextDimmed, fontFamily = LinkUpDesign.monoFont, fontWeight = FontWeight.Bold, fontSize = 14.sp)
            }
        }
        Spacer(Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
            drop.tags.forEach { tag ->
                Box(
                    Modifier.clip(RoundedCornerShape(6.dp)).background(LinkUpZone)
                        .border(1.dp, LinkUpBorder, RoundedCornerShape(6.dp)).padding(horizontal = 8.dp, vertical = 2.dp),
                ) { Text(tag, color = LinkUpTextMuted, fontSize = 10.sp) }
            }
        }
        Spacer(Modifier.height(12.dp))
        LinkUpProgressBar(drop.joined, drop.capacity, showLabel = true)
        Spacer(Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            LinkUpButton("×  Pass", {}, Modifier.weight(1f), LinkUpButtonVariant.SECONDARY)
            LinkUpButton("↗  Join", {}, Modifier.weight(1f), LinkUpButtonVariant.PRIMARY)
        }
    }
}
