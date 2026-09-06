package com.linkup.app.ui.theme

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

val LinkUpBackground = Color(0xFF050506)
val LinkUpSurface = Color(0xFF0D0E10)
val LinkUpElevated = Color(0xFF141518)
val LinkUpZone = Color(0xFF1A1C20)
val LinkUpRed = Color(0xFFFF2D35)
val LinkUpRedSignal = Color(0xFFFF3B42)
val LinkUpRedDeep = Color(0xFF9F171E)
val LinkUpCritical = Color(0xFFEF4444)
val LinkUpSuccess = Color(0xFF22C55E)
val LinkUpWarning = Color(0xFFF59E0B)
val LinkUpInfo = Color(0xFF3B82F6)
val LinkUpTextPrimary = Color(0xFFF7F8FA)
val LinkUpTextDimmed = Color(0xFFA5A9B0)
val LinkUpTextMuted = Color(0xFF747982)
val LinkUpBorder = Color(0xFF24262B)

private val LinkUpColorScheme = darkColorScheme(
    primary = LinkUpRed,
    onPrimary = LinkUpTextPrimary,
    background = LinkUpBackground,
    onBackground = LinkUpTextPrimary,
    surface = LinkUpSurface,
    onSurface = LinkUpTextPrimary,
    error = LinkUpCritical,
    onError = LinkUpTextPrimary,
    secondary = LinkUpInfo,
    tertiary = LinkUpSuccess,
    outline = LinkUpBorder,
)

@Composable
fun LinkUpTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = LinkUpColorScheme,
        content = content,
    )
}
