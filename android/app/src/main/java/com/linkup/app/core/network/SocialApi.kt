package com.linkup.app.core.network

interface SocialApi {
    suspend fun mySlots(view: MySlotsView): List<SlotModel>
    suspend fun pulse(): List<SlotModel>
    suspend fun createSlot(input: CreateSlotInput): SlotModel
    suspend fun getSlot(slotId: String): SlotModel
    suspend fun editSlot(slotId: String, input: EditSlotInput): SlotModel
    suspend fun cancelSlot(slotId: String, expectedVersion: Long): SlotModel
    suspend fun requestSlot(slotId: String): SlotModel
    suspend fun leaveSlot(slotId: String): SlotModel
    suspend fun pendingRequests(slotId: String): List<PendingSlotRequest>
    suspend fun approveRequest(slotId: String, userId: String): SlotModel
    suspend fun rejectRequest(slotId: String, userId: String): SlotModel
    suspend fun startSlot(slotId: String): SlotModel
    suspend fun completeSlot(slotId: String): SlotModel
    suspend fun chatMessages(slotId: String, limit: Int = 100): List<ChatMessage>
    suspend fun sendChatMessage(slotId: String, text: String): ChatMessage
}
