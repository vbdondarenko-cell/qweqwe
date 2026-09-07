package com.linkup.app.ui.me

import androidx.compose.foundation.background
import androidx.compose.foundation.border
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
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.BuildConfig
import com.linkup.app.R
import com.linkup.app.core.network.BlockedUser
import com.linkup.app.core.network.UserProfile
import com.linkup.app.core.social.LoadState
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning
import com.linkup.app.ui.theme.LinkUpZone

@Composable
fun MeScreen(
    user: UserProfile,
    blocked: LoadState<List<BlockedUser>>,
    actionError: String?,
    onRefreshBlocks: () -> Unit,
    onUnblock: (String) -> Unit,
    onLogout: () -> Unit,
    onEditProfile: () -> Unit,
    onMySlots: () -> Unit,
) {
    val uriHandler = LocalUriHandler.current
    val privacyUrl = BuildConfig.LINKUP_PRIVACY_URL.trim()
    val termsUrl = BuildConfig.LINKUP_TERMS_URL.trim()
    val privacyConfigured = privacyUrl.startsWith("https://")
    val termsConfigured = termsUrl.startsWith("https://")

    Column(Modifier.fillMaxSize().padding(horizontal = 20.dp, vertical = 18.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
        Text(stringResource(R.string.me_title), color = LinkUpTextPrimary, fontWeight = FontWeight.Black, fontSize = 24.sp)
        Row(
            Modifier.fillMaxWidth().clip(RoundedCornerShape(16.dp)).background(LinkUpElevated).border(1.dp, LinkUpBorder, RoundedCornerShape(16.dp)).padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Box(Modifier.size(52.dp).clip(RoundedCornerShape(16.dp)).background(LinkUpZone), contentAlignment = Alignment.Center) {
                Text(user.displayName.take(1).uppercase(), color = LinkUpTextPrimary, fontWeight = FontWeight.Black, fontSize = 20.sp)
            }
            Spacer(Modifier.size(12.dp))
            Column(modifier = Modifier.weight(1f)) {
                Text(user.displayName, color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, fontSize = 17.sp)
                Text("@${user.username}", color = LinkUpTextMuted, fontSize = 12.sp)
                Text(user.email, color = LinkUpTextDimmed, fontSize = 12.sp)
            }
            Text(user.profileVisibility, color = LinkUpRed, fontSize = 10.sp, fontWeight = FontWeight.Bold)
        }

        TextButton(onClick = onEditProfile) { Text(stringResource(R.string.me_edit_profile), color = LinkUpRed) }
        TextButton(onClick = onMySlots) { Text(stringResource(R.string.me_my_links), color = LinkUpRed) }

        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(stringResource(R.string.me_blocked_people), color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, modifier = Modifier.weight(1f))
            TextButton(onClick = onRefreshBlocks) { Text(stringResource(R.string.common_refresh), color = LinkUpRed) }
        }

        when (blocked) {
            LoadState.Idle -> Text(stringResource(R.string.me_block_list_idle), color = LinkUpTextMuted, fontSize = 12.sp)
            LoadState.Loading -> CircularProgressIndicator(color = LinkUpRed, modifier = Modifier.size(24.dp))
            LoadState.Empty -> Text(stringResource(R.string.me_no_blocked_accounts), color = LinkUpTextMuted, fontSize = 12.sp)
            is LoadState.Failure -> Text(blocked.error.message, color = LinkUpWarning, fontSize = 12.sp)
            is LoadState.Content -> blocked.value.forEach { item ->
                Row(
                    Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp)).background(LinkUpElevated).border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp)).padding(12.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(item.displayName, color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, fontSize = 13.sp)
                        Text("@${item.username}", color = LinkUpTextMuted, fontSize = 11.sp)
                    }
                    TextButton(onClick = { onUnblock(item.id) }) { Text(stringResource(R.string.common_unblock), color = LinkUpRed) }
                }
            }
        }

        actionError?.let { Text(it, color = LinkUpWarning, fontSize = 12.sp) }
        Spacer(Modifier.weight(1f))

        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            TextButton(
                onClick = { uriHandler.openUri(privacyUrl) },
                enabled = privacyConfigured,
                modifier = Modifier.weight(1f),
            ) {
                Text(stringResource(R.string.me_privacy_policy), color = if (privacyConfigured) LinkUpRed else LinkUpTextMuted, fontSize = 11.sp)
            }
            TextButton(
                onClick = { uriHandler.openUri(termsUrl) },
                enabled = termsConfigured,
                modifier = Modifier.weight(1f),
            ) {
                Text(stringResource(R.string.me_terms_of_service), color = if (termsConfigured) LinkUpRed else LinkUpTextMuted, fontSize = 11.sp)
            }
        }
        if (!privacyConfigured || !termsConfigured) {
            Text(stringResource(R.string.me_legal_unavailable), color = LinkUpTextMuted, fontSize = 10.sp)
        }
        Text(stringResource(R.string.me_language_format, user.language.uppercase()), color = LinkUpTextMuted, fontSize = 11.sp)
        Text(stringResource(R.string.me_version_format, BuildConfig.VERSION_NAME), color = LinkUpTextMuted, fontSize = 10.sp)
        Box(
            Modifier.fillMaxWidth().height(48.dp).clip(RoundedCornerShape(12.dp)).background(LinkUpZone).border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp)),
            contentAlignment = Alignment.Center,
        ) {
            TextButton(onClick = onLogout) { Text(stringResource(R.string.common_logout), color = LinkUpRed, fontWeight = FontWeight.Bold) }
        }
    }
}
