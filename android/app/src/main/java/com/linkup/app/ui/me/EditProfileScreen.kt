package com.linkup.app.ui.me

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.R
import com.linkup.app.core.network.UserProfile
import com.linkup.app.ui.design.LinkUpTextField
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning

@Composable
fun EditProfileScreen(
    user: UserProfile,
    busy: Boolean,
    error: String?,
    onBack: () -> Unit,
    onSave: (String, String, String, String) -> Unit,
) {
    var name by rememberSaveable(user.id) { mutableStateOf(user.displayName) }
    var avatar by rememberSaveable(user.id) { mutableStateOf(user.avatarUrl.orEmpty()) }
    var visibility by rememberSaveable(user.id) { mutableStateOf(user.profileVisibility) }
    var language by rememberSaveable(user.id) { mutableStateOf(user.language) }
    val nameLength = name.trim().codePointCount(0, name.trim().length)
    val backDescription = stringResource(R.string.a11y_back)
    BackHandler { if (!busy) onBack() }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(20.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            TextButton(onClick = onBack, enabled = !busy, modifier = Modifier.semantics { contentDescription = backDescription }) {
                Text("‹", color = LinkUpRed, fontSize = 24.sp)
            }
            Text(stringResource(R.string.profile_edit_title), color = LinkUpTextPrimary, fontWeight = FontWeight.Bold, modifier = Modifier.weight(1f))
            TextButton(
                onClick = { onSave(name.trim(), avatar.trim(), visibility, language) },
                enabled = !busy && nameLength in 1..80 && avatar.trim().length <= 2048,
            ) {
                Text(if (busy) stringResource(R.string.common_saving) else stringResource(R.string.common_save), color = LinkUpRed)
            }
        }
        Text("@${user.username}", color = LinkUpTextMuted)
        LinkUpTextField(
            value = name,
            onValueChange = { name = it },
            enabled = !busy,
            label = stringResource(R.string.auth_display_name),
            modifier = Modifier.fillMaxWidth(),
        )
        LinkUpTextField(
            value = avatar,
            onValueChange = { avatar = it },
            enabled = !busy,
            label = stringResource(R.string.profile_avatar_optional),
            modifier = Modifier.fillMaxWidth(),
        )
        Text(stringResource(R.string.profile_avatar_empty_hint), color = LinkUpTextMuted, fontSize = 12.sp)
        Text(stringResource(R.string.profile_visibility), color = LinkUpTextPrimary, fontWeight = FontWeight.Bold)
        listOf(
            "PUBLIC" to stringResource(R.string.profile_public_label),
            "HIDDEN" to stringResource(R.string.profile_hidden_label),
        ).forEach { (value, label) ->
            Row(verticalAlignment = Alignment.CenterVertically) {
                RadioButton(selected = visibility == value, enabled = !busy, onClick = { visibility = value })
                Text(label, color = LinkUpTextPrimary)
            }
        }
        Text(stringResource(R.string.profile_preferred_language), color = LinkUpTextPrimary, fontWeight = FontWeight.Bold)
        listOf(
            "uk" to stringResource(R.string.profile_language_uk),
            "en" to stringResource(R.string.profile_language_en),
        ).forEach { (value, label) ->
            Row(verticalAlignment = Alignment.CenterVertically) {
                RadioButton(selected = language == value, enabled = !busy, onClick = { language = value })
                Text(label, color = LinkUpTextPrimary)
            }
        }
        error?.let { Text(it, color = LinkUpWarning) }
    }
}
