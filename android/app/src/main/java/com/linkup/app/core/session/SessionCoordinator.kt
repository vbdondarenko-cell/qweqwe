package com.linkup.app.core.session

import com.linkup.app.core.network.ApiException
import com.linkup.app.core.network.LinkUpApiClient
import com.linkup.app.core.network.UserProfile
import java.io.IOException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

sealed interface SessionState {
    data object Checking : SessionState
    data object SignedOut : SessionState
    data class SignedIn(val user: UserProfile) : SessionState
    data class OfflineSession(val expiresAtEpochMillis: Long) : SessionState
    data class RecoverableError(val message: String) : SessionState
}

class SessionCoordinator(
    private val api: LinkUpApiClient,
    private val sessions: SecureSessionStore,
) {
    private val mutableState = MutableStateFlow<SessionState>(SessionState.Checking)
    val state: StateFlow<SessionState> = mutableState.asStateFlow()

    suspend fun bootstrap() {
        mutableState.value = SessionState.Checking
        val local = sessions.load()
        if (local == null) {
            mutableState.value = SessionState.SignedOut
            return
        }

        try {
            mutableState.value = SessionState.SignedIn(api.me())
        } catch (error: ApiException) {
            if (error.status == 401) {
                sessions.clear()
                mutableState.value = SessionState.SignedOut
            } else {
                mutableState.value = SessionState.RecoverableError(error.message)
            }
        } catch (_: IOException) {
            // A still-valid local bearer is retained during temporary network loss.
            mutableState.value = SessionState.OfflineSession(local.expiresAtEpochMillis)
        }
    }

    suspend fun login(identifier: String, password: String, deviceLabel: String): UserProfile {
        val auth = api.login(identifier, password, deviceLabel)
        mutableState.value = SessionState.SignedIn(auth.user)
        return auth.user
    }

    suspend fun register(
        email: String,
        username: String,
        displayName: String,
        password: String,
        language: String,
        deviceLabel: String,
    ): UserProfile {
        val auth = api.register(email, username, displayName, password, language, deviceLabel)
        mutableState.value = SessionState.SignedIn(auth.user)
        return auth.user
    }

    suspend fun updateProfile(displayName: String, avatarUrl: String, visibility: String, language: String): Boolean {
        val userId = (mutableState.value as? SessionState.SignedIn)?.user?.id ?: return false
        val updated = api.updateMe(displayName, avatarUrl, visibility, language)
        // Accept only the canonical response for the account that opened the form.
        if ((mutableState.value as? SessionState.SignedIn)?.user?.id != userId || updated.id != userId) return false
        mutableState.value = SessionState.SignedIn(updated)
        return true
    }

    suspend fun logout() {
        try {
            api.logout()
        } finally {
            // The API clears local credentials even if remote revocation fails.
            mutableState.value = SessionState.SignedOut
        }
    }

    fun clearLocalSession() {
        sessions.clear()
        mutableState.value = SessionState.SignedOut
    }
}
