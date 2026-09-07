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
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
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
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.BuildConfig
import com.linkup.app.core.network.BlockedUser
import com.linkup.app.core.network.UserProfile
import com.linkup.app.core.social.LoadState
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpCritical
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning
import com.linkup.app.ui.theme.LinkUpZone

private val meTabs = listOf("Profile", "Passport", "Settings")

@Composable
fun FrozenMeScreen(
    user: UserProfile,
    blocked: LoadState<List<BlockedUser>>,
    actionError: String?,
    onRefreshBlocks: () -> Unit,
    onUnblock: (String) -> Unit,
    onEditProfile: () -> Unit,
    onMySlots: () -> Unit,
    onLogout: () -> Unit,
    modifier: Modifier = Modifier,
) {
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
            "Profile" -> FrozenProfileTab(user, onEditProfile, onMySlots, Modifier.weight(1f))
            "Passport" -> FrozenPassportGate(Modifier.weight(1f))
            else -> FrozenSettingsTab(
                user = user,
                blocked = blocked,
                actionError = actionError,
                onRefreshBlocks = onRefreshBlocks,
                onUnblock = onUnblock,
                onEditProfile = onEditProfile,
                onMySlots = onMySlots,
                onLogout = onLogout,
                modifier = Modifier.weight(1f),
            )
        }
    }
}

@Composable
private fun FrozenProfileTab(
    user: UserProfile,
    onEditProfile: () -> Unit,
    onMySlots: () -> Unit,
    modifier: Modifier = Modifier,
) {
    LazyColumn(
        modifier = modifier.fillMaxWidth(),
        contentPadding = PaddingValues(start = 20.dp, end = 20.dp, top = 16.dp, bottom = 112.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        item {
            LinkUpCard {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    val initials = user.displayName.trim().split(Regex("\\s+")).filter { it.isNotBlank() }
                        .take(2).joinToString("") { it.take(1).uppercase() }
                        .ifBlank { user.username.take(2).uppercase() }
                    LinkUpAvatar(initials, LinkUpRed, LinkUpAvatarSize.XL)
                    Spacer(Modifier.width(16.dp))
                    Column(Modifier.weight(1f)) {
                        Text(user.displayName, color = LinkUpTextPrimary, fontFamily = LinkUpDesign.displayFont, fontWeight = FontWeight.Bold, fontSize = 20.sp)
                        Text("@${user.username}", color = LinkUpTextDimmed, fontSize = 14.sp)
                        Spacer(Modifier.height(4.dp))
                        Text(user.email, color = LinkUpTextMuted, fontSize = 11.sp)
                    }
                    LinkUpButton("Edit", onEditProfile, variant = LinkUpButtonVariant.SECONDARY, size = LinkUpButtonSize.SM)
                }
                Spacer(Modifier.height(14.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    FrozenProfileFact("Visibility", user.profileVisibility, Modifier.weight(1f))
                    FrozenProfileFact("Language", user.language.uppercase(), Modifier.weight(1f))
                }
            }
        }
        item {
            LinkUpCard(onClick = onMySlots) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text("★", color = LinkUpRed, fontSize = 18.sp)
                    Spacer(Modifier.width(10.dp))
                    Column(Modifier.weight(1f)) {
                        Text("My LinkUps", color = LinkUpTextPrimary, fontWeight = FontWeight.SemiBold, fontSize = 14.sp)
                        Text("Hosting · Joined · Requests", color = LinkUpTextMuted, fontSize = 12.sp)
                    }
                    Text("›", color = LinkUpTextMuted, fontSize = 20.sp)
                }
            }
        }
        item {
            LinkUpCard {
                Text("v1.0 profile", color = LinkUpRed, fontFamily = LinkUpDesign.monoFont, fontWeight = FontWeight.Bold, fontSize = 10.sp)
                Spacer(Modifier.height(6.dp))
                Text(
                    "Reliability, BUMP Vault, Social Passport and city-history metrics are not shown until their server-authoritative v1.1 capability is active.",
                    color = LinkUpTextDimmed,
                    fontSize = 12.sp,
                )
            }
        }
    }
}

