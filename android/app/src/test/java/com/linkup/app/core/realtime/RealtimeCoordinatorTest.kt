package com.linkup.app.core.realtime

import com.linkup.app.core.network.RealtimeApi
import com.linkup.app.core.network.RealtimeBatchModel
import com.linkup.app.core.network.RealtimeEventModel
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertFalse
import kotlin.test.assertTrue

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

    private class FakeRealtimeApi(private val batch: RealtimeBatchModel) : RealtimeApi {
        val afters = mutableListOf<Long>()
        override suspend fun pullRealtime(after: Long, limit: Int): RealtimeBatchModel {
            afters += after
            return batch
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

    private fun runSuspend(block: suspend () -> Unit) = kotlinx.coroutines.test.runTest { block() }

    private companion object {
        const val USER_ID = "00000000-0000-0000-0000-000000000010"
        const val SLOT_ID = "00000000-0000-0000-0000-000000000020"
    }
}
