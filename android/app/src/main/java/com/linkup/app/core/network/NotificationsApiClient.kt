package com.linkup.app.core.network

import com.linkup.app.core.session.SecureSessionStore
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URL
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import org.json.JSONObject

/** One row of GET /v1/me/notifications -- see backend internal/notification.Delivery. */
data class NotificationDeliveryModel(
    val id: String,
    val type: String,
    val title: String,
    val body: String,
    val deepLink: String,
    val slotId: String?,
    val createdAtEpochMillis: Long,
    val read: Boolean,
)

/** GET /v1/me/notifications's parsed response -- see backend internal/notification.InboxSnapshot. */
data class NotificationInboxModel(
    val items: List<NotificationDeliveryModel>,
    val unreadCount: Int,
    val nextCursorEpochMillis: Long?,
)

class NotificationsApiClient(
    baseUrl: String,
    private val sessions: SecureSessionStore,
) {
    private val root = validatedApiRoot(baseUrl)

    suspend fun list(limit: Int? = null, beforeEpochMillis: Long? = null): NotificationInboxModel =
        withContext(Dispatchers.IO) {
            val query = buildString {
                var first = true
                fun sep(): String { val s = if (first) "?" else "&"; first = false; return s }
                if (limit != null) append(sep()).append("limit=").append(limit)
                if (beforeEpochMillis != null) append(sep()).append("before=").append(beforeEpochMillis)
            }
            withRetry { requestOnce("GET", "/v1/me/notifications$query", null, parseInbox = true)!! }
        }

    suspend fun markAllRead(): Unit = withContext(Dispatchers.IO) {
        withRetry { requestOnce("POST", "/v1/me/notifications/read", null, parseInbox = false); Unit }
    }

    private suspend fun <T> withRetry(block: () -> T): T {
        var lastError: Exception? = null
        repeat(2) { attempt ->
            try {
                return block()
            } catch (error: CancellationException) {
                throw error
            } catch (error: Exception) {
                lastError = error
                val retryable = when (error) {
                    is ApiException -> error.status in setOf(408, 429, 502, 503, 504)
                    is IOException -> true
                    else -> false
                }
                if (attempt == 1 || !retryable) throw error
                delay(250L)
            }
        }
        throw lastError ?: IOException("request failed")
    }

    private fun requestOnce(method: String, path: String, body: JSONObject?, parseInbox: Boolean): NotificationInboxModel? {
        val stored = sessions.load() ?: throw ApiException(401, "unauthorized", "authentication required")
        val connection = (URL("$root$path").openConnection() as HttpURLConnection).apply {
            requestMethod = method
            connectTimeout = 10_000
            readTimeout = 15_000
            setRequestProperty("Accept", "application/json")
            setRequestProperty("Authorization", "Bearer ${stored.token}")
            useCaches = false
            if (body != null) {
                doOutput = true
                setRequestProperty("Content-Type", "application/json; charset=utf-8")
            }
        }
        try {
            if (body != null) {
                connection.outputStream.use { it.write(body.toString().toByteArray(Charsets.UTF_8)) }
            }
            val status = connection.responseCode
            val requestId = connection.getHeaderField("X-Request-ID")
            val input = if (status in 200..299) connection.inputStream else connection.errorStream
            val text = try {
                input?.use { readUtf8Bounded(it) }.orEmpty()
            } catch (_: ResponseTooLargeException) {
                throw ApiException(status, "response_too_large", "server response exceeded the client safety limit", requestId)
            }
            if (status !in 200..299) {
                if (status == 401) sessions.clear()
                val problem = runCatching { JSONObject(text) }.getOrNull()
                throw ApiException(
                    status = status,
                    code = problem?.optString("code")?.takeIf { it.isNotBlank() } ?: "request_failed",
                    message = problem?.optString("message")?.takeIf { it.isNotBlank() } ?: "request failed",
                    requestId = problem?.optString("requestId")?.takeIf { it.isNotBlank() } ?: requestId,
                )
            }
            if (!parseInbox) return null
            return parseInboxJson(JSONObject(text))
        } finally {
            connection.disconnect()
        }
    }

    private fun parseInboxJson(json: JSONObject): NotificationInboxModel {
        val itemsJson = json.optJSONArray("items")
        val items = buildList {
            if (itemsJson != null) {
                for (index in 0 until itemsJson.length()) {
                    val item = itemsJson.getJSONObject(index)
                    add(
                        NotificationDeliveryModel(
                            id = item.getString("id"),
                            type = item.getString("type"),
                            title = item.getString("title"),
                            body = item.getString("body"),
                            deepLink = item.getString("deepLink"),
                            slotId = item.optString("slotId").takeIf { it.isNotBlank() },
                            createdAtEpochMillis = item.getLong("createdAtEpochMillis"),
                            read = item.optBoolean("read", false),
                        ),
                    )
                }
            }
        }
        return NotificationInboxModel(
            items = items,
            unreadCount = json.optInt("unreadCount", 0),
            nextCursorEpochMillis = json.optLong("nextCursorEpochMillis", 0L).takeIf { it != 0L },
        )
    }
}
