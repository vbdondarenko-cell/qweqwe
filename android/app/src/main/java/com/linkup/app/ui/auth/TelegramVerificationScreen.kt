package com.linkup.app.ui.auth

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.R
import com.linkup.app.core.network.telegramNativeDeepLink
import com.linkup.app.core.network.telegramVerificationCommandDeepLink
import com.linkup.app.core.session.SessionState
import com.linkup.app.ui.design.LinkUpButton
import com.linkup.app.ui.design.LinkUpButtonSize
import com.linkup.app.ui.design.LinkUpButtonVariant
import com.linkup.app.ui.theme.LinkUpBackground
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive

@Composable
fun TelegramVerificationScreen(
    state: SessionState.TelegramVerification,
    busy: Boolean,
    onRefresh: () -> Unit,
    onCancel: () -> Unit,
) {
    val uriHandler = LocalUriHandler.current

    LaunchedEffect(state.start.verificationToken) {
        while (isActive) {
            delay(10_000)
            onRefresh()
        }
    }

    Column(
        Modifier.fillMaxSize().background(LinkUpBackground).padding(horizontal = 24.dp, vertical = 36.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.Start,
    ) {
        Text(stringResource(R.string.telegram_verify_title), color = LinkUpTextPrimary, fontSize = 26.sp, fontWeight = FontWeight.Black)
        Spacer(Modifier.height(8.dp))
        Text(stringResource(R.string.telegram_verify_body), color = LinkUpTextDimmed, fontSize = 14.sp)
        Spacer(Modifier.height(6.dp))
        Text(stringResource(R.string.telegram_start_hint), color = LinkUpTextDimmed, fontSize = 12.sp)
        if (state.teenMode == true) {
            Spacer(Modifier.height(8.dp))
            Text(stringResource(R.string.onboarding_teen_mode), color = LinkUpTextDimmed, fontSize = 12.sp)
        }
        Spacer(Modifier.height(20.dp))
        LinkUpButton(
            label = stringResource(R.string.telegram_open_bot),
            onClick = {
                val native = telegramNativeDeepLink(state.start.telegramDeepLink, state.start.verificationToken)
                val nativeOpened = native?.let { runCatching { uriHandler.openUri(it) }.isSuccess } ?: false
                if (!nativeOpened) runCatching { uriHandler.openUri(state.start.telegramDeepLink) }
            },
            enabled = !busy,
            modifier = Modifier.fillMaxWidth().height(50.dp),
            variant = LinkUpButtonVariant.PRIMARY,
            size = LinkUpButtonSize.LG,
        )
        Spacer(Modifier.height(10.dp))
        LinkUpButton(
            label = stringResource(R.string.telegram_open_with_command),
            onClick = {
                val commandLink = telegramVerificationCommandDeepLink(
                    state.start.telegramDeepLink,
                    state.start.verificationToken,
                )
                if (commandLink != null) runCatching { uriHandler.openUri(commandLink) }
            },
            enabled = !busy,
            modifier = Modifier.fillMaxWidth().height(50.dp),
            variant = LinkUpButtonVariant.SECONDARY,
            size = LinkUpButtonSize.LG,
        )
        Spacer(Modifier.height(6.dp))
        Text(stringResource(R.string.telegram_command_hint), color = LinkUpTextDimmed, fontSize = 12.sp)
        Spacer(Modifier.height(10.dp))
        LinkUpButton(
            label = stringResource(R.string.telegram_check_status),
            onClick = onRefresh,
            enabled = !busy,
            modifier = Modifier.fillMaxWidth().height(50.dp),
            variant = LinkUpButtonVariant.SECONDARY,
            size = LinkUpButtonSize.LG,
        )
        if (busy) {
            Spacer(Modifier.height(12.dp))
            CircularProgressIndicator(color = LinkUpRed, modifier = Modifier.align(Alignment.CenterHorizontally))
        }
        if (state.statusChecked && state.message.isNullOrBlank()) {
            Spacer(Modifier.height(10.dp))
            Text(stringResource(R.string.telegram_waiting_for_phone), color = LinkUpWarning, fontSize = 12.sp)
        }
        state.message?.takeIf { it.isNotBlank() }?.let {
            Spacer(Modifier.height(10.dp))
            Text(it, color = LinkUpWarning, fontSize = 12.sp)
        }
        Spacer(Modifier.height(8.dp))
        TextButton(onClick = onCancel, enabled = !busy, modifier = Modifier.align(Alignment.CenterHorizontally)) {
            Text(stringResource(R.string.telegram_cancel_registration), color = LinkUpTextDimmed)
        }
    }
}
