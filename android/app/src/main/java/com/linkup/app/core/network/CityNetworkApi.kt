package com.linkup.app.core.network

interface CityNetworkApi {
    suspend fun searchPlaces(
        query: String,
        locality: String? = null,
        limit: Int = 20,
    ): List<CanonicalPlace>

    suspend fun mapViewport(query: MapViewportQuery): List<MapCluster>

    suspend fun mapPlaceSlots(
        placeId: String,
        fromEpochMillis: Long,
        toEpochMillis: Long,
        limit: Int = 50,
    ): List<SlotModel>
}
