package com.linkup.app.core.city

import com.linkup.app.core.network.CanonicalPlace
import com.linkup.app.core.network.MapViewportQuery

private const val MIN_LAT_E6 = -90_000_000
private const val MAX_LAT_E6 = 90_000_000
private const val MIN_LON_E6 = -180_000_000
private const val MAX_LON_E6 = 180_000_000
private const val FULL_LON_E6 = 360_000_000L

fun mapViewportAround(
    center: CanonicalPlace,
    zoom: Int,
    fromEpochMillis: Long,
    toEpochMillis: Long,
    limit: Int = 100,
): MapViewportQuery {
    require(zoom in 1..20)
    require(limit in 1..200)
    require(fromEpochMillis < toEpochMillis)

    val halfSpanE6 = when {
        zoom <= 7 -> 10_000_000L
        zoom <= 10 -> 2_000_000L
        zoom <= 13 -> 500_000L
        zoom <= 15 -> 150_000L
        else -> 50_000L
    }

    val south = (center.latitudeE6.toLong() - halfSpanE6)
        .coerceAtLeast(MIN_LAT_E6.toLong())
        .toInt()
    val north = (center.latitudeE6.toLong() + halfSpanE6)
        .coerceAtMost(MAX_LAT_E6.toLong())
        .toInt()
    require(south < north)

    val west = normalizeLongitudeE6(center.longitudeE6.toLong() - halfSpanE6)
    val east = normalizeLongitudeE6(center.longitudeE6.toLong() + halfSpanE6)
    require(west != east)

    return MapViewportQuery(
        westE6 = west,
        southE6 = south,
        eastE6 = east,
        northE6 = north,
        zoom = zoom,
        fromEpochMillis = fromEpochMillis,
        toEpochMillis = toEpochMillis,
        limit = limit,
    )
}

internal fun longitudeFraction(longitudeE6: Int, westE6: Int, eastE6: Int): Float {
    val west = westE6.toLong()
    val east = eastE6.toLong()
    val point = longitudeE6.toLong()
    val span = if (west < east) east - west else FULL_LON_E6 - west + east
    require(span > 0)
    val distance = if (west < east) {
        point - west
    } else if (point >= west) {
        point - west
    } else {
        FULL_LON_E6 - west + point
    }
    return (distance.toDouble() / span.toDouble()).toFloat().coerceIn(0f, 1f)
}

internal fun latitudeFraction(latitudeE6: Int, southE6: Int, northE6: Int): Float {
    val span = northE6.toLong() - southE6.toLong()
    require(span > 0)
    return ((latitudeE6.toLong() - southE6.toLong()).toDouble() / span.toDouble())
        .toFloat()
        .coerceIn(0f, 1f)
}

private fun normalizeLongitudeE6(value: Long): Int {
    var normalized = value
    while (normalized < MIN_LON_E6) normalized += FULL_LON_E6
    while (normalized > MAX_LON_E6) normalized -= FULL_LON_E6
    return normalized.toInt()
}
