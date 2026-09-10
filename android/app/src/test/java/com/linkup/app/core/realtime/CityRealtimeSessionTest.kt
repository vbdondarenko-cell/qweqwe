package com.linkup.app.core.realtime

import com.linkup.app.core.network.ApiException
import com.linkup.app.core.network.CityContextModel
import com.linkup.app.core.network.CityLocality
import com.linkup.app.core.network.CityPermissionClass
import com.linkup.app.core.social.LoadState
import com.linkup.app.core.social.SocialError
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancelAndJoin
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.yield

class CityRealtimeSessionTest {
    @Test
    fun `expiry recovery survives transient empty city and refreshes snapshot before polling`() = runTest {
        val fixture = Fixture()
        val resumed = CompletableDeferred<Unit>()
        fixture.recover = {
            fixture.context.value = LoadState.Empty
            yield() // Give any state-driven owner a chance to cancel recovery.
            fixture.context.value = LoadState.Content(city())
        }
        fixture.poll = {
            if (fixture.polls == 1) throw expired()
            resumed.complete(Unit)
            awaitCancellation()
        }
        val job = launch { fixture.run() }
        resumed.await()
        job.cancelAndJoin()
        assertEquals(1, fixture.recoveries)
        assertEquals(2, fixture.snapshots)
        assertEquals(2, fixture.polls)
    }

    @Test
    fun `ambiguous resolver failure is reconciled by GET without repeating resolution`() = runTest {
        val fixture = Fixture()
        val resumed = CompletableDeferred<Unit>()
        fixture.recover = {
            fixture.context.value = LoadState.Failure(SocialError("request_failed", "gateway", httpStatus = 503))
        }
        fixture.read = { fixture.context.value = LoadState.Content(city()) }
        fixture.poll = {
            if (fixture.polls == 1) throw expired()
            resumed.complete(Unit)
            awaitCancellation()
        }
        val job = launch { fixture.run() }
        resumed.await()
        job.cancelAndJoin()
        assertEquals(1, fixture.recoveries)
        assertEquals(1, fixture.reads)
        assertEquals(2, fixture.snapshots)
    }

    @Test
    fun `no city suspends polling until explicit resolution supplies a context`() = runTest {
        val fixture = Fixture()
        fixture.context.value = LoadState.Empty
        val resumed = CompletableDeferred<Unit>()
        fixture.poll = { resumed.complete(Unit); awaitCancellation() }
        val job = launch { fixture.run() }
        yield()
        assertEquals(0, fixture.polls)
        assertEquals(0, fixture.reads)
        assertEquals(0, fixture.recoveries)
        fixture.context.value = LoadState.Content(city())
        resumed.await()
        job.cancelAndJoin()
        assertEquals(1, fixture.snapshots)
    }

    @Test
    fun `unresolved expired city waits without another GPS attempt`() = runTest {
        val fixture = Fixture()
        val recovered = CompletableDeferred<Unit>()
        fixture.recover = {
            fixture.context.value = LoadState.Empty
            recovered.complete(Unit)
        }
        fixture.poll = { throw expired() }
        val job = launch { fixture.run() }
        recovered.await()
        yield()
        assertEquals(1, fixture.recoveries)
        assertEquals(1, fixture.polls)
        job.cancelAndJoin()
    }

    @Test
    fun `forbidden stream stops instead of retrying denied endpoint`() = runTest {
        val fixture = Fixture()
        fixture.poll = { throw ApiException(403, "capability_disabled", "disabled") }
        fixture.run()
        assertEquals(1, fixture.forbidden)
        assertEquals(1, fixture.polls)
        assertEquals(0, fixture.recoveries)
    }

    @Test
    fun `unauthorized context stops session before any city poll`() = runTest {
        val fixture = Fixture()
        fixture.context.value = LoadState.Failure(SocialError("request_failed", "unauthorized", httpStatus = 401))
        fixture.run()
        assertEquals(1, fixture.unauthorized)
        assertEquals(0, fixture.polls)
    }

