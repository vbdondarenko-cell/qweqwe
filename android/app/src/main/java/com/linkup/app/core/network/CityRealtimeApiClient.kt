package com.linkup.app.core.network

import com.linkup.app.core.session.SecureSessionStore
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URL
import java.time.Instant
import java.util.UUID
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import org.json.JSONException
import org.json.JSONObject

private const val CITY_REALTIME_MAX_BATCH = 200
private const val CITY_REALTIME_GET_ATTEMPTS = 2
private const val CITY_REALTIME_RETRY_DELAY_MS = 250L

class CityRealtimeApiClient(
    baseUrl: String,
    private val sessions: SecureSessionStore,
) : CityRealtimeApi {
    private val root = validatedApiRoot(baseUrl)

    override suspend fun pullCityRealtime(after: Long, limit: Int): CityRealtimeBatchModel = withContext(Dispatchers.IO) {
        require(after >= 0)
        require(limit in 1..CITY_REALTIME_MAX_BATCH)
        var lastError: Exception? = null
        repeat(CITY_REALTIME_GET_ATTEMPTS) { attempt ->
            try {
                return@withContext requestOnce(after, limit)
            } catch (error: CancellationException) {
                throw error
            } catch (error: Exception) {
                lastError = error
                val retryable = when (error) {
                    is ApiException -> error.status in setOf(408, 429, 502, 503, 504)
                    is IOException -> true
                    else -> false
                }
                if (attempt == CITY_REALTIME_GET_ATTEMPTS - 1 || !retryable) throw error
                delay(CITY_REALTIME_RETRY_DELAY_MS)
            }
        }
        throw lastError ?: IOException("city realtime request failed")
    }

    override suspend fun currentCursor(): Long = withContext(Dispatchers.IO) {
        var lastError: Exception? = null
        repeat(CITY_REALTIME_GET_ATTEMPTS) { attempt ->
            try {
                return@withContext requestCursorOnce()
            } catch (error: CancellationException) {
                throw error
            } catch (error: Exception) {
                lastError = error
                val retryable = when (error) {
                    is ApiException -> error.status in setOf(408, 429, 502, 503, 504)
                    is IOException -> true
                    else -> false
                }
                if (attempt == CITY_REALTIME_GET_ATTEMPTS - 1 || !retryable) throw error
                delay(CITY_REALTIME_RETRY_DELAY_MS)
            }
        }
        throw lastError ?: IOException("city realtime cursor request failed")
    }

    private fun requestCursorOnce(): Long {
        val stored = sessions.load() ?: throw ApiException(401, "unauthorized", "authentication required")
        val connection = (URL("$root/v1/realtime/city/cursor").openConnection() as HttpURLConnection).apply {
            requestMethod = "GET"
            connectTimeout = 10_000
            readTimeout = 15_000
            setRequestProperty("Accept", "application/json")
            setRequestProperty("Authorization", "Bearer ${stored.token}")
            instanceFollowRedirects = false
            useCaches = false
        }
        try {
            val status = connection.responseCode
            val requestId = connection.getHeaderField("X-Request-ID")
            val input = if (status in 200..299) connection.inputStream else connection.errorStream
            val text = try {
                input?.use { readUtf8Bounded(it) }.orEmpty()
            } catch (_: ResponseTooLargeException) {
                throw ApiException(status, "response_too_large", "server response exceeded the client safety limit", requestId)
            }
            if (status !in 200..299) {
                if (status == HttpURLConnection.HTTP_UNAUTHORIZED) sessions.clear()
                val problem = runCatching { JSONObject(text) }.getOrNull()
                throw ApiException(
                    status = status,
                    code = problem?.optString("code")?.takeIf { it.isNotBlank() } ?: "request_failed",
                    message = problem?.optString("message")?.takeIf { it.isNotBlank() } ?: "request failed",
                    requestId = problem?.optString("requestId")?.takeIf { it.isNotBlank() } ?: requestId,
                )
            }
            val json = try {
                JSONObject(text)
            } catch (_: JSONException) {
                throw cityRealtimeProtocolError(requestId)
            }
            val cursor = json.optLong("cursor", -1L)
            if (cursor < 0) throw cityRealtimeProtocolError(requestId)
            return cursor
        } finally {
            connection.disconnect()
        }
    }

    private fun requestOnce(after: Long, limit: Int): CityRealtimeBatchModel {
        val stored = sessions.load() ?: throw ApiException(401, "unauthorized", "authentication required")
        val connection = (URL("$root/v1/realtime/city?after=$after&limit=$limit").openConnection() as HttpURLConnection).apply {
            requestMethod = "GET"
            connectTimeout = 10_000
            readTimeout = 15_000
            setRequestProperty("Accept", "application/json")
            setRequestProperty("Authorization", "Bearer ${stored.token}")
            instanceFollowRedirects = false
            useCaches = false
        }
        try {
            val status = connection.responseCode
            val requestId = connection.getHeaderField("X-Request-ID")
            val input = if (status in 200..299) connection.inputStream else connection.errorStream
            val text = try {
                input?.use { readUtf8Bounded(it) }.orEmpty()
            } catch (_: ResponseTooLargeException) {
                throw ApiException(status, "response_too_large", "server response exceeded the client safety limit", requestId)
            }
            if (status !in 200..299) {
                if (status == HttpURLConnection.HTTP_UNAUTHORIZED) sessions.clear()
                val problem = runCatching { JSONObject(text) }.getOrNull()
                throw ApiException(
                    status = status,
                    code = problem?.optString("code")?.takeIf { it.isNotBlank() } ?: "request_failed",
                    message = problem?.optString("message")?.takeIf { it.isNotBlank() } ?: "request failed",
                    requestId = problem?.optString("requestId")?.takeIf { it.isNotBlank() } ?: requestId,
                )
            }
            val json = try {
                JSONObject(text)
            } catch (_: JSONException) {
                throw cityRealtimeProtocolError(requestId)
            }
            return parseCityRealtimeBatch(json, after, limit, requestId)
        } finally {
            connection.disconnect()
        }
    }
}

