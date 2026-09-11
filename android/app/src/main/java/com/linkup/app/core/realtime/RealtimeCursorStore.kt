package com.linkup.app.core.realtime

import android.content.Context
import java.util.UUID

interface RealtimeCursorStore {
    fun load(userId: String): Long
    fun save(userId: String, cursor: Long)
    fun clear(userId: String)

    // hasSynced distinguishes "never persisted a cursor for this user"
    // from "explicitly persisted at 0" — load() alone cannot, since its
    // default and a genuine zero read the same. RealtimeCoordinator uses
    // this to bootstrap a brand-new/cleared client straight to the
    // channel's current position instead of starting from 0 (see
    // RealtimeCoordinator.bootstrapIfNeeded).
    fun hasSynced(userId: String): Boolean
}

class SharedPreferencesRealtimeCursorStore(context: Context) : RealtimeCursorStore {
    private val preferences = context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    override fun load(userId: String): Long {
        val key = userKey(userId)
        return preferences.getLong(key, 0L).coerceAtLeast(0L)
    }

    override fun hasSynced(userId: String): Boolean = preferences.contains(userKey(userId))

    override fun save(userId: String, cursor: Long) {
        require(cursor >= 0)
        val key = userKey(userId)
        val current = preferences.getLong(key, 0L).coerceAtLeast(0L)
        if (cursor <= current) return
        check(preferences.edit().putLong(key, cursor).commit()) { "failed to persist realtime cursor" }
    }

    override fun clear(userId: String) {
        check(preferences.edit().remove(userKey(userId)).commit()) { "failed to clear realtime cursor" }
    }

    private fun userKey(userId: String): String = "cursor_${UUID.fromString(userId.trim())}"

    private companion object {
        const val PREFS_NAME = "linkup_realtime_cursor_v1"
    }
}
