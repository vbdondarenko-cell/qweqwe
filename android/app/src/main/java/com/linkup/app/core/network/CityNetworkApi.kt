package com.linkup.app.core.network

interface CityNetworkApi {
    suspend fun searchPlaces(
        query: String,
        locality: String? = null,
        limit: Int = 20,
    ): List<CanonicalPlace>

    suspend fun mapViewport(query: MapViewportQuery): List<MapCluster>
}
