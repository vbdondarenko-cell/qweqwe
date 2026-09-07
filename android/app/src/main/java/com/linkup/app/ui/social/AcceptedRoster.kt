package com.linkup.app.ui.social

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
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
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.core.network.SlotOrganizer
import com.linkup.app.core.social.LoadState
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning

@Composable
internal fun AcceptedRoster(
    state: LoadState<List<SlotOrganizer>>,
    busy: Boolean,
    onRefresh: () -> Unit,
    onBlockUser: (SlotOrganizer) -> Unit,
) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Text("Accepted participants", color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, modifier = Modifier.weight(1f))
        TextButton(onClick = onRefresh, enabled = !busy && state !is LoadState.Loading) {
            Text(if (state is LoadState.Idle) "Show" else "Refresh", color = LinkUpRed)
        }
    }
    when (state) {
        LoadState.Idle -> Unit
        LoadState.Loading -> CircularProgressIndicator(color = LinkUpRed, modifier = Modifier.size(24.dp))
        LoadState.Empty -> Text("No accepted participants", color = LinkUpTextMuted, fontSize = 12.sp)
        is LoadState.Failure -> {
            Text(state.error.message, color = LinkUpWarning, fontSize = 12.sp)
            TextButton(onClick = onRefresh, enabled = !busy) { Text("Retry", color = LinkUpRed) }
        }
        is LoadState.Content -> state.value.forEach { participant ->
            Row(
                Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp)).background(LinkUpElevated)
                    .border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp)).padding(12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(Modifier.weight(1f)) {
                    Text(participant.displayName, color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, fontSize = 13.sp, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    Text("@${participant.username}", color = LinkUpTextMuted, fontSize = 11.sp)
                }
                TextButton(onClick = { onBlockUser(participant) }, enabled = !busy) {
                    Text("Block user", color = LinkUpWarning, fontSize = 11.sp)
                }
            }
        }
    }
}
