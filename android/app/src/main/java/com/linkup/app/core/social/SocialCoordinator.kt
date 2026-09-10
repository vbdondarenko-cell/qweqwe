package com.linkup.app.core.social

import com.linkup.app.core.network.ApiException
import com.linkup.app.core.network.ChatMessage
import com.linkup.app.core.network.CreateSlotInput
import com.linkup.app.core.network.EditSlotInput
import com.linkup.app.core.network.MySlotsView
import com.linkup.app.core.network.PendingSlotRequest
import com.linkup.app.core.network.SlotAccessMode
import com.linkup.app.core.network.SlotModel
import com.linkup.app.core.network.SlotOrganizer
import com.linkup.app.core.network.SlotState
import com.linkup.app.core.network.SlotViewerState
import com.linkup.app.core.network.SocialApi
import com.linkup.app.core.network.V11AccessApi
import java.io.IOException
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.sync.Mutex

sealed interface LoadState<out T> {
    data object Idle : LoadState<Nothing>
    data object Loading : LoadState<Nothing>
    data object Empty : LoadState<Nothing>
    data class Content<T>(
        val value: T,
        val refreshing: Boolean = false,
        val refreshError: SocialError? = null,
    ) : LoadState<T>
    data class Failure(val error: SocialError) : LoadState<Nothing>
}