internal fun parseCityRealtimeBatch(
    json: JSONObject,
    after: Long,
    limit: Int,
    requestId: String? = null,
): CityRealtimeBatchModel {
    val cursor = json.optLong("cursor", Long.MIN_VALUE)
    if (cursor < after) throw cityRealtimeProtocolError(requestId)
    val array = json.optJSONArray("invalidations") ?: throw cityRealtimeProtocolError(requestId)
    if (array.length() > limit) throw cityRealtimeProtocolError(requestId)

    var previous = after
    val ids = HashSet<String>(array.length())
    val invalidations = buildList(array.length()) {
        for (index in 0 until array.length()) {
            val item = array.optJSONObject(index) ?: throw cityRealtimeProtocolError(requestId)
            val sequence = item.optLong("sequence", Long.MIN_VALUE)
            val eventId = item.optString("eventId")
            val canonicalEventId = runCatching { UUID.fromString(eventId).toString() }.getOrNull()
            val occurredAt = runCatching { Instant.parse(item.getString("occurredAt")).toEpochMilli() }.getOrNull()
            if (
                sequence <= previous || sequence > cursor ||
                canonicalEventId == null || canonicalEventId != eventId.lowercase() || !ids.add(canonicalEventId) ||
                occurredAt == null
            ) {
                throw cityRealtimeProtocolError(requestId)
            }
            add(CityRealtimeInvalidationModel(sequence, canonicalEventId, occurredAt))
            previous = sequence
        }
    }
    return CityRealtimeBatchModel(cursor = cursor, invalidations = invalidations)
}

private fun cityRealtimeProtocolError(requestId: String?) = ApiException(
    status = HttpURLConnection.HTTP_OK,
    code = "protocol_error",
    message = "invalid city realtime response",
    requestId = requestId,
)
