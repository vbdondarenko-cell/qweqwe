package com.linkup.app.core.city

import com.linkup.app.core.network.ApiException
import com.linkup.app.core.network.CanonicalPlace
import com.linkup.app.core.network.CityNetworkApi
import com.linkup.app.core.network.MapCluster
import com.linkup.app.core.network.MapViewportQuery
import com.linkup.app.core.network.SlotModel
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue
import kotlinx.coroutines.test.runTest

class CityNetworkCoordinatorTest {
    @Test
    fun `city invalidation has no map work before viewport was loaded`() = runTest {
        val api = FakeCityNetworkApi()
        val coordinator = CityNetworkCoordinator(api)

        assertTrue(coordinator.refreshCurrentMapIfLoaded())
        assertEquals(0, api.mapCalls.size)
    }

    @Test
    fun `city invalidation rereads last canonical map viewport`() = runTest {
        val api = FakeCityNetworkApi()
        val coordinator = CityNetworkCoordinator(api)
        val query = viewport()

        coordinator.refreshMap(query)
        assertTrue(coordinator.refreshCurrentMapIfLoaded())

        assertEquals(listOf(query, query), api.mapCalls)
    }

    @Test
    fun `failed canonical map reread blocks cursor acknowledgement`() = runTest {
        val api = FakeCityNetworkApi()
        val coordinator = CityNetworkCoordinator(api)
        coordinator.refreshMap(viewport())
        api.failMap = true

        assertFalse(coordinator.refreshCurrentMapIfLoaded())
    }


    @Test
    fun `place search forwards canonical locality id`() = runTest {
        val api = FakeCityNetworkApi()
        val coordinator = CityNetworkCoordinator(api)
        val localityId = "00000000-0000-0000-0000-000000000123"

        coordinator.searchPlaces("  cafe  ", localityId)

        assertEquals(1, api.placeCalls.size)
        assertEquals("cafe", api.placeCalls.single().first)
        assertEquals(localityId, api.placeCalls.single().second)
    }

    @Test
    fun `map API failure preserves http status for access recovery`() = runTest {
        val api = FakeCityNetworkApi()
        val coordinator = CityNetworkCoordinator(api)
        api.mapError = ApiException(403, "capability_disabled", "Map is disabled", "request-1")

        coordinator.refreshMap(viewport())

        val failure = coordinator.map.value as com.linkup.app.core.social.LoadState.Failure
        assertEquals("capability_disabled", failure.error.code)
        assertEquals(403, failure.error.httpStatus)
        assertEquals("request-1", failure.error.requestId)
    }

    private class FakeCityNetworkApi : CityNetworkApi {
        val placeCalls = mutableListOf<Pair<String, String?>>()
        val mapCalls = mutableListOf<MapViewportQuery>()
        var failMap = false
        var mapError: Exception? = null

        override suspend fun searchPlaces(query: String, localityId: String?, limit: Int): List<CanonicalPlace> {
            placeCalls += query to localityId
            return emptyList()
        }

        override suspend fun mapViewport(query: MapViewportQuery): List<MapCluster> {
            mapCalls += query
            mapError?.let { throw it }
            if (failMap) error("map unavailable")
            return emptyList()
        }

        override suspend fun mapPlaceSlots(
            placeId: String,
            fromEpochMillis: Long,
            toEpochMillis: Long,
            limit: Int,
        ): List<SlotModel> = emptyList()
    }

    private fun viewport() = MapViewportQuery(
        westE6 = 32_000_000,
        southE6 = 49_300_000,
        eastE6 = 32_200_000,
        northE6 = 49_500_000,
        zoom = 13,
        fromEpochMillis = 1_000,
        toEpochMillis = 2_000,
    )
}
