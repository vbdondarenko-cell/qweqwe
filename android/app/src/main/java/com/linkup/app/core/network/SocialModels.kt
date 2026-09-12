package com.linkup.app.core.network

enum class MySlotsView { HOSTING, JOINED, REQUESTED }

enum class SlotState {
    DRAFT,
    PUBLISHED,
    FILLING,
    FULL,
    ACTIVE,
    COMPLETED,
    CANCELLED,
    EXPIRED,
    MODERATED,
}

enum class SlotAccessMode { INSTANT, APPROVAL, WAITLIST }

// PRIVATE and LINKS (README §4.3's first two non-PUBLIC visibility modes,
// backend/IMPLEMENTATION_STATUS.md §52/§55) are discoverability gates, not
// access-control gates: Request/Join work identically regardless of which
// value is set here. valueOf(json.getString("visibility")) in parseSlot
// (LinkUpApiClient.kt, DurableSocialApi.kt) throws for any value not
// listed here, so this enum must stay in sync with slot.Visibility's
// closed set (internal/slot/service.go's validVisibility) or a real
// PRIVATE/LINKS Slot response would crash parsing instead of just failing
// to render one unsupported field.
// A real, live client bug found and fixed here, not just new capability:
// the backend has emitted SELECTED and CITY since earlier work this
// session, and now LASSO/TRAVEL_CORRIDOR too — none were in this enum.
// SlotVisibility.valueOf(json.getString("visibility")) (both call sites:
// LinkUpApiClient.kt and DurableSocialApi.kt) throws for any value not
// listed here, so any build against the real backend would have crashed
// parsing the first such Slot it encountered — found by reading the
// existing code before writing anything, not by running it (no
// SDK/Gradle access in this environment).
enum class SlotVisibility { PUBLIC, PRIVATE, LINKS, SELECTED, CITY, LASSO, TRAVEL_CORRIDOR }
enum class SlotViewerState { NONE, PENDING, ACCEPTED, HOST }

data class SlotOrganizer(
    val id: String,
    val username: String,
    val displayName: String,
    val avatarUrl: String?,
)

data class SlotModel(
    val id: String,
    val organizer: SlotOrganizer,
    val title: String,
    val activity: String,
    val details: String?,
    val placeText: String,
    val zoneText: String?,
    val canonicalPlaceId: String? = null,
    val startAtEpochMillis: Long?,
    val capacity: Int,
    val acceptedCount: Int,
    val state: SlotState,
    val accessMode: SlotAccessMode,
    val visibility: SlotVisibility,
    val viewerState: SlotViewerState,
    val version: Long,
    val createdAtEpochMillis: Long,
    val updatedAtEpochMillis: Long,
)

data class CreateSlotInput(
    val title: String,
    val activity: String,
    val details: String? = null,
    val placeText: String,
    val zoneText: String? = null,
    val canonicalPlaceId: String? = null,
    val startAtEpochMillis: Long? = null,
    val capacity: Int,
    val accessMode: SlotAccessMode = SlotAccessMode.APPROVAL,
    // Nil-safe server-side (defaults to PUBLIC); the legacy /v1/slots
    // endpoint rejects anything else with invalid_slot_state, matching
    // slot.Service.Create's own guard — only /v1/slots/drafts (createDraft)
    // actually accepts PRIVATE/LINKS.
    val visibility: SlotVisibility? = null,
)

data class EditSlotInput(
    val expectedVersion: Long,
    val title: String? = null,
    val details: String? = null,
    val placeText: String? = null,
    val zoneText: String? = null,
    val canonicalPlaceId: String? = null,
    val clearCanonicalPlaceId: Boolean = false,
    val startAtEpochMillis: Long? = null,
    val clearStartAt: Boolean = false,
    val capacity: Int? = null,
    val accessMode: SlotAccessMode? = null,
    // Server only accepts this while the Slot is still DRAFT
    // (invalid_slot_state otherwise) — see backend IMPLEMENTATION_STATUS.md
    // §53.
    val visibility: SlotVisibility? = null,
)

data class PendingSlotRequest(
    val user: SlotOrganizer,
    val requestedAtEpochMillis: Long,
)

data class ChatAuthor(
    val id: String,
    val username: String,
    val displayName: String,
    val avatarUrl: String?,
)

data class ChatMessage(
    val id: String,
    val slotId: String,
    val author: ChatAuthor,
    val text: String,
    val createdAtEpochMillis: Long,
)
