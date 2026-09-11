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

    // bootstrapIfNeeded closes README §6.2's "obtain authoritative
    // snapshot/cursor" reconnect step for a client that has no persisted
    // cursor at all yet (first launch, cleared app storage, a fresh
    // reinstall). Without this, such a client's first pull() started from
    // 0 and correctly, but wastefully, worked forward through the entire
    // realtime history it has no use for — its real starting state
    // already comes from Pulse/Get/ListMine, never from replayed
    // historical deltas. Call this once before the first pull()/whenever
    // a session begins; it is a no-op once a cursor exists (including a
    // genuinely-zero one persisted by an earlier bootstrap or ack).
    // Note: RealtimeCursorStore.save only ever advances (never persists a
    // value <= what is already stored), so on a genuinely empty outbox
    // (cursor == 0, realistically only a brand-new deployment with zero
    // domain events ever) this stays a no-op and hasSynced keeps reading
    // false — the very next real event moves the live cursor above 0 and
    // the following bootstrapIfNeeded call then persists normally. Stated
    // here rather than special-cased: it costs at most a few redundant
    // network calls on a database with no activity at all yet, never an
    // incorrect cursor.
    suspend fun bootstrapIfNeeded(userId: String) = mutex.withLock {
        val normalizedUserId = normalizedUserId(userId)
        if (cursors.hasSynced(normalizedUserId)) return@withLock
        val cursor = api.currentCursor()
        require(cursor >= 0)
        cursors.save(normalizedUserId, cursor)
    }

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
