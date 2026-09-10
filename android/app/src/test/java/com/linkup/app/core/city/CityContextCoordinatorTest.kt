package com.linkup.app.core.city

import com.linkup.app.core.location.LocationObservationSource
import com.linkup.app.core.location.LocationPermissionRequiredException
import com.linkup.app.core.network.ApiException
import com.linkup.app.core.network.CityContextApi
import com.linkup.app.core.network.CityContextModel
import com.linkup.app.core.network.CityLocality
import com.linkup.app.core.network.CityLocationObservation
import com.linkup.app.core.network.CityPermissionClass
import com.linkup.app.core.social.LoadState
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancelAndJoin
import kotlinx.coroutines.launch
import kotlinx.coroutines.test.runTest

class CityContextCoordinatorTest {
    @Test
    fun `resolve keeps only server city lock in coordinator state`() = runTest {
        val observation = CityLocationObservation(
            latitudeE6 = 50_450_100,
            longitudeE6 = 30_523_400,
            accuracyM = 25,
            permissionClass = CityPermissionClass.PRECISE,
            capturedAtEpochMillis = 1_000L,
            mocked = false,
        )
        val expected = context()
        var received: CityLocationObservation? = null
        val coordinator = CityContextCoordinator(
            api = object : CityContextApi {
                override suspend fun currentCityContext() = expected
                override suspend fun resolveCityContext(observation: CityLocationObservation): CityContextModel {
                    received = observation
                    return expected
                }
            },
            locations = object : LocationObservationSource {
                override fun permissionClass() = CityPermissionClass.PRECISE
                override suspend fun currentObservation() = observation
            },
        )

        coordinator.resolveFromDevice()

        assertEquals(observation, received)
        val state = assertIs<LoadState.Content<CityContextModel>>(coordinator.context.value)
        assertEquals("Kyiv", state.value.locality.name)
        assertEquals(50_450_100, state.value.locality.centroidLatitudeE6)
        assertEquals(30_523_400, state.value.locality.centroidLongitudeE6)
    }

    @Test
    fun `missing fresh server lock becomes empty instead of fake city`() = runTest {
        val coordinator = CityContextCoordinator(
            api = object : CityContextApi {
                override suspend fun currentCityContext(): CityContextModel =
                    throw ApiException(404, "city_context_unavailable", "no fresh city context")

                override suspend fun resolveCityContext(observation: CityLocationObservation): CityContextModel = error("unused")
            },
            locations = object : LocationObservationSource {
                override fun permissionClass(): CityPermissionClass? = null
                override suspend fun currentObservation(): CityLocationObservation = error("unused")
            },
        )

        coordinator.loadCurrent()

        assertIs<LoadState.Empty>(coordinator.context.value)
    }

    @Test
    fun `authorization loss discards cached city and surfaces failure`() = runTest {
        for (status in listOf(401, 403)) {
            val api = ControlledCityApi()
            val coordinator = CityContextCoordinator(api, observations())
            coordinator.loadCurrent()
            api.failure = ApiException(status, if (status == 401) "unauthorized" else "capability_disabled", "denied")

            coordinator.loadCurrent()

            val state = assertIs<LoadState.Failure>(coordinator.context.value)
            assertEquals(api.failure!!.code, state.error.code)
            assertEquals(status, state.error.httpStatus)
        }
    }

    @Test
    fun `resolver unavailable response removes previously cached city`() = runTest {
        val api = ControlledCityApi()
        val coordinator = CityContextCoordinator(api, observations())
        coordinator.loadCurrent()
        api.failure = ApiException(404, "city_context_unavailable", "expired")

        coordinator.resolveFromDevice()

        assertIs<LoadState.Empty>(coordinator.context.value)
    }

    @Test
    fun `temporary outage preserves cached city with explicit refresh error`() = runTest {
        val api = ControlledCityApi()
        val coordinator = CityContextCoordinator(api, observations())
        coordinator.loadCurrent()
        api.failure = ApiException(503, "unavailable", "temporary outage")

        coordinator.loadCurrent()

        val state = assertIs<LoadState.Content<CityContextModel>>(coordinator.context.value)
        assertEquals(context(), state.value)
        assertEquals(false, state.refreshing)
        assertEquals("unavailable", state.refreshError?.code)
    }

    @Test
    fun `clear while GPS is pending prevents observation from reaching server`() = runTest {
        val waiting = CompletableDeferred<Unit>()
        val release = CompletableDeferred<Unit>()
        val api = ControlledCityApi()
        val coordinator = CityContextCoordinator(api, observations {
            waiting.complete(Unit)
            release.await()
        })
        val request = launch { coordinator.resolveFromDevice() }
        waiting.await()

        coordinator.clear()
        release.complete(Unit)
        request.join()

        assertEquals(0, api.resolveCalls)
        assertIs<LoadState.Idle>(coordinator.context.value)
    }

    @Test
    fun `newer city request prevents older GPS result from being submitted`() = runTest {
        val waiting = CompletableDeferred<Unit>()
        val release = CompletableDeferred<Unit>()
        val api = ControlledCityApi()
        val coordinator = CityContextCoordinator(api, observations {
            waiting.complete(Unit)
            release.await()
        })
        val request = launch { coordinator.resolveFromDevice() }
        waiting.await()
        coordinator.loadCurrent()
        release.complete(Unit)
        request.join()

        assertEquals(0, api.resolveCalls)
        assertEquals(context(), assertIs<LoadState.Content<CityContextModel>>(coordinator.context.value).value)
    }