@Composable
private fun FrozenProfileFact(label: String, value: String, modifier: Modifier = Modifier) {
    Column(
        modifier.clip(RoundedCornerShape(12.dp)).background(LinkUpZone.copy(alpha = .55f))
            .border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp)).padding(horizontal = 12.dp, vertical = 10.dp),
    ) {
        Text(label, color = LinkUpTextMuted, fontFamily = LinkUpDesign.monoFont, fontSize = 9.sp)
        Spacer(Modifier.height(3.dp))
        Text(value, color = LinkUpTextPrimary, fontWeight = FontWeight.SemiBold, fontSize = 13.sp)
    }
}

@Composable
private fun FrozenPassportGate(modifier: Modifier = Modifier) {
    Box(modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 16.dp), contentAlignment = Alignment.TopCenter) {
        LinkUpCard(modifier = Modifier.fillMaxWidth()) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                FrozenLineIcon(FrozenIconKind.ME, LinkUpRed, Modifier.size(20.dp))
                Spacer(Modifier.width(10.dp))
                Text("Social Passport", color = LinkUpTextPrimary, fontFamily = LinkUpDesign.displayFont, fontWeight = FontWeight.Bold, fontSize = 18.sp)
            }
            Spacer(Modifier.height(12.dp))
            Text("Planned for LinkUp v1.1", color = LinkUpRed, fontFamily = LinkUpDesign.monoFont, fontWeight = FontWeight.Bold, fontSize = 11.sp)
            Spacer(Modifier.height(8.dp))
            Text(
                "This build intentionally does not fabricate meetups, cities, reliability, BUMP history or activity metrics. The screen becomes active only when those values are backed by canonical server data.",
                color = LinkUpTextDimmed,
                fontSize = 13.sp,
            )
        }
    }
}

@Composable
private fun FrozenSettingsTab(
    user: UserProfile,
    blocked: LoadState<List<BlockedUser>>,
    actionError: String?,
    onRefreshBlocks: () -> Unit,
    onUnblock: (String) -> Unit,
    onEditProfile: () -> Unit,
    onMySlots: () -> Unit,
    onLogout: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val uriHandler = LocalUriHandler.current
    val privacyUrl = BuildConfig.LINKUP_PRIVACY_URL.trim()
    val termsUrl = BuildConfig.LINKUP_TERMS_URL.trim()
    val privacyConfigured = privacyUrl.startsWith("https://")
    val termsConfigured = termsUrl.startsWith("https://")

    LazyColumn(
        modifier = modifier.fillMaxWidth(),
        contentPadding = PaddingValues(start = 20.dp, end = 20.dp, top = 16.dp, bottom = 112.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        item {
            FrozenSettingsSection("ACCOUNT") {
                FrozenSettingRow("◉", "Edit profile", value = "@${user.username}", onClick = onEditProfile)
                FrozenSettingRow("★", "My LinkUps", onClick = onMySlots)
                FrozenSettingRow("◎", "Language", value = user.language.uppercase())
                FrozenSettingRow("◌", "Profile visibility", value = user.profileVisibility)
            }
        }
        item {
            FrozenSettingsSection("BLOCKED PEOPLE") {
                Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 10.dp), verticalAlignment = Alignment.CenterVertically) {
                    Text("Server block list", color = LinkUpTextDimmed, fontSize = 12.sp, modifier = Modifier.weight(1f))
                    Text("Refresh", color = LinkUpRed, fontWeight = FontWeight.SemiBold, fontSize = 12.sp, modifier = Modifier.clickable(onClick = onRefreshBlocks))
                }
                when (blocked) {
                    LoadState.Idle -> Text("Open Settings to load the block list.", color = LinkUpTextMuted, fontSize = 12.sp, modifier = Modifier.padding(horizontal = 16.dp, vertical = 10.dp))
                    LoadState.Loading -> Box(Modifier.fillMaxWidth().padding(16.dp), contentAlignment = Alignment.Center) { CircularProgressIndicator(color = LinkUpRed, modifier = Modifier.size(24.dp)) }
                    LoadState.Empty -> Text("No blocked accounts.", color = LinkUpTextMuted, fontSize = 12.sp, modifier = Modifier.padding(horizontal = 16.dp, vertical = 10.dp))
                    is LoadState.Failure -> Text(blocked.error.message, color = LinkUpWarning, fontSize = 12.sp, modifier = Modifier.padding(horizontal = 16.dp, vertical = 10.dp))
                    is LoadState.Content -> blocked.value.forEach { item -> FrozenBlockedUserRow(item, onUnblock) }
                }
                actionError?.let { Text(it, color = LinkUpWarning, fontSize = 12.sp, modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp)) }
            }
        }
        item {
            FrozenSettingsSection("LEGAL & APP") {
                FrozenSettingRow(
                    icon = "◈",
                    label = "Privacy Policy",
                    value = if (privacyConfigured) null else "Not configured",
                    enabled = privacyConfigured,
                    onClick = if (privacyConfigured) ({ uriHandler.openUri(privacyUrl) }) else null,
                )
                FrozenSettingRow(
                    icon = "▤",
                    label = "Terms of Service",
                    value = if (termsConfigured) null else "Not configured",
                    enabled = termsConfigured,
                    onClick = if (termsConfigured) ({ uriHandler.openUri(termsUrl) }) else null,
                )
                FrozenSettingRow("◇", "Version", value = BuildConfig.VERSION_NAME)
            }
        }
        item {
            FrozenSettingsSection("SESSION") {
                FrozenSettingRow("↪", "Log out", danger = true, onClick = onLogout)
            }
        }
    }
}

