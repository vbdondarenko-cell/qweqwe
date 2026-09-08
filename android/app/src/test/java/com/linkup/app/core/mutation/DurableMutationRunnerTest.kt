package com.linkup.app.core.mutation

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertNull
import kotlin.test.assertTrue
import kotlinx.coroutines.test.runTest

class DurableMutationRunnerTest {
    @Test
    fun `submit persists before attempt and removes only after acknowledgement`() = runTest {
        val outbox = MemoryOutbox()
        val runner = DurableMutationRunner(outbox) { 2_000L }
        val command = command("00000000-0000-0000-0000-000000000001", createdAt = 1_000L)
        var sawPersistedAttempt = false

        val result = runner.submit(command) { attempted ->
            sawPersistedAttempt = outbox.items.single().firstAttemptAtEpochMillis == 2_000L
            assertEquals(attempted.idempotencyKey, outbox.items.single().idempotencyKey)
            DurableAttemptResult(DurableAttemptDisposition.ACKNOWLEDGED, responseJson = "{}")
        }

        assertTrue(sawPersistedAttempt)
        assertEquals(DurableAttemptDisposition.ACKNOWLEDGED, result.disposition)
        assertTrue(outbox.items.isEmpty())
    }

    @Test
    fun `ambiguous submit remains queued with original first attempt time`() = runTest {
        val outbox = MemoryOutbox()
        val runner = DurableMutationRunner(outbox) { 2_000L }
        val command = command("00000000-0000-0000-0000-000000000001", createdAt = 1_000L)

        runner.submit(command) { DurableAttemptResult(DurableAttemptDisposition.AMBIGUOUS_FAILURE) }

        assertEquals(1, outbox.items.size)
        assertEquals(2_000L, outbox.items.single().firstAttemptAtEpochMillis)
    }

    @Test
    fun `replay stops in causal order after ambiguous failure`() = runTest {
        val outbox = MemoryOutbox().apply {
            enqueue(command("00000000-0000-0000-0000-000000000001", createdAt = 1_000L))
            enqueue(command("00000000-0000-0000-0000-000000000002", createdAt = 2_000L))
        }
        val runner = DurableMutationRunner(outbox) { 3_000L }
        val attempted = mutableListOf<String>()

        val report = runner.replay(OWNER) { command ->
            attempted += command.idempotencyKey
            DurableAttemptResult(DurableAttemptDisposition.AMBIGUOUS_FAILURE)
        }

        assertEquals(listOf("00000000-0000-0000-0000-000000000001"), attempted)
        assertEquals("00000000-0000-0000-0000-000000000001", report.ambiguousFailureKey)
        assertEquals(2, outbox.items.size)
    }

    @Test
    fun `unsafe expired ambiguity blocks later automatic replay`() = runTest {
        val old = command("00000000-0000-0000-0000-000000000001", createdAt = 1_000L)
            .markAttempt(2_000L)
        val later = command("00000000-0000-0000-0000-000000000002", createdAt = 3_000L)
        val outbox = MemoryOutbox().apply { enqueue(old); enqueue(later) }
        val runner = DurableMutationRunner(outbox) { 2_001L + DURABLE_MUTATION_REPLAY_WINDOW_MS }
        var transportCalled = false

        val report = runner.replay(OWNER) {
            transportCalled = true
            DurableAttemptResult(DurableAttemptDisposition.ACKNOWLEDGED)
        }

        assertFalse(transportCalled)
        assertEquals(1, report.unsafeAmbiguousCount)
        assertFalse(report.completed)
        assertNull(report.definitiveFailureKey)
    }

    @Test
    fun `definitive replay failure is removed but stops dependent commands`() = runTest {
        val outbox = MemoryOutbox().apply {
            enqueue(command("00000000-0000-0000-0000-000000000001", createdAt = 1_000L))
            enqueue(command("00000000-0000-0000-0000-000000000002", createdAt = 2_000L))
        }
        val runner = DurableMutationRunner(outbox) { 3_000L }

        val report = runner.replay(OWNER) {
            DurableAttemptResult(DurableAttemptDisposition.DEFINITIVE_FAILURE, errorCode = "conflict")
        }

        assertEquals("00000000-0000-0000-0000-000000000001", report.definitiveFailureKey)
        assertEquals(listOf("00000000-0000-0000-0000-000000000002"), outbox.items.map { it.idempotencyKey })
    }

    private class MemoryOutbox : MutationOutbox {
        val items = mutableListOf<DurableMutationCommand>()

        override fun enqueue(command: DurableMutationCommand) {
            if (items.none { it.idempotencyKey == command.idempotencyKey }) items += command
        }

        override fun markAttempt(idempotencyKey: String, nowEpochMillis: Long): DurableMutationCommand? {
            val index = items.indexOfFirst { it.idempotencyKey == idempotencyKey }
            if (index < 0) return null
            items[index] = items[index].markAttempt(nowEpochMillis)
            return items[index]
        }

        override fun remove(idempotencyKey: String) {
            items.removeAll { it.idempotencyKey == idempotencyKey }
        }

        override fun pendingReplayable(ownerFingerprint: String, nowEpochMillis: Long): List<DurableMutationCommand> =
            items.filter { it.ownerFingerprint == ownerFingerprint && it.canAutoReplay(nowEpochMillis) }
                .sortedBy { it.createdAtEpochMillis }

        override fun unsafeAmbiguousCount(ownerFingerprint: String, nowEpochMillis: Long): Int =
            items.count { it.ownerFingerprint == ownerFingerprint && it.firstAttemptAtEpochMillis != null && !it.canAutoReplay(nowEpochMillis) }

        override fun clearOwner(ownerFingerprint: String) {
            items.removeAll { it.ownerFingerprint == ownerFingerprint }
        }

        override fun clearAll() {
            items.clear()
        }
    }

    private fun command(key: String, createdAt: Long) = DurableMutationCommand(
        idempotencyKey = key,
        ownerFingerprint = OWNER,
        method = "POST",
        path = "/v1/slots/00000000-0000-0000-0000-000000000099/request",
        bodyJson = null,
        responseKind = DurableResponseKind.SLOT,
        createdAtEpochMillis = createdAt,
    )

    private companion object {
        const val OWNER = "a".repeat(64)
    }
}
