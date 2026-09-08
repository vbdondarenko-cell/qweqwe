package com.linkup.app.core.city

import com.linkup.app.core.location.LocationObservationSource
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

    private fun context() = CityContextModel(
        locality = CityLocality(
            id = "00000000-0000-0000-0000-000000000001",
            name = "Kyiv",
            countryCode = "UA",
            timezone = "Europe/Kyiv",
        ),
        permissionClass = CityPermissionClass.PRECISE,
        accuracyM = 25,
        observedAtEpochMillis = 1_000L,
        expiresAtEpochMillis = 2_000L,
        switchPending = false,
    )
}
