package com.linkup.app.core.network

interface V11HostingApi {
    suspend fun createDraft(input: CreateSlotInput): SlotModel
    suspend fun publishDraft(slotId: String, expectedVersion: Long): SlotModel
}
