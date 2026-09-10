package com.linkup.app.core.network

data class CityRealtimeInvalidationModel(
    val sequence: Long,
    val eventId: String,
    val occurredAtEpochMillis: Long,
)

data class CityRealtimeBatchModel(
    val cursor: Long,
    val invalidations: List<CityRealtimeInvalidationModel>,
)

interface CityRealtimeApi {
    suspend fun pullCityRealtime(after: Long, limit: Int = 100): CityRealtimeBatchModel
}
