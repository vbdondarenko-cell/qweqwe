package com.linkup.app.core.network

data class RealtimeEventModel(
    val sequence: Long,
    val eventId: String,
    val eventType: String,
    val aggregateType: String,
    val aggregateId: String,
    val subjectUserId: String?,
    val slotId: String?,
    val payloadJson: String,
    val occurredAtEpochMillis: Long,
)

data class RealtimeBatchModel(
    val cursor: Long,
    val events: List<RealtimeEventModel>,
)

interface RealtimeApi {
    suspend fun pullRealtime(after: Long, limit: Int = 100): RealtimeBatchModel
}
