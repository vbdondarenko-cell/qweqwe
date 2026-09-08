package com.linkup.app.core.location

import android.Manifest
import android.annotation.SuppressLint
import android.content.Context
import android.content.pm.PackageManager
import android.location.Location
import android.location.LocationListener
import android.location.LocationManager
import android.os.Build
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import com.linkup.app.core.network.CityLocationObservation
import com.linkup.app.core.network.CityPermissionClass
import java.util.concurrent.Executor
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException
import kotlin.math.roundToInt
import kotlinx.coroutines.suspendCancellableCoroutine

class LocationPermissionRequiredException : IllegalStateException("Location permission is required")
class LocationUnavailableException : IllegalStateException("No enabled location provider is available")

interface LocationObservationSource {
    fun permissionClass(): CityPermissionClass?
    suspend fun currentObservation(): CityLocationObservation
}

class AndroidLocationObservationSource(context: Context) : LocationObservationSource {
    private val appContext = context.applicationContext
    private val locationManager = appContext.getSystemService(Context.LOCATION_SERVICE) as LocationManager
    private val mainHandler = Handler(Looper.getMainLooper())
    private val mainExecutor = Executor { runnable -> mainHandler.post(runnable) }

    override fun permissionClass(): CityPermissionClass? = when {
        appContext.checkSelfPermission(Manifest.permission.ACCESS_FINE_LOCATION) == PackageManager.PERMISSION_GRANTED ->
            CityPermissionClass.PRECISE
        appContext.checkSelfPermission(Manifest.permission.ACCESS_COARSE_LOCATION) == PackageManager.PERMISSION_GRANTED ->
            CityPermissionClass.APPROXIMATE
        else -> null
    }

    @SuppressLint("MissingPermission")
    override suspend fun currentObservation(): CityLocationObservation {
        val permission = permissionClass() ?: throw LocationPermissionRequiredException()
        val providers = candidateProviders(permission)
        if (providers.isEmpty()) throw LocationUnavailableException()

        newestUsableLastKnown(providers)?.let { return it.toObservation(permission) }

        var lastError: Exception? = null
        for (provider in providers) {
            try {
                return requestOne(provider).toObservation(permission)
            } catch (error: SecurityException) {
                throw LocationPermissionRequiredException()
            } catch (error: Exception) {
                lastError = error
            }
        }
        throw lastError ?: LocationUnavailableException()
    }

    @SuppressLint("MissingPermission")
    private fun newestUsableLastKnown(providers: List<String>): Location? {
        val now = System.currentTimeMillis()
        return providers
            .mapNotNull { provider -> runCatching { locationManager.getLastKnownLocation(provider) }.getOrNull() }
            .filter { it.time > 0L && it.time <= now + MAX_FUTURE_LOCATION_MS && now - it.time <= MAX_LAST_KNOWN_AGE_MS }
            .maxByOrNull { it.time }
    }

    @SuppressLint("MissingPermission")
    private suspend fun requestOne(provider: String): Location = suspendCancellableCoroutine { continuation ->
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            try {
                locationManager.getCurrentLocation(provider, null, mainExecutor) { location ->
                    if (!continuation.isActive) return@getCurrentLocation
                    if (location == null) continuation.resumeWithException(LocationUnavailableException())
                    else continuation.resume(location)
                }
            } catch (error: Exception) {
                if (continuation.isActive) continuation.resumeWithException(error)
            }
            return@suspendCancellableCoroutine
        }

        lateinit var listener: LocationListener
        listener = object : LocationListener {
            override fun onLocationChanged(location: Location) {
                locationManager.removeUpdates(this)
                if (continuation.isActive) continuation.resume(location)
            }

            @Deprecated("Deprecated in Android framework")
            override fun onStatusChanged(provider: String?, status: Int, extras: Bundle?) = Unit
            override fun onProviderEnabled(provider: String) = Unit
            override fun onProviderDisabled(provider: String) {
                locationManager.removeUpdates(this)
                if (continuation.isActive) continuation.resumeWithException(LocationUnavailableException())
            }
        }
        continuation.invokeOnCancellation { runCatching { locationManager.removeUpdates(listener) } }
        try {
            locationManager.requestSingleUpdate(provider, listener, Looper.getMainLooper())
        } catch (error: Exception) {
            if (continuation.isActive) continuation.resumeWithException(error)
        }
    }

    private fun candidateProviders(permission: CityPermissionClass): List<String> {
        val preferred = if (permission == CityPermissionClass.PRECISE) {
            listOf(LocationManager.GPS_PROVIDER, LocationManager.NETWORK_PROVIDER, LocationManager.PASSIVE_PROVIDER)
        } else {
            listOf(LocationManager.NETWORK_PROVIDER, LocationManager.PASSIVE_PROVIDER)
        }
        return preferred.filter { provider ->
            runCatching { locationManager.getProvider(provider) != null && locationManager.isProviderEnabled(provider) }
                .getOrDefault(false)
        }
    }

    private fun Location.toObservation(permission: CityPermissionClass): CityLocationObservation {
        require(latitude in -90.0..90.0 && longitude in -180.0..180.0)
        val accuracyMeters = if (hasAccuracy()) accuracy.roundToInt().coerceAtLeast(1) else Int.MAX_VALUE
        require(accuracyMeters <= 10_000) { "location accuracy is outside city-context policy bounds" }
        val capturedAt = time.takeIf { it > 0L } ?: System.currentTimeMillis()
        return CityLocationObservation(
            latitudeE6 = (latitude * 1_000_000.0).roundToInt(),
            longitudeE6 = (longitude * 1_000_000.0).roundToInt(),
            accuracyM = accuracyMeters,
            permissionClass = permission,
            capturedAtEpochMillis = capturedAt,
            mocked = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) isMock else isFromMockProvider,
        )
    }

    private companion object {
        const val MAX_LAST_KNOWN_AGE_MS = 2L * 60L * 1000L
        const val MAX_FUTURE_LOCATION_MS = 30L * 1000L
    }
}
