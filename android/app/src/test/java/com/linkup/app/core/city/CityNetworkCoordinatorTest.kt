package com.linkup.app.core.city

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

    private class FakeCityNetworkApi : CityNetworkApi {
        val mapCalls = mutableListOf<MapViewportQuery>()
        var failMap = false

        override suspend fun searchPlaces(query: String, locality: String?, limit: Int): List<CanonicalPlace> = emptyList()

        override suspend fun mapViewport(query: MapViewportQuery): List<MapCluster> {
            mapCalls += query
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
