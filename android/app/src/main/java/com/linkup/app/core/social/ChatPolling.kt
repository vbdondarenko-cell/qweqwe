package com.linkup.app.core.social

import kotlinx.coroutines.delay

enum class ChatRefreshResult { SUCCESS, RETRY, STOP }

// Operational defaults, not a realtime latency guarantee. Requests are sequential.
data class ChatPollingPolicy(val intervalMillis: Long = 5_000, val maxRetryMillis: Long = 60_000) {
    init { require(intervalMillis > 0 && maxRetryMillis >= intervalMillis) }
}

suspend fun pollChat(
    policy: ChatPollingPolicy = ChatPollingPolicy(),
    pause: suspend (Long) -> Unit = { delay(it) },
    refresh: suspend () -> ChatRefreshResult,
) {
    var nextDelay = policy.intervalMillis
    while (true) {
        when (refresh()) {
            ChatRefreshResult.STOP -> return
            ChatRefreshResult.SUCCESS -> nextDelay = policy.intervalMillis
            ChatRefreshResult.RETRY -> Unit
        }
        pause(nextDelay)
        nextDelay = if (nextDelay > policy.maxRetryMillis / 2) policy.maxRetryMillis
            else (nextDelay * 2).coerceAtMost(policy.maxRetryMillis)
    }
}
