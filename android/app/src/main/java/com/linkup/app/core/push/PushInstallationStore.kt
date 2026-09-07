package com.linkup.app.core.push

import android.content.Context
import java.util.UUID

class PushInstallationStore(context: Context) {
    private val prefs = context.getSharedPreferences("linkup.push.installation", Context.MODE_PRIVATE)

    fun installationId(): String {
        prefs.getString(KEY_INSTALLATION_ID, null)?.let { existing ->
            runCatching { UUID.fromString(existing) }.getOrNull()?.let { return existing.lowercase() }
        }
        val created = UUID.randomUUID().toString().lowercase()
        prefs.edit().putString(KEY_INSTALLATION_ID, created).apply()
        return created
    }

    private companion object {
        const val KEY_INSTALLATION_ID = "installation_id"
    }
}
