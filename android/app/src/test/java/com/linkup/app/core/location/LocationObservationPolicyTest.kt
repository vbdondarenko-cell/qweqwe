package com.linkup.app.core.location

import com.linkup.app.core.network.CityPermissionClass
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertNull
import kotlin.test.assertTrue

class LocationObservationPolicyTest {
    @Test
    fun `unknown capture time cannot satisfy observation freshness`() {
        for (capturedAt in listOf(0L, -1L)) {
            assertFalse(deviceObservationMetadataIsUsable(CityPermissionClass.PRECISE, capturedAt, 1_000_000L, 10, false))
        }
    }

    @Test
    fun `missing nonfinite or nonpositive accuracy cannot become a one meter fix`() {
        assertNull(deviceAccuracyMeters(false, 10f))
        for (accuracy in listOf(Float.NaN, Float.POSITIVE_INFINITY, Float.NEGATIVE_INFINITY, 0f, -1f)) {
            assertNull(deviceAccuracyMeters(true, accuracy))
        }
    }

    @Test
    fun `positive accuracy preserves existing integer meter representation`() {
        assertEquals(1, deviceAccuracyMeters(true, 0.3f))
        assertEquals(25, deviceAccuracyMeters(true, 25.2f))
        assertEquals(26, deviceAccuracyMeters(true, 25.8f))
    }

    @Test
    fun `precise rejects coarse fix while approximate accepts it`() {
        val now = 1_000_000L
        assertFalse(deviceObservationMetadataIsUsable(CityPermissionClass.PRECISE, now - 1_000L, now, 1_200, false))
        assertTrue(deviceObservationMetadataIsUsable(CityPermissionClass.APPROXIMATE, now - 1_000L, now, 1_200, false))
    }

    @Test
    fun `mocked observation is always rejected`() {
        val now = 1_000_000L
        assertFalse(deviceObservationMetadataIsUsable(CityPermissionClass.PRECISE, now, now, 10, true))
        assertFalse(deviceObservationMetadataIsUsable(CityPermissionClass.APPROXIMATE, now, now, 10, true))
    }

    @Test
    fun `permission class controls freshness window`() {
        val now = 1_000_000L
        val threeMinutesAgo = now - 3L * 60L * 1000L
        assertFalse(deviceObservationMetadataIsUsable(CityPermissionClass.PRECISE, threeMinutesAgo, now, 100, false))
        assertTrue(deviceObservationMetadataIsUsable(CityPermissionClass.APPROXIMATE, threeMinutesAgo, now, 100, false))
    }

    @Test
    fun `future observation beyond skew is rejected`() {
        val now = 1_000_000L
        assertFalse(deviceObservationMetadataIsUsable(CityPermissionClass.PRECISE, now + 30_001L, now, 100, false))
        assertTrue(deviceObservationMetadataIsUsable(CityPermissionClass.PRECISE, now + 30_000L, now, 100, false))
    }
}
