package com.linkup.app.core.push

import android.content.Context
import com.google.firebase.messaging.FirebaseMessaging
import com.linkup.app.BuildConfig
import com.linkup.app.core.network.PushApiClient
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException
import kotlinx.coroutines.suspendCancellableCoroutine

class PushCoordinator(
    context: Context,
    private val api: PushApiClient,
) {
    private val appContext = context.applicationContext
    private val store = PushInstallationStore(appContext)

    fun configure(): Boolean = FirebaseBootstrap.configure(appContext)

    suspend fun sync() {
        if (!configure()) return
        api.registerAndroid(
            installationId = store.installationId(),
            token = firebaseToken(),
            appVersion = BuildConfig.VERSION_NAME,
        )
    }

    private suspend fun firebaseToken(): String = suspendCancellableCoroutine { continuation ->
        FirebaseMessaging.getInstance().token
            .addOnSuccessListener { token -> if (continuation.isActive) continuation.resume(token) }
            .addOnFailureListener { error -> if (continuation.isActive) continuation.resumeWithException(error) }
    }
}
