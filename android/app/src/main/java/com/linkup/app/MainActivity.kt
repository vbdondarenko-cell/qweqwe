package com.linkup.app

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.linkup.app.core.network.LinkUpApiClient
import com.linkup.app.core.session.SecureSessionStore
import com.linkup.app.core.session.SessionCoordinator
import com.linkup.app.core.social.SocialCoordinator
import com.linkup.app.ui.LinkUpApp
import com.linkup.app.ui.theme.LinkUpRed
import com.linkup.app.ui.theme.LinkUpTextDimmed
import com.linkup.app.ui.theme.LinkUpTextPrimary
import com.linkup.app.ui.theme.LinkUpTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        val apiBaseUrl = BuildConfig.LINKUP_API_BASE_URL.trim()
        if (apiBaseUrl.isBlank()) {
            setContent {
                LinkUpTheme {
                    Column(
                        Modifier.fillMaxSize().padding(28.dp),
                        verticalArrangement = Arrangement.Center,
                        horizontalAlignment = Alignment.CenterHorizontally,
                    ) {
                        Text("LinkUp configuration required", color = LinkUpTextPrimary, fontSize = 20.sp, fontWeight = FontWeight.Bold)
                        Text(
                            "Release builds require the LINKUP_API_BASE_URL Gradle property. No fallback production endpoint is hardcoded.",
                            color = LinkUpTextDimmed,
                            modifier = Modifier.padding(top = 10.dp),
                        )
                        Text("Fail-closed", color = LinkUpRed, modifier = Modifier.padding(top = 10.dp), fontWeight = FontWeight.Bold)
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
                )
            }
        }
    }
}
