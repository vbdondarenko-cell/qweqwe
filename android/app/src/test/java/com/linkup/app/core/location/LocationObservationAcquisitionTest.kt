package com.linkup.app.core.location

import com.linkup.app.core.network.CityLocationObservation
import com.linkup.app.core.network.CityPermissionClass
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertTrue
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.launch
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.withTimeout

class LocationObservationAcquisitionTest {
    @Test
    fun `first usable observation stops provider acquisition`() = runTest {
        val calls = mutableListOf<String>()
        val result = acquireLocationFromProviders(PROVIDERS, 100) { provider ->
            calls += provider
            OBSERVATION
        }
        assertEquals(OBSERVATION, result)
        assertEquals(listOf("gps"), calls)
    }

    @Test
    fun `provider timeout releases current attempt before starting fallback`() = runTest {
        val events = mutableListOf<String>()
        val result = acquireLocationFromProviders(PROVIDERS, 100) { provider ->
            events += "start:$provider"
            if (provider == "gps") {
                try { awaitCancellation() }
                finally { events += "stop:$provider" }
            }
            OBSERVATION
        }
        assertEquals(OBSERVATION, result)
        assertEquals(listOf("start:gps", "stop:gps", "start:network"), events)
    }

    @Test
    fun `caller cancellation releases attempt without starting another provider`() = runTest {
        val started = CompletableDeferred<Unit>()
        val calls = mutableListOf<String>()
        var released = false
        val job = launch {
            acquireLocationFromProviders(PROVIDERS, 100) { provider ->
                calls += provider
                started.complete(Unit)
                try { awaitCancellation() }
                finally { released = true }
            }
        }
        started.await()
        job.cancel()
        job.join()
        assertTrue(job.isCancelled)
        assertTrue(released)
        assertEquals(listOf("gps"), calls)
    }

    @Test
    fun `outer timeout cannot become provider fallback`() = runTest {
        val calls = mutableListOf<String>()
        assertFailsWith<TimeoutCancellationException> {
            withTimeout(50) {
                acquireLocationFromProviders(PROVIDERS, 100) { provider ->
                    calls += provider
                    awaitCancellation()
                }
            }
        }
        assertEquals(listOf("gps"), calls)
    }

    @Test
    fun `explicit cancellation is not treated as provider failure`() = runTest {
        val calls = mutableListOf<String>()
        assertFailsWith<CancellationException> {
            acquireLocationFromProviders(PROVIDERS, 100) { provider ->
                calls += provider
                throw CancellationException("cancelled")
            }
        }
        assertEquals(listOf("gps"), calls)
    }

    @Test
    fun `permission errors stop all provider attempts`() = runTest {
        for (failure in listOf(SecurityException(), LocationPermissionRequiredException())) {
            val calls = mutableListOf<String>()
            assertFailsWith<LocationPermissionRequiredException> {
                acquireLocationFromProviders(PROVIDERS, 100) { provider ->
                    calls += provider
                    throw failure
                }
            }
            assertEquals(listOf("gps"), calls)
        }
    }

    @Test
    fun `invalid observation can fall back to next provider`() = runTest {
        val calls = mutableListOf<String>()
        val result = acquireLocationFromProviders(PROVIDERS, 100) { provider ->
            calls += provider
            if (provider == "gps") throw IllegalArgumentException("invalid observation")
            OBSERVATION
        }
        assertEquals(OBSERVATION, result)
        assertEquals(PROVIDERS, calls)
    }

    @Test
    fun `exhausted providers finish with unavailable instead of an unbounded retry`() = runTest {
        val calls = mutableListOf<String>()
        assertFailsWith<LocationUnavailableException> {
            acquireLocationFromProviders(PROVIDERS, 100) { provider ->
                calls += provider
                awaitCancellation()
            }
        }
        assertEquals(PROVIDERS, calls)
    }

    @Test
    fun `empty provider list does not attempt acquisition`() = runTest {
        assertFailsWith<LocationUnavailableException> {
            acquireLocationFromProviders(emptyList(), 100) { error("no provider should be called") }
        }
    }

    private companion object {
        val PROVIDERS = listOf("gps", "network")
        val OBSERVATION = CityLocationObservation(
            latitudeE6 = 50_450_100,
            longitudeE6 = 30_523_400,
            accuracyM = 25,
            permissionClass = CityPermissionClass.PRECISE,
            capturedAtEpochMillis = 1_000L,
            mocked = false,
        )
    }
}
