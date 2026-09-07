package com.linkup.app.ui.auth

import androidx.activity.compose.BackHandler
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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.R
import com.linkup.app.core.network.passwordResetToken
import com.linkup.app.ui.theme.LinkUpBorder
import com.linkup.app.ui.theme.LinkUpElevated
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextMuted
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning
import kotlinx.coroutines.launch

enum class AuthMode { LOGIN, REGISTER, RECOVERY, RESET }

@Composable
fun AuthScreen(
    busy: Boolean,
    errorMessage: String?,
    infoMessage: String?,
    initialResetToken: String? = null,
    onLogin: (String, String) -> Unit,
    onRegister: (String, String, String, String) -> Unit,
    onRecovery: (String) -> Unit,
    onResetPassword: suspend (String, String) -> Boolean,
) {
    var mode by remember(initialResetToken) { mutableStateOf(if (initialResetToken != null) AuthMode.RESET else AuthMode.LOGIN) }
    var email by remember { mutableStateOf("") }
    var username by remember { mutableStateOf("") }
    var displayName by remember { mutableStateOf("") }
    var identifier by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    // Reset credentials are intentionally memory-only, never saved instance state.
    var resetInput by remember(initialResetToken) { mutableStateOf(initialResetToken.orEmpty()) }
    var newPassword by remember { mutableStateOf("") }
    var confirmation by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    val loginLabel = stringResource(R.string.auth_login)
    val registerLabel = stringResource(R.string.auth_register)

    BackHandler(enabled = mode == AuthMode.RECOVERY || mode == AuthMode.RESET) {
        if (!busy) {
            mode = AuthMode.LOGIN
            resetInput = ""
            newPassword = ""
            confirmation = ""
        }
    }

    Column(
        Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(horizontal = 24.dp, vertical = 36.dp),
        verticalArrangement = Arrangement.Center,
    ) {
        Text(stringResource(R.string.app_name), color = LinkUpTextPrimary, fontSize = 34.sp, fontWeight = FontWeight.Black)
        Text(stringResource(R.string.tagline_real_world_first), color = LinkUpRed, fontSize = 13.sp, fontWeight = FontWeight.Bold)
        Spacer(Modifier.height(28.dp))

        if (mode == AuthMode.LOGIN || mode == AuthMode.REGISTER) {
            Row(
                Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp)).background(LinkUpElevated).border(1.dp, LinkUpBorder, RoundedCornerShape(12.dp)).padding(4.dp),
            ) {
                ModeButton(loginLabel, mode == AuthMode.LOGIN, Modifier.weight(1f)) { mode = AuthMode.LOGIN }
                ModeButton(registerLabel, mode == AuthMode.REGISTER, Modifier.weight(1f)) { mode = AuthMode.REGISTER }
            }
            Spacer(Modifier.height(18.dp))
        }

        when (mode) {
            AuthMode.LOGIN -> {
                AuthField(stringResource(R.string.auth_email_or_username), identifier) { identifier = it.take(320) }
                Spacer(Modifier.height(10.dp))
                AuthPasswordField(password) { password = it.take(256) }
                Spacer(Modifier.height(14.dp))
                SubmitButton(loginLabel, busy, identifier.isNotBlank() && password.isNotBlank()) { onLogin(identifier.trim(), password) }
                TextButton(onClick = { mode = AuthMode.RECOVERY }, modifier = Modifier.align(Alignment.End)) {
                    Text(stringResource(R.string.auth_forgot_password), color = LinkUpTextDimmed, fontSize = 12.sp)
                }
            }
            AuthMode.REGISTER -> {
                AuthField(stringResource(R.string.auth_email), email) { email = it.take(320) }
                Spacer(Modifier.height(10.dp))
                AuthField(stringResource(R.string.auth_username), username) { username = it.take(32) }
                Spacer(Modifier.height(10.dp))
                AuthField(stringResource(R.string.auth_display_name), displayName) { displayName = it.take(80) }
                Spacer(Modifier.height(10.dp))
                AuthPasswordField(password) { password = it.take(256) }
                Spacer(Modifier.height(14.dp))
                SubmitButton(
                    stringResource(R.string.auth_create_account),
                    busy,
                    email.isNotBlank() && username.isNotBlank() && displayName.isNotBlank() && password.isNotBlank(),
                ) {
                    onRegister(email.trim(), username.trim(), displayName.trim(), password)
                }
            }
            AuthMode.RECOVERY -> {
                Text(stringResource(R.string.auth_password_recovery), color = LinkUpTextPrimary, fontSize = 20.sp, fontWeight = FontWeight.Bold)
                Text(stringResource(R.string.auth_recovery_description), color = LinkUpTextDimmed, fontSize = 13.sp)
                Spacer(Modifier.height(16.dp))
                AuthField(stringResource(R.string.auth_email), email) { email = it.take(320) }
                Spacer(Modifier.height(14.dp))
                SubmitButton(stringResource(R.string.auth_send_reset_link), busy, email.isNotBlank()) { onRecovery(email.trim()) }
                TextButton(enabled = !busy, onClick = { mode = AuthMode.RESET }) {
                    Text(stringResource(R.string.auth_have_reset_link), color = LinkUpRed)
                }
                TextButton(enabled = !busy, onClick = { mode = AuthMode.LOGIN }) {
                    Text(stringResource(R.string.auth_back_to_login), color = LinkUpRed)
                }
            }
            AuthMode.RESET -> {
                Text(stringResource(R.string.auth_set_new_password), color = LinkUpTextPrimary, fontSize = 20.sp, fontWeight = FontWeight.Bold)
                Text(stringResource(R.string.auth_reset_description), color = LinkUpTextDimmed, fontSize = 13.sp)
                Spacer(Modifier.height(16.dp))
                AuthField(stringResource(R.string.auth_reset_link_or_code), resetInput) { resetInput = it.take(4096) }
                Spacer(Modifier.height(10.dp))
                AuthPasswordField(newPassword, stringResource(R.string.auth_new_password)) { newPassword = it.take(1024) }
                Spacer(Modifier.height(10.dp))
                AuthPasswordField(confirmation, stringResource(R.string.auth_confirm_password)) { confirmation = it.take(1024) }
                Spacer(Modifier.height(14.dp))
                val token = passwordResetToken(resetInput)
                val valid = token != null && newPassword == confirmation && newPassword.toByteArray(Charsets.UTF_8).size in 8..1024
                if (resetInput.isNotBlank() && token == null) {
                    Text(stringResource(R.string.auth_invalid_reset), color = LinkUpWarning, fontSize = 12.sp)
                }
                if (confirmation.isNotEmpty() && newPassword != confirmation) {
                    Text(stringResource(R.string.auth_passwords_mismatch), color = LinkUpWarning, fontSize = 12.sp)
                }
                Text(stringResource(R.string.auth_password_hint), color = LinkUpTextMuted, fontSize = 12.sp)
                SubmitButton(stringResource(R.string.auth_change_password), busy, valid) {
                    if (token != null) scope.launch {
                        if (onResetPassword(token, newPassword)) {
                            resetInput = ""
                            newPassword = ""
                            confirmation = ""
                            password = ""
                            mode = AuthMode.LOGIN
                        }
                    }
                }
                TextButton(enabled = !busy, onClick = {
                    resetInput = ""
                    newPassword = ""
                    confirmation = ""
                    mode = AuthMode.LOGIN
                }) {
                    Text(stringResource(R.string.auth_back_to_login), color = LinkUpRed)
                }
            }
        }

        if (busy) {
            Spacer(Modifier.height(12.dp))
            CircularProgressIndicator(color = LinkUpRed, modifier = Modifier.align(Alignment.CenterHorizontally))
        }
        errorMessage?.let {
            Spacer(Modifier.height(10.dp))
            Text(it, color = LinkUpWarning, fontSize = 12.sp)
        }
        infoMessage?.let {
            Spacer(Modifier.height(10.dp))
            Text(it, color = LinkUpTextDimmed, fontSize = 12.sp)
        }
    }
}

