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

private const val REALTIME_MAX_BATCH = 200
private const val REALTIME_GET_ATTEMPTS = 2
private const val REALTIME_RETRY_DELAY_MS = 250L

class RealtimeApiClient(
    baseUrl: String,
    private val sessions: SecureSessionStore,
) : RealtimeApi {
    private val root = validatedApiRoot(baseUrl)

    override suspend fun pullRealtime(after: Long, limit: Int): RealtimeBatchModel = withContext(Dispatchers.IO) {
        require(after >= 0)
        require(limit in 1..REALTIME_MAX_BATCH)
        var lastError: Exception? = null
        repeat(REALTIME_GET_ATTEMPTS) { attempt ->
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
                if (attempt == REALTIME_GET_ATTEMPTS - 1 || !retryable) throw error
                delay(REALTIME_RETRY_DELAY_MS)
            }
        }
        throw lastError ?: IOException("realtime request failed")
    }

    override suspend fun currentCursor(): Long = withContext(Dispatchers.IO) {
        var lastError: Exception? = null
        repeat(REALTIME_GET_ATTEMPTS) { attempt ->
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
                if (attempt == REALTIME_GET_ATTEMPTS - 1 || !retryable) throw error
                delay(REALTIME_RETRY_DELAY_MS)
            }
        }
        throw lastError ?: IOException("realtime cursor request failed")
    }

    private fun requestCursorOnce(): Long {
        val stored = sessions.load() ?: throw ApiException(401, "unauthorized", "authentication required")
        val connection = (URL("$root/v1/realtime/cursor").openConnection() as HttpURLConnection).apply {
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
                throw ApiException(status, "protocol_error", "invalid realtime cursor response", requestId)
            }
            val cursor = json.optLong("cursor", -1L)
            if (cursor < 0) {
                throw ApiException(status, "protocol_error", "invalid realtime cursor response", requestId)
            }
            return cursor
        } finally {
            connection.disconnect()
        }
    }

    private fun requestOnce(after: Long, limit: Int): RealtimeBatchModel {
        val stored = sessions.load() ?: throw ApiException(401, "unauthorized", "authentication required")
        val connection = (URL("$root/v1/realtime/events?after=$after&limit=$limit").openConnection() as HttpURLConnection).apply {
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
                throw ApiException(status, "protocol_error", "invalid realtime response", requestId)
            }
            return parseBatch(json, after, limit, requestId)
        } finally {
            connection.disconnect()
        }
    }

    private fun parseBatch(json: JSONObject, after: Long, limit: Int, requestId: String?): RealtimeBatchModel {
        val cursor = json.optLong("cursor", Long.MIN_VALUE)
        if (cursor < after) throw protocolError(requestId)
        val array = json.optJSONArray("events") ?: throw protocolError(requestId)
        if (array.length() > limit) throw protocolError(requestId)

        var previous = after
        val eventIds = HashSet<String>(array.length())
        val events = buildList(array.length()) {
            for (index in 0 until array.length()) {
                val item = array.optJSONObject(index) ?: throw protocolError(requestId)
                val sequence = item.optLong("sequence", Long.MIN_VALUE)
                val eventId = item.optString("eventId")
                val aggregateId = item.optString("aggregateId")
                if (sequence <= previous || sequence > cursor || eventId.isBlank() || !eventIds.add(eventId)) {
                    throw protocolError(requestId)
                }
                if (!validUuid(aggregateId)) throw protocolError(requestId)
                val eventType = item.optString("eventType")
                val aggregateType = item.optString("aggregateType")
                val occurredAt = runCatching { Instant.parse(item.getString("occurredAt")).toEpochMilli() }.getOrNull()
                    ?: throw protocolError(requestId)
                if (eventType.isBlank() || aggregateType.isBlank()) throw protocolError(requestId)
                val payload = item.optJSONObject("payload")?.toString() ?: throw protocolError(requestId)
                add(
                    RealtimeEventModel(
                        sequence = sequence,
                        eventId = eventId,
                        eventType = eventType,
                        aggregateType = aggregateType,
                        aggregateId = aggregateId,
                        subjectUserId = optionalUuid(item, "subjectUserId", requestId),
                        slotId = optionalUuid(item, "slotId", requestId),
                        payloadJson = payload,
                        occurredAtEpochMillis = occurredAt,
                    ),
                )
                previous = sequence
            }
        }
        return RealtimeBatchModel(cursor = cursor, events = events)
    }

    private fun optionalUuid(json: JSONObject, key: String, requestId: String?): String? {
        if (!json.has(key) || json.isNull(key)) return null
        val value = json.optString(key)
        if (!validUuid(value)) throw protocolError(requestId)
        return UUID.fromString(value).toString()
    }

    private fun validUuid(value: String): Boolean = runCatching { UUID.fromString(value).toString() == value.lowercase() }.getOrDefault(false)

    private fun protocolError(requestId: String?) = ApiException(
        status = HttpURLConnection.HTTP_OK,
        code = "protocol_error",
        message = "invalid realtime response",
        requestId = requestId,
    )
}
