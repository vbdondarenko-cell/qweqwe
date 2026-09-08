package com.linkup.app.core.mutation

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class DurableMutationTest {
    @Test
    fun `unattempted command remains replayable after long offline period`() {
        val command = command(createdAt = 1_000L)
        assertTrue(command.canAutoReplay(30L * 24L * 60L * 60L * 1000L))
    }

    @Test
    fun `ambiguous attempted command stops before server idempotency horizon`() {
        val attempted = command(createdAt = 1_000L).markAttempt(2_000L)
        assertTrue(attempted.canAutoReplay(2_000L + DURABLE_MUTATION_REPLAY_WINDOW_MS))
        assertFalse(attempted.canAutoReplay(2_001L + DURABLE_MUTATION_REPLAY_WINDOW_MS))
    }

    @Test
    fun `mark attempt is stable across retries`() {
        val first = command(createdAt = 1_000L).markAttempt(2_000L)
        val second = first.markAttempt(9_000L)
        assertEquals(2_000L, second.firstAttemptAtEpochMillis)
    }

    @Test
    fun `owner fingerprint is deterministic without retaining bearer`() {
        val first = mutationOwnerFingerprint("opaque-bearer-token")
        val second = mutationOwnerFingerprint("opaque-bearer-token")
        assertEquals(first, second)
        assertEquals(64, first.length)
        assertFalse(first.contains("opaque"))
    }

    private fun command(createdAt: Long) = DurableMutationCommand(
        idempotencyKey = "00000000-0000-0000-0000-000000000001",
        ownerFingerprint = "a".repeat(64),
        method = "POST",
        path = "/v1/slots/00000000-0000-0000-0000-000000000002/request",
        bodyJson = null,
        responseKind = DurableResponseKind.SLOT,
        createdAtEpochMillis = createdAt,
    )
}
