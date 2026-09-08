package com.linkup.app.core.network

enum class CityPermissionClass {
    APPROXIMATE,
    PRECISE,
}

data class CityLocality(
    val id: String,
    val name: String,
    val countryCode: String,
    val timezone: String,
)

data class CityContextModel(
    val locality: CityLocality,
    val permissionClass: CityPermissionClass,
    val accuracyM: Int,
    val observedAtEpochMillis: Long,
    val expiresAtEpochMillis: Long,
    val switchPending: Boolean,
)

data class CityLocationObservation(
    val latitudeE6: Int,
    val longitudeE6: Int,
    val accuracyM: Int,
    val permissionClass: CityPermissionClass,
    val capturedAtEpochMillis: Long,
    val mocked: Boolean,
)

interface CityContextApi {
    suspend fun currentCityContext(): CityContextModel
    suspend fun resolveCityContext(observation: CityLocationObservation): CityContextModel
}
