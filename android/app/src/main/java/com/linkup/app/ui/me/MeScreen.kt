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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
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
    Column(Modifier.fillMaxSize().padding(horizontal = 20.dp, vertical = 18.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
        Text("Me", color = LinkUpTextPrimary, fontWeight = FontWeight.Black, fontSize = 24.sp)
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

        TextButton(onClick = onEditProfile) { Text("Edit profile", color = LinkUpRed) }

        TextButton(onClick = onMySlots) { Text("My LINKs", color = LinkUpRed) }

        Row(verticalAlignment = Alignment.CenterVertically) {
            Text("Blocked people", color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, modifier = Modifier.weight(1f))
            TextButton(onClick = onRefreshBlocks) { Text("Refresh", color = LinkUpRed) }
        }

        when (blocked) {
            LoadState.Idle -> Text("Open this section to load your server block list.", color = LinkUpTextMuted, fontSize = 12.sp)
            LoadState.Loading -> CircularProgressIndicator(color = LinkUpRed, modifier = Modifier.size(24.dp))
            LoadState.Empty -> Text("No blocked accounts", color = LinkUpTextMuted, fontSize = 12.sp)
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
                    TextButton(onClick = { onUnblock(item.id) }) { Text("Unblock", color = LinkUpRed) }
                }
            }
        }

        actionError?.let { Text(it, color = LinkUpWarning, fontSize = 12.sp) }
        Spacer(Modifier.weight(1f))
        Text("Language: ${user.language.uppercase()}", color = LinkUpTextMuted, fontSize = 11.sp)
        Text("LinkUp 1.0.0-dev", color = LinkUpTextMuted, fontSize = 10.sp)
        Box(
            Modifier.fillMaxWidth().height(48.dp).clip(RoundedCornerShape(12.dp)).background(LinkUpZone).border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp)),
            contentAlignment = Alignment.Center,
        ) {
            TextButton(onClick = onLogout) { Text("Logout", color = LinkUpRed, fontWeight = FontWeight.Bold) }
        }
    }
}
