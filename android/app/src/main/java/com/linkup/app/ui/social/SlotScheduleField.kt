package com.linkup.app.ui.social

import android.app.DatePickerDialog
import android.app.TimePickerDialog
import android.text.format.DateFormat
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.R
import com.linkup.app.core.scheduling.scheduleInstants
import com.linkup.app.core.scheduling.scheduleLabel
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpWarning
import java.time.Instant
import java.time.LocalDate
import java.time.LocalDateTime
import java.time.ZoneId

@Composable
fun SlotScheduleField(value: Long?, enabled: Boolean, onChange: (Long?) -> Unit) {
    val context = LocalContext.current
    val zoneName = rememberSaveable { ZoneId.systemDefault().id }
    val zone = remember(zoneName) { ZoneId.of(zoneName) }
    var picker by remember { mutableStateOf<String?>(null) }
    var selectedDate by remember { mutableStateOf<LocalDate?>(null) }
    var choices by remember { mutableStateOf<List<Long>>(emptyList()) }
    var error by remember { mutableStateOf<String?>(null) }
    val gapTemplate = stringResource(R.string.schedule_gap_format, zoneName)

    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Text(stringResource(R.string.field_date_time), color = LinkUpTextDimmed, fontSize = 12.sp)
        Text(scheduleLabel(value, zone), color = LinkUpTextDimmed, fontSize = 12.sp)
        Row {
            TextButton(enabled = enabled, onClick = { error = null; picker = "date" }) {
                Text(
                    if (value == null) stringResource(R.string.schedule_action_schedule) else stringResource(R.string.schedule_action_change),
                    color = LinkUpRed,
                )
            }
            if (value != null) {
                TextButton(enabled = enabled, onClick = { error = null; onChange(null) }) {
                    Text(stringResource(R.string.common_now), color = LinkUpRed)
                }
            }
        }
        error?.let { Text(it, color = LinkUpWarning, fontSize = 12.sp) }
    }

    if (picker != null && enabled) {
        DisposableEffect(picker, context) {
            val initial = (value?.let(Instant::ofEpochMilli) ?: Instant.now()).atZone(zone)
            val dialog = if (picker == "date") {
                DatePickerDialog(context, { _, year, month, day ->
                    selectedDate = LocalDate.of(year, month + 1, day)
                    picker = "time"
                }, initial.year, initial.monthValue - 1, initial.dayOfMonth)
            } else {
                TimePickerDialog(context, { _, hour, minute ->
                    val local = LocalDateTime.of(selectedDate ?: initial.toLocalDate(), java.time.LocalTime.of(hour, minute))
                    val candidates = scheduleInstants(local, zone)
                    when (candidates.size) {
                        0 -> error = gapTemplate
                        1 -> onChange(candidates.single())
                        else -> choices = candidates
                    }
                    picker = null
                }, initial.hour, initial.minute, DateFormat.is24HourFormat(context))
            }
            dialog.setOnCancelListener { picker = null }
            dialog.show()
            onDispose { dialog.dismiss() }
        }
    }

    if (choices.isNotEmpty() && enabled) {
        AlertDialog(
            onDismissRequest = { choices = emptyList() },
            title = { Text(stringResource(R.string.schedule_offset_title)) },
            text = { Text(stringResource(R.string.schedule_offset_body)) },
            confirmButton = {
                Column {
                    choices.forEach { instant ->
                        TextButton(onClick = { onChange(instant); choices = emptyList() }) {
                            Text(scheduleLabel(instant, zone), color = LinkUpRed)
                        }
                    }
                }
            },
            dismissButton = {
                TextButton(onClick = { choices = emptyList() }) { Text(stringResource(R.string.common_cancel)) }
            },
        )
    }
}
