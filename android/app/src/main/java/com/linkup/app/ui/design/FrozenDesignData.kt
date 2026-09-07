package com.linkup.app.ui.design

data class FrozenOrganizer(
    val name: String,
    val initials: String,
    val color: Long,
    val reliability: Int,
)

data class FrozenSlot(
    val emoji: String,
    val status: LinkUpVisualStatus,
    val title: String,
    val location: String,
    val time: String,
    val distance: String,
    val description: String,
    val organizer: FrozenOrganizer,
    val joined: Int,
    val capacity: Int,
    val tags: List<String>,
    val approval: Boolean,
)
