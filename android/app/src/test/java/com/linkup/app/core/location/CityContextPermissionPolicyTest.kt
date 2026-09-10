package com.linkup.app.core.location

import com.linkup.app.core.network.CityPermissionClass
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class CityContextPermissionPolicyTest {
    @Test
    fun `disabled capability never requests location`() {
        assertEquals(
            CityContextPermissionAction.UNAVAILABLE,
            cityContextPermissionAction(false, null),
        )
        assertEquals(
            CityContextPermissionAction.UNAVAILABLE,
            cityContextPermissionAction(false, CityPermissionClass.PRECISE),
        )
    }

    @Test
    fun `enabled capability requests permission only when none exists`() {
        assertEquals(
            CityContextPermissionAction.REQUEST_PERMISSION,
            cityContextPermissionAction(true, null),
        )
        assertEquals(
            CityContextPermissionAction.RESOLVE_FROM_DEVICE,
            cityContextPermissionAction(true, CityPermissionClass.APPROXIMATE),
        )
        assertEquals(
            CityContextPermissionAction.RESOLVE_FROM_DEVICE,
            cityContextPermissionAction(true, CityPermissionClass.PRECISE),
        )
    }

    @Test
    fun `approximate or precise grant is sufficient for city resolution`() {
        assertTrue(cityLocationPermissionGranted(fineGranted = false, coarseGranted = true, observedPermissionClass = null))
        assertTrue(cityLocationPermissionGranted(fineGranted = true, coarseGranted = false, observedPermissionClass = null))
        assertTrue(cityLocationPermissionGranted(fineGranted = false, coarseGranted = false, observedPermissionClass = CityPermissionClass.APPROXIMATE))
        assertFalse(cityLocationPermissionGranted(fineGranted = false, coarseGranted = false, observedPermissionClass = null))
    }
}
