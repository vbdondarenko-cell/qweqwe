package com.linkup.app.core.social

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.runBlocking

class ChatPollingTest {
    @Test
    fun backsOffWithinCapResetsAfterRecoveryAndStopsOnDenial() = runBlocking {
        val outcomes = ArrayDeque(listOf(
            ChatRefreshResult.RETRY, ChatRefreshResult.RETRY, ChatRefreshResult.RETRY,
            ChatRefreshResult.SUCCESS, ChatRefreshResult.STOP,
        ))
        val pauses = mutableListOf<Long>()
        pollChat(ChatPollingPolicy(10, 30), pause = { pauses.add(it) }) { outcomes.removeFirst() }
        assertEquals(listOf(10L, 20L, 30L, 10L), pauses)
        assertEquals(0, outcomes.size)
    }

    @Test
    fun cancellationDoesNotScheduleAnotherRead() = runBlocking {
        var reads = 0
        assertFailsWith<CancellationException> {
            pollChat(pause = { throw CancellationException("background") }) {
                reads++
                ChatRefreshResult.SUCCESS
            }
        }
        assertEquals(1, reads)
    }
}
