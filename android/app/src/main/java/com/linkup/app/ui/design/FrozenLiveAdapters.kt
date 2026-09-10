package com.linkup.app.ui.design

import com.linkup.app.core.network.SlotAccessMode
import com.linkup.app.core.network.SlotModel
import com.linkup.app.core.network.SlotState
import com.linkup.app.core.network.SlotViewerState
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

internal fun SlotModel.toFrozenSlot(): FrozenSlot {
    val initials = organizer.displayName
        .trim()
        .split(Regex("\\s+"))
        .filter { it.isNotBlank() }
        .take(2)
        .joinToString("") { it.take(1).uppercase() }
        .ifBlank { organizer.username.take(2).uppercase() }
    val visualStatus = when {
        state == SlotState.ACTIVE -> LinkUpVisualStatus.LIVE
        state == SlotState.FULL -> LinkUpVisualStatus.FULL
        viewerState == SlotViewerState.PENDING || accessMode != SlotAccessMode.INSTANT -> LinkUpVisualStatus.APPROVAL
        else -> LinkUpVisualStatus.OPEN
    }
    val emoji = when (activity.lowercase()) {
        "coffee" -> "☕"
        "running", "run" -> "🏃"
        "gym", "workout", "fitness" -> "🏋️"
        "food", "dinner", "lunch" -> "🍽️"
        "walk" -> "🚶"
        "music" -> "🎵"
        "games", "gaming" -> "🎮"
        "travel" -> "✈️"
        "photo", "photography" -> "📷"
        else -> "✦"
    }
    val time = startAtEpochMillis?.let {
        DateTimeFormatter.ofPattern("EEE · HH:mm")
            .withZone(ZoneId.systemDefault())
            .format(Instant.ofEpochMilli(it))
    } ?: if (state == SlotState.ACTIVE) "Happening now" else "Open now"
    val tags = buildList {
        add(activity)
        zoneText?.takeIf { it.isNotBlank() }?.let(::add)
        when (viewerState) {
            SlotViewerState.HOST -> add("hosting")
            SlotViewerState.ACCEPTED -> add("joined")
            SlotViewerState.PENDING -> add("pending")
            SlotViewerState.NONE -> Unit
        }
    }.distinct().take(3)
    return FrozenSlot(
        emoji = emoji,
        status = visualStatus,
        title = title,
        location = placeText,
        time = time,
        distance = zoneText.orEmpty(),
        description = details.orEmpty().ifBlank { activity.replaceFirstChar { c -> c.uppercase() } },
        organizer = FrozenOrganizer(organizer.displayName, initials, 0xFFFF2D35),
        joined = acceptedCount,
        capacity = capacity,
        tags = tags,
        approval = accessMode != SlotAccessMode.INSTANT,
    )
}

internal enum class FrozenPrimaryAction {
    REQUEST, JOIN, WAITLIST, PENDING, WAITLISTED, OPEN, MANAGE, FULL, ACTIVE, CLOSED, UNAVAILABLE
}

internal fun SlotModel.primaryActionKind(waitlistEnabled: Boolean = false): FrozenPrimaryAction = when (viewerState) {
    SlotViewerState.PENDING -> if (accessMode == SlotAccessMode.WAITLIST) FrozenPrimaryAction.WAITLISTED else FrozenPrimaryAction.PENDING
    SlotViewerState.ACCEPTED -> FrozenPrimaryAction.OPEN
    SlotViewerState.HOST -> FrozenPrimaryAction.MANAGE
    SlotViewerState.NONE -> when {
        state == SlotState.ACTIVE -> FrozenPrimaryAction.ACTIVE
        state !in setOf(SlotState.PUBLISHED, SlotState.FILLING, SlotState.FULL) -> FrozenPrimaryAction.CLOSED
        accessMode == SlotAccessMode.WAITLIST && !waitlistEnabled -> FrozenPrimaryAction.UNAVAILABLE
        accessMode == SlotAccessMode.WAITLIST && (state == SlotState.FULL || acceptedCount >= capacity) -> FrozenPrimaryAction.WAITLIST
        state == SlotState.FULL || acceptedCount >= capacity -> FrozenPrimaryAction.FULL
        accessMode == SlotAccessMode.INSTANT || accessMode == SlotAccessMode.WAITLIST -> FrozenPrimaryAction.JOIN
        else -> FrozenPrimaryAction.REQUEST
    }
}

internal fun SlotModel.canParticipate(waitlistEnabled: Boolean): Boolean =
    viewerState == SlotViewerState.NONE && when (accessMode) {
        SlotAccessMode.WAITLIST -> waitlistEnabled && state in setOf(SlotState.PUBLISHED, SlotState.FILLING, SlotState.FULL)
        else -> state in setOf(SlotState.PUBLISHED, SlotState.FILLING) && acceptedCount < capacity
    }

internal fun SlotModel.primaryActionEnabled(waitlistEnabled: Boolean = false): Boolean =
    primaryActionKind(waitlistEnabled) in setOf(
        FrozenPrimaryAction.REQUEST,
        FrozenPrimaryAction.JOIN,
        FrozenPrimaryAction.WAITLIST,
        FrozenPrimaryAction.OPEN,
        FrozenPrimaryAction.MANAGE,
    )
