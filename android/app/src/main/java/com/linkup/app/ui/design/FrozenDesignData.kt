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

val FrozenPulseSlots = listOf(
    FrozenSlot(
        emoji = "☕", status = LinkUpVisualStatus.LIVE,
        title = "Morning Espresso Run", location = "Green Hills Coffee, Podil",
        time = "Happening now", distance = "0.4km",
        description = "Quick coffee before work. Trying a new roastery — join for a flat white and a chat.",
        organizer = FrozenOrganizer("Andriy Melnyk", "AM", 0xFFFF2D35, 94),
        joined = 4, capacity = 6, tags = listOf("coffee", "morning", "chill"), approval = false,
    ),
    FrozenSlot(
        emoji = "🏃", status = LinkUpVisualStatus.OPEN,
        title = "Evening Riverside Run", location = "Poshtova Square",
        time = "Today · 18:30", distance = "1.2km",
        description = "5K along the Dnipro promenade. All paces welcome, we regroup at bridges.",
        organizer = FrozenOrganizer("Maksym Dmytrenko", "MD", 0xFF22C55E, 91),
        joined = 7, capacity = 12, tags = listOf("running", "5k", "riverside"), approval = false,
    ),
    FrozenSlot(
        emoji = "🍽️", status = LinkUpVisualStatus.APPROVAL,
        title = "Khachapuri & Wine Night", location = "Mimino Restaurant",
        time = "Today · 19:00", distance = "0.8km",
        description = "Georgian dinner at a hidden gem in Podil. 6 people, split the bill. Request to join.",
        organizer = FrozenOrganizer("Sofia Hrytsenko", "SH", 0xFFF59E0B, 97),
        joined = 4, capacity = 6, tags = listOf("food", "georgian", "wine"), approval = true,
    ),
    FrozenSlot(
        emoji = "📷", status = LinkUpVisualStatus.OPEN,
        title = "Golden Hour Photo Walk", location = "Andriyivskyy Descent",
        time = "Today · 17:45", distance = "1.5km",
        description = "Capture the old town at sunset. Bring any camera, even a phone. We share tips after.",
        organizer = FrozenOrganizer("Olena Kovalenko", "OK", 0xFF3B82F6, 88),
        joined = 3, capacity = 8, tags = listOf("photography", "sunset", "walk"), approval = false,
    ),
)


data class FrozenMapPin(
    val x: Float,
    val y: Float,
    val slot: FrozenSlot,
    val color: Long,
)

val FrozenMapPins = listOf(
    FrozenMapPin(.32f, .40f, FrozenPulseSlots[0], 0xFFF59E0B),
    FrozenMapPin(.55f, .28f, FrozenPulseSlots[1], 0xFF22C55E),
    FrozenMapPin(.28f, .62f, FrozenPulseSlots[2], 0xFFEF4444),
    FrozenMapPin(.68f, .50f, FrozenPulseSlots[3], 0xFF3B82F6),
    FrozenMapPin(.48f, .69f, FrozenPulseSlots[0].copy(emoji = "♟️", title = "Park Chess Blitz", joined = 5, capacity = 8), 0xFFA5A9B0),
    FrozenMapPin(.72f, .35f, FrozenPulseSlots[2].copy(emoji = "🍸", title = "Cocktail Crawl · 3 Bars", joined = 6, capacity = 8), 0xFFFF2D35),
)

data class FrozenFlashDrop(
    val emoji: String,
    val title: String,
    val location: String,
    val distance: String,
    val time: String,
    val joined: Int,
    val capacity: Int,
    val tags: List<String>,
    val urgent: Boolean = false,
)

val FrozenFlashDrops = listOf(
    FrozenFlashDrop("☕", "Flash Coffee · Free Round", "Green Hills, Podil", "0.3km", "14:02", 3, 5, listOf("coffee", "flash", "free")),
    FrozenFlashDrop("🛹", "Pop-up Skate Session", "Kontraktova Square", "0.6km", "08:17", 5, 8, listOf("skate", "outdoor", "quick")),
    FrozenFlashDrop("🍕", "Pizza Slice Sprint", "Podil Food Hall", "0.8km", "03:42", 4, 6, listOf("food", "flash", "social"), urgent = true),
)
