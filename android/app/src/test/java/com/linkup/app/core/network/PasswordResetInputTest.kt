package com.linkup.app.core.network

import java.util.Base64
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class PasswordResetInputTest {
    private val token = Base64.getUrlEncoder().withoutPadding().encodeToString(ByteArray(32) { it.toByte() })

    @Test
    fun acceptsCanonicalTokenAndPastedHttpsLink() {
        assertEquals(token, passwordResetToken(" $token "))
        assertEquals(token, passwordResetToken("https://example.com/reset?token=$token"))
        assertEquals(token, passwordResetToken("https://example.com/reset?source=email&token=$token"))
    }

    @Test
    fun rejectsMalformedOrAmbiguousCredentials() {
        for (input in listOf(
            "", "short", "$token=", "a".repeat(43),
            "http://example.com/reset?token=$token",
            "https://example.com/reset?token=$token&token=$token",
            "https://user@example.com/reset?token=$token",
            "https://example.com/reset#token=$token",
            "https://example.com/reset?token=%ZZ", "x".repeat(4097),
        )) assertNull(passwordResetToken(input), input)
    }
}
