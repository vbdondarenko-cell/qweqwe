package com.linkup.app.core.network

import java.time.LocalDate
import java.time.Period

internal enum class OnboardingDraftIssue {
    EMAIL,
    USERNAME,
    DISPLAY_NAME,
    PASSWORD,
    BIRTH_DATE,
    CITY,
    TIME_PREFERENCES,
    PEOPLE_PREFERENCES,
    GOAL_PREFERENCES,
    TEEN_ROMANTIC,
}

private val allowedTime = setOf(
    "walks", "cafes", "sports", "games", "arts_culture",
    "nightlife", "travel", "networking", "unsure",
)
private val allowedPeople = setOf(
    "new_people", "friends", "romantic", "groups", "anyone", "unsure",
)
private val allowedGoals = setOf(
    "new_friends", "romantic", "activities_events", "networking", "browsing", "unsure",
)

internal fun validateOnboardingDraft(
    draft: OnboardingRegistrationDraft,
    today: LocalDate = LocalDate.now(),
): OnboardingDraftIssue? {
    val email = draft.email.trim()
    val username = draft.username.trim().lowercase()
    val displayName = draft.displayName.trim()
    val cityName = draft.cityName.trim()

    if (!validOnboardingEmail(email)) return OnboardingDraftIssue.EMAIL
    if (!Regex("^[a-z0-9_.]{3,32}$").matches(username)) return OnboardingDraftIssue.USERNAME
    if (displayName.codePointCount(0, displayName.length) !in 1..80) return OnboardingDraftIssue.DISPLAY_NAME
    if (draft.password.toByteArray(Charsets.UTF_8).size !in 8..1024) return OnboardingDraftIssue.PASSWORD

    val birthDate = runCatching { LocalDate.parse(draft.birthDate.trim()) }.getOrNull()
        ?: return OnboardingDraftIssue.BIRTH_DATE
    val age = Period.between(birthDate, today).years
    if (age !in 14..120) return OnboardingDraftIssue.BIRTH_DATE

    if (cityName.codePointCount(0, cityName.length) !in 1..160) return OnboardingDraftIssue.CITY
    if (draft.preferences.time.isEmpty() || !allowedTime.containsAll(draft.preferences.time)) {
        return OnboardingDraftIssue.TIME_PREFERENCES
    }
    if (draft.preferences.people.isEmpty() || !allowedPeople.containsAll(draft.preferences.people)) {
        return OnboardingDraftIssue.PEOPLE_PREFERENCES
    }
    if (draft.preferences.goals.isEmpty() || !allowedGoals.containsAll(draft.preferences.goals)) {
        return OnboardingDraftIssue.GOAL_PREFERENCES
    }
    if (age < 18 && ("romantic" in draft.preferences.people || "romantic" in draft.preferences.goals)) {
        return OnboardingDraftIssue.TEEN_ROMANTIC
    }
    return null
}

private fun validOnboardingEmail(value: String): Boolean {
    if (value.length !in 3..320 || value.any(Char::isWhitespace)) return false
    val at = value.lastIndexOf('@')
    return at > 0 && at < value.length - 3 && value.substring(at + 1).contains('.')
}
