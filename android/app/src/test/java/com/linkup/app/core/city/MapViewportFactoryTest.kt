package com.linkup.app.core.city

import com.linkup.app.core.network.CanonicalPlace
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class MapViewportFactoryTest {
    @Test
    fun `viewport stays valid around antimeridian`() {
        val place = place(latitudeE6 = 10_000_000, longitudeE6 = 179_900_000)
        val query = mapViewportAround(place, zoom = 13, fromEpochMillis = 1_000, toEpochMillis = 2_000)

        assertTrue(query.westE6 > query.eastE6)
        assertTrue(longitudeFraction(place.longitudeE6, query.westE6, query.eastE6) in 0.49f..0.51f)
    }

    @Test
    fun `viewport clamps latitude at poles without collapsing`() {
        val place = place(latitudeE6 = 90_000_000, longitudeE6 = 0)
        val query = mapViewportAround(place, zoom = 16, fromEpochMillis = 1_000, toEpochMillis = 2_000)

        assertEquals(90_000_000, query.northE6)
        assertTrue(query.southE6 < query.northE6)
        assertEquals(1f, latitudeFraction(place.latitudeE6, query.southE6, query.northE6))
    }

    @Test
    fun `center projects near middle of ordinary viewport`() {
        val place = place(latitudeE6 = 50_450_000, longitudeE6 = 30_523_000)
        val query = mapViewportAround(place, zoom = 13, fromEpochMillis = 1_000, toEpochMillis = 2_000)

        assertTrue(latitudeFraction(place.latitudeE6, query.southE6, query.northE6) in 0.49f..0.51f)
        assertTrue(longitudeFraction(place.longitudeE6, query.westE6, query.eastE6) in 0.49f..0.51f)
    }

    private fun place(latitudeE6: Int, longitudeE6: Int) = CanonicalPlace(
        id = "00000000-0000-0000-0000-000000000001",
        name = "Place",
        category = null,
        locality = null,
        countryCode = null,
        latitudeE6 = latitudeE6,
        longitudeE6 = longitudeE6,
        precisionM = 100,
    )
}
