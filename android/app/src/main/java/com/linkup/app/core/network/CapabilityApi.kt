package com.linkup.app.core.network

enum class CapabilityKey(val wireName: String) {
    REALTIME("realtime"),
    CITY_CONTEXT("city_context"),
    MAP("map"),
    WAITLIST("waitlist"),
    CHAT_V2("chat_v2"),
    NOTIFICATIONS("notifications"),
    BUMP("bump"),
    CITY_BPM("city_bpm"),
    SWARMS("swarms"),
    FLY_NOW("fly_now"),
    FLY_TRAVEL("fly_travel"),
    FLY_MOTION("fly_motion"),
}

data class CapabilitySnapshot(
    val revision: Long,
    val values: Map<CapabilityKey, Boolean>,
) {
    fun enabled(key: CapabilityKey): Boolean = values[key] == true

    companion object {
        fun disabled(): CapabilitySnapshot = CapabilitySnapshot(
            revision = 0L,
            values = CapabilityKey.entries.associateWith { false },
        )
    }
}

interface CapabilityApi {
    suspend fun capabilities(): CapabilitySnapshot
}

internal fun parseCapabilitySnapshot(json: org.json.JSONObject): CapabilitySnapshot {
    val revision = json.optLong("revision", -1L)
    require(revision >= 0L)
    val capabilities = json.optJSONObject("capabilities") ?: org.json.JSONObject()
    val values = CapabilityKey.entries.associateWith { key ->
        capabilities.optBoolean(key.wireName, false)
    }
    return CapabilitySnapshot(revision = revision, values = values)
}
