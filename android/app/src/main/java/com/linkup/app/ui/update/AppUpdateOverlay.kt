package com.linkup.app.ui.update

import android.net.Uri
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.width
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import com.linkup.app.R
import com.linkup.app.core.network.AppUpdateInfo
import com.linkup.app.core.update.AppUpdateState
import com.linkup.app.ui.design.LinkUpButton
import com.linkup.app.ui.design.LinkUpButtonVariant
import com.linkup.app.ui.design.LinkUpCard
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpWarning

/**
 * A minimal, always-explicit surface for the OTA update flow, styled like
 * every other LinkUp modal ([LinkUpCard] in a plain [Dialog]) rather than a
 * default Material3 [androidx.compose.material3.AlertDialog] -- every step
 * (download, install, granting the one-time "install unknown apps"
 * permission) needs a tap. There is no auto-download or auto-install path.
 */
@Composable
fun AppUpdateOverlay(
    state: AppUpdateState,
    canInstallUnknownApps: () -> Boolean,
    onDownload: (AppUpdateInfo) -> Unit,
    onInstall: (Uri) -> Unit,
    onRequestInstallPermission: () -> Unit,
    onDismiss: () -> Unit,
) {
    if (state is AppUpdateState.Idle) return

    val dismissible = when (state) {
        is AppUpdateState.Available -> !state.info.mandatory
        is AppUpdateState.ReadyToInstall -> !state.info.mandatory
        is AppUpdateState.Downloading -> false
        is AppUpdateState.Failed -> true
        AppUpdateState.Idle -> true
    }

    Dialog(
        onDismissRequest = { if (dismissible) onDismiss() },
        properties = DialogProperties(dismissOnBackPress = dismissible, dismissOnClickOutside = dismissible),
    ) {
        LinkUpCard(modifier = Modifier.fillMaxWidth()) {
            Text(
                stringResource(R.string.app_update_available_title),
                color = LinkUpTextPrimary,
                fontWeight = FontWeight.Bold,
                fontSize = 18.sp,
            )
            Spacer(Modifier.height(10.dp))

            when (state) {
                is AppUpdateState.Available -> {
                    Text(
                        stringResource(
                            if (state.info.mandatory) R.string.app_update_mandatory_body else R.string.app_update_available_body,
                            state.info.versionName,
                        ),
                        color = LinkUpTextDimmed,
                        fontSize = 13.sp,
                    )
                    Spacer(Modifier.height(16.dp))
                    UpdateActionRow(dismissible, onDismiss) {
                        LinkUpButton(
                            stringResource(R.string.app_update_download),
                            { onDownload(state.info) },
                            variant = LinkUpButtonVariant.PRIMARY,
                        )
                    }
                }

                is AppUpdateState.Downloading -> {
                    Row(verticalAlignment = androidx.compose.ui.Alignment.CenterVertically) {
                        CircularProgressIndicator(color = LinkUpRed, modifier = Modifier.width(18.dp).height(18.dp), strokeWidth = 2.dp)
                        Spacer(Modifier.width(10.dp))
                        Text(stringResource(R.string.app_update_downloading), color = LinkUpTextDimmed, fontSize = 13.sp)
                    }
                }

                is AppUpdateState.ReadyToInstall -> {
                    if (!canInstallUnknownApps()) {
                        Text(stringResource(R.string.app_update_allow_unknown_sources_body), color = LinkUpTextDimmed, fontSize = 13.sp)
                        Spacer(Modifier.height(16.dp))
                    }
                    UpdateActionRow(dismissible, onDismiss) {
                        LinkUpButton(
                            stringResource(
                                if (canInstallUnknownApps()) R.string.app_update_install else R.string.app_update_open_settings,
                            ),
                            { if (canInstallUnknownApps()) onInstall(state.apkUri) else onRequestInstallPermission() },
                            variant = LinkUpButtonVariant.PRIMARY,
                        )
                    }
                }

                is AppUpdateState.Failed -> {
                    Text(stringResource(R.string.app_update_failed, state.message), color = LinkUpWarning, fontSize = 13.sp)
                    Spacer(Modifier.height(16.dp))
                    LinkUpButton(
                        stringResource(R.string.common_close),
                        onDismiss,
                        modifier = Modifier.fillMaxWidth(),
                        variant = LinkUpButtonVariant.SECONDARY,
                    )
                }

                AppUpdateState.Idle -> Unit
            }
        }
    }
}

@Composable
private fun UpdateActionRow(dismissible: Boolean, onDismiss: () -> Unit, primary: @Composable () -> Unit) {
    Row(horizontalArrangement = Arrangement.spacedBy(10.dp), modifier = Modifier.fillMaxWidth()) {
        if (dismissible) {
            LinkUpButton(
                stringResource(R.string.app_update_dismiss),
                onDismiss,
                modifier = Modifier.weight(1f),
                variant = LinkUpButtonVariant.SECONDARY,
            )
        }
        Column(modifier = if (dismissible) Modifier.weight(1f) else Modifier.fillMaxWidth()) {
            primary()
        }
    }
}
