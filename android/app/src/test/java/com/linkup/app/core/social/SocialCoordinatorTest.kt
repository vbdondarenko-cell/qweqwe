package com.linkup.app.core.social

import com.linkup.app.core.network.ApiException
import com.linkup.app.core.network.MySlotsView
import com.linkup.app.core.network.ChatMessage
import com.linkup.app.core.network.CreateSlotInput
import com.linkup.app.core.network.EditSlotInput
import com.linkup.app.core.network.PendingSlotRequest
import com.linkup.app.core.network.SocialApi
import com.linkup.app.core.network.SlotAccessMode
import com.linkup.app.core.network.SlotModel
import com.linkup.app.core.network.SlotOrganizer
import com.linkup.app.core.network.SlotState
import com.linkup.app.core.network.SlotViewerState
import com.linkup.app.core.network.SlotVisibility
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs
import kotlin.test.assertTrue
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineStart
import kotlinx.coroutines.launch
import kotlinx.coroutines.cancelAndJoin
import kotlinx.coroutines.runBlocking

class SocialCoordinatorTest {
    @Test
    fun refreshPulseUsesServerDataAndEmptyState(): Unit = runBlocking {
        val api = FakeSocialApi()
        val coordinator = SocialCoordinator(api)

        coordinator.refreshPulse()
        assertIs<LoadState.Empty>(coordinator.pulse.value)

        api.pulseItems = listOf(slot(viewer = SlotViewerState.NONE))
        coordinator.refreshPulse()
        val content = assertIs<LoadState.Content<List<SlotModel>>>(coordinator.pulse.value)
        assertEquals("slot-1", content.value.single().id)
    }

    @Test
    fun requestUsesServerReturnedViewerState(): Unit = runBlocking {
        val api = FakeSocialApi().apply {
            mutationResult = slot(viewer = SlotViewerState.PENDING, version = 2)
        }
        val coordinator = SocialCoordinator(api)

        coordinator.requestSlot("slot-1")

        val selected = assertIs<LoadState.Content<SlotModel>>(coordinator.selectedSlot.value)
        assertEquals(SlotViewerState.PENDING, selected.value.viewerState)
        assertEquals(2, selected.value.version)
        assertIs<MutationState.Idle>(coordinator.mutation.value)
    }

    @Test
    fun terminalServerStateClearsChatState(): Unit = runBlocking {
        val api = FakeSocialApi().apply {
            messages = listOf(ChatMessage("m1", "slot-1", com.linkup.app.core.network.ChatAuthor("u", "u", "User", null), "hello", 1))
        }
        val coordinator = SocialCoordinator(api)
        coordinator.refreshChat("slot-1")
        assertIs<LoadState.Content<List<ChatMessage>>>(coordinator.chat.value)

        api.mutationResult = slot(viewer = SlotViewerState.HOST, state = SlotState.COMPLETED, version = 5)
        coordinator.completeSlot("slot-1")

        assertIs<LoadState.Idle>(coordinator.chat.value)
    }

    @Test
    fun lateChatReadCannotRestoreTerminalThread(): Unit = runBlocking {
        val response = CompletableDeferred<List<ChatMessage>>()
        val api = FakeSocialApi().apply { chatResponse = response }
        val coordinator = SocialCoordinator(api)
        val read = launch(start = CoroutineStart.UNDISPATCHED) { coordinator.refreshChat("slot-1") }
        api.mutationResult = slot(SlotViewerState.HOST, SlotState.COMPLETED)
        coordinator.completeSlot("slot-1")
        response.complete(listOf(message()))
        read.join()
        assertIs<LoadState.Idle>(coordinator.chat.value)
    }

