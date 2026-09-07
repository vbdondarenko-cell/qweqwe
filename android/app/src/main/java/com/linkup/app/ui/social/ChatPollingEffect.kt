package com.linkup.app.ui.social

import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.rememberCoroutineScope
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import com.linkup.app.core.social.SocialCoordinator
import com.linkup.app.core.social.pollChat
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch

@Composable
internal fun ChatPollingEffect(lifecycle: Lifecycle, slotId: String, social: SocialCoordinator) {
    val scope = rememberCoroutineScope()
    DisposableEffect(lifecycle, slotId, social) {
        var job: Job? = null
        fun stop() { job?.cancel(); job = null }
        fun start() {
            if (job == null) job = scope.launch { pollChat { social.refreshChat(slotId) } }
        }
        val observer = LifecycleEventObserver { _, event ->
            when (event) {
                Lifecycle.Event.ON_RESUME -> start()
                Lifecycle.Event.ON_PAUSE, Lifecycle.Event.ON_STOP, Lifecycle.Event.ON_DESTROY -> stop()
                else -> Unit
            }
        }
        lifecycle.addObserver(observer)
        if (lifecycle.currentState.isAtLeast(Lifecycle.State.RESUMED)) start()
        onDispose {
            lifecycle.removeObserver(observer)
            stop()
            social.clearChat()
        }
    }
}
