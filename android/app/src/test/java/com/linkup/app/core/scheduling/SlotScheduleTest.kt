package com.linkup.app.core.scheduling

import java.time.Instant
import java.time.LocalDateTime
import java.time.ZoneId
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class SlotScheduleTest {
    @Test
    fun convertsLocalDateToExactUtcInstant() {
        val result = scheduleInstants(LocalDateTime.parse("2026-09-07T12:30"), ZoneId.of("Asia/Kolkata"))
        assertEquals(listOf(Instant.parse("2026-09-07T07:00:00Z").toEpochMilli()), result)
    }

    @Test
    fun springGapDoesNotSilentlyMoveMeeting() {
        assertTrue(scheduleInstants(LocalDateTime.parse("2026-03-29T02:30"), ZoneId.of("Europe/Berlin")).isEmpty())
    }

    @Test
    fun autumnOverlapExposesBothInstants() {
        val result = scheduleInstants(LocalDateTime.parse("2026-10-25T02:30"), ZoneId.of("Europe/Berlin"))
        assertEquals(listOf("2026-10-25T00:30:00Z", "2026-10-25T01:30:00Z").map { Instant.parse(it).toEpochMilli() }, result)
    }
}
