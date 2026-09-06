package com.linkup.app.ui.social

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
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.core.network.CreateSlotInput
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpSuccess
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning
import com.linkup.app.ui.theme.LinkUpZone

private data class ActivityOption(val key: String, val label: String, val emoji: String)

private val activityOptions = listOf(
    ActivityOption("coffee", "Coffee", "☕"),
    ActivityOption("walk", "Walk", "🚶"),
    ActivityOption("running", "Run", "🏃"),
    ActivityOption("food", "Food", "🍜"),
    ActivityOption("gym", "Gym", "🏋️"),
    ActivityOption("music", "Music", "🎵"),
    ActivityOption("games", "Games", "🎮"),
    ActivityOption("social", "Social", "✨"),
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
            TextButton(onClick = onClose) { Text("×", color = LinkUpTextDimmed, fontSize = 24.sp) }
            Column(modifier = Modifier.weight(1f), horizontalAlignment = Alignment.CenterHorizontally) {
                Text("Create LINK", color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, fontSize = 15.sp)
                Text("Step $step of 3", color = LinkUpTextMuted, fontSize = 10.sp)
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
                    SectionTitle("What's happening?", "Pick an activity to get started.")
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
                                    Text(option.label, color = if (selected) LinkUpRed else LinkUpTextDimmed, fontSize = 10.sp, fontWeight = FontWeight.SemiBold)
                                }
                            }
                            repeat(4 - row.size) { Spacer(Modifier.weight(1f)) }
                        }
                    }
                    StyledField("Title (${title.length}/60)", title, { title = it.take(60) }, "e.g. Morning Coffee at Green Hills")
                    StyledField("Description", description, { description = it.take(1000) }, "Tell people what to expect...", singleLine = false)
                }
                2 -> {
                    SectionTitle("Where & when?", "Set the details for your LinkUp.")
                    StyledField("Location", location, { location = it.take(200) }, "e.g. Green Hills Coffee, Podil")
                    Text("Capacity", color = LinkUpTextDimmed, fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        CounterButton("−") { capacity = (capacity - 1).coerceAtLeast(2) }
                        Text("$capacity", color = LinkUpTextPrimary, fontSize = 30.sp, fontWeight = FontWeight.Bold, modifier = Modifier.weight(1f), textAlign = androidx.compose.ui.text.style.TextAlign.Center)
                        CounterButton("+") { capacity = (capacity + 1).coerceAtMost(50) }
                    }
                    Text("Access level", color = LinkUpTextDimmed, fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
                    FixedOption("✓", "Approval Required", "You approve each request manually", LinkUpWarning)
                    Text("Visibility", color = LinkUpTextDimmed, fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
                    FixedOption("◎", "Public", "Visible to people allowed by server privacy rules", LinkUpRed)
                }
                3 -> {
                    SectionTitle("Preview", "Review before publishing.")
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
                                Text("OPEN", color = LinkUpSuccess, fontSize = 10.sp, fontWeight = FontWeight.Bold)
                                Text(title.trim(), color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, fontSize = 16.sp)
                                Text(location.trim(), color = LinkUpTextDimmed, fontSize = 12.sp)
                            }
                        }
                        if (description.isNotBlank()) Text(description.trim(), color = LinkUpTextDimmed, fontSize = 13.sp)
                        Text("0/$capacity going · Approval required · Public", color = LinkUpTextMuted, fontSize = 11.sp)
                    }
                    Text("Publishing creates a real server-backed LinkUp.", color = LinkUpTextDimmed, fontSize = 12.sp, modifier = Modifier.fillMaxWidth().background(LinkUpZone.copy(alpha = .5f)).padding(12.dp))
                    errorMessage?.let { Text(it, color = LinkUpWarning, fontSize = 12.sp) }
                }
            }
        }

        Row(
            Modifier.fillMaxWidth().border(1.dp, LinkUpBorder).padding(horizontal = 20.dp, vertical = 12.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            if (step > 1) SecondaryButton("Back", Modifier.weight(1f)) { step-- }
            if (step < 3) PrimaryButton("Continue", Modifier.weight(2f), enabled = canNext && !submitting) { step++ }
            else PrimaryButton(if (submitting) "Publishing…" else "Publish LINK", Modifier.weight(2f), enabled = !submitting) {
                val selected = activity ?: return@PrimaryButton
                onPublish(
                    CreateSlotInput(
                        title = title.trim(),
                        activity = selected.key,
                        details = description.trim().ifBlank { null },
                        placeText = location.trim(),
                        capacity = capacity,
                    ),
                )
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