    @Test
    fun lateResponsesCannotRestoreDisposedAccount(): Unit = runBlocking {
        val pulseResponse = CompletableDeferred<List<SlotModel>>()
        val chatResponse = CompletableDeferred<List<ChatMessage>>()
        val api = FakeSocialApi().apply {
            this.pulseResponse = pulseResponse
            this.chatResponse = chatResponse
        }
        val coordinator = SocialCoordinator(api)
        val pulse = launch(start = CoroutineStart.UNDISPATCHED) { coordinator.refreshPulse() }
        val chat = launch(start = CoroutineStart.UNDISPATCHED) { coordinator.refreshChat("slot-1") }
        coordinator.clearAll()
        pulseResponse.complete(listOf(slot(SlotViewerState.HOST)))
        chatResponse.complete(listOf(message()))
        pulse.join(); chat.join()
        assertIs<LoadState.Idle>(coordinator.pulse.value)
        assertIs<LoadState.Idle>(coordinator.chat.value)
    }

    @Test
    fun duplicateTapDoesNotQueueSecondMutation(): Unit = runBlocking {
        val response = CompletableDeferred<SlotModel>()
        val api = FakeSocialApi().apply { requestResponse = response }
        val coordinator = SocialCoordinator(api)
        val first = launch(start = CoroutineStart.UNDISPATCHED) { coordinator.requestSlot("slot-1") }
        val duplicate = launch(start = CoroutineStart.UNDISPATCHED) { coordinator.requestSlot("slot-1") }
        assertTrue(duplicate.isCompleted)
        assertEquals(1, api.requestCalls)
        response.complete(slot(SlotViewerState.PENDING))
        first.join()
        assertEquals(1, api.requestCalls)
    }

    @Test
    fun cancelledMutationReleasesLockAndPropagatesCancellation(): Unit = runBlocking {
        val api = FakeSocialApi().apply { requestResponse = CompletableDeferred() }
        val coordinator = SocialCoordinator(api)
        val action = launch(start = CoroutineStart.UNDISPATCHED) { coordinator.requestSlot("slot-1") }
        action.cancelAndJoin()
        assertTrue(action.isCancelled)
        assertIs<MutationState.Idle>(coordinator.mutation.value)
        api.requestResponse = null
        assertTrue(coordinator.requestSlot("slot-1"))
        assertEquals(2, api.requestCalls)
    }

    @Test
    fun latePulseSnapshotCannotOverwriteMutation(): Unit = runBlocking {
        val response = CompletableDeferred<List<SlotModel>>()
        val api = FakeSocialApi().apply { pulseResponse = response }
        val coordinator = SocialCoordinator(api)
        val read = launch(start = CoroutineStart.UNDISPATCHED) { coordinator.refreshPulse() }
        api.mutationResult = slot(SlotViewerState.PENDING, version = 2)
        coordinator.requestSlot("slot-1")
        response.complete(listOf(slot(SlotViewerState.NONE)))
        read.join()
        val pulse = assertIs<LoadState.Content<List<SlotModel>>>(coordinator.pulse.value)
        assertEquals(SlotViewerState.PENDING, pulse.value.single().viewerState)
    }

    @Test
    fun lateMutationCannotRestoreClosedSelection(): Unit = runBlocking {
        val response = CompletableDeferred<SlotModel>()
        val api = FakeSocialApi().apply { requestResponse = response }
        val coordinator = SocialCoordinator(api)
        val action = launch(start = CoroutineStart.UNDISPATCHED) { coordinator.requestSlot("slot-1") }
        coordinator.clearSelected()
        response.complete(slot(SlotViewerState.PENDING))
        action.join()
        assertIs<LoadState.Idle>(coordinator.selectedSlot.value)
    }

