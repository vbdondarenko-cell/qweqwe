package com.linkup.app.core.scheduling

import java.time.Instant
import java.time.LocalDateTime
import java.time.ZoneId
import java.time.format.DateTimeFormatter

/** Zero choices means a DST gap; two choices require the user to choose an offset. */
fun scheduleInstants(local: LocalDateTime, zone: ZoneId): List<Long> =
    zone.rules.getValidOffsets(local).map { local.toInstant(it).toEpochMilli() }.sorted()

fun scheduleLabel(epochMillis: Long?, zone: ZoneId = ZoneId.systemDefault()): String =
    if (epochMillis == null) "Now"
    else DateTimeFormatter.ofPattern("dd MMM yyyy · HH:mm XXX")
        .withZone(zone).format(Instant.ofEpochMilli(epochMillis)) + " · " + zone.id
