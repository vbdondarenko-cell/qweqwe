package com.linkup.app.core.city

import com.linkup.app.core.location.LocationObservationSource
import com.linkup.app.core.location.LocationPermissionRequiredException
import com.linkup.app.core.network.ApiException
import com.linkup.app.core.network.CityContextApi
import com.linkup.app.core.network.CityContextModel
import com.linkup.app.core.network.CityPermissionClass
import com.linkup.app.core.social.LoadState
import com.linkup.app.core.social.SocialError
import java.io.IOException
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

class CityContextCoordinator(
    private val api: CityContextApi,
    private val locations: LocationObservationSource,
) {
    private var requestGeneration = 0L
    private val mutableContext = MutableStateFlow<LoadState<CityContextModel>>(LoadState.Idle)
    val context: StateFlow<LoadState<CityContextModel>> = mutableContext.asStateFlow()

    fun permissionClass() = locations.permissionClass()

    suspend fun loadCurrent() {
        val request = ++requestGeneration
        val previous = mutableContext.value
        mutableContext.value = previous.asRefreshingOrLoading()
        try {
            val current = api.currentCityContext()
            if (request == requestGeneration) mutableContext.value = LoadState.Content(current)
        } catch (error: CancellationException) {
            if (request == requestGeneration) mutableContext.value = previous
            throw error
        } catch (error: ApiException) {
            if (request != requestGeneration) return
            mutableContext.value = if (error.status == 404 && error.code == "city_context_unavailable") {
                LoadState.Empty
            } else {
                previous.afterRefreshFailure(error)
            }
        } catch (error: Exception) {
            if (request == requestGeneration) mutableContext.value = previous.afterRefreshFailure(error)
        }
    }

    suspend fun resolveFromDevice() {
        val request = ++requestGeneration
        val previous = mutableContext.value
        mutableContext.value = previous.asRefreshingOrLoading()
        try {
            // The raw observation only exists on this stack frame and is sent directly
            // to the resolver. Coordinator state retains only the server City-Lock.
            val observation = locations.currentObservation()
            currentCoroutineContext().ensureActive()
            // GPS may finish after sign-out, capability revocation or a newer request.
            // In that case even sending the old observation is no longer authorized.
            if (request != requestGeneration) return
            val permission = locations.permissionClass()
            if (permission == null ||
                (observation.permissionClass == CityPermissionClass.PRECISE && permission != CityPermissionClass.PRECISE)
            ) throw LocationPermissionRequiredException()
            val resolved = api.resolveCityContext(observation)
            if (request == requestGeneration) mutableContext.value = LoadState.Content(resolved)
        } catch (error: CancellationException) {
            if (request == requestGeneration) mutableContext.value = previous
            throw error
        } catch (error: Exception) {
            if (request == requestGeneration) mutableContext.value = previous.afterRefreshFailure(error)
        }
    }

    fun clear() {
        ++requestGeneration
        mutableContext.value = LoadState.Idle
    }

    private fun <T> LoadState<T>.asRefreshingOrLoading(): LoadState<T> = when (this) {
        is LoadState.Content -> copy(refreshing = true, refreshError = null)
        else -> LoadState.Loading
    }

    private fun <T> LoadState<T>.afterRefreshFailure(error: Exception): LoadState<T> {
        // A denied/expired server scope must stop city realtime and expose auth errors
        // to the session owner. Only recoverable transport failures retain cached city.
        if (error is ApiException && error.status == 404 && error.code == "city_context_unavailable") {
            return LoadState.Empty
        }
        if (error is LocationPermissionRequiredException ||
            (error is ApiException && error.status in setOf(401, 403))
        ) return LoadState.Failure(error.toSocialError())
        return when (this) {
            is LoadState.Content -> copy(refreshing = false, refreshError = error.toSocialError())
            else -> LoadState.Failure(error.toSocialError())
        }
    }

    private fun Exception.toSocialError(): SocialError = when (this) {
        is ApiException -> SocialError(code = code, message = message, requestId = requestId)
        is IOException -> SocialError(code = "network_error", message = message ?: "Network request failed")
        else -> SocialError(code = "location_error", message = message ?: "City context could not be resolved")
    }
}
