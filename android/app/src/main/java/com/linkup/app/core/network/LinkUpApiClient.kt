package com.linkup.app.core.network

import com.linkup.app.core.session.SecureSessionStore
import java.io.ByteArrayOutputStream
import java.io.IOException
import java.io.InputStream
import java.net.HttpURLConnection
import java.net.URL
import java.time.Instant
import java.util.UUID
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONException
import org.json.JSONObject

internal const val MAX_API_RESPONSE_BYTES = 1024 * 1024

internal class ResponseTooLargeException : IOException("API response exceeds client limit")

internal fun readUtf8Bounded(input: InputStream, maxBytes: Int = MAX_API_RESPONSE_BYTES): String {
    require(maxBytes > 0)
    val output = ByteArrayOutputStream(minOf(maxBytes, 8192))
    val buffer = ByteArray(8192)
    var total = 0
    while (true) {
        val read = input.read(buffer)
        if (read < 0) break
        total += read
        if (total > maxBytes) throw ResponseTooLargeException()
        output.write(buffer, 0, read)
    }
    return output.toString(Charsets.UTF_8.name())
}

class LinkUpApiClient(
    baseUrl: String,
    private val sessions: SecureSessionStore,
) : SocialApi {
    private val root = validatedApiRoot(baseUrl)

    @Volatile
    private var unauthorizedHandler: (() -> Unit)? = null

    // v1.0 keeps only in-process ambiguous chat sends. Durable process-death
    // mutation replay belongs to v1.1. The same slot+text retry reuses the key
    // until the server acknowledges the message or returns a definitive 4xx.
    private val pendingChatKeys = mutableMapOf<String, String>()

    fun setUnauthorizedHandler(handler: (() -> Unit)?) {
        unauthorizedHandler = handler
    }

    suspend fun register(
        email: String,
        username: String,
        displayName: String,
        password: String,
        language: String,
        deviceLabel: String,
    ): AuthSession {
        val body = JSONObject()
            .put("email", email)
            .put("username", username)
            .put("displayName", displayName)
            .put("password", password)
            .put("language", language)
            .put("deviceLabel", deviceLabel)
        return persistAuth(request("POST", "/v1/auth/register", body, false)!!)
    }

    suspend fun login(identifier: String, password: String, deviceLabel: String): AuthSession {
        val body = JSONObject()
            .put("identifier", identifier)
            .put("password", password)
            .put("deviceLabel", deviceLabel)
        return persistAuth(request("POST", "/v1/auth/login", body, false)!!)
    }

    suspend fun requestPasswordRecovery(email: String) {
        request("POST", "/v1/auth/recovery/request", JSONObject().put("email", email), false)
    }

    suspend fun resetPassword(token: String, newPassword: String) {
        request("POST", "/v1/auth/recovery/reset", JSONObject().put("token", token).put("newPassword", newPassword), false)
    }

    suspend fun logout() {
        try { request("POST", "/v1/auth/logout", null, true) } finally {
            synchronized(pendingChatKeys) { pendingChatKeys.clear() }
            sessions.clear()
        }
    }

    suspend fun me(): UserProfile = parseUser(request("GET", "/v1/me", null, true)!!)

    suspend fun updateMe(
        displayName: String? = null,
        avatarUrl: String? = null,
        profileVisibility: String? = null,
        language: String? = null,
    ): UserProfile {
        val body = JSONObject()
        displayName?.let { body.put("displayName", it) }
        avatarUrl?.let { body.put("avatarUrl", it) }
        profileVisibility?.let { body.put("profileVisibility", it) }
        language?.let { body.put("language", it) }
        return parseUser(request("PATCH", "/v1/me", body, true)!!)
    }

    suspend fun blockedUsers(): List<BlockedUser> {
        val items = request("GET", "/v1/me/blocks", null, true)!!.getJSONArray("items")
        return buildList(items.length()) {
            for (index in 0 until items.length()) {
                val item = items.getJSONObject(index)
                add(BlockedUser(
                    id = item.getString("id"),
                    username = item.getString("username"),
                    displayName = item.getString("displayName"),
                    avatarUrl = nullableString(item, "avatarUrl"),
                ))
            }
        }
    }

    suspend fun blockUser(userId: String) {
        request("PUT", "/v1/me/blocks/${uuid(userId)}", null, true)
    }

    suspend fun unblockUser(userId: String) {
        request("DELETE", "/v1/me/blocks/${uuid(userId)}", null, true)
    }

    override suspend fun mySlots(view: MySlotsView): List<SlotModel> {
        val items = request("GET", "/v1/me/slots?view=${view.name}", null, true)!!.getJSONArray("items")
        return buildList(items.length()) { for (index in 0 until items.length()) add(parseSlot(items.getJSONObject(index))) }
    }

    override suspend fun pulse(): List<SlotModel> {
        val items = request("GET", "/v1/pulse", null, true)!!.getJSONArray("items")
        return buildList(items.length()) { for (index in 0 until items.length()) add(parseSlot(items.getJSONObject(index))) }
    }

    override suspend fun createSlot(input: CreateSlotInput): SlotModel {
        val body = JSONObject()
            .put("title", input.title)
            .put("activity", input.activity)
            .put("placeText", input.placeText)
            .put("capacity", input.capacity)
        input.details?.let { body.put("details", it) }
        input.zoneText?.let { body.put("zoneText", it) }
        input.startAtEpochMillis?.let { body.put("startAt", Instant.ofEpochMilli(it).toString()) }
        return parseSlot(request("POST", "/v1/slots", body, true, mutationHeaders())!!)
    }

    override suspend fun getSlot(slotId: String): SlotModel =
        parseSlot(request("GET", "/v1/slots/${uuid(slotId)}", null, true)!!)

    override suspend fun editSlot(slotId: String, input: EditSlotInput): SlotModel {
        require(input.expectedVersion > 0)
        val body = JSONObject().put("expectedVersion", input.expectedVersion)
        input.title?.let { body.put("title", it) }
        input.details?.let { body.put("details", it) }
        input.placeText?.let { body.put("placeText", it) }
        input.zoneText?.let { body.put("zoneText", it) }
        input.startAtEpochMillis?.let { body.put("startAt", Instant.ofEpochMilli(it).toString()) }
        if (input.clearStartAt) body.put("clearStartAt", true)
        input.capacity?.let { body.put("capacity", it) }
        return parseSlot(request("PATCH", "/v1/slots/${uuid(slotId)}", body, true, mutationHeaders())!!)
    }

    override suspend fun cancelSlot(slotId: String, expectedVersion: Long): SlotModel {
        require(expectedVersion > 0)
        return parseSlot(request(
            "POST",
            "/v1/slots/${uuid(slotId)}/cancel",
            JSONObject().put("expectedVersion", expectedVersion),
            true,
            mutationHeaders(),
        )!!)
    }

    override suspend fun requestSlot(slotId: String): SlotModel =
        parseSlot(request("POST", "/v1/slots/${uuid(slotId)}/request", null, true, mutationHeaders())!!)

    override suspend fun leaveSlot(slotId: String): SlotModel =
        parseSlot(request("POST", "/v1/slots/${uuid(slotId)}/leave", null, true, mutationHeaders())!!)

    override suspend fun removeParticipant(slotId: String, userId: String, expectedVersion: Long): SlotModel {
        require(expectedVersion > 0)
        return parseSlot(request("POST", "/v1/slots/${uuid(slotId)}/members/${uuid(userId)}/remove",
            JSONObject().put("expectedVersion", expectedVersion), true, mutationHeaders())!!)
    }

    override suspend fun acceptedParticipants(slotId: String): List<SlotOrganizer> {
        val items = request("GET", "/v1/slots/${uuid(slotId)}/accepted", null, true)!!.getJSONArray("items")
        return buildList(items.length()) {
            for (index in 0 until items.length()) add(parseOrganizer(items.getJSONObject(index)))
        }
    }

    override suspend fun pendingRequests(slotId: String): List<PendingSlotRequest> {
        val items = request("GET", "/v1/slots/${uuid(slotId)}/requests", null, true)!!.getJSONArray("items")
        return buildList(items.length()) {
            for (index in 0 until items.length()) {
                val item = items.getJSONObject(index)
                add(PendingSlotRequest(
                    user = parseOrganizer(item.getJSONObject("user")),
                    requestedAtEpochMillis = Instant.parse(item.getString("requestedAt")).toEpochMilli(),
                ))
            }
        }
    }

    override suspend fun approveRequest(slotId: String, userId: String): SlotModel =
        parseSlot(request(
            "POST",
            "/v1/slots/${uuid(slotId)}/requests/${uuid(userId)}/approve",
            null,
            true,
            mutationHeaders(),
        )!!)

    override suspend fun rejectRequest(slotId: String, userId: String): SlotModel =
        parseSlot(request(
            "POST",
            "/v1/slots/${uuid(slotId)}/requests/${uuid(userId)}/reject",
            null,
            true,
            mutationHeaders(),
        )!!)

    override suspend fun startSlot(slotId: String): SlotModel =
        parseSlot(request("POST", "/v1/slots/${uuid(slotId)}/start", null, true, mutationHeaders())!!)

    override suspend fun completeSlot(slotId: String): SlotModel =
        parseSlot(request("POST", "/v1/slots/${uuid(slotId)}/complete", null, true, mutationHeaders())!!)

    override suspend fun chatMessages(slotId: String, limit: Int): List<ChatMessage> {
        require(limit in 1..100)
        val items = request("GET", "/v1/slots/${uuid(slotId)}/chat/messages?limit=$limit", null, true)!!.getJSONArray("items")
        return buildList(items.length()) { for (index in 0 until items.length()) add(parseChatMessage(items.getJSONObject(index))) }
    }

    override suspend fun sendChatMessage(slotId: String, text: String): ChatMessage {
        val normalized = text.trim()
        require(normalized.isNotEmpty())
        val fingerprint = "$slotId\u0000$normalized"
        val idempotencyKey = synchronized(pendingChatKeys) {
            pendingChatKeys.getOrPut(fingerprint) { UUID.randomUUID().toString() }
        }
        try {
            val message = parseChatMessage(request(
                "POST",
                "/v1/slots/${uuid(slotId)}/chat/messages",
                JSONObject().put("text", normalized),
                true,
                mapOf("Idempotency-Key" to idempotencyKey),
            )!!)
            synchronized(pendingChatKeys) { pendingChatKeys.remove(fingerprint) }
            return message
        } catch (error: Exception) {
            if (error is ApiException && error.status in 400..499 && error.status != 408 && error.status != 429) {
                synchronized(pendingChatKeys) { pendingChatKeys.remove(fingerprint) }
            }
            throw error
        }
    }

    private fun persistAuth(json: JSONObject): AuthSession {
        val token = json.getString("token")
        val expiresAt = Instant.parse(json.getString("expiresAt")).toEpochMilli()
        val session = AuthSession(parseUser(json.getJSONObject("user")), token, expiresAt)
        sessions.save(token, expiresAt)
        return session
    }

    private fun parseUser(json: JSONObject): UserProfile = UserProfile(
        id = json.getString("id"),
        email = json.getString("email"),
        username = json.getString("username"),
        displayName = json.getString("displayName"),
        avatarUrl = nullableString(json, "avatarUrl"),
        profileVisibility = json.getString("profileVisibility"),
        language = json.getString("language"),
    )

    private fun parseOrganizer(json: JSONObject): SlotOrganizer = SlotOrganizer(
        id = json.getString("id"),
        username = json.getString("username"),
        displayName = json.getString("displayName"),
        avatarUrl = nullableString(json, "avatarUrl"),
    )

    private fun parseSlot(json: JSONObject): SlotModel = SlotModel(
        id = json.getString("id"),
        organizer = parseOrganizer(json.getJSONObject("organizer")),
        title = json.getString("title"),
        activity = json.getString("activity"),
        details = nullableString(json, "details"),
        placeText = json.getString("placeText"),
        zoneText = nullableString(json, "zoneText"),
        startAtEpochMillis = nullableInstant(json, "startAt"),
        capacity = json.getInt("capacity"),
        acceptedCount = json.getInt("acceptedCount"),
        state = SlotState.valueOf(json.getString("state")),
        accessMode = SlotAccessMode.valueOf(json.getString("accessMode")),
        visibility = SlotVisibility.valueOf(json.getString("visibility")),
        viewerState = SlotViewerState.valueOf(json.getString("viewerState")),
        version = json.getLong("version"),
        createdAtEpochMillis = Instant.parse(json.getString("createdAt")).toEpochMilli(),
        updatedAtEpochMillis = Instant.parse(json.getString("updatedAt")).toEpochMilli(),
    )

    private fun parseChatMessage(json: JSONObject): ChatMessage {
        val author = json.getJSONObject("author")
        return ChatMessage(
            id = json.getString("id"),
            slotId = json.getString("slotId"),
            author = ChatAuthor(
                id = author.getString("id"),
                username = author.getString("username"),
                displayName = author.getString("displayName"),
                avatarUrl = nullableString(author, "avatarUrl"),
            ),
            text = json.getString("text"),
            createdAtEpochMillis = Instant.parse(json.getString("createdAt")).toEpochMilli(),
        )
    }

    private fun nullableString(json: JSONObject, key: String): String? =
        if (!json.has(key) || json.isNull(key)) null else json.getString(key)

    private fun nullableInstant(json: JSONObject, key: String): Long? =
        nullableString(json, key)?.let { Instant.parse(it).toEpochMilli() }

    private fun uuid(raw: String): String = UUID.fromString(raw).toString()
    private fun mutationHeaders(): Map<String, String> = mapOf("Idempotency-Key" to UUID.randomUUID().toString())

    private suspend fun request(
        method: String,
        path: String,
        body: JSONObject?,
        authenticated: Boolean,
        headers: Map<String, String> = emptyMap(),
    ): JSONObject? = withContext(Dispatchers.IO) {
        val connection = URL(root + path).openConnection() as HttpURLConnection
        try {
            connection.instanceFollowRedirects = false
            connection.useCaches = false
            connection.requestMethod = method
            connection.connectTimeout = 10_000
            connection.readTimeout = 15_000
            connection.setRequestProperty("Accept", "application/json")
            headers.forEach { (name, value) -> connection.setRequestProperty(name, value) }
            if (authenticated) {
                val token = sessions.load()?.token ?: throw ApiException(401, "unauthorized", "Authentication required")
                connection.setRequestProperty("Authorization", "Bearer $token")
            }
            if (body != null) {
                connection.doOutput = true
                connection.setRequestProperty("Content-Type", "application/json; charset=utf-8")
                connection.outputStream.use { it.write(body.toString().toByteArray(Charsets.UTF_8)) }
            }

            val status = connection.responseCode
            if (status == HttpURLConnection.HTTP_NO_CONTENT) return@withContext null
            val requestId = connection.getHeaderField("X-Request-ID")
            val stream = if (status in 200..299) connection.inputStream else connection.errorStream
            val responseText = try {
                stream?.use { readUtf8Bounded(it) }.orEmpty()
            } catch (_: ResponseTooLargeException) {
                throw ApiException(status, "response_too_large", "Server response exceeded the client safety limit", requestId)
            }
            val json = try { JSONObject(responseText) } catch (_: JSONException) { null }
            if (status !in 200..299) {
                if (authenticated && status == HttpURLConnection.HTTP_UNAUTHORIZED) {
                    sessions.clear()
                    synchronized(pendingChatKeys) { pendingChatKeys.clear() }
                    unauthorizedHandler?.invoke()
                }
                throw ApiException(
                    status = status,
                    code = json?.optString("code")?.ifBlank { null } ?: "http_error",
                    message = json?.optString("message")?.ifBlank { null } ?: "Request failed (HTTP $status)",
                    requestId = json?.optString("requestId")?.ifBlank { null } ?: requestId,
                )
            }
            json ?: throw ApiException(status, "protocol_error", "Invalid server response", requestId)
        } finally {
            connection.disconnect()
        }
    }
}
