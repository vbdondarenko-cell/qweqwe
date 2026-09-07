package com.linkup.app.core.city

import com.linkup.app.core.network.ApiException
import com.linkup.app.core.network.CanonicalPlace
import com.linkup.app.core.network.CityNetworkApi
import com.linkup.app.core.network.MapCluster
import com.linkup.app.core.network.MapViewportQuery
import com.linkup.app.core.network.SlotModel
import com.linkup.app.core.social.LoadState
import com.linkup.app.core.social.SocialError
import java.io.IOException
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

class CityNetworkCoordinator(
    private val api: CityNetworkApi,
) {
    private var placeRequest = 0L
    private var mapRequest = 0L
    private var mapSlotsRequest = 0L

    private val mutablePlaces = MutableStateFlow<LoadState<List<CanonicalPlace>>>(LoadState.Idle)
    val places: StateFlow<LoadState<List<CanonicalPlace>>> = mutablePlaces.asStateFlow()

    private val mutableMap = MutableStateFlow<LoadState<List<MapCluster>>>(LoadState.Idle)
    val map: StateFlow<LoadState<List<MapCluster>>> = mutableMap.asStateFlow()

    private val mutableMapSlots = MutableStateFlow<LoadState<List<SlotModel>>>(LoadState.Idle)
    val mapSlots: StateFlow<LoadState<List<SlotModel>>> = mutableMapSlots.asStateFlow()

    suspend fun searchPlaces(query: String, locality: String? = null) {
        val normalized = query.trim()
        if (normalized.length < 2) {
            clearPlaces()
            return
        }
        val request = ++placeRequest
        val previous = mutablePlaces.value
        mutablePlaces.value = previous.asRefreshingOrLoading()
        try {
            val items = api.searchPlaces(normalized, locality, 20)
            if (request != placeRequest) return
            mutablePlaces.value = if (items.isEmpty()) LoadState.Empty else LoadState.Content(items)
        } catch (error: CancellationException) {
            if (request == placeRequest) mutablePlaces.value = previous
            throw error
        } catch (error: Exception) {
            if (request == placeRequest) mutablePlaces.value = previous.afterRefreshFailure(error)
        }
    }

    suspend fun refreshMap(query: MapViewportQuery) {
        val request = ++mapRequest
        val previous = mutableMap.value
        mutableMap.value = previous.asRefreshingOrLoading()
        clearMapSlots()
        try {
            val items = api.mapViewport(query)
            if (request != mapRequest) return
            mutableMap.value = if (items.isEmpty()) LoadState.Empty else LoadState.Content(items)
        } catch (error: CancellationException) {
            if (request == mapRequest) mutableMap.value = previous
            throw error
        } catch (error: Exception) {
            if (request == mapRequest) mutableMap.value = previous.afterRefreshFailure(error)
        }
    }

    suspend fun refreshMapPlaceSlots(
        placeId: String,
        fromEpochMillis: Long,
        toEpochMillis: Long,
    ) {
        val request = ++mapSlotsRequest
        val previous = mutableMapSlots.value
        mutableMapSlots.value = previous.asRefreshingOrLoading()
        try {
            val items = api.mapPlaceSlots(placeId, fromEpochMillis, toEpochMillis, 50)
            if (request != mapSlotsRequest) return
            mutableMapSlots.value = if (items.isEmpty()) LoadState.Empty else LoadState.Content(items)
        } catch (error: CancellationException) {
            if (request == mapSlotsRequest) mutableMapSlots.value = previous
            throw error
        } catch (error: Exception) {
            if (request == mapSlotsRequest) mutableMapSlots.value = previous.afterRefreshFailure(error)
        }
    }

    fun clearPlaces() {
        ++placeRequest
        mutablePlaces.value = LoadState.Idle
    }

    fun clearMapSlots() {
        ++mapSlotsRequest
        mutableMapSlots.value = LoadState.Idle
    }

    fun clearMap() {
        ++mapRequest
        mutableMap.value = LoadState.Idle
        clearMapSlots()
    }

    fun clearAll() {
        clearPlaces()
        clearMap()
    }

    private fun <T> LoadState<T>.asRefreshingOrLoading(): LoadState<T> = when (this) {
        is LoadState.Content -> copy(refreshing = true, refreshError = null)
        else -> LoadState.Loading
    }

    private fun <T> LoadState<T>.afterRefreshFailure(error: Exception): LoadState<T> = when (this) {
        is LoadState.Content -> copy(refreshing = false, refreshError = error.toSocialError())
        else -> LoadState.Failure(error.toSocialError())
    }

    private fun Exception.toSocialError(): SocialError = when (this) {
        is ApiException -> SocialError(code = code, message = message, requestId = requestId)
        is IOException -> SocialError(code = "network_error", message = message ?: "Network request failed")
        else -> SocialError(code = "client_error", message = message ?: "Request failed")
    }
}
