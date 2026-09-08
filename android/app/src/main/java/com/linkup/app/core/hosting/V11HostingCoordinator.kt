package com.linkup.app.core.hosting

import com.linkup.app.core.network.ApiException
import com.linkup.app.core.network.CreateSlotInput
import com.linkup.app.core.network.SlotModel
import com.linkup.app.core.network.SlotState
import com.linkup.app.core.network.V11HostingApi
import com.linkup.app.core.social.LoadState
import com.linkup.app.core.social.SocialError
import java.io.IOException
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.sync.Mutex

class V11HostingCoordinator(
    private val api: V11HostingApi,
) {
    private val mutationMutex = Mutex()
    private var generation = 0L
    private val mutableDraft = MutableStateFlow<LoadState<SlotModel>>(LoadState.Idle)
    val draft: StateFlow<LoadState<SlotModel>> = mutableDraft.asStateFlow()

    suspend fun createDraft(input: CreateSlotInput): Boolean {
        if (!mutationMutex.tryLock()) return false
        val request = ++generation
        mutableDraft.value = LoadState.Loading
        try {
            val draft = api.createDraft(input)
            if (request != generation) return false
            if (draft.state != SlotState.DRAFT) {
                mutableDraft.value = LoadState.Failure(
                    SocialError("protocol_error", "Server did not return a DRAFT LINK"),
                )
                return false
            }
            mutableDraft.value = LoadState.Content(draft)
            return true
        } catch (error: CancellationException) {
            if (request == generation) mutableDraft.value = LoadState.Idle
            throw error
        } catch (error: Exception) {
            if (request == generation) mutableDraft.value = LoadState.Failure(error.toHostingError())
            return false
        } finally {
            mutationMutex.unlock()
        }
    }

    suspend fun publishCurrent(): Boolean {
        val current = (mutableDraft.value as? LoadState.Content)?.value ?: return false
        if (current.state != SlotState.DRAFT || current.version <= 0) return false
        if (!mutationMutex.tryLock()) return false
        val request = ++generation
        mutableDraft.value = LoadState.Content(current, refreshing = true)
        try {
            val published = api.publishDraft(current.id, current.version)
            if (request != generation) return false
            if (published.state !in setOf(SlotState.PUBLISHED, SlotState.FILLING, SlotState.FULL)) {
                mutableDraft.value = LoadState.Content(
                    current,
                    refreshError = SocialError("protocol_error", "Server did not publish the LINK"),
                )
                return false
            }
            mutableDraft.value = LoadState.Content(published)
            return true
        } catch (error: CancellationException) {
            if (request == generation) mutableDraft.value = LoadState.Content(current)
            throw error
        } catch (error: Exception) {
            if (request == generation) {
                mutableDraft.value = LoadState.Content(current, refreshError = error.toHostingError())
            }
            return false
        } finally {
            mutationMutex.unlock()
        }
    }

    fun clear() {
        ++generation
        mutableDraft.value = LoadState.Idle
    }

    private fun Exception.toHostingError(): SocialError = when (this) {
        is ApiException -> SocialError(code = code, message = message, requestId = requestId)
        is IOException -> SocialError(code = "network_error", message = message ?: "Network request failed")
        else -> SocialError(code = "client_error", message = message ?: "Hosting request failed")
    }
}
