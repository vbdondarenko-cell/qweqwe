package com.linkup.app.core.capability

import com.linkup.app.core.network.CapabilityApi
import com.linkup.app.core.network.CapabilitySnapshot
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

class CapabilityCoordinator(
    private val api: CapabilityApi,
) {
    private val mutableSnapshot = MutableStateFlow(CapabilitySnapshot.disabled())
    val snapshot: StateFlow<CapabilitySnapshot> = mutableSnapshot.asStateFlow()

    suspend fun refresh() {
        mutableSnapshot.value = try {
            api.capabilities()
        } catch (error: CancellationException) {
            throw error
        } catch (_: Exception) {
            CapabilitySnapshot.disabled()
        }
    }

    fun reset() {
        mutableSnapshot.value = CapabilitySnapshot.disabled()
    }
}
