package com.linkup.app.core.mutation

import com.linkup.app.core.network.ChatMessage
import com.linkup.app.core.network.CreateSlotInput
import com.linkup.app.core.network.EditSlotInput
import com.linkup.app.core.network.MySlotsView
import com.linkup.app.core.network.PendingSlotRequest
import com.linkup.app.core.network.SlotModel
import com.linkup.app.core.network.SlotOrganizer
import com.linkup.app.core.network.SocialApi
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertFalse
import kotlin.test.assertTrue
import kotlinx.coroutines.test.runTest

class DurableSocialApiTest {
    @Test
    fun `create slot persists exact body and idempotency key before transport`() = runTest {
        val outbox = MemoryOutbox()
        var observed: DurableMutationCommand? = null
        val api = DurableSocialApi(
            delegate = UnusedSocialApi,
            currentBearerToken = { TOKEN },
            runner = DurableMutationRunner(outbox) { 2_000L },
            transport = DurableMutationTransport { command ->
                observed = command
                assertEquals(command, outbox.items.single())
                DurableAttemptResult(
                    disposition = DurableAttemptDisposition.ACKNOWLEDGED,
                    responseJson = slotJson(),
                    httpStatus = 201,
                )
            },
        )

        val slot = api.createSlot(
            CreateSlotInput(
                title = "Coffee",
                activity = "Coffee",
                details = "Meet outside",
                placeText = "Central Cafe",
                zoneText = "Center",
                canonicalPlaceId = PLACE_ID,
                capacity = 4,
            ),
        )

        val command = requireNotNull(observed)
        assertEquals("POST", command.method)
        assertEquals("/v1/slots", command.path)
        assertEquals(DurableResponseKind.SLOT, command.responseKind)
        assertTrue(command.bodyJson!!.contains("\"canonicalPlaceId\":\"$PLACE_ID\""))
        assertTrue(requireNotNull(command.firstAttemptAtEpochMillis) >= command.createdAtEpochMillis)
        assertEquals(SLOT_ID, slot.id)
        assertTrue(outbox.items.isEmpty())
    }

    @Test
    fun `ambiguous request survives reconstruction and retries with same key`() = runTest {
        val outbox = MemoryOutbox()
        var ambiguous = true
        val transport = DurableMutationTransport { command ->
            if (ambiguous) {
                DurableAttemptResult(DurableAttemptDisposition.AMBIGUOUS_FAILURE, errorCode = "network_error")
            } else {
                DurableAttemptResult(
                    DurableAttemptDisposition.ACKNOWLEDGED,
                    responseJson = slotJson(viewerState = "PENDING"),
                    httpStatus = 200,
                )
            }
        }
        val firstProcess = DurableSocialApi(
            delegate = UnusedSocialApi,
            currentBearerToken = { TOKEN },
            runner = DurableMutationRunner(outbox) { 2_000L },
            transport = transport,
        )

        assertFailsWith<DurableMutationQueuedException> { firstProcess.requestSlot(SLOT_ID) }
        val persisted = outbox.items.single()
        val originalKey = persisted.idempotencyKey
        val originalAttemptAt = requireNotNull(persisted.firstAttemptAtEpochMillis)
        assertEquals("/v1/slots/$SLOT_ID/request", persisted.path)
        assertTrue(originalAttemptAt >= persisted.createdAtEpochMillis)

        assertFailsWith<DurableMutationQueuedException> { firstProcess.requestSlot(SLOT_ID) }
        assertEquals(1, outbox.items.size)
        assertEquals(originalKey, outbox.items.single().idempotencyKey)
        assertEquals(originalAttemptAt, outbox.items.single().firstAttemptAtEpochMillis)

        ambiguous = false
        val afterProcessDeath = DurableSocialApi(
            delegate = UnusedSocialApi,
            currentBearerToken = { TOKEN },
            runner = DurableMutationRunner(outbox) { 3_000L },
            transport = transport,
        )
        val report = requireNotNull(afterProcessDeath.replayPending())

        assertEquals(listOf(originalKey), report.acknowledgedKeys)
        assertTrue(report.completed)
        assertTrue(outbox.items.isEmpty())
    }