    @Test
    fun mySlotsIgnoresPreviousViewAndDisposedAccount(): Unit = runBlocking {
        val response = CompletableDeferred<List<SlotModel>>()
        val api = FakeSocialApi().apply { mySlotsResponse = response }
        val coordinator = SocialCoordinator(api)
        val oldView = launch(start = CoroutineStart.UNDISPATCHED) { coordinator.refreshMySlots(MySlotsView.HOSTING) }
        api.mySlotsResponse = null
        coordinator.refreshMySlots(MySlotsView.JOINED)
        response.complete(listOf(slot(SlotViewerState.HOST)))
        oldView.join()
        assertIs<LoadState.Empty>(coordinator.mySlots.value)

        val late = CompletableDeferred<List<SlotModel>>()
        api.mySlotsResponse = late
        val read = launch(start = CoroutineStart.UNDISPATCHED) { coordinator.refreshMySlots(MySlotsView.HOSTING) }
        coordinator.clearAll()
        late.complete(listOf(slot(SlotViewerState.HOST)))
        read.join()
        assertIs<LoadState.Idle>(coordinator.mySlots.value)
    }

    @Test
    fun lateRosterCannotRestoreAfterCompletionOrNavigation(): Unit = runBlocking {
        val api = FakeSocialApi()
        val coordinator = SocialCoordinator(api)
        coordinator.openSlot("slot-1")
        val response = CompletableDeferred<List<SlotOrganizer>>()
        api.acceptedResponse = response
        val read = launch(start = CoroutineStart.UNDISPATCHED) { coordinator.refreshAccepted("slot-1") }
        api.mutationResult = slot(SlotViewerState.HOST, SlotState.COMPLETED)
        coordinator.completeSlot("slot-1")
        response.complete(listOf(slot(SlotViewerState.ACCEPTED).organizer))
        read.join()
        assertIs<LoadState.Idle>(coordinator.accepted.value)

        api.mutationResult = slot(SlotViewerState.HOST)
        coordinator.openSlot("slot-1")
        val late = CompletableDeferred<List<SlotOrganizer>>()
        api.acceptedResponse = late
        val second = launch(start = CoroutineStart.UNDISPATCHED) { coordinator.refreshAccepted("slot-1") }
        coordinator.clearSelected()
        late.complete(listOf(slot(SlotViewerState.ACCEPTED).organizer))
        second.join()
        assertIs<LoadState.Idle>(coordinator.accepted.value)
    }

    @Test
    fun removingParticipantUsesServerVersionAndReloadsRoster(): Unit = runBlocking {
        val api = FakeSocialApi()
        val coordinator = SocialCoordinator(api)
        coordinator.openSlot("slot-1")
        api.mutationResult = slot(SlotViewerState.HOST, SlotState.FILLING, version = 4)
        assertTrue(coordinator.removeParticipant("slot-1", "member", 3))
        assertEquals(4, assertIs<LoadState.Content<SlotModel>>(coordinator.selectedSlot.value).value.version)
        assertIs<LoadState.Empty>(coordinator.accepted.value)
    }

    @Test
    fun oldChatSnapshotCannotEraseAcknowledgedSend(): Unit = runBlocking {
        val response = CompletableDeferred<List<ChatMessage>>()
        val api = FakeSocialApi().apply { chatResponse = response; messages = listOf(message()) }
        val coordinator = SocialCoordinator(api)
        val read = launch(start = CoroutineStart.UNDISPATCHED) { coordinator.refreshChat("slot-1") }
        coordinator.sendChatMessage("slot-1", "hello")
        response.complete(emptyList())
        read.join()
        assertEquals(listOf("m1"), assertIs<LoadState.Content<List<ChatMessage>>>(coordinator.chat.value).value.map { it.id })
    }

    @Test
    fun chatRefreshKeepsContentWhileLoadingAndStopsOnRevocation(): Unit = runBlocking {
        val api = FakeSocialApi().apply { messages = listOf(message()) }
        val coordinator = SocialCoordinator(api)
        coordinator.refreshChat("slot-1")
        val response = CompletableDeferred<List<ChatMessage>>()
        api.chatResponse = response
        val read = launch(start = CoroutineStart.UNDISPATCHED) { coordinator.refreshChat("slot-1") }
        assertIs<LoadState.Content<List<ChatMessage>>>(coordinator.chat.value)
        read.cancelAndJoin()
        assertIs<LoadState.Content<List<ChatMessage>>>(coordinator.chat.value)
        api.chatResponse = null
        api.chatError = ApiException(403, "chat_forbidden", "Access removed")
        assertEquals(ChatRefreshResult.STOP, coordinator.refreshChat("slot-1"))
        assertIs<LoadState.Failure>(coordinator.chat.value)
        api.chatError = java.io.IOException("offline")
        assertEquals(ChatRefreshResult.RETRY, coordinator.refreshChat("slot-1"))
    }

