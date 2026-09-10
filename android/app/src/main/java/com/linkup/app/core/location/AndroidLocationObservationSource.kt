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
import android.os.CancellationSignal
import android.os.Handler
import android.os.Looper
import com.linkup.app.core.network.CityLocationObservation
import com.linkup.app.core.network.CityPermissionClass
import java.util.concurrent.Executor
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException
import kotlin.math.roundToInt
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.ensureActive
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
        currentCoroutineContext().ensureActive()
        val permission = permissionClass() ?: throw LocationPermissionRequiredException()
        val providers = candidateProviders(permission)
        if (providers.isEmpty()) throw LocationUnavailableException()

        newestUsableLastKnown(providers, permission)?.let { return it.toObservation(permission) }

        return acquireLocationFromProviders(providers, PROVIDER_TIMEOUT_MS) { provider ->
            requestOne(provider).toObservation(permission)
        }
    }

    @SuppressLint("MissingPermission")
    private fun newestUsableLastKnown(providers: List<String>, permission: CityPermissionClass): Location? {
        val now = System.currentTimeMillis()
        return providers
            .mapNotNull { provider -> runCatching { locationManager.getLastKnownLocation(provider) }.getOrNull() }
            .filter { location ->
                location.time > 0L &&
                    now - location.time <= MAX_LAST_KNOWN_AGE_MS &&
                    deviceObservationMetadataIsUsable(
                        permission = permission,
                        capturedAtEpochMillis = location.time,
                        nowEpochMillis = now,
                        accuracyM = deviceAccuracyMeters(location.hasAccuracy(), location.accuracy),
                        mocked = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) location.isMock else location.isFromMockProvider,
                    )
            }
            .maxByOrNull { it.time }
    }

    @SuppressLint("MissingPermission")
    private suspend fun requestOne(provider: String): Location = suspendCancellableCoroutine { continuation ->
        if (!continuation.isActive) return@suspendCancellableCoroutine
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            val cancellation = CancellationSignal()
            continuation.invokeOnCancellation { cancellation.cancel() }
            try {
                locationManager.getCurrentLocation(provider, cancellation, mainExecutor) { location ->
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
                runCatching { locationManager.removeUpdates(this) }
                if (continuation.isActive) continuation.resume(location)
            }

            @Deprecated("Deprecated in Android framework")
            override fun onStatusChanged(provider: String?, status: Int, extras: Bundle?) = Unit
            override fun onProviderEnabled(provider: String) = Unit
            override fun onProviderDisabled(provider: String) {
                runCatching { locationManager.removeUpdates(this) }
                if (continuation.isActive) continuation.resumeWithException(LocationUnavailableException())
            }
        }
        continuation.invokeOnCancellation { runCatching { locationManager.removeUpdates(listener) } }
        try {
            if (!continuation.isActive) return@suspendCancellableCoroutine
            locationManager.requestSingleUpdate(provider, listener, Looper.getMainLooper())
            // Cancellation can race registration after the cancellation handler ran.
            if (!continuation.isActive) runCatching { locationManager.removeUpdates(listener) }
        } catch (error: Exception) {
            runCatching { locationManager.removeUpdates(listener) }
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
        val accuracyMeters = deviceAccuracyMeters(hasAccuracy(), accuracy)
            ?: throw IllegalArgumentException("location accuracy is unavailable or invalid")
        val now = System.currentTimeMillis()
        // Never turn an unknown capture time into a fresh device observation.
        val capturedAt = time
        val mocked = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) isMock else isFromMockProvider
        require(deviceObservationMetadataIsUsable(permission, capturedAt, now, accuracyMeters, mocked)) {
            "location observation is outside city-context policy bounds"
        }
        return CityLocationObservation(
            latitudeE6 = (latitude * 1_000_000.0).roundToInt(),
            longitudeE6 = (longitude * 1_000_000.0).roundToInt(),
            accuracyM = accuracyMeters,
            permissionClass = permission,
            capturedAtEpochMillis = capturedAt,
            mocked = false,
        )
    }

    private companion object {
        const val MAX_LAST_KNOWN_AGE_MS = 2L * 60L * 1000L
        const val MAX_FUTURE_LOCATION_MS = 30L * 1000L
        const val PROVIDER_TIMEOUT_MS = 12_000L
    }
}

internal fun deviceAccuracyMeters(hasAccuracy: Boolean, accuracy: Float): Int? {
    if (!hasAccuracy || !accuracy.isFinite() || accuracy <= 0f) return null
    return accuracy.roundToInt().coerceAtLeast(1)
}

internal fun deviceObservationMetadataIsUsable(
    permission: CityPermissionClass,
    capturedAtEpochMillis: Long,
    nowEpochMillis: Long,
    accuracyM: Int?,
    mocked: Boolean,
): Boolean {
    if (mocked || capturedAtEpochMillis <= 0L || nowEpochMillis <= 0L || accuracyM == null || accuracyM <= 0) return false
    if (capturedAtEpochMillis > nowEpochMillis + 30_000L) return false
    val age = nowEpochMillis - capturedAtEpochMillis
    if (age < -30_000L) return false
    val maxAge = when (permission) {
        CityPermissionClass.PRECISE -> 2L * 60L * 1000L
        CityPermissionClass.APPROXIMATE -> 10L * 60L * 1000L
    }
    val maxAccuracy = when (permission) {
        CityPermissionClass.PRECISE -> 500
        CityPermissionClass.APPROXIMATE -> 5_000
    }
    return age <= maxAge && accuracyM <= maxAccuracy
}
