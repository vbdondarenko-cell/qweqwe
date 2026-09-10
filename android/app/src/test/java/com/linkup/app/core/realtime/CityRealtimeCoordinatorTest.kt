package com.linkup.app.core.realtime

import com.linkup.app.core.network.CityRealtimeApi
import com.linkup.app.core.network.CityRealtimeBatchModel
import com.linkup.app.core.network.CityRealtimeInvalidationModel
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertFalse
import kotlin.test.assertTrue
import kotlinx.coroutines.test.runTest

class CityRealtimeCoordinatorTest {
    @Test
    fun `city cursor is durable only after canonical reconciliation acknowledgement`() = runTest {
        val cursors = MemoryCityCursorStore()
        val api = FakeCityRealtimeApi(batch(12, listOf(invalidation(12))))
        val coordinator = CityRealtimeCoordinator(api, cursors)

        val pulled = coordinator.pull(USER_ID)

        assertEquals(0, pulled.fromCursor)
        assertEquals(12, pulled.nextCursor)
        assertTrue(pulled.hasChanges)
        assertEquals(0, cursors.load(USER_ID))

        coordinator.acknowledge(USER_ID, pulled.nextCursor)
        assertEquals(12, cursors.load(USER_ID))
    }

    @Test
    fun `process death before city acknowledgement replays from durable cursor`() = runTest {
        val cursors = MemoryCityCursorStore().apply { save(USER_ID, 5) }
        val api = FakeCityRealtimeApi(batch(8, listOf(invalidation(8))))

        CityRealtimeCoordinator(api, cursors).pull(USER_ID)
        val afterRestart = CityRealtimeCoordinator(api, cursors).pull(USER_ID)

        assertEquals(5, afterRestart.fromCursor)
        assertEquals(listOf(5L, 5L), api.afters)
    }

    @Test
    fun `coordinator refuses unseen city cursor acknowledgement`() = runTest {
        val coordinator = CityRealtimeCoordinator(
            FakeCityRealtimeApi(batch(4, emptyList())),
            MemoryCityCursorStore(),
        )
        coordinator.pull(USER_ID)
        assertFailsWith<IllegalArgumentException> { coordinator.acknowledge(USER_ID, 5) }
    }

    @Test
    fun `cursor-only progress has no canonical refresh work`() = runTest {
        val coordinator = CityRealtimeCoordinator(
            FakeCityRealtimeApi(batch(25, emptyList())),
            MemoryCityCursorStore(),
        )
        val pulled = coordinator.pull(USER_ID)
        assertFalse(pulled.hasChanges)
        coordinator.acknowledge(USER_ID, pulled.nextCursor)
        assertEquals(25, pulled.nextCursor)
    }

    private class FakeCityRealtimeApi(private val batch: CityRealtimeBatchModel) : CityRealtimeApi {
        val afters = mutableListOf<Long>()
        override suspend fun pullCityRealtime(after: Long, limit: Int): CityRealtimeBatchModel {
            afters += after
            return batch
        }
    }

    private class MemoryCityCursorStore : CityRealtimeCursorStore {
        private val values = mutableMapOf<String, Long>()
        override fun load(userId: String): Long = values[userId] ?: 0L
        override fun save(userId: String, cursor: Long) {
            values[userId] = maxOf(values[userId] ?: 0L, cursor)
        }
        override fun clear(userId: String) {
            values.remove(userId)
        }
    }

    private fun batch(cursor: Long, invalidations: List<CityRealtimeInvalidationModel>) =
        CityRealtimeBatchModel(cursor, invalidations)

    private fun invalidation(sequence: Long) = CityRealtimeInvalidationModel(
        sequence = sequence,
        eventId = "00000000-0000-0000-0000-${sequence.toString().padStart(12, '0')}",
        occurredAtEpochMillis = sequence,
    )

    private companion object {
        const val USER_ID = "00000000-0000-0000-0000-000000000010"
    }
}
