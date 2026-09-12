package com.linkup.app.ui.update

import android.net.Uri
import androidx.compose.foundation.layout.Column
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.res.stringResource
import com.linkup.app.R
import com.linkup.app.core.network.AppUpdateInfo
import com.linkup.app.core.update.AppUpdateState

/**
 * A minimal, always-explicit surface for the OTA update flow: every step
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
    when (state) {
        is AppUpdateState.Available -> AlertDialog(
            onDismissRequest = { if (!state.info.mandatory) onDismiss() },
            title = { Text(stringResource(R.string.app_update_available_title)) },
            text = {
                Text(
                    if (state.info.mandatory) {
                        stringResource(R.string.app_update_mandatory_body, state.info.versionName)
                    } else {
                        stringResource(R.string.app_update_available_body, state.info.versionName)
                    },
                )
            },
            confirmButton = {
                TextButton(onClick = { onDownload(state.info) }) {
                    Text(stringResource(R.string.app_update_download))
                }
            },
            dismissButton = if (state.info.mandatory) null else {
                { TextButton(onClick = onDismiss) { Text(stringResource(R.string.app_update_dismiss)) } }
            },
        )

        is AppUpdateState.Downloading -> AlertDialog(
            onDismissRequest = {},
            title = { Text(stringResource(R.string.app_update_available_title)) },
            text = { Text(stringResource(R.string.app_update_downloading)) },
            confirmButton = {},
        )

        is AppUpdateState.ReadyToInstall -> AlertDialog(
            onDismissRequest = { if (!state.info.mandatory) onDismiss() },
            title = { Text(stringResource(R.string.app_update_available_title)) },
            text = {
                Column {
                    if (!canInstallUnknownApps()) {
                        Text(stringResource(R.string.app_update_allow_unknown_sources_body))
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = {
                    if (canInstallUnknownApps()) onInstall(state.apkUri) else onRequestInstallPermission()
                }) {
                    Text(
                        if (canInstallUnknownApps()) {
                            stringResource(R.string.app_update_install)
                        } else {
                            stringResource(R.string.app_update_open_settings)
                        },
                    )
                }
            },
            dismissButton = if (state.info.mandatory) null else {
                { TextButton(onClick = onDismiss) { Text(stringResource(R.string.app_update_dismiss)) } }
            },
        )

        is AppUpdateState.Failed -> AlertDialog(
            onDismissRequest = onDismiss,
            title = { Text(stringResource(R.string.app_update_available_title)) },
            text = { Text(stringResource(R.string.app_update_failed, state.message)) },
            confirmButton = { TextButton(onClick = onDismiss) { Text(stringResource(R.string.common_close)) } },
        )

        AppUpdateState.Idle -> Unit
    }
}
