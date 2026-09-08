package com.linkup.app.core.hosting

import com.linkup.app.core.network.CreateSlotInput
import com.linkup.app.core.network.SlotAccessMode
import com.linkup.app.core.network.SlotModel
import com.linkup.app.core.network.SlotOrganizer
import com.linkup.app.core.network.SlotState
import com.linkup.app.core.network.SlotViewerState
import com.linkup.app.core.network.SlotVisibility
import com.linkup.app.core.network.V11HostingApi
import com.linkup.app.core.social.LoadState
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertIs
import kotlin.test.assertTrue
import kotlinx.coroutines.test.runTest

class V11HostingCoordinatorTest {
    @Test
    fun `draft remains non discoverable until explicit publish`() = runTest {
        val api = FakeHostingApi()
        val coordinator = V11HostingCoordinator(api)

        assertTrue(coordinator.createDraft(input()))
        val draft = assertIs<LoadState.Content<SlotModel>>(coordinator.draft.value).value
        assertEquals(SlotState.DRAFT, draft.state)
        assertEquals(0, api.publishCalls)

        val publishedResult = assertIs<SlotModel>(coordinator.publishCurrent())
        assertEquals(SlotState.FILLING, publishedResult.state)
        val published = assertIs<LoadState.Content<SlotModel>>(coordinator.draft.value).value
        assertEquals(SlotState.FILLING, published.state)
        assertEquals(1, api.publishCalls)
        assertEquals(1L, api.lastExpectedVersion)
    }

    @Test
    fun `invalid server state fails closed`() = runTest {
        val coordinator = V11HostingCoordinator(object : V11HostingApi {
            override suspend fun createDraft(input: CreateSlotInput): SlotModel = slot(SlotState.FILLING, 1)
            override suspend fun publishDraft(slotId: String, expectedVersion: Long): SlotModel = error("unused")
        })

        assertFalse(coordinator.createDraft(input()))
        assertIs<LoadState.Failure>(coordinator.draft.value)
    }

    private class FakeHostingApi : V11HostingApi {
        var publishCalls = 0
        var lastExpectedVersion = 0L

        override suspend fun createDraft(input: CreateSlotInput): SlotModel = slot(SlotState.DRAFT, 1)

        override suspend fun publishDraft(slotId: String, expectedVersion: Long): SlotModel {
            publishCalls++
            lastExpectedVersion = expectedVersion
            return slot(SlotState.FILLING, expectedVersion + 1)
        }
    }

    private fun input() = CreateSlotInput(
        title = "Coffee",
        activity = "coffee",
        placeText = "Center",
        capacity = 4,
    )

    companion object {
        private fun slot(state: SlotState, version: Long) = SlotModel(
            id = "00000000-0000-0000-0000-000000000010",
            organizer = SlotOrganizer(
                id = "00000000-0000-0000-0000-000000000011",
                username = "host",
                displayName = "Host",
                avatarUrl = null,
            ),
            title = "Coffee",
            activity = "coffee",
            details = null,
            placeText = "Center",
            zoneText = null,
            canonicalPlaceId = null,
            startAtEpochMillis = null,
            capacity = 4,
            acceptedCount = 0,
            state = state,
            accessMode = SlotAccessMode.APPROVAL,
            visibility = SlotVisibility.PUBLIC,
            viewerState = SlotViewerState.HOST,
            version = version,
            createdAtEpochMillis = 1_000L,
            updatedAtEpochMillis = 1_000L,
        )
    }
}
