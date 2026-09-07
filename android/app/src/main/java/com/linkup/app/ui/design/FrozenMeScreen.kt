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
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
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
import com.linkup.app.ui.theme.LinkUpZone

private val meTabs = listOf("Profile", "Passport", "Settings")
private val cities = listOf(
    Triple("🇺🇦", "Kyiv", "Ukraine"), Triple("🇵🇱", "Warsaw", "Poland"),
    Triple("🇩🇪", "Berlin", "Germany"), Triple("🇨🇿", "Prague", "Czechia"),
)
private val monthValues = listOf("Apr" to 5, "May" to 9, "Jun" to 7, "Jul" to 14, "Aug" to 11, "Sep" to 16)

@Composable
fun FrozenMeScreen(modifier: Modifier = Modifier) {
    var tab by remember { mutableStateOf("Profile") }
    Column(modifier.fillMaxSize().background(Color(0xFF050506))) {
        Column(
            Modifier.fillMaxWidth().background(LinkUpElevated.copy(alpha = .72f)).border(1.dp, LinkUpBorder)
                .padding(start = 20.dp, end = 20.dp, top = 56.dp, bottom = 12.dp),
        ) {
            Text("Me", color = LinkUpTextPrimary, fontFamily = LinkUpDesign.displayFont, fontWeight = FontWeight.Black, fontSize = 24.sp)
            Spacer(Modifier.height(12.dp))
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                meTabs.forEach { item -> LinkUpChip(item, tab == item, { tab = item }) }
            }
        }
        when (tab) {
            "Profile" -> FrozenProfileTab(Modifier.weight(1f))
            "Passport" -> FrozenPassportTab(Modifier.weight(1f))
            else -> FrozenSettingsTab(Modifier.weight(1f))
        }
    }
}

@Composable
private fun FrozenProfileTab(modifier: Modifier = Modifier) {
    LazyColumn(
        modifier = modifier.fillMaxWidth(),
        contentPadding = PaddingValues(start = 20.dp, end = 20.dp, top = 16.dp, bottom = 112.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        item {
            Row(verticalAlignment = Alignment.CenterVertically) {
                LinkUpAvatar("OS", LinkUpRed, LinkUpAvatarSize.XL)
                Spacer(Modifier.width(16.dp))
                Column {
                    Text("Oleh Sitnik", color = LinkUpTextPrimary, fontFamily = LinkUpDesign.displayFont, fontWeight = FontWeight.Bold, fontSize = 20.sp)
                    Text("@oleh", color = LinkUpTextDimmed, fontSize = 14.sp)
                    Spacer(Modifier.height(5.dp))
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        FrozenLineIcon(FrozenIconKind.MAP, LinkUpRed, Modifier.size(12.dp))
                        Spacer(Modifier.width(4.dp))
                        Text("Kyiv · Podil", color = LinkUpTextDimmed, fontSize = 12.sp)
                    }
                }
            }
        }
        item {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                FrozenCityBadge("🇺🇦", "Kyiv")
                FrozenCityBadge("🇵🇱", "Warsaw")
                FrozenCityBadge("🇩🇪", "Berlin")
            }
        }
        item { FrozenReliabilityCard() }
        item { FrozenBumpVault() }
        item { FrozenMyLinkUps() }
    }
}

@Composable
private fun FrozenReliabilityCard() {
    LinkUpCard {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text("↗", color = LinkUpRed, fontSize = 18.sp)
            Spacer(Modifier.width(8.dp))
            Text("Reliability", color = LinkUpTextPrimary, fontWeight = FontWeight.SemiBold, fontSize = 14.sp)
            Spacer(Modifier.weight(1f))
            Text("94%", color = LinkUpTextPrimary, fontFamily = LinkUpDesign.monoFont, fontWeight = FontWeight.Bold, fontSize = 18.sp)
        }
        Spacer(Modifier.height(12.dp))
        LinkUpProgressBar(94, 100, fill = LinkUpRed)
        Spacer(Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            FrozenMetric("✓", "Showed up", "32", LinkUpSuccess, Modifier.weight(1f))
            FrozenMetric("★", "Hosted", "12", LinkUpRed, Modifier.weight(1f))
        }
        Spacer(Modifier.height(8.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            FrozenMetric("♥", "BUMP verified", "18", LinkUpInfo, Modifier.weight(1f))
            FrozenMetric("×", "No-show", "1", LinkUpTextMuted, Modifier.weight(1f))
        }
    }
}

