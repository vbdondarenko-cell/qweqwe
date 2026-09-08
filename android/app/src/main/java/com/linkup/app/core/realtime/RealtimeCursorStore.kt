package com.linkup.app.core.realtime

import android.content.Context
import java.util.UUID

interface RealtimeCursorStore {
    fun load(userId: String): Long
    fun save(userId: String, cursor: Long)
    fun clear(userId: String)
}

class SharedPreferencesRealtimeCursorStore(context: Context) : RealtimeCursorStore {
    private val preferences = context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    override fun load(userId: String): Long {
        val key = userKey(userId)
        return preferences.getLong(key, 0L).coerceAtLeast(0L)
    }

    override fun save(userId: String, cursor: Long) {
        require(cursor >= 0)
        val key = userKey(userId)
        val current = preferences.getLong(key, 0L).coerceAtLeast(0L)
        if (cursor <= current) return
        preferences.edit().putLong(key, cursor).apply()
    }

    override fun clear(userId: String) {
        preferences.edit().remove(userKey(userId)).apply()
    }

    private fun userKey(userId: String): String = "cursor_${UUID.fromString(userId.trim())}"

    private companion object {
        const val PREFS_NAME = "linkup_realtime_cursor_v1"
    }
}