    @Test
    fun `permission revoked or downgraded before GPS returns prevents precise submission`() = runTest {
        for (currentPermission in listOf(null, CityPermissionClass.APPROXIMATE)) {
            val api = ControlledCityApi()
            val source = object : LocationObservationSource {
                override fun permissionClass() = currentPermission
                override suspend fun currentObservation() = observation()
            }
            val coordinator = CityContextCoordinator(api, source)
            coordinator.loadCurrent()

            coordinator.resolveFromDevice()

            assertEquals(0, api.resolveCalls)
            assertIs<LoadState.Failure>(coordinator.context.value)
        }
    }

    @Test
    fun `clear while server response is pending prevents cached city resurrection`() = runTest {
        val waiting = CompletableDeferred<Unit>()
        val release = CompletableDeferred<Unit>()
        val api = object : CityContextApi {
            override suspend fun currentCityContext(): CityContextModel {
                waiting.complete(Unit)
                release.await()
                return context()
            }
            override suspend fun resolveCityContext(observation: CityLocationObservation) = context()
        }
        val coordinator = CityContextCoordinator(api, observations())
        val request = launch { coordinator.loadCurrent() }
        waiting.await()
        coordinator.clear()
        release.complete(Unit)
        request.join()

        assertIs<LoadState.Idle>(coordinator.context.value)
    }

    @Test
    fun `location permission failure clears previously loaded context`() = runTest {
        val coordinator = CityContextCoordinator(ControlledCityApi(), observations {
            throw LocationPermissionRequiredException()
        })
        coordinator.loadCurrent()
        coordinator.resolveFromDevice()
        assertIs<LoadState.Failure>(coordinator.context.value)
    }

    @Test
    fun `cancelled resolve overlapping a load restores settled state without a stuck spinner`() = runTest {
        for (preload in listOf(false, true)) {
            var waitForLoad = false
            val loadStarted = CompletableDeferred<Unit>()
            val releaseLoad = CompletableDeferred<Unit>()
            val gpsStarted = CompletableDeferred<Unit>()
            val api = object : CityContextApi {
                override suspend fun currentCityContext(): CityContextModel {
                    if (waitForLoad) {
                        loadStarted.complete(Unit)
                        releaseLoad.await()
                    }
                    return context()
                }
                override suspend fun resolveCityContext(observation: CityLocationObservation): CityContextModel =
                    error("cancelled GPS must not be submitted")
            }
            val coordinator = CityContextCoordinator(api, observations {
                gpsStarted.complete(Unit)
                awaitCancellation()
            })
            if (preload) coordinator.loadCurrent()
            val settled = coordinator.context.value
            waitForLoad = true
            val older = launch { coordinator.loadCurrent() }
            loadStarted.await()
            val newer = launch { coordinator.resolveFromDevice() }
            gpsStarted.await()

            newer.cancelAndJoin()
            assertEquals(settled, coordinator.context.value)
            releaseLoad.complete(Unit)
            older.join()
            assertEquals(settled, coordinator.context.value)
        }
    }

    @Test
    fun `cancelled load overlapping GPS restores idle and suppresses older submission`() = runTest {
        val gpsStarted = CompletableDeferred<Unit>()
        val releaseGps = CompletableDeferred<Unit>()
        val loadStarted = CompletableDeferred<Unit>()
        val api = object : CityContextApi {
            override suspend fun currentCityContext(): CityContextModel {
                loadStarted.complete(Unit)
                awaitCancellation()
            }
            override suspend fun resolveCityContext(observation: CityLocationObservation): CityContextModel =
                error("superseded GPS must not be submitted")
        }
        val coordinator = CityContextCoordinator(api, observations {
            gpsStarted.complete(Unit)
            releaseGps.await()
        })
        val older = launch { coordinator.resolveFromDevice() }
        gpsStarted.await()
        val newer = launch { coordinator.loadCurrent() }
        loadStarted.await()

        newer.cancelAndJoin()
        assertIs<LoadState.Idle>(coordinator.context.value)
        releaseGps.complete(Unit)
        older.join()
        assertIs<LoadState.Idle>(coordinator.context.value)
    }

    private inner class ControlledCityApi : CityContextApi {
        var failure: ApiException? = null
        var resolveCalls = 0
        override suspend fun currentCityContext(): CityContextModel {
            failure?.let { throw it }
            return context()
        }
        override suspend fun resolveCityContext(observation: CityLocationObservation): CityContextModel {
            resolveCalls++
            failure?.let { throw it }
            return context()
        }
    }

    private fun observations(beforeObservation: suspend () -> Unit = {}) = object : LocationObservationSource {
        override fun permissionClass() = CityPermissionClass.PRECISE
        override suspend fun currentObservation(): CityLocationObservation {
            beforeObservation()
            return observation()
        }
    }

    private fun observation() = CityLocationObservation(
        latitudeE6 = 50_450_100,
        longitudeE6 = 30_523_400,
        accuracyM = 25,
        permissionClass = CityPermissionClass.PRECISE,
        capturedAtEpochMillis = 1_000L,
        mocked = false,
    )

    private fun context() = CityContextModel(
        locality = CityLocality(
            id = "00000000-0000-0000-0000-000000000001",
            name = "Kyiv",
            countryCode = "UA",
            timezone = "Europe/Kyiv",
            centroidLatitudeE6 = 50_450_100,
            centroidLongitudeE6 = 30_523_400,
        ),
        permissionClass = CityPermissionClass.PRECISE,
        accuracyM = 25,
        observedAtEpochMillis = 1_000L,
        expiresAtEpochMillis = 2_000L,
        switchPending = false,
    )
}