@Composable
private fun FrozenMetric(symbol: String, label: String, value: String, color: Color, modifier: Modifier = Modifier) {
    Row(modifier.clip(RoundedCornerShape(12.dp)).background(LinkUpZone.copy(alpha = .5f)).padding(horizontal = 12.dp, vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) {
        Text(symbol, color = color, fontWeight = FontWeight.Bold, fontSize = 14.sp)
        Spacer(Modifier.width(8.dp))
        Column {
            Text(value, color = LinkUpTextPrimary, fontFamily = LinkUpDesign.monoFont, fontWeight = FontWeight.Bold, fontSize = 14.sp)
            Text(label, color = LinkUpTextMuted, fontSize = 10.sp)
        }
    }
}

@Composable
private fun FrozenBumpVault() {
    LinkUpCard {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text("♥", color = LinkUpRed, fontSize = 18.sp)
            Spacer(Modifier.width(8.dp))
            Text("BUMP Vault", color = LinkUpTextPrimary, fontWeight = FontWeight.SemiBold, fontSize = 14.sp)
            Spacer(Modifier.weight(1f))
            Text("3 verified", color = LinkUpTextMuted, fontFamily = LinkUpDesign.monoFont, fontSize = 10.sp)
        }
        Spacer(Modifier.height(10.dp))
        listOf(
            Triple("Andriy Melnyk", "Morning Espresso Run", "Sep 03"),
            Triple("Sofia Hrytsenko", "Khachapuri & Wine Night", "Aug 29"),
            Triple("Olena Kovalenko", "Golden Hour Photo Walk", "Aug 21"),
        ).forEach { row ->
            Row(Modifier.fillMaxWidth().padding(vertical = 6.dp), verticalAlignment = Alignment.CenterVertically) {
                Box(Modifier.size(32.dp).clip(RoundedCornerShape(8.dp)).background(LinkUpRed.copy(alpha = .10f)).border(1.dp, LinkUpRed.copy(alpha = .20f), RoundedCornerShape(8.dp)), contentAlignment = Alignment.Center) {
                    Text("♥", color = LinkUpRed, fontSize = 14.sp)
                }
                Spacer(Modifier.width(12.dp))
                Column(Modifier.weight(1f)) {
                    Text(row.first, color = LinkUpTextPrimary, fontWeight = FontWeight.SemiBold, fontSize = 14.sp)
                    Text(row.second, color = LinkUpTextMuted, fontSize = 12.sp)
                }
                Column(horizontalAlignment = Alignment.End) {
                    Text(row.third, color = LinkUpTextMuted, fontFamily = LinkUpDesign.monoFont, fontSize = 10.sp)
                    Text("✓", color = LinkUpSuccess, fontSize = 12.sp)
                }
            }
        }
    }
}

@Composable
private fun FrozenMyLinkUps() {
    LinkUpCard {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text("★", color = LinkUpRed, fontSize = 16.sp)
            Spacer(Modifier.width(8.dp))
            Text("My LinkUps", color = LinkUpTextPrimary, fontWeight = FontWeight.SemiBold, fontSize = 14.sp)
        }
        Spacer(Modifier.height(6.dp))
        listOf(
            Triple("Morning Espresso Run", "LIVE", "4/6"),
            Triple("Cocktail Crawl · 3 Bars", "OPEN", "6/8"),
            Triple("Evening Riverside Run", "OPEN", "7/12"),
        ).forEach { row ->
            Row(Modifier.fillMaxWidth().padding(vertical = 9.dp), verticalAlignment = Alignment.CenterVertically) {
                Text(row.first, color = LinkUpTextDimmed, fontSize = 14.sp, modifier = Modifier.weight(1f))
                Text("${row.second} · ${row.third}", color = if (row.second == "LIVE") LinkUpSuccess else LinkUpInfo, fontFamily = LinkUpDesign.monoFont, fontSize = 10.sp)
            }
        }
    }
}

