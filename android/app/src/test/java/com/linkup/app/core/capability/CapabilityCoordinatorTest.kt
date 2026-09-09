package com.linkup.app.core.capability

import com.linkup.app.core.network.CapabilityApi
import com.linkup.app.core.network.CapabilityKey
import com.linkup.app.core.network.CapabilitySnapshot
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class CapabilityCoordinatorTest {
    @Test
    fun `registry failure fails closed`() = runBlocking {
        val coordinator = CapabilityCoordinator(object : CapabilityApi {
            override suspend fun capabilities(): CapabilitySnapshot = error("offline")
        })
        coordinator.refresh()
        CapabilityKey.entries.forEach { key -> assertFalse(coordinator.snapshot.value.enabled(key)) }
    }

    @Test
    fun `only server enabled keys become active`() = runBlocking {
        val coordinator = CapabilityCoordinator(object : CapabilityApi {
            override suspend fun capabilities(): CapabilitySnapshot = CapabilitySnapshot(
                revision = 9,
                values = mapOf(CapabilityKey.REALTIME to true),
            )
        })
        coordinator.refresh()
        assertTrue(coordinator.snapshot.value.enabled(CapabilityKey.REALTIME))
        assertFalse(coordinator.snapshot.value.enabled(CapabilityKey.MAP))
    }
}
