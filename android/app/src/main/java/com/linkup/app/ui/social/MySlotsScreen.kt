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
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.R
import com.linkup.app.core.network.MySlotsView
import com.linkup.app.core.network.SlotModel
import com.linkup.app.core.social.LoadState
import com.linkup.app.ui.design.FrozenSlotCard
import com.linkup.app.ui.design.toFrozenSlot
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
            TextButton(onClick = onBack, modifier = Modifier.weight(1f)) { Text(stringResource(R.string.common_back), color = LinkUpRed) }
            TextButton(onClick = onRefresh) { Text(stringResource(R.string.common_refresh), color = LinkUpRed) }
        }
        Text(stringResource(R.string.my_links_title), color = LinkUpTextPrimary, fontSize = 24.sp, fontWeight = FontWeight.Bold)
        Row {
            MySlotsView.values().forEach { item ->
                TextButton(onClick = { onViewChange(item) }, modifier = Modifier.weight(1f)) {
                    Text(
                        when (item) {
                            MySlotsView.HOSTING -> stringResource(R.string.my_links_hosting)
                            MySlotsView.JOINED -> stringResource(R.string.my_links_joined)
                            MySlotsView.REQUESTED -> stringResource(R.string.my_links_requests)
                        },
                        color = if (view == item) LinkUpRed else LinkUpTextMuted,
                        fontWeight = if (view == item) FontWeight.Bold else FontWeight.Normal,
                    )
                }
            }
        }
        when (state) {
            LoadState.Idle, LoadState.Loading -> CircularProgressIndicator(color = LinkUpRed)
            LoadState.Empty -> Text(stringResource(R.string.my_links_empty), color = LinkUpTextMuted)
            is LoadState.Failure -> {
                Text(state.error.message.ifBlank { stringResource(R.string.my_links_error) }, color = LinkUpWarning)
                TextButton(onClick = onRefresh) { Text(stringResource(R.string.common_retry), color = LinkUpRed) }
            }
            is LoadState.Content -> LazyColumn(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                items(state.value, key = { it.id }) { item ->
                    FrozenSlotCard(
                        slot = item.toFrozenSlot(),
                        onClick = { onSlotClick(item) },
                        actionLabel = stringResource(R.string.common_open),
                        actionEnabled = true,
                        onPrimaryAction = { onSlotClick(item) },
                    )
                }
            }
        }
    }
}
