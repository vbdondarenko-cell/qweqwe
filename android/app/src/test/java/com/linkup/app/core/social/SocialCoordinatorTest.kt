package com.linkup.app.core.social

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
import kotlinx.coroutines.runBlocking

class SocialCoordinatorTest {
    @Test
    fun refreshPulseUsesServerDataAndEmptyState() = runBlocking {
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
    fun requestUsesServerReturnedViewerState() = runBlocking {
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
    fun terminalServerStateClearsChatState() = runBlocking {
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

    private class FakeSocialApi : SocialApi {
        var pulseItems: List<SlotModel> = emptyList()
        var mutationResult: SlotModel = slot(viewer = SlotViewerState.HOST)
        var messages: List<ChatMessage> = emptyList()

        override suspend fun pulse() = pulseItems
        override suspend fun createSlot(input: CreateSlotInput) = mutationResult
        override suspend fun getSlot(slotId: String) = mutationResult
        override suspend fun editSlot(slotId: String, input: EditSlotInput) = mutationResult
        override suspend fun cancelSlot(slotId: String, expectedVersion: Long) = mutationResult
        override suspend fun requestSlot(slotId: String) = mutationResult
        override suspend fun leaveSlot(slotId: String) = mutationResult
        override suspend fun pendingRequests(slotId: String): List<PendingSlotRequest> = emptyList()
        override suspend fun approveRequest(slotId: String, userId: String) = mutationResult
        override suspend fun rejectRequest(slotId: String, userId: String) = mutationResult
        override suspend fun startSlot(slotId: String) = mutationResult
        override suspend fun completeSlot(slotId: String) = mutationResult
        override suspend fun chatMessages(slotId: String, limit: Int) = messages
        override suspend fun sendChatMessage(slotId: String, text: String) = messages.first()
    }

    companion object {
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