@Composable
private fun ModeButton(label: String, active: Boolean, modifier: Modifier, onClick: () -> Unit) {
    Box(
        modifier.height(42.dp).clip(RoundedCornerShape(9.dp)).background(if (active) LinkUpRed else androidx.compose.ui.graphics.Color.Transparent).clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Text(label, color = if (active) LinkUpTextPrimary else LinkUpTextMuted, fontWeight = FontWeight.Bold, fontSize = 12.sp)
    }
}

@Composable
private fun AuthField(label: String, value: String, onChange: (String) -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(5.dp)) {
        Text(label, color = LinkUpTextDimmed, fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
        OutlinedTextField(value = value, onValueChange = onChange, modifier = Modifier.fillMaxWidth(), singleLine = true, shape = RoundedCornerShape(12.dp))
    }
}

@Composable
private fun AuthPasswordField(value: String, label: String? = null, onChange: (String) -> Unit) {
    Column(verticalArrangement = Arrangement.spacedBy(5.dp)) {
        Text(label ?: stringResource(R.string.auth_password), color = LinkUpTextDimmed, fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
        OutlinedTextField(
            value = value,
            onValueChange = onChange,
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
            visualTransformation = PasswordVisualTransformation(),
            shape = RoundedCornerShape(12.dp),
        )
    }
}

@Composable
private fun SubmitButton(label: String, busy: Boolean, valid: Boolean, onClick: () -> Unit) {
    Button(
        onClick = onClick,
        enabled = !busy && valid,
        modifier = Modifier.fillMaxWidth().height(50.dp),
        shape = RoundedCornerShape(12.dp),
        colors = ButtonDefaults.buttonColors(containerColor = LinkUpRed, contentColor = LinkUpTextPrimary),
    ) {
        Text(label, fontWeight = FontWeight.Bold)
    }
}