@Composable
private fun FrozenSettingsSection(title: String, content: @Composable () -> Unit) {
    Column {
        Text(title, color = LinkUpTextMuted, fontFamily = LinkUpDesign.monoFont, fontWeight = FontWeight.SemiBold, fontSize = 10.sp, modifier = Modifier.padding(start = 4.dp, bottom = 6.dp))
        Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(16.dp)).background(LinkUpElevated).border(1.dp, LinkUpBorder, RoundedCornerShape(16.dp))) {
            content()
        }
    }
}

@Composable
private fun FrozenSettingRow(
    icon: String,
    label: String,
    value: String? = null,
    enabled: Boolean = true,
    danger: Boolean = false,
    onClick: (() -> Unit)? = null,
) {
    val labelColor = when {
        !enabled -> LinkUpTextMuted
        danger -> LinkUpCritical
        else -> LinkUpTextPrimary
    }
    Row(
        Modifier.fillMaxWidth()
            .then(if (enabled && onClick != null) Modifier.clickable(onClick = onClick) else Modifier)
            .padding(horizontal = 16.dp, vertical = 13.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(icon, color = if (danger) LinkUpCritical else LinkUpTextMuted, fontSize = 18.sp)
        Spacer(Modifier.width(12.dp))
        Text(label, color = labelColor, fontSize = 14.sp, modifier = Modifier.weight(1f))
        value?.let { Text(it, color = LinkUpTextMuted, fontSize = 11.sp) }
        if (enabled && onClick != null) Text("  ›", color = LinkUpTextMuted, fontSize = 18.sp)
    }
}

@Composable
private fun FrozenBlockedUserRow(item: BlockedUser, onUnblock: (String) -> Unit) {
    Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 10.dp), verticalAlignment = Alignment.CenterVertically) {
        val initials = item.displayName.trim().split(Regex("\\s+")).filter { it.isNotBlank() }
            .take(2).joinToString("") { it.take(1).uppercase() }
            .ifBlank { item.username.take(2).uppercase() }
        LinkUpAvatar(initials, LinkUpZone, LinkUpAvatarSize.SM)
        Spacer(Modifier.width(10.dp))
        Column(Modifier.weight(1f)) {
            Text(item.displayName, color = LinkUpTextPrimary, fontWeight = FontWeight.SemiBold, fontSize = 13.sp)
            Text("@${item.username}", color = LinkUpTextMuted, fontSize = 11.sp)
        }
        Text("Unblock", color = LinkUpRed, fontWeight = FontWeight.SemiBold, fontSize = 12.sp, modifier = Modifier.clickable { onUnblock(item.id) })
    }
}