@Composable
private fun FrozenPassportTab(modifier: Modifier = Modifier) {
    LazyColumn(
        modifier = modifier.fillMaxWidth(),
        contentPadding = PaddingValues(start = 20.dp, end = 20.dp, top = 16.dp, bottom = 112.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        item {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                FrozenMetricCard("42", "Meetups", Modifier.weight(1f))
                FrozenMetricCard("4", "Cities", Modifier.weight(1f))
                FrozenMetricCard("12", "Hosted", Modifier.weight(1f))
            }
        }
        item {
            LinkUpCard {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text("◎", color = LinkUpRed, fontSize = 16.sp)
                    Spacer(Modifier.width(8.dp))
                    Text("Cities Explored", color = LinkUpTextPrimary, fontWeight = FontWeight.SemiBold, fontSize = 14.sp)
                }
                Spacer(Modifier.height(8.dp))
                cities.forEachIndexed { index, city ->
                    Row(Modifier.fillMaxWidth().padding(vertical = 7.dp), verticalAlignment = Alignment.CenterVertically) {
                        Text(city.first, fontSize = 20.sp)
                        Spacer(Modifier.width(12.dp))
                        Column(Modifier.weight(1f)) {
                            Text(city.second, color = LinkUpTextPrimary, fontWeight = FontWeight.SemiBold, fontSize = 14.sp)
                            Text(city.third, color = LinkUpTextMuted, fontSize = 12.sp)
                        }
                        Text("${listOf(18,5,3,2)[index]} visits", color = LinkUpTextDimmed, fontFamily = LinkUpDesign.monoFont, fontSize = 12.sp)
                    }
                }
            }
        }
        item {
            LinkUpCard {
                Text("Interests", color = LinkUpTextPrimary, fontWeight = FontWeight.SemiBold, fontSize = 14.sp)
                Spacer(Modifier.height(12.dp))
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        LinkUpChip("Coffee", true, {})
                        LinkUpChip("Running", true, {})
                        LinkUpChip("Photography", true, {})
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        LinkUpChip("Food", true, {})
                        LinkUpChip("Travel", true, {})
                    }
                }
            }
        }
        item { FrozenActivityChart() }
    }
}

@Composable
private fun FrozenMetricCard(value: String, label: String, modifier: Modifier = Modifier) {
    Column(
        modifier.clip(RoundedCornerShape(16.dp)).background(LinkUpElevated).border(1.dp, LinkUpBorder, RoundedCornerShape(16.dp)).padding(16.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(value, color = LinkUpTextPrimary, fontFamily = LinkUpDesign.displayFont, fontWeight = FontWeight.Black, fontSize = 24.sp)
        Text(label, color = LinkUpTextMuted, fontSize = 12.sp)
    }
}

@Composable
private fun FrozenActivityChart() {
    LinkUpCard {
        Text("Activity · 6 months", color = LinkUpTextPrimary, fontWeight = FontWeight.SemiBold, fontSize = 14.sp)
        Spacer(Modifier.height(14.dp))
        Row(Modifier.fillMaxWidth().height(132.dp), horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.Bottom) {
            val max = monthValues.maxOf { it.second }
            monthValues.forEach { (month, value) ->
                Column(Modifier.weight(1f), horizontalAlignment = Alignment.CenterHorizontally) {
                    Box(Modifier.width(28.dp).height((92f * value / max).dp).clip(RoundedCornerShape(topStart = 8.dp, topEnd = 8.dp)).background(LinkUpRed))
                    Spacer(Modifier.height(6.dp))
                    Text(month, color = LinkUpTextMuted, fontFamily = LinkUpDesign.monoFont, fontSize = 10.sp)
                    Text(value.toString(), color = LinkUpTextDimmed, fontFamily = LinkUpDesign.monoFont, fontWeight = FontWeight.Bold, fontSize = 10.sp)
                }
            }
        }
    }
}

@Composable
private fun FrozenSettingsTab(modifier: Modifier = Modifier) {
    val sections = listOf(
        "PRIVACY & SAFETY" to listOf(
            FrozenSetting("◈", "Privacy Center"), FrozenSetting("◈", "Safety Center"),
            FrozenSetting("♥", "Guardian", badge = "NEW"), FrozenSetting("◌", "Ghost Mode", toggle = true),
        ),
        "ACCOUNT" to listOf(
            FrozenSetting("◉", "Notifications"), FrozenSetting("◎", "Language", value = "English"),
            FrozenSetting("♿", "Accessibility"), FrozenSetting("▣", "Data & Privacy"),
        ),
        "LINKUP+" to listOf(
            FrozenSetting("♛", "Upgrade to LinkUp+", highlight = true), FrozenSetting("◆", "Rewarded Free Day", badge = "FREE"),
        ),
        "APP" to listOf(
            FrozenSetting("◐", "Themes", value = "OLED Dark"), FrozenSetting("▤", "Legal"),
            FrozenSetting("▤", "Version", value = "1.0.0-prototype"), FrozenSetting("↪", "Log out", danger = true),
        ),
    )
    LazyColumn(
        modifier = modifier.fillMaxWidth(),
        contentPadding = PaddingValues(start = 20.dp, end = 20.dp, top = 16.dp, bottom = 112.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        items(sections) { section ->
            Column {
                Text(section.first, color = LinkUpTextMuted, fontFamily = LinkUpDesign.monoFont, fontWeight = FontWeight.SemiBold, fontSize = 10.sp, modifier = Modifier.padding(start = 4.dp, bottom = 6.dp))
                Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(16.dp)).background(LinkUpElevated).border(1.dp, LinkUpBorder, RoundedCornerShape(16.dp))) {
                    section.second.forEachIndexed { index, item -> FrozenSettingRow(item, index < section.second.lastIndex) }
                }
            }
        }
    }
}

