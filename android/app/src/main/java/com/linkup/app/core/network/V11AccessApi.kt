package com.linkup.app.core.network

interface V11AccessApi {
    suspend fun joinSlot(slotId: String): SlotModel
}
