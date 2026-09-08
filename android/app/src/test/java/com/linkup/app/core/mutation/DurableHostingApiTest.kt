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
import kotlin.test.assertTrue
import kotlinx.coroutines.test.runTest

class DurableHostingApiTest {
    @Test
    fun `draft create uses explicit v1_1 endpoint and survives durable contract`() = runTest {
        val outbox = MemoryOutbox()
        var observed: DurableMutationCommand? = null
        val api = DurableSocialApi(
            delegate = UnusedSocial,
            currentBearerToken = { TOKEN },
            runner = DurableMutationRunner(outbox) { 2_000L },
            transport = DurableMutationTransport { command ->
                observed = command
                DurableAttemptResult(
                    disposition = DurableAttemptDisposition.ACKNOWLEDGED,
                    responseJson = slotJson("DRAFT", 1),
                    httpStatus = 201,
                )
            },
        )

        val out = api.createDraft(CreateSlotInput("Coffee", "coffee", placeText = "Center", capacity = 4))

        assertEquals("DRAFT", out.state.name)
        assertEquals("/v1/slots/drafts", observed?.path)
        assertEquals("POST", observed?.method)
        assertTrue(outbox.items.isEmpty())
    }

    @Test
    fun `publish carries optimistic version through same durable write path`() = runTest {
        val outbox = MemoryOutbox()
        var observed: DurableMutationCommand? = null
        val api = DurableSocialApi(
            delegate = UnusedSocial,
            currentBearerToken = { TOKEN },
            runner = DurableMutationRunner(outbox) { 3_000L },
            transport = DurableMutationTransport { command ->
                observed = command
                DurableAttemptResult(
                    disposition = DurableAttemptDisposition.ACKNOWLEDGED,
                    responseJson = slotJson("FILLING", 8),
                    httpStatus = 200,
                )
            },
        )

        val out = api.publishDraft(SLOT_ID, 7)

        assertEquals("/v1/slots/$SLOT_ID/publish", observed?.path)
        assertTrue(observed!!.bodyJson!!.contains("\"expectedVersion\":7"))
        assertEquals(8, out.version)
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
        override fun remove(idempotencyKey: String) { items.removeAll { it.idempotencyKey == idempotencyKey } }
        override fun pendingReplayable(ownerFingerprint: String, nowEpochMillis: Long) =
            items.filter { it.ownerFingerprint == ownerFingerprint && it.canAutoReplay(nowEpochMillis) }
        override fun unsafeAmbiguousCount(ownerFingerprint: String, nowEpochMillis: Long) =
            items.count { it.ownerFingerprint == ownerFingerprint && it.firstAttemptAtEpochMillis != null && !it.canAutoReplay(nowEpochMillis) }
        override fun clearOwner(ownerFingerprint: String) { items.removeAll { it.ownerFingerprint == ownerFingerprint } }
        override fun clearAll() { items.clear() }
    }

    private object UnusedSocial : SocialApi {
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

    private fun slotJson(state: String, version: Long) = """
        {
          "id":"$SLOT_ID",
          "organizer":{"id":"00000000-0000-0000-0000-000000000011","username":"host","displayName":"Host","avatarUrl":null},
          "title":"Coffee","activity":"coffee","details":null,"placeText":"Center","zoneText":null,
          "canonicalPlaceId":null,"startAt":null,"capacity":4,"acceptedCount":0,"state":"$state",
          "accessMode":"APPROVAL","visibility":"PUBLIC","viewerState":"HOST","version":$version,
          "createdAt":"2026-09-08T01:00:00Z","updatedAt":"2026-09-08T01:00:00Z"
        }
    """.trimIndent()

    private companion object {
        const val TOKEN = "opaque-token"
        const val SLOT_ID = "00000000-0000-0000-0000-000000000010"
    }
}
