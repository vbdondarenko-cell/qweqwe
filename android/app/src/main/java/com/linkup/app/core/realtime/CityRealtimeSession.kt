package com.linkup.app.core.realtime

import com.linkup.app.core.network.ApiException
import com.linkup.app.core.network.CityContextModel
import com.linkup.app.core.social.LoadState
import com.linkup.app.core.social.SocialError
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.isActive

/** Owned by the signed-in account/capability lifecycle, not by transient city state. */
internal suspend fun runCityRealtimeSession(
    context: StateFlow<LoadState<CityContextModel>>,
    retryMillis: Long,
    readContext: suspend () -> Unit,
    recoverContext: suspend () -> Unit,
    refreshSnapshot: suspend () -> Boolean,
    poll: suspend () -> Long,
    onUnauthorized: suspend () -> Unit,
    onForbidden: suspend () -> Unit,
    pause: suspend (Long) -> Unit = { delay(it) },
) {
    require(retryMillis > 0)
    var snapshotRequired = true
    var localityId: String? = null
    while (currentCoroutineContext().isActive) {
        // No polling/GPS loop when permission or a supported locality is absent.
        val state = context.first {
            it is LoadState.Content || (it is LoadState.Failure &&
                (it.error.canRetryCityRead() || it.error.isCityAccessLoss()))
        }
        if (state is LoadState.Failure) {
            when {
                state.error.httpStatus == 401 || state.error.code == "unauthorized" -> {
                    onUnauthorized()
                    return
                }
                state.error.httpStatus == 403 || state.error.code == "capability_disabled" -> {
                    onForbidden()
                    return
                }
                else -> {
                    pause(retryMillis)
                    // A manual resolution may have completed while we were waiting.
                    // Ambiguous resolver POSTs are reconciled by GET, never replayed.
                    if (context.value === state) readContext()
                    continue
                }
            }
        }
        val current = (state as LoadState.Content<CityContextModel>).value
        if (localityId != current.locality.id) snapshotRequired = true
        try {
            if (snapshotRequired) {
                if (!refreshSnapshot()) {
                    pause(retryMillis)
                    continue
                }
                localityId = current.locality.id
                snapshotRequired = false
            }
            pause(poll())
        } catch (error: CancellationException) {
            throw error
        } catch (error: ApiException) {
            when {
                error.status == 401 -> {
                    onUnauthorized()
                    return
                }
                error.status == 403 -> {
                    onForbidden()
                    return
                }
                error.status == 404 && error.code == "city_context_unavailable" -> {
                    snapshotRequired = true
                    recoverContext()
                }
            }
            pause(retryMillis)
        } catch (_: Exception) {
            pause(retryMillis)
        }
    }
}

private fun SocialError.canRetryCityRead(): Boolean =
    code == "network_error" || httpStatus in setOf(408, 429, 500, 502, 503, 504)

private fun SocialError.isCityAccessLoss(): Boolean =
    httpStatus in setOf(401, 403) || code in setOf("unauthorized", "capability_disabled")
