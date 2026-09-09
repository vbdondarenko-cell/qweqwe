package com.linkup.app.core.session

import com.linkup.app.core.network.ApiException
import com.linkup.app.core.network.LinkUpApiClient
import com.linkup.app.core.network.OnboardingRegistrationDraft
import com.linkup.app.core.network.OnboardingStart
import com.linkup.app.core.network.UserProfile
import java.io.IOException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

sealed interface SessionState {
    data object Checking : SessionState
    data object SignedOut : SessionState
    data class TelegramVerification(
        val start: OnboardingStart,
        val teenMode: Boolean? = null,
        val message: String? = null,
    ) : SessionState
    data class SignedIn(val user: UserProfile) : SessionState
    data class OfflineSession(val expiresAtEpochMillis: Long) : SessionState
    data class RecoverableError(val message: String) : SessionState
}

class SessionCoordinator(
    private val api: LinkUpApiClient,
    private val sessions: SecureSessionStore,
    private val onboarding: SecureOnboardingStore,
) {
    private val mutableState = MutableStateFlow<SessionState>(SessionState.Checking)
    val state: StateFlow<SessionState> = mutableState.asStateFlow()

    init {
        api.setUnauthorizedHandler { clearLocalSession() }
    }

    suspend fun bootstrap() {
        mutableState.value = SessionState.Checking
        val local = sessions.load()
        if (local == null) {
            val pending = onboarding.load()
            if (pending == null) {
                mutableState.value = SessionState.SignedOut
            } else {
                resumePending(pending)
            }
            return
        }

        try {
            mutableState.value = SessionState.SignedIn(api.me())
        } catch (error: ApiException) {
            if (error.status == 401) clearLocalSession()
            else mutableState.value = SessionState.RecoverableError(error.message)
        } catch (_: IOException) {
            mutableState.value = SessionState.OfflineSession(local.expiresAtEpochMillis)
        }
    }

    suspend fun login(identifier: String, password: String, deviceLabel: String): UserProfile {
        val auth = api.login(identifier, password, deviceLabel)
        onboarding.clear()
        mutableState.value = SessionState.SignedIn(auth.user)
        return auth.user
    }

    suspend fun startRegistration(
        draft: OnboardingRegistrationDraft,
        language: String,
        deviceLabel: String,
    ): OnboardingStart {
        val started = api.startOnboarding(draft, language, deviceLabel)
        onboarding.save(
            SecureOnboardingStore.Pending(
                verificationToken = started.verificationToken,
                telegramDeepLink = started.telegramDeepLink,
                expiresAtEpochMillis = started.expiresAtEpochMillis,
            ),
        )
        mutableState.value = SessionState.TelegramVerification(started)
        return started
    }

    suspend fun refreshRegistration(): Boolean {
        val pending = onboarding.load() ?: run {
            mutableState.value = SessionState.SignedOut
            return false
        }
        return try {
            val status = api.onboardingStatus(pending.verificationToken)
            if (!status.phoneVerified) {
                mutableState.value = SessionState.TelegramVerification(pending.toStart(), status.teenMode)
                false
            } else {
                val auth = api.completeOnboarding(pending.verificationToken)
                onboarding.clear()
                mutableState.value = SessionState.SignedIn(auth.user)
                true
            }
        } catch (error: ApiException) {
            if (error.status == 404 || error.status == 410) {
                onboarding.clear()
                mutableState.value = SessionState.SignedOut
            } else {
                mutableState.value = SessionState.TelegramVerification(pending.toStart(), message = error.message)
            }
            false
        } catch (error: IOException) {
            mutableState.value = SessionState.TelegramVerification(pending.toStart(), message = error.message)
            false
        }
    }

    fun cancelRegistration() {
        onboarding.clear()
        mutableState.value = SessionState.SignedOut
    }

    private suspend fun resumePending(pending: SecureOnboardingStore.Pending) {
        try {
            val status = api.onboardingStatus(pending.verificationToken)
            if (status.phoneVerified) {
                val auth = api.completeOnboarding(pending.verificationToken)
                onboarding.clear()
                mutableState.value = SessionState.SignedIn(auth.user)
            } else {
                mutableState.value = SessionState.TelegramVerification(pending.toStart(), status.teenMode)
            }
        } catch (error: ApiException) {
            if (error.status == 404 || error.status == 410) {
                onboarding.clear()
                mutableState.value = SessionState.SignedOut
            } else {
                mutableState.value = SessionState.TelegramVerification(pending.toStart(), message = error.message)
            }
        } catch (error: IOException) {
            mutableState.value = SessionState.TelegramVerification(pending.toStart(), message = error.message)
        }
    }

    suspend fun updateProfile(displayName: String, avatarUrl: String, visibility: String, language: String): Boolean {
        val userId = (mutableState.value as? SessionState.SignedIn)?.user?.id ?: return false
        val updated = api.updateMe(displayName, avatarUrl, visibility, language)
        if ((mutableState.value as? SessionState.SignedIn)?.user?.id != userId || updated.id != userId) return false
        mutableState.value = SessionState.SignedIn(updated)
        return true
    }

    suspend fun refreshSignedInProfile(): Boolean {
        val userId = (mutableState.value as? SessionState.SignedIn)?.user?.id ?: return false
        return try {
            val updated = api.me()
            if ((mutableState.value as? SessionState.SignedIn)?.user?.id != userId || updated.id != userId) false
            else {
                mutableState.value = SessionState.SignedIn(updated)
                true
            }
        } catch (error: ApiException) {
            if (error.status == 401) clearLocalSession()
            false
        } catch (_: IOException) {
            false
        }
    }

    suspend fun logout() {
        try { api.logout() } finally { mutableState.value = SessionState.SignedOut }
    }

    fun clearLocalSession() {
        sessions.clear()
        mutableState.value = SessionState.SignedOut
    }
}

private fun SecureOnboardingStore.Pending.toStart() = OnboardingStart(
    verificationToken = verificationToken,
    telegramDeepLink = telegramDeepLink,
    expiresAtEpochMillis = expiresAtEpochMillis,
)
