package com.linkup.app.core.realtime

import com.linkup.app.core.network.RealtimeApi
import com.linkup.app.core.network.RealtimeEventModel
import java.util.UUID
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock

data class RealtimeInvalidationHints(
    val refreshPulse: Boolean,
    val refreshProfile: Boolean,
    val refreshRelationships: Boolean,
    val slotIds: Set<String>,
    val chatSlotIds: Set<String>,
)

data class RealtimePull(
    val fromCursor: Long,
    val nextCursor: Long,
    val events: List<RealtimeEventModel>,
    val hints: RealtimeInvalidationHints,
) {
    val hasChanges: Boolean get() = events.isNotEmpty()
}

class RealtimeCoordinator(
    private val api: RealtimeApi,
    private val cursors: RealtimeCursorStore,
) {
    private val mutex = Mutex()
    private val offeredCursorByUser = mutableMapOf<String, Long>()

    suspend fun pull(userId: String, limit: Int = 100): RealtimePull = mutex.withLock {
        val normalizedUserId = normalizedUserId(userId)
        require(limit in 1..200)
        val from = cursors.load(normalizedUserId)
        val batch = api.pullRealtime(from, limit)
        require(batch.cursor >= from)
        offeredCursorByUser[normalizedUserId] = maxOf(offeredCursorByUser[normalizedUserId] ?: from, batch.cursor)
        RealtimePull(
            fromCursor = from,
            nextCursor = batch.cursor,
            events = batch.events,
            hints = hintsFor(batch.events),
        )
    }

    suspend fun acknowledge(userId: String, cursor: Long) = mutex.withLock {
        val normalizedUserId = normalizedUserId(userId)
        require(cursor >= 0)
        val persisted = cursors.load(normalizedUserId)
        if (cursor <= persisted) return@withLock
        val offered = offeredCursorByUser[normalizedUserId] ?: persisted
        require(cursor <= offered) { "cannot acknowledge an unseen realtime cursor" }
        cursors.save(normalizedUserId, cursor)
        if (cursor >= offered) offeredCursorByUser.remove(normalizedUserId)
    }

    suspend fun reset(userId: String) = mutex.withLock {
        val normalizedUserId = normalizedUserId(userId)
        offeredCursorByUser.remove(normalizedUserId)
        cursors.clear(normalizedUserId)
    }

    private fun normalizedUserId(userId: String): String = UUID.fromString(userId.trim()).toString()
}

internal fun hintsFor(events: List<RealtimeEventModel>): RealtimeInvalidationHints {
    val slotIds = linkedSetOf<String>()
    val chatSlotIds = linkedSetOf<String>()
    var refreshPulse = false
    var refreshProfile = false
    var refreshRelationships = false

    events.forEach { event ->
        event.slotId?.let(slotIds::add)
        when (event.eventType) {
            "slot.created", "slot.updated", "slot.state_changed" -> refreshPulse = true
            "slot.request_created", "slot.request_removed",
            "slot.membership_added", "slot.membership_removed" -> {
                refreshPulse = true
                refreshRelationships = true
            }
            "slot.chat_message_created" -> event.slotId?.let(chatSlotIds::add)
            "user.block_created", "user.block_removed" -> {
                refreshPulse = true
                refreshProfile = true
                refreshRelationships = true
            }
            "user.profile_changed" -> refreshProfile = true
            else -> {
                // Future server invalidations remain safe: refresh canonical snapshots
                // instead of interpreting an unknown delta as client authority.
                refreshPulse = true
                if (event.slotId != null) refreshRelationships = true
            }
        }
    }

    return RealtimeInvalidationHints(
        refreshPulse = refreshPulse,
        refreshProfile = refreshProfile,
        refreshRelationships = refreshRelationships,
        slotIds = slotIds,
        chatSlotIds = chatSlotIds,
    )
}
