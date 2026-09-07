package com.linkup.app.core.push

import android.content.Context
import com.google.firebase.FirebaseApp
import com.google.firebase.FirebaseOptions
import com.linkup.app.BuildConfig

object FirebaseBootstrap {
    fun configure(context: Context): Boolean {
        val apiKey = BuildConfig.LINKUP_FIREBASE_API_KEY.trim()
        val appId = BuildConfig.LINKUP_FIREBASE_APP_ID.trim()
        val projectId = BuildConfig.LINKUP_FIREBASE_PROJECT_ID.trim()
        val senderId = BuildConfig.LINKUP_FIREBASE_SENDER_ID.trim()
        if (listOf(apiKey, appId, projectId, senderId).any { it.isBlank() }) return false
        if (FirebaseApp.getApps(context).isNotEmpty()) return true
        val options = FirebaseOptions.Builder()
            .setApiKey(apiKey)
            .setApplicationId(appId)
            .setProjectId(projectId)
            .setGcmSenderId(senderId)
            .build()
        return FirebaseApp.initializeApp(context, options) != null
    }
}
