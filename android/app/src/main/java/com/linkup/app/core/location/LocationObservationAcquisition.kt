package com.linkup.app.core.location

import com.linkup.app.core.network.CityLocationObservation
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.withTimeoutOrNull

/** One bounded attempt per provider; caller cancellation never starts a fallback. */
internal suspend fun acquireLocationFromProviders(
    providers: List<String>,
    timeoutMillis: Long,
    request: suspend (String) -> CityLocationObservation,
): CityLocationObservation {
    require(timeoutMillis > 0)
    var lastError: Exception = LocationUnavailableException()
    for (provider in providers) {
        currentCoroutineContext().ensureActive()
        try {
            val observation = withTimeoutOrNull(timeoutMillis) { request(provider) }
            currentCoroutineContext().ensureActive()
            if (observation != null) return observation
            lastError = LocationUnavailableException()
        } catch (error: CancellationException) {
            throw error
        } catch (error: SecurityException) {
            throw LocationPermissionRequiredException()
        } catch (error: LocationPermissionRequiredException) {
            throw error
        } catch (error: Exception) {
            lastError = error
        }
    }
    throw lastError
}
