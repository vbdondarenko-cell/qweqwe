package com.linkup.app.core.network

// Mirrors backend/internal/friend.Outcome (backend IMPLEMENTATION_STATUS.md
// §55). REQUESTED = a new pending request was created; ACCEPTED = the
// target already had a pending request out to the caller, so this call
// resolved as an immediate mutual match instead; ALREADY_PENDING = a
// repeat call in the same direction, a harmless no-op.
enum class FriendRequestOutcome { REQUESTED, ACCEPTED, ALREADY_PENDING }

data class FriendUserSummary(
    val id: String,
    val username: String,
    val displayName: String,
    val avatarUrl: String?,
)

data class FriendPendingRequest(
    val requestId: String,
    val user: FriendUserSummary,
    val createdAtEpochMillis: Long,
)

data class FriendModel(
    val user: FriendUserSummary,
    val sinceEpochMillis: Long,
)
