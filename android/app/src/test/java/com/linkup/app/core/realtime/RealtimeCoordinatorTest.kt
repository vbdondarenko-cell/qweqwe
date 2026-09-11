package com.linkup.app.core.realtime

import com.linkup.app.core.network.RealtimeApi
import com.linkup.app.core.network.RealtimeBatchModel
import com.linkup.app.core.network.RealtimeEventModel
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertFalse
import kotlin.test.assertTrue
import kotlinx.coroutines.runBlocking

class RealtimeCoordinatorTest {
    @Test
    fun `cursor is durable only after caller acknowledges reconciliation`() = runSuspend {
        val cursors = MemoryCursorStore()
        val api = FakeRealtimeApi(
            RealtimeBatchModel(
                cursor = 12,
                events = listOf(event(12, "slot.updated", SLOT_ID)),
            ),
        )
        val coordinator = RealtimeCoordinator(api, cursors)

        val pulled = coordinator.pull(USER_ID)
        assertEquals(0, pulled.fromCursor)
        assertEquals(12, pulled.nextCursor)
        assertEquals(0, cursors.load(USER_ID))
        assertTrue(pulled.hints.refreshPulse)
        assertEquals(setOf(SLOT_ID), pulled.hints.slotIds)

        coordinator.acknowledge(USER_ID, pulled.nextCursor)
        assertEquals(12, cursors.load(USER_ID))
    }

    @Test
    fun `process death before acknowledgement replays from durable cursor`() = runSuspend {
        val cursors = MemoryCursorStore().apply { save(USER_ID, 7) }
        val api = FakeRealtimeApi(RealtimeBatchModel(9, listOf(event(9, "slot.state_changed", SLOT_ID))))
        val first = RealtimeCoordinator(api, cursors)

        val unacknowledged = first.pull(USER_ID)
        assertEquals(7, unacknowledged.fromCursor)
        assertEquals(7, cursors.load(USER_ID))

        val afterProcessDeath = RealtimeCoordinator(api, cursors).pull(USER_ID)
        assertEquals(7, afterProcessDeath.fromCursor)
        assertEquals(listOf(7L, 7L), api.afters)
    }

    @Test
    fun `coordinator refuses acknowledging cursor never offered by server`() = runSuspend {
        val coordinator = RealtimeCoordinator(
            FakeRealtimeApi(RealtimeBatchModel(5, emptyList())),
            MemoryCursorStore(),
        )
        coordinator.pull(USER_ID)
        assertFailsWith<IllegalArgumentException> { coordinator.acknowledge(USER_ID, 6) }
    }

    @Test
    fun `private event hints refresh only relevant canonical surfaces`() {
        val hints = hintsFor(
            listOf(
                event(1, "slot.chat_message_created", SLOT_ID),
                event(2, "slot.membership_removed", SLOT_ID),
                event(3, "user.block_created", null),
            ),
        )
        assertTrue(hints.refreshPulse)
        assertTrue(hints.refreshRelationships)
        assertTrue(hints.refreshProfile)
        assertEquals(setOf(SLOT_ID), hints.slotIds)
        assertEquals(setOf(SLOT_ID), hints.chatSlotIds)
    }

    @Test
    fun `bootstrapIfNeeded adopts server cursor for a never-synced user without pulling history`() = runSuspend {
        val cursors = MemoryCursorStore()
        val api = FakeRealtimeApi(RealtimeBatchModel(0, emptyList()), cursor = 500)
        val coordinator = RealtimeCoordinator(api, cursors)

        coordinator.bootstrapIfNeeded(USER_ID)

        assertEquals(500, cursors.load(USER_ID))
        assertEquals(1, api.cursorCalls)
        assertTrue(api.afters.isEmpty())
    }

    @Test
    fun `bootstrapIfNeeded is a no-op once a cursor already exists`() = runSuspend {
        val cursors = MemoryCursorStore().apply { save(USER_ID, 7) }
        val api = FakeRealtimeApi(RealtimeBatchModel(0, emptyList()), cursor = 999)
        val coordinator = RealtimeCoordinator(api, cursors)

        coordinator.bootstrapIfNeeded(USER_ID)

        assertEquals(7, cursors.load(USER_ID))
        assertEquals(0, api.cursorCalls)
    }

    @Test
    fun `a real event after bootstrap is still reached going forward`() = runSuspend {
        val cursors = MemoryCursorStore()
        val api = FakeRealtimeApi(RealtimeBatchModel(51, listOf(event(51, "slot.updated", SLOT_ID))), cursor = 50)
        val coordinator = RealtimeCoordinator(api, cursors)

        coordinator.bootstrapIfNeeded(USER_ID)
        val pulled = coordinator.pull(USER_ID)

        assertEquals(50, pulled.fromCursor)
        assertEquals(51, pulled.nextCursor)
        assertEquals(listOf(50L), api.afters)
    }

    @Test
    fun `empty batch produces no invalidation work`() = runSuspend {
        val coordinator = RealtimeCoordinator(
            FakeRealtimeApi(RealtimeBatchModel(25, emptyList())),
            MemoryCursorStore(),
        )
        val pulled = coordinator.pull(USER_ID)
        assertFalse(pulled.hasChanges)
        assertFalse(pulled.hints.refreshPulse)
        assertTrue(pulled.hints.slotIds.isEmpty())
        coordinator.acknowledge(USER_ID, 25)
    }

    private class FakeRealtimeApi(
        private val batch: RealtimeBatchModel,
        private val cursor: Long = 0,
    ) : RealtimeApi {
        val afters = mutableListOf<Long>()
        var cursorCalls = 0
            private set
        override suspend fun pullRealtime(after: Long, limit: Int): RealtimeBatchModel {
            afters += after
            return batch
        }
        override suspend fun currentCursor(): Long {
            cursorCalls++
            return cursor
        }
    }

    private class MemoryCursorStore : RealtimeCursorStore {
        private val values = mutableMapOf<String, Long>()
        override fun load(userId: String): Long = values[userId] ?: 0L
        override fun save(userId: String, cursor: Long) {
            values[userId] = maxOf(values[userId] ?: 0L, cursor)
        }
        override fun clear(userId: String) {
            values.remove(userId)
        }
        override fun hasSynced(userId: String): Boolean = values.containsKey(userId)
    }

    private fun event(sequence: Long, type: String, slotId: String?) = RealtimeEventModel(
        sequence = sequence,
        eventId = "event-$sequence",
        eventType = type,
        aggregateType = "slot",
        aggregateId = SLOT_ID,
        subjectUserId = null,
        slotId = slotId,
        payloadJson = "{}",
        occurredAtEpochMillis = sequence,
    )

    private fun runSuspend(block: suspend () -> Unit) = runBlocking { block() }

    private companion object {
        const val USER_ID = "00000000-0000-0000-0000-000000000010"
        const val SLOT_ID = "00000000-0000-0000-0000-000000000020"
    }
}