    @Test
    fun `expired ambiguous mutation blocks new writes instead of risking duplicate side effects`() = runTest {
        val outbox = MemoryOutbox()
        val owner = mutationOwnerFingerprint(TOKEN)
        outbox.enqueue(
            DurableMutationCommand(
                idempotencyKey = "00000000-0000-0000-0000-000000000099",
                ownerFingerprint = owner,
                method = "POST",
                path = "/v1/slots/$SLOT_ID/request",
                bodyJson = null,
                responseKind = DurableResponseKind.SLOT,
                createdAtEpochMillis = 1_000L,
            ).markAttempt(2_000L),
        )
        var transportCalled = false
        val api = DurableSocialApi(
            delegate = UnusedSocialApi,
            currentBearerToken = { TOKEN },
            runner = DurableMutationRunner(outbox) { 2_001L + DURABLE_MUTATION_REPLAY_WINDOW_MS },
            transport = DurableMutationTransport {
                transportCalled = true
                DurableAttemptResult(DurableAttemptDisposition.ACKNOWLEDGED, slotJson(), httpStatus = 200)
            },
        )

        assertFailsWith<DurableMutationSafetyException> { api.startSlot(SLOT_ID) }
        assertFalse(transportCalled)
        assertEquals(1, outbox.items.size)
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

    private object UnusedSocialApi : SocialApi {
        override suspend fun removeParticipant(slotId: String, userId: String, expectedVersion: Long): SlotModel = error("unused")
        override suspend fun acceptedParticipants(slotId: String): List<SlotOrganizer> = error("unused")
        override suspend fun mySlots(view: MySlotsView): List<SlotModel> = error("unused")
        override suspend fun pulse(): List<SlotModel> = error("unused")
        override suspend fun createSlot(input: CreateSlotInput): SlotModel = error("unused")
        override suspend fun getSlot(slotId: String): SlotModel = error("unused")
        override suspend fun editSlot(slotId: String, input: EditSlotInput): SlotModel = error("unused")
        override suspend fun cancelSlot(slotId: String, expectedVersion: Long): SlotModel = error("unused")
        override suspend fun requestSlot(slotId: String): SlotModel = error("unused")
        override suspend fun leaveSlot(slotId: String): SlotModel = error("unused")
        override suspend fun pendingRequests(slotId: String): List<PendingSlotRequest> = error("unused")
        override suspend fun approveRequest(slotId: String, userId: String): SlotModel = error("unused")
        override suspend fun rejectRequest(slotId: String, userId: String): SlotModel = error("unused")
        override suspend fun startSlot(slotId: String): SlotModel = error("unused")
        override suspend fun completeSlot(slotId: String): SlotModel = error("unused")
        override suspend fun chatMessages(slotId: String, limit: Int): List<ChatMessage> = error("unused")
        override suspend fun sendChatMessage(slotId: String, text: String): ChatMessage = error("unused")
    }

    private fun slotJson(viewerState: String = "HOST") = """
        {
          "id":"$SLOT_ID",
          "organizer":{"id":"$USER_ID","username":"owner","displayName":"Owner","avatarUrl":null},
          "title":"Coffee","activity":"Coffee","details":"Meet outside","placeText":"Central Cafe","zoneText":"Center",
          "canonicalPlaceId":"$PLACE_ID","startAt":null,"capacity":4,"acceptedCount":0,"state":"FILLING",
          "accessMode":"APPROVAL","visibility":"PUBLIC","viewerState":"$viewerState","version":1,
          "createdAt":"2026-09-08T01:00:00Z","updatedAt":"2026-09-08T01:00:00Z"
        }
    """.trimIndent()

    private companion object {
        const val TOKEN = "opaque-token"
        const val USER_ID = "00000000-0000-0000-0000-000000000010"
        const val SLOT_ID = "00000000-0000-0000-0000-000000000020"
        const val PLACE_ID = "00000000-0000-0000-0000-000000000030"
    }
}
