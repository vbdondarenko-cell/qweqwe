package com.linkup.app.core.network

import java.net.URI
import java.time.Instant
import org.json.JSONArray
import org.json.JSONObject

data class OnboardingPreferences(
    val time: Set<String>,
    val people: Set<String>,
    val goals: Set<String>,
)

data class OnboardingRegistrationDraft(
    val email: String,
    val username: String,
    val displayName: String,
    val password: String,
    val birthDate: String,
    val cityId: String = "",
    val cityName: String,
    val preferences: OnboardingPreferences,
)

data class OnboardingStart(
    val verificationToken: String,
    val telegramDeepLink: String,
    val expiresAtEpochMillis: Long,
)

data class OnboardingStatus(
    val phoneVerified: Boolean,
    val teenMode: Boolean,
    val expiresAtEpochMillis: Long,
    val completedAtEpochMillis: Long?,
)

internal fun parseOnboardingStart(json: JSONObject): OnboardingStart {
    val token = json.getString("verificationToken")
    val link = json.getString("telegramDeepLink")
    require(Regex("^[A-Za-z0-9_-]{43}$").matches(token))
    require(validTelegramDeepLink(link, token))
    return OnboardingStart(token, link, Instant.parse(json.getString("expiresAt")).toEpochMilli())
}

internal fun parseOnboardingStatus(json: JSONObject) = OnboardingStatus(
    phoneVerified = json.getBoolean("phoneVerified"),
    teenMode = json.getBoolean("teenMode"),
    expiresAtEpochMillis = Instant.parse(json.getString("expiresAt")).toEpochMilli(),
    completedAtEpochMillis = json.optString("completedAt").takeIf { it.isNotBlank() }?.let { Instant.parse(it).toEpochMilli() },
)

internal fun onboardingPreferencesJson(value: OnboardingPreferences) = JSONObject()
    .put("time", JSONArray(value.time.toList().sorted()))
    .put("people", JSONArray(value.people.toList().sorted()))
    .put("goals", JSONArray(value.goals.toList().sorted()))

internal fun validTelegramDeepLink(value: String, token: String): Boolean = runCatching {
    val uri = URI(value)
    uri.scheme == "https" && uri.host == "t.me" && uri.port == -1 && uri.userInfo == null &&
        uri.fragment == null && Regex("/[A-Za-z][A-Za-z0-9_]{1,31}").matches(uri.rawPath.orEmpty()) &&
        uri.rawQuery == "start=$token"
}.getOrDefault(false)

internal fun telegramNativeDeepLink(value: String, token: String): String? {
    val botUsername = telegramBotUsername(value, token) ?: return null
    return "tg://resolve?domain=$botUsername&start=$token"
}

internal fun telegramVerificationCommandDeepLink(value: String, token: String): String? {
    val botUsername = telegramBotUsername(value, token) ?: return null
    return "tg://resolve?domain=$botUsername&text=%2Fstart%20$token"
}

private fun telegramBotUsername(value: String, token: String): String? {
    if (!validTelegramDeepLink(value, token)) return null
    val path = runCatching { URI(value).rawPath }.getOrNull() ?: return null
    return path.removePrefix("/").takeIf { it.isNotBlank() }
}
