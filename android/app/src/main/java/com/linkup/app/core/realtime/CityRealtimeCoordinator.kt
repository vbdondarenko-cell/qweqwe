package com.linkup.app.core.realtime

import com.linkup.app.core.network.CityRealtimeApi
import com.linkup.app.core.network.CityRealtimeInvalidationModel
import java.util.UUID
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock

data class CityRealtimePull(
    val fromCursor: Long,
    val nextCursor: Long,
    val invalidations: List<CityRealtimeInvalidationModel>,
) {
    val hasChanges: Boolean get() = invalidations.isNotEmpty()
}

class CityRealtimeCoordinator(
    private val api: CityRealtimeApi,
    private val cursors: CityRealtimeCursorStore,
) {
    private val mutex = Mutex()
    private val offeredCursorByUser = mutableMapOf<String, Long>()

    suspend fun pull(userId: String, limit: Int = 100): CityRealtimePull = mutex.withLock {
        val normalizedUserId = normalizedUserId(userId)
        require(limit in 1..200)
        val from = cursors.load(normalizedUserId)
        val batch = api.pullCityRealtime(from, limit)
        require(batch.cursor >= from)
        offeredCursorByUser[normalizedUserId] = maxOf(offeredCursorByUser[normalizedUserId] ?: from, batch.cursor)
        CityRealtimePull(from, batch.cursor, batch.invalidations)
    }

    suspend fun acknowledge(userId: String, cursor: Long) = mutex.withLock {
        val normalizedUserId = normalizedUserId(userId)
        require(cursor >= 0)
        val persisted = cursors.load(normalizedUserId)
        if (cursor <= persisted) return@withLock
        val offered = offeredCursorByUser[normalizedUserId] ?: persisted
        require(cursor <= offered) { "cannot acknowledge an unseen city realtime cursor" }
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