    @Test
    fun `account or lifecycle cancellation cancels recovery and prevents further polls`() = runTest {
        val fixture = Fixture()
        val started = CompletableDeferred<Unit>()
        var released = false
        fixture.poll = { throw expired() }
        fixture.recover = {
            started.complete(Unit)
            try { awaitCancellation() }
            finally { released = true }
        }
        val job = launch { fixture.run() }
        started.await()
        job.cancelAndJoin()
        assertTrue(released)
        assertEquals(1, fixture.polls)
    }

    @Test
    fun `failed authoritative snapshot prevents polling until refresh succeeds`() = runTest {
        val fixture = Fixture()
        val polled = CompletableDeferred<Unit>()
        fixture.snapshot = { fixture.snapshots > 1 }
        fixture.poll = { polled.complete(Unit); awaitCancellation() }
        val job = launch { fixture.run() }
        polled.await()
        job.cancelAndJoin()
        assertEquals(2, fixture.snapshots)
        assertEquals(1, fixture.polls)
    }

    @Test
    fun `city switch refreshes snapshot before consuming another delta batch`() = runTest {
        val fixture = Fixture()
        val polled = CompletableDeferred<Unit>()
        fixture.poll = {
            if (fixture.polls == 1) {
                fixture.context.value = LoadState.Content(city("00000000-0000-0000-0000-000000000002"))
                1L
            } else {
                polled.complete(Unit)
                awaitCancellation()
            }
        }
        val job = launch { fixture.run() }
        polled.await()
        job.cancelAndJoin()
        assertEquals(2, fixture.snapshots)
    }

    @Test
    fun `expired snapshot enters recovery before first stream poll`() = runTest {
        val fixture = Fixture()
        val polled = CompletableDeferred<Unit>()
        fixture.snapshot = {
            if (fixture.snapshots == 1) throw expired()
            true
        }
        fixture.recover = { fixture.context.value = LoadState.Content(city()) }
        fixture.poll = { polled.complete(Unit); awaitCancellation() }
        val job = launch { fixture.run() }
        polled.await()
        job.cancelAndJoin()
        assertEquals(1, fixture.recoveries)
        assertEquals(2, fixture.snapshots)
        assertEquals(1, fixture.polls)
    }

    @Test
    fun `readback showing no lock after ambiguous resolution waits for user action`() = runTest {
        val fixture = Fixture()
        val readCompleted = CompletableDeferred<Unit>()
        fixture.context.value = LoadState.Failure(SocialError("network_error", "offline"))
        fixture.read = {
            fixture.context.value = LoadState.Empty
            readCompleted.complete(Unit)
        }
        val job = launch { fixture.run() }
        readCompleted.await()
        yield()
        assertEquals(1, fixture.reads)
        assertEquals(0, fixture.recoveries)
        assertEquals(0, fixture.polls)
        job.cancelAndJoin()
    }

    private class Fixture {
        val context = MutableStateFlow<LoadState<CityContextModel>>(LoadState.Content(city()))
        var polls = 0
        var reads = 0
        var recoveries = 0
        var snapshots = 0
        var unauthorized = 0
        var forbidden = 0
        var poll: suspend () -> Long = { error("unexpected poll") }
        var read: suspend () -> Unit = { error("unexpected GET") }
        var recover: suspend () -> Unit = { error("unexpected resolution") }
        var snapshot: suspend () -> Boolean = { true }

        suspend fun run() = runCityRealtimeSession(
            context = context,
            retryMillis = 5,
            readContext = { reads++; read() },
            recoverContext = { recoveries++; recover() },
            refreshSnapshot = { snapshots++; snapshot() },
            poll = { polls++; poll() },
            onUnauthorized = { unauthorized++ },
            onForbidden = { forbidden++ },
        )
    }

    private companion object {
        fun expired() = ApiException(404, "city_context_unavailable", "expired")
        fun city(id: String = "00000000-0000-0000-0000-000000000001") = CityContextModel(
            locality = CityLocality(id, "Kyiv", "UA", "Europe/Kyiv", 50_450_100, 30_523_400),
            permissionClass = CityPermissionClass.PRECISE,
            accuracyM = 25,
            observedAtEpochMillis = 1_000,
            expiresAtEpochMillis = 2_000,
            switchPending = false,
        )
    }
}
