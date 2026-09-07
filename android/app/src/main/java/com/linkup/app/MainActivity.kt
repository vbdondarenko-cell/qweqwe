package com.linkup.app

import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.core.network.LinkUpApiClient
import com.linkup.app.core.network.passwordResetToken
import com.linkup.app.core.session.SecureSessionStore
import com.linkup.app.core.session.SessionCoordinator
import com.linkup.app.core.social.SocialCoordinator
import com.linkup.app.ui.LinkUpApp
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpTheme

class MainActivity : ComponentActivity() {
    // Password reset credentials are intentionally process-memory only.
    private var pendingResetToken by mutableStateOf<String?>(null)

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        captureResetToken(intent)

        val apiBaseUrl = BuildConfig.LINKUP_API_BASE_URL.trim()
        if (apiBaseUrl.isBlank()) {
            setContent {
                LinkUpTheme {
                    Column(
                        Modifier.fillMaxSize().padding(28.dp),
                        verticalArrangement = Arrangement.Center,
                        horizontalAlignment = Alignment.CenterHorizontally,
                    ) {
                        Text(stringResource(R.string.config_required_title), color = LinkUpTextPrimary, fontSize = 20.sp, fontWeight = FontWeight.Bold)
                        Text(
                            stringResource(R.string.config_required_body),
                            color = LinkUpTextDimmed,
                            modifier = Modifier.padding(top = 10.dp),
                        )
                        Text(stringResource(R.string.config_fail_closed), color = LinkUpRed, modifier = Modifier.padding(top = 10.dp), fontWeight = FontWeight.Bold)
                    }
                }
            }
            return
        }

        val sessionStore = SecureSessionStore(applicationContext)
        val api = LinkUpApiClient(apiBaseUrl, sessionStore)
        val sessionCoordinator = SessionCoordinator(api, sessionStore)
        val socialCoordinator = SocialCoordinator(api)

        setContent {
            LinkUpTheme {
                LinkUpApp(
                    lifecycle = lifecycle,
                    api = api,
                    sessions = sessionCoordinator,
                    social = socialCoordinator,
                    resetToken = pendingResetToken,
                    onResetTokenConsumed = { pendingResetToken = null },
                )
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        captureResetToken(intent)
    }

    private fun captureResetToken(intent: Intent?) {
        if (intent?.action != Intent.ACTION_VIEW) return
        val token = passwordResetToken(intent.dataString.orEmpty()) ?: return
        pendingResetToken = token
        // Drop the URI reference after extracting the credential so it is not retained by Activity intent state.
        intent.data = null
    }
}