    private class FakeSocialApi : SocialApi {
        var chatError: Exception? = null
        override suspend fun removeParticipant(slotId: String, userId: String, expectedVersion: Long) = mutationResult
        var acceptedResponse: CompletableDeferred<List<SlotOrganizer>>? = null
        override suspend fun acceptedParticipants(slotId: String): List<SlotOrganizer> = acceptedResponse?.await() ?: emptyList()
        var mySlotsResponse: CompletableDeferred<List<SlotModel>>? = null
        var pulseResponse: CompletableDeferred<List<SlotModel>>? = null
        var chatResponse: CompletableDeferred<List<ChatMessage>>? = null
        var requestResponse: CompletableDeferred<SlotModel>? = null
        var requestCalls = 0
        var pulseItems: List<SlotModel> = emptyList()
        var mutationResult: SlotModel = slot(viewer = SlotViewerState.HOST)
        var messages: List<ChatMessage> = emptyList()

        override suspend fun mySlots(view: MySlotsView) = mySlotsResponse?.await() ?: pulseItems.filter {
            when (view) {
                MySlotsView.HOSTING -> it.viewerState == SlotViewerState.HOST
                MySlotsView.JOINED -> it.viewerState == SlotViewerState.ACCEPTED
                MySlotsView.REQUESTED -> it.viewerState == SlotViewerState.PENDING
            }
        }
        override suspend fun pulse() = pulseResponse?.await() ?: pulseItems
        override suspend fun createSlot(input: CreateSlotInput) = mutationResult
        override suspend fun getSlot(slotId: String) = mutationResult
        override suspend fun editSlot(slotId: String, input: EditSlotInput) = mutationResult
        override suspend fun cancelSlot(slotId: String, expectedVersion: Long) = mutationResult
        override suspend fun requestSlot(slotId: String): SlotModel {
            requestCalls++
            return requestResponse?.await() ?: mutationResult
        }
        override suspend fun leaveSlot(slotId: String) = mutationResult
        override suspend fun pendingRequests(slotId: String): List<PendingSlotRequest> = emptyList()
        override suspend fun approveRequest(slotId: String, userId: String) = mutationResult
        override suspend fun rejectRequest(slotId: String, userId: String) = mutationResult
        override suspend fun startSlot(slotId: String) = mutationResult
        override suspend fun completeSlot(slotId: String) = mutationResult
        override suspend fun chatMessages(slotId: String, limit: Int): List<ChatMessage> {
            chatError?.let { throw it }
            return chatResponse?.await() ?: messages
        }
        override suspend fun sendChatMessage(slotId: String, text: String) = messages.first()
    }

    companion object {
        private fun message() = ChatMessage("m1", "slot-1", com.linkup.app.core.network.ChatAuthor("u", "u", "User", null), "hello", 1)

        private fun slot(
            viewer: SlotViewerState,
            state: SlotState = SlotState.FILLING,
            version: Long = 1,
        ) = SlotModel(
            id = "slot-1",
            organizer = SlotOrganizer("host", "host", "Host", null),
            title = "Coffee",
            activity = "coffee",
            details = null,
            placeText = "Podil",
            zoneText = null,
            startAtEpochMillis = null,
            capacity = 4,
            acceptedCount = if (viewer == SlotViewerState.ACCEPTED) 1 else 0,
            state = state,
            accessMode = SlotAccessMode.APPROVAL,
            visibility = SlotVisibility.PUBLIC,
            viewerState = viewer,
            version = version,
            createdAtEpochMillis = 1,
            updatedAtEpochMillis = 1,
        )
    }
}
