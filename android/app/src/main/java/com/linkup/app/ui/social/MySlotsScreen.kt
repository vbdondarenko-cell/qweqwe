package com.linkup.app.ui.social

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.core.network.MySlotsView
import com.linkup.app.core.network.SlotModel
import com.linkup.app.core.social.LoadState
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning

@Composable
fun MySlotsScreen(
    state: LoadState<List<SlotModel>>,
    view: MySlotsView,
    onViewChange: (MySlotsView) -> Unit,
    onBack: () -> Unit,
    onRefresh: () -> Unit,
    onSlotClick: (SlotModel) -> Unit,
) {
    BackHandler(onBack = onBack)
    Column(Modifier.fillMaxSize().padding(20.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Row {
            TextButton(onClick = onBack, modifier = Modifier.weight(1f)) { Text("Back", color = LinkUpRed) }
            TextButton(onClick = onRefresh) { Text("Refresh", color = LinkUpRed) }
        }
        Text("My LINKs", color = LinkUpTextPrimary, fontSize = 24.sp, fontWeight = FontWeight.Bold)
        Text("Current LINKs · up to 100 most recently updated", color = LinkUpTextMuted, fontSize = 12.sp)
        Row {
            MySlotsView.values().forEach { item ->
                TextButton(onClick = { onViewChange(item) }, modifier = Modifier.weight(1f)) {
                    Text(
                        when (item) { MySlotsView.HOSTING -> "Hosting"; MySlotsView.JOINED -> "Joined"; MySlotsView.REQUESTED -> "Requests" },
                        color = if (view == item) LinkUpRed else LinkUpTextMuted,
                        fontWeight = if (view == item) FontWeight.Bold else FontWeight.Normal,
                    )
                }
            }
        }
        when (state) {
            LoadState.Idle, LoadState.Loading -> CircularProgressIndicator(color = LinkUpRed)
            LoadState.Empty -> Text(
                when (view) {
                    MySlotsView.HOSTING -> "You are not hosting any current LINKs."
                    MySlotsView.JOINED -> "You have not joined any current LINKs."
                    MySlotsView.REQUESTED -> "You have no pending requests."
                }, color = LinkUpTextMuted,
            )
            is LoadState.Failure -> {
                Text(state.error.message, color = LinkUpWarning)
                TextButton(onClick = onRefresh) { Text("Retry", color = LinkUpRed) }
            }
            is LoadState.Content -> LazyColumn(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                items(state.value, key = { it.id }) { item ->
                    SlotCard(item, onSlotClick = { onSlotClick(item) }, onPrimaryAction = { onSlotClick(item) })
                }
            }
        }
    }
}
