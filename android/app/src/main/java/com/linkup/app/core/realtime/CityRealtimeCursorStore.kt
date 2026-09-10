package com.linkup.app.core.realtime

import android.content.Context
import java.util.UUID

interface CityRealtimeCursorStore {
    fun load(userId: String): Long
    fun save(userId: String, cursor: Long)
    fun clear(userId: String)
}

class SharedPreferencesCityRealtimeCursorStore(context: Context) : CityRealtimeCursorStore {
    private val preferences = context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    override fun load(userId: String): Long = preferences.getLong(userKey(userId), 0L).coerceAtLeast(0L)

    override fun save(userId: String, cursor: Long) {
        require(cursor >= 0)
        val key = userKey(userId)
        val current = preferences.getLong(key, 0L).coerceAtLeast(0L)
        if (cursor <= current) return
        check(preferences.edit().putLong(key, cursor).commit()) { "failed to persist city realtime cursor" }
    }

    override fun clear(userId: String) {
        check(preferences.edit().remove(userKey(userId)).commit()) { "failed to clear city realtime cursor" }
    }

    private fun userKey(userId: String): String = "cursor_${UUID.fromString(userId.trim())}"

    private companion object {
        const val PREFS_NAME = "linkup_city_realtime_cursor_v1"
    }
}