private data class FrozenSetting(
    val icon: String,
    val label: String,
    val value: String? = null,
    val badge: String? = null,
    val toggle: Boolean = false,
    val highlight: Boolean = false,
    val danger: Boolean = false,
)

@Composable
private fun FrozenSettingRow(item: FrozenSetting, divider: Boolean) {
    val accent = when { item.danger -> com.linkup.app.ui.theme.LinkUpCritical; item.highlight -> LinkUpRed; else -> LinkUpTextMuted }
    Row(
        Modifier.fillMaxWidth().then(if (divider) Modifier.border(width = 0.dp, color = Color.Transparent) else Modifier)
            .padding(horizontal = 16.dp, vertical = 13.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(item.icon, color = accent, fontSize = 18.sp)
        Spacer(Modifier.width(12.dp))
        Text(item.label, color = when { item.danger -> com.linkup.app.ui.theme.LinkUpCritical; item.highlight -> LinkUpRed; else -> LinkUpTextPrimary }, fontSize = 14.sp, modifier = Modifier.weight(1f))
        item.badge?.let {
            Box(Modifier.clip(RoundedCornerShape(6.dp)).background(LinkUpRed.copy(alpha = .15f)).border(1.dp, LinkUpRed.copy(alpha = .30f), RoundedCornerShape(6.dp)).padding(horizontal = 6.dp, vertical = 2.dp)) {
                Text(it, color = LinkUpRed, fontFamily = LinkUpDesign.monoFont, fontWeight = FontWeight.Bold, fontSize = 9.sp)
            }
        }
        item.value?.let { Text(it, color = LinkUpTextMuted, fontSize = 12.sp) }
        if (item.toggle) {
            Box(Modifier.width(36.dp).height(20.dp).clip(RoundedCornerShape(99.dp)).background(LinkUpRed).padding(2.dp), contentAlignment = Alignment.CenterEnd) {
                Box(Modifier.size(16.dp).clip(CircleShape).background(LinkUpTextPrimary))
            }
        }
        if (!item.toggle && item.value == null && item.badge == null) Text("›", color = LinkUpTextMuted, fontSize = 18.sp)
    }
}

@Composable
private fun FrozenCityBadge(flag: String, city: String) {
    Row(
        Modifier.clip(RoundedCornerShape(8.dp)).background(LinkUpElevated).border(1.dp, LinkUpBorder, RoundedCornerShape(8.dp)).padding(horizontal = 10.dp, vertical = 5.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(flag, fontSize = 12.sp)
        Spacer(Modifier.width(4.dp))
        Text(city, color = LinkUpTextDimmed, fontSize = 12.sp)
    }
}
