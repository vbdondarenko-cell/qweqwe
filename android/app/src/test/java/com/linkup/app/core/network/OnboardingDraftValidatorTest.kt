package com.linkup.app.core.network

import java.time.LocalDate
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class OnboardingDraftValidatorTest {
    private val today = LocalDate.of(2026, 9, 9)

    private fun validDraft(
        password: String = "password123",
        username: String = "valid_user",
    ) = OnboardingRegistrationDraft(
        email = "user@example.com",
        username = username,
        displayName = "User",
        password = password,
        birthDate = "2000-01-01",
        cityName = "Cherkasy",
        preferences = OnboardingPreferences(
            setOf("walks"), setOf("friends"), setOf("new_friends"),
        ),
    )

    @Test fun validDraftPasses() {
        assertNull(validateOnboardingDraft(validDraft(), today))
    }
    @Test fun shortPasswordIsRejectedBeforeNetwork() {
        assertEquals(
            OnboardingDraftIssue.PASSWORD,
            validateOnboardingDraft(validDraft(password = "1234567"), today),
        )
    }

    @Test fun invalidUsernameIsRejectedBeforeNetwork() {
        assertEquals(
            OnboardingDraftIssue.USERNAME,
            validateOnboardingDraft(validDraft(username = "ab-c"), today),
        )
    }

    @Test fun teenRomanticPreferenceIsRejected() {
        val draft = validDraft().copy(
            birthDate = "2010-01-01",
            preferences = OnboardingPreferences(
                setOf("walks"), setOf("romantic"), setOf("romantic"),
            ),
        )
        assertEquals(
            OnboardingDraftIssue.TEEN_ROMANTIC,
            validateOnboardingDraft(draft, today),
        )
    }
}
