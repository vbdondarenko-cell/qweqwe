package com.linkup.app.ui.design

import com.linkup.app.core.network.SlotAccessMode
import com.linkup.app.core.network.SlotModel
import com.linkup.app.core.network.SlotOrganizer
import com.linkup.app.core.network.SlotState
import com.linkup.app.core.network.SlotViewerState
import com.linkup.app.core.network.SlotVisibility
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class FrozenLiveAdaptersTest {
    @Test
    fun strangerCanRequestOnlyDiscoverableNonFullSlot() {
        val open = slot(state = SlotState.FILLING, viewer = SlotViewerState.NONE, accepted = 1, capacity = 4)
        assertEquals(FrozenPrimaryAction.REQUEST, open.primaryActionKind())
        assertTrue(open.primaryActionEnabled())
        assertTrue(open.canJoin())

        val full = open.copy(state = SlotState.FULL, acceptedCount = 4)
        assertEquals(FrozenPrimaryAction.FULL, full.primaryActionKind())
        assertFalse(full.primaryActionEnabled())
        assertFalse(full.canJoin())
    }

    @Test
    fun activeAndTerminalSlotsNeverOfferStrangerJoin() {
        val active = slot(state = SlotState.ACTIVE, viewer = SlotViewerState.NONE)
        assertEquals(FrozenPrimaryAction.ACTIVE, active.primaryActionKind())
        assertFalse(active.primaryActionEnabled())

        val completed = active.copy(state = SlotState.COMPLETED)
        assertEquals(FrozenPrimaryAction.CLOSED, completed.primaryActionKind())
        assertFalse(completed.primaryActionEnabled())
        assertFalse(completed.canJoin())
    }
    @Test
    fun existingRelationshipGetsOpenOrManageAction() {
        val accepted = slot(state = SlotState.FILLING, viewer = SlotViewerState.ACCEPTED)
        assertEquals(FrozenPrimaryAction.OPEN, accepted.primaryActionKind())
        assertTrue(accepted.primaryActionEnabled())

        val host = accepted.copy(viewerState = SlotViewerState.HOST)
        assertEquals(FrozenPrimaryAction.MANAGE, host.primaryActionKind())
        assertTrue(host.primaryActionEnabled())
    }

    private fun slot(
        state: SlotState,
        viewer: SlotViewerState,
        accepted: Int = 0,
        capacity: Int = 4,
    ) = SlotModel(
        id = "00000000-0000-0000-0000-000000000001",
        organizer = SlotOrganizer(
            id = "00000000-0000-0000-0000-000000000002",
            username = "host",
            displayName = "Host",
            avatarUrl = null,
        ),
        title = "Coffee",
        activity = "coffee",
        details = null,
        placeText = "Podil",
        zoneText = null,
        startAtEpochMillis = null,
        capacity = capacity,
        acceptedCount = accepted,
        state = state,
        accessMode = SlotAccessMode.APPROVAL,
        visibility = SlotVisibility.PUBLIC,
        viewerState = viewer,
        version = 1,
        createdAtEpochMillis = 1,
        updatedAtEpochMillis = 1,
    )
}
