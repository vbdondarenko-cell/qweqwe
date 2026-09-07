package com.linkup.app.ui.social

import androidx.annotation.StringRes
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
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
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.R
import com.linkup.app.core.network.CreateSlotInput
import com.linkup.app.core.scheduling.scheduleLabel
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpSuccess
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning
import com.linkup.app.ui.theme.LinkUpZone

private data class ActivityOption(val key: String, @StringRes val labelRes: Int, val emoji: String)

private val activityOptions = listOf(
    ActivityOption("coffee", R.string.activity_coffee, "☕"),
    ActivityOption("walk", R.string.activity_walk, "🚶"),
    ActivityOption("running", R.string.activity_run, "🏃"),
    ActivityOption("food", R.string.activity_food, "🍜"),
    ActivityOption("gym", R.string.activity_gym, "🏋️"),
    ActivityOption("music", R.string.activity_music, "🎵"),
    ActivityOption("games", R.string.activity_games, "🎮"),
    ActivityOption("social", R.string.activity_social, "✨"),
)

@Composable
fun CreateLinkScreen(
    submitting: Boolean,
    errorMessage: String?,
    onClose: () -> Unit,
    onPublish: (CreateSlotInput) -> Unit,
) {
    var step by remember { mutableIntStateOf(1) }
    var activity by remember { mutableStateOf<ActivityOption?>(null) }
    var title by remember { mutableStateOf("") }
    var description by remember { mutableStateOf("") }
    var location by remember { mutableStateOf("") }
    var capacity by remember { mutableIntStateOf(6) }
    var startAt by rememberSaveable { mutableStateOf<Long?>(null) }
    val closeDescription = stringResource(R.string.a11y_close)

    val canNext = when (step) {
        1 -> activity != null && title.trim().isNotEmpty()
        2 -> location.trim().isNotEmpty()
        else -> true
    }

    Column(Modifier.fillMaxSize()) {
        Row(
            modifier = Modifier.fillMaxWidth().border(1.dp, LinkUpBorder).padding(horizontal = 20.dp, vertical = 14.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TextButton(onClick = onClose, modifier = Modifier.semantics { contentDescription = closeDescription }) {
                Text("×", color = LinkUpTextDimmed, fontSize = 24.sp)
            }
            Column(modifier = Modifier.weight(1f), horizontalAlignment = Alignment.CenterHorizontally) {
                Text(stringResource(R.string.create_link_title), color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, fontSize = 15.sp)
                Text(stringResource(R.string.create_step_format, step, 3), color = LinkUpTextMuted, fontSize = 10.sp)
            }
            Spacer(Modifier.width(48.dp))
        }

        Row(Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 9.dp), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
            repeat(3) { index ->
                Box(
                    Modifier.weight(1f).height(4.dp).clip(RoundedCornerShape(99.dp))
                        .background(if (index + 1 <= step) LinkUpRed else LinkUpZone),
                )
            }
        }

        Column(
            modifier = Modifier.weight(1f).verticalScroll(rememberScrollState()).padding(horizontal = 20.dp, vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            when (step) {
                1 -> {
                    SectionTitle(stringResource(R.string.create_what_happening), stringResource(R.string.create_pick_activity))
                    activityOptions.chunked(4).forEach { row ->
                        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            row.forEach { option ->
                                val selected = activity?.key == option.key
                                Column(
                                    modifier = Modifier.weight(1f).clip(RoundedCornerShape(16.dp))
                                        .background(if (selected) LinkUpRed.copy(alpha = .15f) else LinkUpElevated)
                                        .border(1.dp, if (selected) LinkUpRed else LinkUpBorder, RoundedCornerShape(16.dp))
                                        .clickable { activity = option }.padding(vertical = 14.dp),
                                    horizontalAlignment = Alignment.CenterHorizontally,
                                ) {
                                    Text(option.emoji, fontSize = 24.sp)
                                    Text(stringResource(option.labelRes), color = if (selected) LinkUpRed else LinkUpTextDimmed, fontSize = 10.sp, fontWeight = FontWeight.SemiBold)
                                }
                            }
                            repeat(4 - row.size) { Spacer(Modifier.weight(1f)) }
                        }
                    }
                    StyledField(
                        stringResource(R.string.create_title_counter, title.length),
                        title,
                        { title = it.take(60) },
                        stringResource(R.string.create_title_example),
                    )
                    StyledField(
                        stringResource(R.string.field_details),
                        description,
                        { description = it.take(1000) },
                        stringResource(R.string.create_description_example),
                        singleLine = false,
                    )
                }
                2 -> {
                    SectionTitle(stringResource(R.string.create_where_when), stringResource(R.string.create_details_subtitle))
                    StyledField(
                        stringResource(R.string.create_location),
                        location,
                        { location = it.take(200) },
                        stringResource(R.string.create_location_example),
                    )
                    SlotScheduleField(startAt, !submitting) { startAt = it }
                    Text(stringResource(R.string.field_capacity), color = LinkUpTextDimmed, fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        CounterButton("−") { capacity = (capacity - 1).coerceAtLeast(2) }
                        Text("$capacity", color = LinkUpTextPrimary, fontSize = 30.sp, fontWeight = FontWeight.Bold, modifier = Modifier.weight(1f), textAlign = TextAlign.Center)
                        CounterButton("+") { capacity = (capacity + 1).coerceAtMost(50) }
                    }
                    Text(stringResource(R.string.create_access_level), color = LinkUpTextDimmed, fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
                    FixedOption("✓", stringResource(R.string.create_approval_required), stringResource(R.string.create_approval_subtitle), LinkUpWarning)
                    Text(stringResource(R.string.create_visibility), color = LinkUpTextDimmed, fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
                    FixedOption("◎", stringResource(R.string.create_public), stringResource(R.string.create_public_subtitle), LinkUpRed)
                }
                3 -> {
                    SectionTitle(stringResource(R.string.create_preview), stringResource(R.string.create_review_subtitle))
                    Column(
                        Modifier.fillMaxWidth().clip(RoundedCornerShape(16.dp)).background(LinkUpElevated)
                            .border(1.dp, LinkUpBorder, RoundedCornerShape(16.dp)).padding(16.dp),
                        verticalArrangement = Arrangement.spacedBy(10.dp),
                    ) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Box(Modifier.size(56.dp).clip(RoundedCornerShape(16.dp)).background(LinkUpZone), contentAlignment = Alignment.Center) {
                                Text(activity?.emoji ?: "🎯", fontSize = 28.sp)
                            }
                            Spacer(Modifier.width(12.dp))
                            Column {
                                Text(stringResource(R.string.slot_badge_open), color = LinkUpSuccess, fontSize = 10.sp, fontWeight = FontWeight.Bold)
                                Text(title.trim(), color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, fontSize = 16.sp)
                                Text(location.trim(), color = LinkUpTextDimmed, fontSize = 12.sp)
                            }
                        }
                        if (description.isNotBlank()) Text(description.trim(), color = LinkUpTextDimmed, fontSize = 13.sp)
                        Text(scheduleLabel(startAt), color = LinkUpTextDimmed, fontSize = 12.sp)
                        Text(stringResource(R.string.create_preview_capacity_format, capacity), color = LinkUpTextMuted, fontSize = 11.sp)
                    }
                    Text(
                        stringResource(R.string.create_server_backed_notice),
                        color = LinkUpTextDimmed,
                        fontSize = 12.sp,
                        modifier = Modifier.fillMaxWidth().background(LinkUpZone.copy(alpha = .5f)).padding(12.dp),
                    )
                    errorMessage?.let { Text(it, color = LinkUpWarning, fontSize = 12.sp) }
                }
            }
        }

        Row(
            Modifier.fillMaxWidth().border(1.dp, LinkUpBorder).padding(horizontal = 20.dp, vertical = 12.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            if (step > 1) SecondaryButton(stringResource(R.string.common_back), Modifier.weight(1f)) { step-- }
            if (step < 3) {
                PrimaryButton(stringResource(R.string.create_continue), Modifier.weight(2f), enabled = canNext && !submitting) { step++ }
            } else {
                PrimaryButton(
                    if (submitting) stringResource(R.string.create_publishing) else stringResource(R.string.create_publish),
                    Modifier.weight(2f),
                    enabled = !submitting,
                ) {
                    val selected = activity ?: return@PrimaryButton
                    onPublish(
                        CreateSlotInput(
                            title = title.trim(),
                            activity = selected.key,
                            details = description.trim().ifBlank { null },
                            placeText = location.trim(),
                            capacity = capacity,
                            startAtEpochMillis = startAt,
                        ),
                    )
                }
            }
        }
    }
}

@Composable
private fun SectionTitle(title: String, subtitle: String) {
    Column {
        Text(title, color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, fontSize = 20.sp)
        Text(subtitle, color = LinkUpTextDimmed, fontSize = 14.sp)
    }
}

@Composable
private fun StyledField(label: String, value: String, onChange: (String) -> Unit, placeholder: String, singleLine: Boolean = true) {
    Column(verticalArrangement = Arrangement.spacedBy(5.dp)) {
        Text(label, color = LinkUpTextDimmed, fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
        OutlinedTextField(
            value = value,
            onValueChange = onChange,
            modifier = Modifier.fillMaxWidth(),
            placeholder = { Text(placeholder, color = LinkUpTextMuted, fontSize = 13.sp) },
            singleLine = singleLine,
            minLines = if (singleLine) 1 else 3,
            shape = RoundedCornerShape(12.dp),
        )
    }
}

@Composable
private fun CounterButton(label: String, onClick: () -> Unit) {
    Box(
        Modifier.size(48.dp).clip(RoundedCornerShape(16.dp)).background(LinkUpElevated)
            .border(1.dp, LinkUpBorder, RoundedCornerShape(16.dp)).clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) { Text(label, color = LinkUpTextDimmed, fontSize = 22.sp) }
}

@Composable
private fun FixedOption(icon: String, title: String, subtitle: String, color: androidx.compose.ui.graphics.Color) {
    Row(
        Modifier.fillMaxWidth().clip(RoundedCornerShape(14.dp)).background(color.copy(alpha = .10f))
            .border(1.dp, color.copy(alpha = .45f), RoundedCornerShape(14.dp)).padding(14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(icon, color = color, fontSize = 18.sp)
        Spacer(Modifier.width(12.dp))
        Column {
            Text(title, color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, fontSize = 13.sp)
            Text(subtitle, color = LinkUpTextMuted, fontSize = 11.sp)
        }
    }
}

@Composable
private fun PrimaryButton(label: String, modifier: Modifier = Modifier, enabled: Boolean = true, onClick: () -> Unit) {
    Button(
        onClick = onClick,
        enabled = enabled,
        modifier = modifier.height(48.dp),
        shape = RoundedCornerShape(12.dp),
        colors = ButtonDefaults.buttonColors(containerColor = LinkUpRed, contentColor = LinkUpTextPrimary),
    ) { Text(label, fontWeight = FontWeight.Bold) }
}

@Composable
private fun SecondaryButton(label: String, modifier: Modifier = Modifier, onClick: () -> Unit) {
    Box(
        modifier.height(48.dp).clip(RoundedCornerShape(12.dp)).background(LinkUpElevated)
            .border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp)).clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) { Text(label, color = LinkUpTextDimmed, fontWeight = FontWeight.Bold) }
}
