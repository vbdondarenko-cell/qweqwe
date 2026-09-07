package com.linkup.app.core.social

import com.linkup.app.core.network.ApiException
import com.linkup.app.core.network.ChatMessage
import com.linkup.app.core.network.CreateSlotInput
import com.linkup.app.core.network.EditSlotInput
import com.linkup.app.core.network.MySlotsView
import com.linkup.app.core.network.PendingSlotRequest
import com.linkup.app.core.network.SlotAccessMode
import com.linkup.app.core.network.SlotModel
import com.linkup.app.core.network.SlotOrganizer
import com.linkup.app.core.network.SlotState
import com.linkup.app.core.network.SlotViewerState
import com.linkup.app.core.network.SlotVisibility
import com.linkup.app.core.network.SocialApi
import java.io.IOException
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs
import kotlin.test.assertNotNull
import kotlinx.coroutines.runBlocking

class RefreshRetentionTest {
    @Test
    fun transientPulseFailureKeepsCachedContent(): Unit = runBlocking {
        val api = RefreshApi()
        val coordinator = SocialCoordinator(api)
        api.pulseItems = listOf(slot())
        coordinator.refreshPulse()

        api.pulseError = IOException("offline")
        coordinator.refreshPulse()

        val content = assertIs<LoadState.Content<List<SlotModel>>>(coordinator.pulse.value)
        assertEquals("slot-1", content.value.single().id)
        assertNotNull(content.refreshError)
        assertEquals(false, content.refreshing)
    }

    @Test
    fun definitiveChatRevocationDropsCachedThread(): Unit = runBlocking {
        val api = RefreshApi().apply { messages = listOf(message()) }
        val coordinator = SocialCoordinator(api)
        coordinator.refreshChat("slot-1")
        assertIs<LoadState.Content<List<ChatMessage>>>(coordinator.chat.value)

        api.chatError = ApiException(403, "chat_forbidden", "Access removed")
        assertEquals(ChatRefreshResult.STOP, coordinator.refreshChat("slot-1"))
        assertIs<LoadState.Failure>(coordinator.chat.value)
    }

    @Test
    fun transientChatFailureKeepsCachedThread(): Unit = runBlocking {
        val api = RefreshApi().apply { messages = listOf(message()) }
        val coordinator = SocialCoordinator(api)
        coordinator.refreshChat("slot-1")

        api.chatError = IOException("offline")
        assertEquals(ChatRefreshResult.RETRY, coordinator.refreshChat("slot-1"))
        val content = assertIs<LoadState.Content<List<ChatMessage>>>(coordinator.chat.value)
        assertEquals("m1", content.value.single().id)
        assertNotNull(content.refreshError)
    }

    private class RefreshApi : SocialApi {
        var pulseItems: List<SlotModel> = emptyList()
        var pulseError: Exception? = null
        var messages: List<ChatMessage> = emptyList()
        var chatError: Exception? = null

        override suspend fun pulse(): List<SlotModel> {
            pulseError?.let { throw it }
            return pulseItems
        }

        override suspend fun chatMessages(slotId: String, limit: Int): List<ChatMessage> {
            chatError?.let { throw it }
            return messages
        }

        override suspend fun createSlot(input: CreateSlotInput) = slot(SlotViewerState.HOST)
        override suspend fun getSlot(slotId: String) = slot(SlotViewerState.HOST)
        override suspend fun editSlot(slotId: String, input: EditSlotInput) = slot(SlotViewerState.HOST)
        override suspend fun cancelSlot(slotId: String, expectedVersion: Long) = slot(SlotViewerState.HOST, SlotState.CANCELLED)
        override suspend fun requestSlot(slotId: String) = slot(SlotViewerState.PENDING)
        override suspend fun leaveSlot(slotId: String) = slot(SlotViewerState.NONE)
        override suspend fun pendingRequests(slotId: String): List<PendingSlotRequest> = emptyList()
        override suspend fun approveRequest(slotId: String, userId: String) = slot(SlotViewerState.HOST)
        override suspend fun rejectRequest(slotId: String, userId: String) = slot(SlotViewerState.HOST)
        override suspend fun startSlot(slotId: String) = slot(SlotViewerState.HOST, SlotState.ACTIVE)
        override suspend fun completeSlot(slotId: String) = slot(SlotViewerState.HOST, SlotState.COMPLETED)
        override suspend fun sendChatMessage(slotId: String, text: String) = message()
        override suspend fun mySlots(view: MySlotsView): List<SlotModel> = pulseItems
        override suspend fun acceptedParticipants(slotId: String): List<SlotOrganizer> = emptyList()
        override suspend fun removeParticipant(slotId: String, userId: String, expectedVersion: Long) = slot(SlotViewerState.HOST)
    }

    companion object {
        private fun message() = ChatMessage(
            id = "m1",
            slotId = "slot-1",
            author = com.linkup.app.core.network.ChatAuthor("u1", "user", "User", null),
            text = "hello",
            createdAtEpochMillis = 1,
        )

        private fun slot(
            viewer: SlotViewerState = SlotViewerState.NONE,
            state: SlotState = SlotState.FILLING,
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
            version = 1,
            createdAtEpochMillis = 1,
            updatedAtEpochMillis = 1,
        )
    }
}
