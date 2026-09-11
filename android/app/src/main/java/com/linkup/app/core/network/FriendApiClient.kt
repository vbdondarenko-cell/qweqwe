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

// Client for backend/internal/httpserver/friend_handlers.go (backend
// IMPLEMENTATION_STATUS.md §55). Gated server-side by capability "friends"
// (fail-closed, off by default like every other v1.1 capability) — calls
// made while the capability is disabled fail the same way any other
// disabled-capability call does (see CapabilityApi/requireCapability),
// not specially handled here.
//
// NOTE: written against the real, tested backend contract in this same
// commit, but this Android module could not be compiled or run in the
// environment this code was written in (no Android SDK/Gradle network
// access) — see IMPLEMENTATION_STATUS.md's explicit statement of that
// limitation. Treat this as best-effort, unverified client code until it
// has actually been built and exercised.
class FriendApiClient(
    baseUrl: String,
    private val sessions: SecureSessionStore,
) {
    private val root = validatedApiRoot(baseUrl)

    suspend fun sendRequest(targetUserId: String): FriendRequestOutcome {
        val json = request("POST", "/v1/me/friends/requests/${uuid(targetUserId)}", null)
            ?: throw ApiException(502, "protocol_error", "Expected a response body")
        return FriendRequestOutcome.valueOf(json.getString("outcome"))
    }

    suspend fun cancelRequest(targetUserId: String) {
        request("DELETE", "/v1/me/friends/requests/${uuid(targetUserId)}", null)
    }

    suspend fun acceptRequest(requesterUserId: String) {
        request("POST", "/v1/me/friends/requests/${uuid(requesterUserId)}/accept", null)
    }

    suspend fun rejectRequest(requesterUserId: String) {
        request("POST", "/v1/me/friends/requests/${uuid(requesterUserId)}/reject", null)
    }

    suspend fun removeFriend(friendUserId: String) {
        request("DELETE", "/v1/me/friends/${uuid(friendUserId)}", null)
    }

    suspend fun listFriends(): List<FriendModel> {
        val items = request("GET", "/v1/me/friends", null)?.optJSONArray("items") ?: return emptyList()
        return buildList {
            for (index in 0 until items.length()) {
                val item = items.getJSONObject(index)
                add(FriendModel(
                    user = parseUserSummary(item.getJSONObject("user")),
                    sinceEpochMillis = Instant.parse(item.getString("since")).toEpochMilli(),
                ))
            }
        }
    }

    suspend fun listIncomingRequests(): List<FriendPendingRequest> =
        listPendingRequests("/v1/me/friends/requests/incoming")

    suspend fun listOutgoingRequests(): List<FriendPendingRequest> =
        listPendingRequests("/v1/me/friends/requests/outgoing")

    private suspend fun listPendingRequests(path: String): List<FriendPendingRequest> {
        val items = request("GET", path, null)?.optJSONArray("items") ?: return emptyList()
        return buildList {
            for (index in 0 until items.length()) {
                val item = items.getJSONObject(index)
                add(FriendPendingRequest(
                    requestId = item.getString("requestId"),
                    user = parseUserSummary(item.getJSONObject("user")),
                    createdAtEpochMillis = Instant.parse(item.getString("createdAt")).toEpochMilli(),
                ))
            }
        }
    }

    private fun parseUserSummary(json: JSONObject): FriendUserSummary = FriendUserSummary(
        id = json.getString("id"),
        username = json.getString("username"),
        displayName = json.getString("displayName"),
        avatarUrl = if (json.has("avatarUrl") && !json.isNull("avatarUrl")) json.getString("avatarUrl") else null,
    )

    private fun uuid(raw: String): String = UUID.fromString(raw).toString()

    private suspend fun request(method: String, path: String, body: JSONObject?): JSONObject? =
        withContext(Dispatchers.IO) {
            val maxAttempts = if (method == "GET") 2 else 1
            var lastError: Exception? = null
            for (attempt in 1..maxAttempts) {
                try {
                    return@withContext requestOnce(method, path, body)
                } catch (error: CancellationException) {
                    throw error
                } catch (error: Exception) {
                    lastError = error
                    val retryable = when (error) {
                        is ApiException -> error.status in setOf(408, 429, 502, 503, 504)
                        is IOException -> true
                        else -> false
                    }
                    if (attempt >= maxAttempts || !retryable) throw error
                    delay(250L)
                }
            }
            throw lastError ?: IOException("request failed")
        }

    private fun requestOnce(method: String, path: String, body: JSONObject?): JSONObject? {
        val stored = sessions.load() ?: throw ApiException(401, "unauthorized", "authentication required")
        val connection = (URL("$root$path").openConnection() as HttpURLConnection).apply {
            requestMethod = method
            connectTimeout = 10_000
            readTimeout = 15_000
            useCaches = false
            instanceFollowRedirects = false
            setRequestProperty("Accept", "application/json")
            setRequestProperty("Authorization", "Bearer ${stored.token}")
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
            if (status == HttpURLConnection.HTTP_NO_CONTENT) return null
            val requestId = connection.getHeaderField("X-Request-ID")
            val stream = if (status in 200..299) connection.inputStream else connection.errorStream
            val text = stream?.use { readUtf8Bounded(it) }.orEmpty()
            val json = try { JSONObject(text) } catch (_: JSONException) { null }
            if (status !in 200..299) {
                if (status == HttpURLConnection.HTTP_UNAUTHORIZED) sessions.clear()
                throw ApiException(
                    status = status,
                    code = json?.optString("code")?.takeIf { it.isNotBlank() } ?: "request_failed",
                    message = json?.optString("message")?.takeIf { it.isNotBlank() } ?: "request failed",
                    requestId = json?.optString("requestId")?.takeIf { it.isNotBlank() } ?: requestId,
                )
            }
            return json
        } finally {
            connection.disconnect()
        }
    }
}
