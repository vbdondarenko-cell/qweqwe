package com.linkup.app.core.network

data class CanonicalPlace(
    val id: String,
    val name: String,
    val category: String?,
    val locality: String?,
    val countryCode: String?,
    val latitudeE6: Int,
    val longitudeE6: Int,
    val precisionM: Int,
)

data class MapViewportQuery(
    val westE6: Int,
    val southE6: Int,
    val eastE6: Int,
    val northE6: Int,
    val zoom: Int,
    val fromEpochMillis: Long,
    val toEpochMillis: Long,
    val limit: Int = 100,
)

data class MapCluster(
    val key: String,
    val latitudeE6: Int,
    val longitudeE6: Int,
    val placeCount: Int,
    val slotCount: Int,
    val placeId: String?,
    val placeName: String?,
)
