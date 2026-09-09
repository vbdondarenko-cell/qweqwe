package com.linkup.app.core.network

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertFailsWith
import kotlin.test.assertTrue
import org.json.JSONObject

class OnboardingModelsTest {
    private val token = "a".repeat(43)

    @Test
    fun parsesCanonicalStartAndStatus() {
        val start = parseOnboardingStart(JSONObject("""{
            "verificationToken":"$token",
            "telegramDeepLink":"https://t.me/LinkUpBot?start=$token",
            "expiresAt":"2030-01-01T00:00:00Z"
        }"""))
        assertEquals(token, start.verificationToken)
        assertTrue(start.expiresAtEpochMillis > 0)

        val status = parseOnboardingStatus(JSONObject("""{
            "phoneVerified":true,
            "teenMode":false,
            "expiresAt":"2030-01-01T00:00:00Z"
        }"""))
        assertTrue(status.phoneVerified)
        assertFalse(status.teenMode)
        assertEquals(null, status.completedAtEpochMillis)
    }

    @Test
    fun telegramDeepLinkIsFailClosed() {
        assertTrue(validTelegramDeepLink("https://t.me/LinkUpBot?start=$token", token))
        for (value in listOf(
            "http://t.me/LinkUpBot?start=$token",
            "https://evil.example/LinkUpBot?start=$token",
            "https://t.me.evil.example/LinkUpBot?start=$token",
            "https://t.me/LinkUpBot?start=${"b".repeat(43)}",
            "https://t.me/LinkUpBot?start=$token&x=1",
            "https://user@t.me/LinkUpBot?start=$token",
        )) {
            assertFalse(validTelegramDeepLink(value, token), value)
        }
    }

    @Test
    fun nativeTelegramDeepLinkPreservesBotAndStartPayload() {
        assertEquals(
            "tg://resolve?domain=LinkUpBot&start=$token",
            telegramNativeDeepLink("https://t.me/LinkUpBot?start=$token", token),
        )
        assertEquals(null, telegramNativeDeepLink("https://evil.example/LinkUpBot?start=$token", token))
    }

    @Test
    fun verificationCommandDeepLinkPrefillsCanonicalStartCommand() {
        assertEquals(
            "tg://resolve?domain=LinkUpBot&text=%2Fstart%20$token",
            telegramVerificationCommandDeepLink("https://t.me/LinkUpBot?start=$token", token),
        )
        assertEquals(null, telegramVerificationCommandDeepLink("https://evil.example/LinkUpBot?start=$token", token))
    }

    @Test
    fun startRejectsNonCanonicalTokenLength() {
        assertFailsWith<IllegalArgumentException> {
            parseOnboardingStart(JSONObject("""{
                "verificationToken":"short",
                "telegramDeepLink":"https://t.me/LinkUpBot?start=short",
                "expiresAt":"2030-01-01T00:00:00Z"
            }"""))
        }
    }

    @Test
    fun preferencesSerializeDeterministically() {
        val json = onboardingPreferencesJson(
            OnboardingPreferences(
                time = setOf("sports", "walks"),
                people = setOf("friends"),
                goals = setOf("networking", "new_friends"),
            ),
        )
        assertEquals("[\"sports\",\"walks\"]", json.getJSONArray("time").toString())
        assertEquals("[\"friends\"]", json.getJSONArray("people").toString())
        assertEquals("[\"networking\",\"new_friends\"]", json.getJSONArray("goals").toString())
    }
}