data class SocialError(
    val code: String,
    val message: String,
    val requestId: String? = null,
    val httpStatus: Int? = null,
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
    // Called by UI coroutines on Main. Tokens invalidate responses after navigation,
    // access changes, or account disposal; cancellation alone cannot do that.
    private var selectionGeneration = 0L
    private var pulseRequest = 0L
    private var mySlotsRequest = 0L
    private var pulseItems = emptyList<SlotModel>()
    private var slotRequest = 0L
    private var acceptedRequest = 0L
    private var pendingRequest = 0L
    private var chatRequest = 0L

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

    private val mutableMySlots = MutableStateFlow<LoadState<List<SlotModel>>>(LoadState.Idle)
    val mySlots: StateFlow<LoadState<List<SlotModel>>> = mutableMySlots.asStateFlow()

    suspend fun refreshMySlots(view: MySlotsView) {
        val request = ++mySlotsRequest
        val previous = mutableMySlots.value
        mutableMySlots.value = previous.asRefreshingOrLoading()
        try {
            val items = api.mySlots(view)
            if (request == mySlotsRequest) mutableMySlots.value = if (items.isEmpty()) LoadState.Empty else LoadState.Content(items)
        } catch (error: CancellationException) {
            if (request == mySlotsRequest) mutableMySlots.value = previous
            throw error
        } catch (error: Exception) {
            if (request == mySlotsRequest) mutableMySlots.value = previous.afterRefreshFailure(error)
        }
    }

    suspend fun refreshPulse() {
        val request = ++pulseRequest
        val previous = mutablePulse.value
        mutablePulse.value = previous.asRefreshingOrLoading()
        try {
            val items = api.pulse()
            if (request != pulseRequest) return
            pulseItems = items
            mutablePulse.value = if (items.isEmpty()) LoadState.Empty else LoadState.Content(items)
        } catch (error: CancellationException) {
            if (request == pulseRequest) mutablePulse.value = previous
            throw error
        } catch (error: Exception) {
            if (request != pulseRequest) return
            mutablePulse.value = previous.afterRefreshFailure(error)
        }
    }

    suspend fun openSlot(slotId: String) {
        val current = (mutableSelectedSlot.value as? LoadState.Content)?.value
        if (current?.id != slotId) clearSelected()
        val generation = selectionGeneration
        val request = ++slotRequest
        val previous = mutableSelectedSlot.value
        mutableSelectedSlot.value = previous.asRefreshingOrLoading()
        try {
            val slot = api.getSlot(slotId)
            if (generation != selectionGeneration || request != slotRequest) return
            applySelected(slot)
        } catch (error: CancellationException) {
            if (generation == selectionGeneration && request == slotRequest) mutableSelectedSlot.value = previous
            throw error
        } catch (error: Exception) {
            if (generation != selectionGeneration || request != slotRequest) return
            mutableSelectedSlot.value = previous.afterRefreshFailure(error)
        }
    }

    suspend fun createSlot(input: CreateSlotInput) = mutateSlot { api.createSlot(input) }
    suspend fun editSlot(slotId: String, input: EditSlotInput) = mutateSlot { api.editSlot(slotId, input) }
    suspend fun cancelSlot(slotId: String, expectedVersion: Long) = mutateSlot { api.cancelSlot(slotId, expectedVersion) }
    suspend fun requestSlot(slotId: String) = mutateSlot { api.requestSlot(slotId) }

    suspend fun participate(slot: SlotModel): Boolean = when (slot.accessMode) {
        SlotAccessMode.INSTANT -> {
            val accessApi = api as? V11AccessApi ?: return false
            mutateSlot { accessApi.joinSlot(slot.id) }
        }
        SlotAccessMode.APPROVAL, SlotAccessMode.WAITLIST -> requestSlot(slot.id)
    }

    suspend fun leaveSlot(slotId: String) = mutateSlot { api.leaveSlot(slotId) }

    suspend fun approveRequest(slotId: String, userId: String) {
        if (mutateSlot { api.approveRequest(slotId, userId) }) refreshPending(slotId)
    }

    suspend fun rejectRequest(slotId: String, userId: String) {
        if (mutateSlot { api.rejectRequest(slotId, userId) }) refreshPending(slotId)
    }

    suspend fun startSlot(slotId: String) = mutateSlot { api.startSlot(slotId) }
    suspend fun completeSlot(slotId: String) = mutateSlot { api.completeSlot(slotId) }

    suspend fun refreshPending(slotId: String) {
        val generation = selectionGeneration
        val request = ++pendingRequest
        val previous = mutablePending.value
        mutablePending.value = previous.asRefreshingOrLoading()
        try {
            val items = api.pendingRequests(slotId)
            if (generation != selectionGeneration || request != pendingRequest) return
            mutablePending.value = if (items.isEmpty()) LoadState.Empty else LoadState.Content(items)
        } catch (error: CancellationException) {
            if (generation == selectionGeneration && request == pendingRequest) mutablePending.value = previous
            throw error
        } catch (error: Exception) {
            if (generation != selectionGeneration || request != pendingRequest) return
            mutablePending.value = previous.afterRefreshFailure(error)
        }
    }

    private val mutableAccepted = MutableStateFlow<LoadState<List<SlotOrganizer>>>(LoadState.Idle)
    val accepted: StateFlow<LoadState<List<SlotOrganizer>>> = mutableAccepted.asStateFlow()

    suspend fun removeParticipant(slotId: String, userId: String, expectedVersion: Long): Boolean {
        val success = mutateSlot { api.removeParticipant(slotId, userId, expectedVersion) }
        if (success) refreshAccepted(slotId)
        return success
    }

    suspend fun refreshAccepted(slotId: String) {
        val current = (mutableSelectedSlot.value as? LoadState.Content)?.value ?: return
        if (current.id != slotId || current.viewerState != SlotViewerState.HOST || current.state in terminalStates) return
        val generation = selectionGeneration
        val request = ++acceptedRequest
        val previous = mutableAccepted.value
        mutableAccepted.value = previous.asRefreshingOrLoading()
        try {
            val items = api.acceptedParticipants(slotId)
            if (generation == selectionGeneration && request == acceptedRequest) {
                mutableAccepted.value = if (items.isEmpty()) LoadState.Empty else LoadState.Content(items)
            }
        } catch (error: CancellationException) {
            if (generation == selectionGeneration && request == acceptedRequest) mutableAccepted.value = previous
            throw error
        } catch (error: Exception) {
            if (generation == selectionGeneration && request == acceptedRequest) mutableAccepted.value = previous.afterRefreshFailure(error)
        }
    }

    private fun clearAccepted() {
        ++acceptedRequest
        mutableAccepted.value = LoadState.Idle
    }

    suspend fun refreshChat(slotId: String, limit: Int = 100): ChatRefreshResult {
        // A snapshot started during a send could replace its acknowledged message.
        if (actionMutex.isLocked) return ChatRefreshResult.SUCCESS
        val generation = selectionGeneration
        val request = ++chatRequest
        val previous = mutableChat.value
        mutableChat.value = previous.asRefreshingOrLoading()
        try {
            val items = api.chatMessages(slotId, limit)
            if (generation != selectionGeneration || request != chatRequest) return ChatRefreshResult.SUCCESS
            mutableChat.value = if (items.isEmpty()) LoadState.Empty else LoadState.Content(items)
            return ChatRefreshResult.SUCCESS
        } catch (error: CancellationException) {
            if (generation == selectionGeneration && request == chatRequest) mutableChat.value = previous
            throw error
        } catch (error: Exception) {
            if (generation != selectionGeneration || request != chatRequest) return ChatRefreshResult.SUCCESS
            val stop = error.isDefinitiveAccessError()
            mutableChat.value = if (stop) LoadState.Failure(error.toSocialError()) else previous.afterRefreshFailure(error)
            return if (stop) ChatRefreshResult.STOP else ChatRefreshResult.RETRY
        }
    }

    fun clearChat() {
        ++chatRequest
        mutableChat.value = LoadState.Idle
    }

    suspend fun sendChatMessage(slotId: String, text: String) {
        // A Mutex queue would send a second mutation after a rapid double tap.
        if (!actionMutex.tryLock()) return
        val generation = selectionGeneration
        val thread = ++chatRequest
        mutableMutation.value = MutationState.Running
        try {
            val message = api.sendChatMessage(slotId, text)
            if (generation != selectionGeneration) return
            if (thread == chatRequest) {
                val current = (mutableChat.value as? LoadState.Content)?.value.orEmpty()
                mutableChat.value = LoadState.Content((current + message).distinctBy { it.id }.takeLast(100))
            }
            mutableMutation.value = MutationState.Idle
        } catch (error: CancellationException) {
            if (generation == selectionGeneration) mutableMutation.value = MutationState.Idle
            throw error
        } catch (error: Exception) {
            if (generation == selectionGeneration) {
                if (error.isDefinitiveAccessError()) {
                    ++chatRequest
                    mutableChat.value = LoadState.Failure(error.toSocialError())
                }
                mutableMutation.value = MutationState.Failed(error.toSocialError())
            }
        } finally {
            actionMutex.unlock()
        }
    }

    fun clearSelected() {
        clearAccepted()
        ++selectionGeneration
        ++slotRequest
        ++pendingRequest
        ++chatRequest
        mutableSelectedSlot.value = LoadState.Idle
        mutablePending.value = LoadState.Idle
        mutableChat.value = LoadState.Idle
        mutableMutation.value = MutationState.Idle
    }

    fun clearAll() {
        ++mySlotsRequest
        mutableMySlots.value = LoadState.Idle
        ++pulseRequest
        pulseItems = emptyList()
        mutablePulse.value = LoadState.Idle
        clearSelected()
    }

    private suspend fun mutateSlot(action: suspend () -> SlotModel): Boolean {
        if (!actionMutex.tryLock()) return false
        val generation = selectionGeneration
        mutableMutation.value = MutationState.Running
        try {
            val slot = action()
            if (generation != selectionGeneration) return false
            ++slotRequest
            applySelected(slot)
            reconcilePulse(slot)
            mutableMutation.value = MutationState.Idle
            return true
        } catch (error: CancellationException) {
            if (generation == selectionGeneration) mutableMutation.value = MutationState.Idle
            throw error
        } catch (error: Exception) {
            if (generation == selectionGeneration) mutableMutation.value = MutationState.Failed(error.toSocialError())
            return false
        } finally {
            actionMutex.unlock()
        }
    }

    private fun applySelected(slot: SlotModel) {
        clearAccepted()
        mutableSelectedSlot.value = LoadState.Content(slot)
        when (slot.viewerState) {
            SlotViewerState.HOST -> Unit
            SlotViewerState.ACCEPTED -> {
                ++pendingRequest
                mutablePending.value = LoadState.Idle
            }
            SlotViewerState.PENDING, SlotViewerState.NONE -> {
                ++pendingRequest
                ++chatRequest
                mutablePending.value = LoadState.Idle
                mutableChat.value = LoadState.Idle
            }
        }
        if (slot.state in terminalStates) {
            ++pendingRequest
            ++chatRequest
            mutableChat.value = LoadState.Idle
            mutablePending.value = LoadState.Idle
        }
    }

    private fun reconcilePulse(slot: SlotModel) {
        ++pulseRequest
        val current = pulseItems
        val discoverable = slot.state == SlotState.PUBLISHED || slot.state == SlotState.FILLING || slot.state == SlotState.FULL
        val without = current.filterNot { it.id == slot.id }
        val updated = if (discoverable) listOf(slot) + without else without
        pulseItems = updated
        mutablePulse.value = if (updated.isEmpty()) LoadState.Empty else LoadState.Content(updated)
    }

    private fun <T> LoadState<T>.asRefreshingOrLoading(): LoadState<T> = when (this) {
        is LoadState.Content -> copy(refreshing = true, refreshError = null)
        else -> LoadState.Loading
    }

    private fun <T> LoadState<T>.afterRefreshFailure(error: Exception): LoadState<T> {
        if (error.isDefinitiveAccessError()) return LoadState.Failure(error.toSocialError())
        return when (this) {
            is LoadState.Content -> copy(refreshing = false, refreshError = error.toSocialError())
            else -> LoadState.Failure(error.toSocialError())
        }
    }

    private fun Exception.isDefinitiveAccessError(): Boolean =
        this is ApiException && status in setOf(400, 401, 403, 404, 409)

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
