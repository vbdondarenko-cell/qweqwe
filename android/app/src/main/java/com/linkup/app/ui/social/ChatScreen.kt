package com.linkup.app.ui.social

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.R
import com.linkup.app.core.network.ChatMessage
import com.linkup.app.core.social.LoadState
import com.linkup.app.core.social.MutationState
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

@Composable
fun ChatScreen(
    state: LoadState<List<ChatMessage>>,
    mutation: MutationState,
    onBack: () -> Unit,
    onRefresh: () -> Unit,
    onSend: (String) -> Unit,
) {
    var text by remember { mutableStateOf("") }
    var submittedText by remember { mutableStateOf<String?>(null) }
    var sawRunning by remember { mutableStateOf(false) }
    val busy = mutation is MutationState.Running
    val backDescription = stringResource(R.string.a11y_back)
    BackHandler(onBack = onBack)

    LaunchedEffect(mutation) {
        when (mutation) {
            MutationState.Running -> sawRunning = true
            MutationState.Idle -> {
                val submitted = submittedText
                if (sawRunning && submitted != null) {
                    if (text.trim() == submitted) text = ""
                    submittedText = null
                    sawRunning = false
                }
            }
            is MutationState.Failed -> {
                submittedText = null
                sawRunning = false
            }
        }
    }

    Column(Modifier.fillMaxSize()) {
        Row(
            Modifier.fillMaxWidth().border(1.dp, LinkUpBorder).padding(horizontal = 14.dp, vertical = 10.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TextButton(onClick = onBack, modifier = Modifier.semantics { contentDescription = backDescription }) {
                Text("‹", color = LinkUpTextDimmed, fontSize = 28.sp)
            }
            Column(modifier = Modifier.weight(1f)) {
                Text(stringResource(R.string.chat_title), color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, fontSize = 15.sp)
                Text(stringResource(R.string.chat_subtitle), color = LinkUpTextMuted, fontSize = 10.sp)
            }
            TextButton(onClick = onRefresh) { Text(stringResource(R.string.common_refresh), color = LinkUpRed, fontSize = 12.sp) }
        }

        when (state) {
            LoadState.Idle, LoadState.Loading -> Column(Modifier.weight(1f).fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) {
                CircularProgressIndicator(color = LinkUpRed)
            }
            LoadState.Empty -> Column(Modifier.weight(1f).fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) {
                Text(stringResource(R.string.chat_no_messages), color = LinkUpTextPrimary, fontWeight = FontWeight.Bold)
                Text(stringResource(R.string.chat_no_messages_body), color = LinkUpTextMuted, fontSize = 12.sp)
            }
            is LoadState.Failure -> Column(Modifier.weight(1f).fillMaxWidth().padding(24.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) {
                Text(stringResource(R.string.chat_unavailable), color = LinkUpTextPrimary, fontWeight = FontWeight.Bold)
                Text(state.error.message, color = LinkUpTextDimmed, fontSize = 12.sp)
                Spacer(Modifier.height(8.dp))
                TextButton(onClick = onRefresh) { Text(stringResource(R.string.common_retry), color = LinkUpRed) }
            }
            is LoadState.Content -> LazyColumn(
                modifier = Modifier.weight(1f).fillMaxWidth(),
                contentPadding = androidx.compose.foundation.layout.PaddingValues(horizontal = 16.dp, vertical = 14.dp),
                verticalArrangement = Arrangement.spacedBy(9.dp),
            ) {
                items(state.value, key = { it.id }) { message -> MessageBubble(message) }
            }
        }

        if (mutation is MutationState.Failed) {
            Text(mutation.error.message, color = LinkUpWarning, fontSize = 11.sp, modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp))
        }

        Row(
            Modifier.fillMaxWidth().border(1.dp, LinkUpBorder).padding(horizontal = 12.dp, vertical = 10.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            OutlinedTextField(
                value = text,
                onValueChange = { if (it.length <= 2000) text = it },
                modifier = Modifier.weight(1f),
                placeholder = { Text(stringResource(R.string.chat_message_hint), color = LinkUpTextMuted) },
                maxLines = 4,
                shape = RoundedCornerShape(14.dp),
            )
            Spacer(Modifier.width(8.dp))
            TextButton(
                onClick = {
                    val outgoing = text.trim()
                    if (outgoing.isNotEmpty()) {
                        submittedText = outgoing
                        sawRunning = false
                        onSend(outgoing)
                    }
                },
                enabled = !busy && (state is LoadState.Content || state is LoadState.Empty) && text.trim().isNotEmpty(),
            ) {
                Text(stringResource(R.string.chat_send), color = LinkUpRed, fontWeight = FontWeight.Bold)
            }
        }
    }
}

@Composable
private fun MessageBubble(message: ChatMessage) {
    Column(
        Modifier.fillMaxWidth().clip(RoundedCornerShape(14.dp)).background(LinkUpElevated).border(1.dp, LinkUpBorder, RoundedCornerShape(14.dp)).padding(12.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(message.author.displayName, color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, fontSize = 12.sp)
            Spacer(Modifier.width(6.dp))
            Text("@${message.author.username}", color = LinkUpTextMuted, fontSize = 10.sp, modifier = Modifier.weight(1f))
            Text(chatTime(message.createdAtEpochMillis), color = LinkUpTextMuted, fontSize = 10.sp)
        }
        Spacer(Modifier.height(5.dp))
        Text(message.text, color = LinkUpTextDimmed, fontSize = 14.sp)
    }
}

private fun chatTime(epochMillis: Long): String = DateTimeFormatter.ofPattern("HH:mm")
    .withZone(ZoneId.systemDefault())
    .format(Instant.ofEpochMilli(epochMillis))
