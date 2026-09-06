package com.linkup.app.core.social

import com.linkup.app.core.network.ApiException
import com.linkup.app.core.network.ChatMessage
import com.linkup.app.core.network.CreateSlotInput
import com.linkup.app.core.network.EditSlotInput
import com.linkup.app.core.network.PendingSlotRequest
import com.linkup.app.core.network.SocialApi
import com.linkup.app.core.network.SlotModel
import com.linkup.app.core.network.SlotState
import com.linkup.app.core.network.SlotViewerState
import java.io.IOException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock

sealed interface LoadState<out T> {
    data object Idle : LoadState<Nothing>
    data object Loading : LoadState<Nothing>
    data object Empty : LoadState<Nothing>
    data class Content<T>(val value: T) : LoadState<T>
    data class Failure(val error: SocialError) : LoadState<Nothing>
}

data class SocialError(
    val code: String,
    val message: String,
    val requestId: String? = null,
)

sealed interface MutationState {
    data object Idle : MutationState
    data object Running : MutationState
    data class Failed(val error: SocialError) : MutationState
}

class SocialCoordinator(
    private val api: SocialApi,
) {
    private val actionMutex = Mutex()

    private val mutablePulse = MutableStateFlow<LoadState<List<SlotModel>>>(LoadState.Idle)
    val pulse: StateFlow<LoadState<List<SlotModel>>> = mutablePulse.asStateFlow()

    private val mutableSelectedSlot = MutableStateFlow<LoadState<SlotModel>>(LoadState.Idle)
    val selectedSlot: StateFlow<LoadState<SlotModel>> = mutableSelectedSlot.asStateFlow()

    private val mutablePending = MutableStateFlow<LoadState<List<PendingSlotRequest>>>(LoadState.Idle)
    val pending: StateFlow<LoadState<List<PendingSlotRequest>>> = mutablePending.asStateFlow()

    private val mutableChat = MutableStateFlow<LoadState<List<ChatMessage>>>(LoadState.Idle)
    val chat: StateFlow<LoadState<List<ChatMessage>>> = mutableChat.asStateFlow()

    private val mutableMutation = MutableStateFlow<MutationState>(MutationState.Idle)
    val mutation: StateFlow<MutationState> = mutableMutation.asStateFlow()

    suspend fun refreshPulse() {
        mutablePulse.value = LoadState.Loading
        try {
            val items = api.pulse()
            mutablePulse.value = if (items.isEmpty()) LoadState.Empty else LoadState.Content(items)
        } catch (error: Exception) {
            mutablePulse.value = LoadState.Failure(error.toSocialError())
        }
    }

    suspend fun openSlot(slotId: String) {
        mutableSelectedSlot.value = LoadState.Loading
        try {
            applySelected(api.getSlot(slotId))
        } catch (error: Exception) {
            mutableSelectedSlot.value = LoadState.Failure(error.toSocialError())
        }
    }

    suspend fun createSlot(input: CreateSlotInput) = mutateSlot { api.createSlot(input) }
    suspend fun editSlot(slotId: String, input: EditSlotInput) = mutateSlot { api.editSlot(slotId, input) }
    suspend fun cancelSlot(slotId: String, expectedVersion: Long) = mutateSlot { api.cancelSlot(slotId, expectedVersion) }
    suspend fun requestSlot(slotId: String) = mutateSlot { api.requestSlot(slotId) }
    suspend fun leaveSlot(slotId: String) = mutateSlot { api.leaveSlot(slotId) }

    suspend fun approveRequest(slotId: String, userId: String) {
        mutateSlot { api.approveRequest(slotId, userId) }
        if (mutableMutation.value is MutationState.Idle) refreshPending(slotId)
    }

    suspend fun rejectRequest(slotId: String, userId: String) {
        mutateSlot { api.rejectRequest(slotId, userId) }
        if (mutableMutation.value is MutationState.Idle) refreshPending(slotId)
    }

    suspend fun startSlot(slotId: String) = mutateSlot { api.startSlot(slotId) }
    suspend fun completeSlot(slotId: String) = mutateSlot { api.completeSlot(slotId) }

    suspend fun refreshPending(slotId: String) {
        mutablePending.value = LoadState.Loading
        try {
            val items = api.pendingRequests(slotId)
            mutablePending.value = if (items.isEmpty()) LoadState.Empty else LoadState.Content(items)
        } catch (error: Exception) {
            mutablePending.value = LoadState.Failure(error.toSocialError())
        }
    }

    suspend fun refreshChat(slotId: String, limit: Int = 100) {
        mutableChat.value = LoadState.Loading
        try {
            val items = api.chatMessages(slotId, limit)
            mutableChat.value = if (items.isEmpty()) LoadState.Empty else LoadState.Content(items)
        } catch (error: Exception) {
            mutableChat.value = LoadState.Failure(error.toSocialError())
        }
    }

    suspend fun sendChatMessage(slotId: String, text: String) {
        actionMutex.withLock {
            mutableMutation.value = MutationState.Running
            try {
                val message = api.sendChatMessage(slotId, text)
                val current = (mutableChat.value as? LoadState.Content)?.value.orEmpty()
                mutableChat.value = LoadState.Content(current + message)
                mutableMutation.value = MutationState.Idle
            } catch (error: Exception) {
                mutableMutation.value = MutationState.Failed(error.toSocialError())
            }
        }
    }

    fun clearSelected() {
        mutableSelectedSlot.value = LoadState.Idle
        mutablePending.value = LoadState.Idle
        mutableChat.value = LoadState.Idle
        mutableMutation.value = MutationState.Idle
    }

    private suspend fun mutateSlot(action: suspend () -> SlotModel) {
        actionMutex.withLock {
            mutableMutation.value = MutationState.Running
            try {
                val slot = action()
                applySelected(slot)
                reconcilePulse(slot)
                mutableMutation.value = MutationState.Idle
            } catch (error: Exception) {
                mutableMutation.value = MutationState.Failed(error.toSocialError())
            }
        }
    }

    private fun applySelected(slot: SlotModel) {
        mutableSelectedSlot.value = LoadState.Content(slot)
        when (slot.viewerState) {
            SlotViewerState.HOST -> Unit
            SlotViewerState.ACCEPTED -> mutablePending.value = LoadState.Idle
            SlotViewerState.PENDING, SlotViewerState.NONE -> {
                mutablePending.value = LoadState.Idle
                mutableChat.value = LoadState.Idle
            }
        }
        if (slot.state in terminalStates) {
            mutableChat.value = LoadState.Idle
            mutablePending.value = LoadState.Idle
        }
    }

    private fun reconcilePulse(slot: SlotModel) {
        val current = when (val state = mutablePulse.value) {
            is LoadState.Content -> state.value
            else -> return
        }
        val discoverable = slot.state == SlotState.PUBLISHED || slot.state == SlotState.FILLING || slot.state == SlotState.FULL
        val without = current.filterNot { it.id == slot.id }
        val updated = if (discoverable) listOf(slot) + without else without
        mutablePulse.value = if (updated.isEmpty()) LoadState.Empty else LoadState.Content(updated)
    }

    private fun Exception.toSocialError(): SocialError = when (this) {
        is ApiException -> SocialError(code = code, message = message, requestId = requestId)
        is IOException -> SocialError(code = "network_error", message = message ?: "Network request failed")
        else -> SocialError(code = "client_error", message = message ?: "Request failed")
    }

    private companion object {
        val terminalStates = setOf(
            SlotState.COMPLETED,
            SlotState.CANCELLED,
            SlotState.EXPIRED,
            SlotState.MODERATED,
        )
    }
}
