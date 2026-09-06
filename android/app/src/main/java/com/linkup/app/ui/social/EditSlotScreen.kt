package com.linkup.app.ui.social

import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.core.network.EditSlotInput
import com.linkup.app.core.network.SlotModel
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning

@Composable
fun EditSlotScreen(
    slot: SlotModel,
    submitting: Boolean,
    errorMessage: String?,
    onClose: () -> Unit,
    onSave: (EditSlotInput) -> Unit,
) {
    var title by remember(slot.id, slot.version) { mutableStateOf(slot.title) }
    var details by remember(slot.id, slot.version) { mutableStateOf(slot.details.orEmpty()) }
    var place by remember(slot.id, slot.version) { mutableStateOf(slot.placeText) }
    var capacity by remember(slot.id, slot.version) { mutableIntStateOf(slot.capacity) }

    Column(Modifier.fillMaxSize()) {
        Row(
            Modifier.fillMaxWidth().border(1.dp, LinkUpBorder).padding(horizontal = 14.dp, vertical = 10.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TextButton(onClick = onClose) { Text("×", color = LinkUpTextDimmed, fontSize = 24.sp) }
            Column(modifier = Modifier.weight(1f), horizontalAlignment = Alignment.CenterHorizontally) {
                Text("Edit LINK", color = LinkUpTextPrimary, fontWeight = FontWeight.Bold)
                Text("Server version ${slot.version}", color = LinkUpTextMuted, fontSize = 10.sp)
            }
            TextButton(
                onClick = {
                    onSave(
                        EditSlotInput(
                            expectedVersion = slot.version,
                            title = title.trim(),
                            details = details.trim(),
                            placeText = place.trim(),
                            capacity = capacity,
                        ),
                    )
                },
                enabled = !submitting && title.trim().isNotEmpty() && place.trim().isNotEmpty() && capacity >= slot.acceptedCount,
            ) { Text(if (submitting) "Saving…" else "Save", color = LinkUpRed, fontWeight = FontWeight.Bold) }
        }

        Column(
            Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            EditField("Title", title, { title = it.take(60) })
            EditField("Description", details, { details = it.take(1000) }, singleLine = false)
            EditField("Location", place, { place = it.take(200) })
            Text("Capacity", color = LinkUpTextDimmed, fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
            Row(verticalAlignment = Alignment.CenterVertically) {
                TextButton(onClick = { capacity = (capacity - 1).coerceAtLeast(slot.acceptedCount.coerceAtLeast(2)) }) { Text("−", color = LinkUpTextDimmed, fontSize = 22.sp) }
                Text("$capacity", color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, fontSize = 28.sp, modifier = Modifier.weight(1f), textAlign = androidx.compose.ui.text.style.TextAlign.Center)
                TextButton(onClick = { capacity = (capacity + 1).coerceAtMost(50) }) { Text("+", color = LinkUpTextDimmed, fontSize = 22.sp) }
            }
            if (capacity < slot.acceptedCount) {
                Text("Capacity cannot be lower than ${slot.acceptedCount} accepted participants.", color = LinkUpWarning, fontSize = 12.sp)
            }
            errorMessage?.let { Text(it, color = LinkUpWarning, fontSize = 12.sp) }
        }
    }
}

@Composable
private fun EditField(label: String, value: String, onChange: (String) -> Unit, singleLine: Boolean = true) {
    Column(verticalArrangement = Arrangement.spacedBy(5.dp)) {
        Text(label, color = LinkUpTextDimmed, fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
        OutlinedTextField(
            value = value,
            onValueChange = onChange,
            modifier = Modifier.fillMaxWidth(),
            singleLine = singleLine,
            minLines = if (singleLine) 1 else 3,
            shape = RoundedCornerShape(12.dp),
        )
    }
}
