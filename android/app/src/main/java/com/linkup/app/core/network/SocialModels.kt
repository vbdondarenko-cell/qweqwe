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
enum class SlotVisibility { PUBLIC }
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
