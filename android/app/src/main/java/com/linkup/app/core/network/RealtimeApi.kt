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

    // currentCursor is the reconnect/first-run bootstrap call (backend
    // GET /v1/realtime/cursor): the channel's live position with no
    // events attached, so a client with no persisted cursor can adopt it
    // directly instead of pulling forward through the entire outbox
    // history it has no use for (its actual current state already comes
    // from Pulse/Get/ListMine).
    suspend fun currentCursor(): Long
}
