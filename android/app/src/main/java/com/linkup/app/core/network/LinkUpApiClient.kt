package com.linkup.app.core.network

import com.linkup.app.core.session.SecureSessionStore
import java.io.ByteArrayOutputStream
import java.io.IOException
import java.io.InputStream
import java.net.HttpURLConnection
import java.net.URL
import java.net.URLEncoder
import java.time.Instant
import java.util.UUID
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import org.json.JSONException
import org.json.JSONObject

internal const val MAX_API_RESPONSE_BYTES = 1024 * 1024
private const val MAX_GET_ATTEMPTS = 2
private const val GET_RETRY_DELAY_MS = 250L
private const val MAX_MAP_WINDOW_MS = 7L * 24L * 60L * 60L * 1000L

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

internal fun shouldRetryGet(method: String, error: Exception): Boolean {
    if (method != "GET") return false
    return when (error) {
        is ApiException -> error.status in setOf(408, 429, 502, 503, 504)
        is IOException -> true
        else -> false
    }
}

class LinkUpApiClient(
    baseUrl: String,
    private val sessions: SecureSessionStore,
) : SocialApi, CityNetworkApi, CapabilityApi, V11AccessApi {
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

    suspend fun startOnboarding(
        draft: OnboardingRegistrationDraft,
        language: String,
        deviceLabel: String,
    ): OnboardingStart {
        val body = JSONObject()
            .put("email", draft.email)
            .put("username", draft.username)
            .put("displayName", draft.displayName)
            .put("password", draft.password)
            .put("language", language)
            .put("deviceLabel", deviceLabel)
            .put("birthDate", draft.birthDate)
            .put("cityId", draft.cityId)
            .put("cityName", draft.cityName)
            .put("preferences", onboardingPreferencesJson(draft.preferences))
        return parseOnboardingStart(request("POST", "/v1/auth/register", body, false)!!)
    }

    suspend fun onboardingStatus(verificationToken: String): OnboardingStatus =
        parseOnboardingStatus(
            request(
                "POST",
                "/v1/auth/register/status",
                JSONObject().put("verificationToken", verificationToken),
                false,
            )!!,
        )

    suspend fun completeOnboarding(verificationToken: String): AuthSession =
        persistAuth(
            request(
                "POST",
                "/v1/auth/register/complete",
                JSONObject().put("verificationToken", verificationToken),
                false,
            )!!,
        )

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
    override suspend fun capabilities(): CapabilitySnapshot =
        parseCapabilitySnapshot(request("GET", "/v1/capabilities", null, true)!!)

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

    override suspend fun searchPlaces(query: String, localityId: String?, limit: Int): List<CanonicalPlace> {
        val normalizedQuery = query.trim()
        val normalizedLocalityId = localityId?.trim()?.takeIf { it.isNotEmpty() }?.let(::uuid)
        require(normalizedQuery.length in 2..80)
        require(limit in 1..50)

        val path = buildString {
            append("/v1/places/search?q=")
            append(queryParam(normalizedQuery))
            normalizedLocalityId?.let {
                append("&localityId=")
                append(queryParam(it))
            }
            append("&limit=")
            append(limit)
        }
        val items = request("GET", path, null, true)!!.getJSONArray("items")
        return buildList(items.length()) {
            for (index in 0 until items.length()) add(parseCanonicalPlace(items.getJSONObject(index)))
        }
    }

    override suspend fun mapViewport(query: MapViewportQuery): List<MapCluster> {
        require(query.westE6 in -180_000_000..180_000_000)
        require(query.eastE6 in -180_000_000..180_000_000)
        require(query.southE6 in -90_000_000..90_000_000)
        require(query.northE6 in -90_000_000..90_000_000)
        require(query.southE6 < query.northE6)
        require(query.westE6 != query.eastE6)
        require(query.zoom in 1..20)
        require(query.limit in 1..200)
        requireValidMapWindow(query.fromEpochMillis, query.toEpochMillis)

        val path = buildString {
            append("/v1/map?westE6=").append(query.westE6)
            append("&southE6=").append(query.southE6)
            append("&eastE6=").append(query.eastE6)
            append("&northE6=").append(query.northE6)
            append("&zoom=").append(query.zoom)
            append("&from=").append(queryParam(Instant.ofEpochMilli(query.fromEpochMillis).toString()))
            append("&to=").append(queryParam(Instant.ofEpochMilli(query.toEpochMillis).toString()))
            append("&limit=").append(query.limit)
        }
        val items = request("GET", path, null, true)!!.getJSONArray("items")
        return buildList(items.length()) {
            for (index in 0 until items.length()) add(parseMapCluster(items.getJSONObject(index)))
        }
    }

    override suspend fun mapPlaceSlots(
        placeId: String,
        fromEpochMillis: Long,
        toEpochMillis: Long,
        limit: Int,
    ): List<SlotModel> {
        val normalizedPlaceId = uuid(placeId)
        requireValidMapWindow(fromEpochMillis, toEpochMillis)
        require(limit in 1..100)
        val path = buildString {
            append("/v1/map/places/").append(normalizedPlaceId).append("/slots?from=")
            append(queryParam(Instant.ofEpochMilli(fromEpochMillis).toString()))
            append("&to=").append(queryParam(Instant.ofEpochMilli(toEpochMillis).toString()))
            append("&limit=").append(limit)
        }
        val items = request("GET", path, null, true)!!.getJSONArray("items")
        return buildList(items.length()) {
            for (index in 0 until items.length()) add(parseSlot(items.getJSONObject(index)))
        }
    }

    override suspend fun createSlot(input: CreateSlotInput): SlotModel {
        val body = JSONObject()
            .put("title", input.title)
            .put("activity", input.activity)
            .put("placeText", input.placeText)
            .put("capacity", input.capacity)
            .put("accessMode", input.accessMode.name)
        input.details?.let { body.put("details", it) }
        input.zoneText?.let { body.put("zoneText", it) }
        input.canonicalPlaceId?.let { body.put("canonicalPlaceId", uuid(it)) }
        input.startAtEpochMillis?.let { body.put("startAt", Instant.ofEpochMilli(it).toString()) }
        return parseSlot(request("POST", "/v1/slots", body, true, mutationHeaders())!!)
    }

    override suspend fun getSlot(slotId: String): SlotModel =
        parseSlot(request("GET", "/v1/slots/${uuid(slotId)}", null, true)!!)

    override suspend fun editSlot(slotId: String, input: EditSlotInput): SlotModel {
        require(input.expectedVersion > 0)
        require(input.canonicalPlaceId == null || !input.clearCanonicalPlaceId)
        val body = JSONObject().put("expectedVersion", input.expectedVersion)
        input.title?.let { body.put("title", it) }
        input.details?.let { body.put("details", it) }
        input.placeText?.let { body.put("placeText", it) }
        input.zoneText?.let { body.put("zoneText", it) }
        input.canonicalPlaceId?.let { body.put("canonicalPlaceId", uuid(it)) }
        if (input.clearCanonicalPlaceId) body.put("clearCanonicalPlaceId", true)
        input.startAtEpochMillis?.let { body.put("startAt", Instant.ofEpochMilli(it).toString()) }
        if (input.clearStartAt) body.put("clearStartAt", true)
        input.capacity?.let { body.put("capacity", it) }
        input.accessMode?.let { body.put("accessMode", it.name) }
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

    override suspend fun joinSlot(slotId: String): SlotModel =
        parseSlot(request("POST", "/v1/slots/${uuid(slotId)}/join", null, true, mutationHeaders())!!)

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
        canonicalPlaceId = nullableString(json, "canonicalPlaceId"),
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

    private fun parseCanonicalPlace(json: JSONObject): CanonicalPlace = CanonicalPlace(
        id = uuid(json.getString("id")),
        name = json.getString("name"),
        category = nullableString(json, "category"),
        locality = nullableString(json, "locality"),
        localityId = nullableString(json, "localityId")?.let(::uuid),
        countryCode = nullableString(json, "countryCode"),
        latitudeE6 = json.getInt("latitudeE6"),
        longitudeE6 = json.getInt("longitudeE6"),
        precisionM = json.getInt("precisionM"),
    )

    private fun parseMapCluster(json: JSONObject): MapCluster = MapCluster(
        key = json.getString("key"),
        latitudeE6 = json.getInt("latitudeE6"),
        longitudeE6 = json.getInt("longitudeE6"),
        placeCount = json.getInt("placeCount"),
        slotCount = json.getInt("slotCount"),
        placeId = nullableString(json, "placeId")?.let(::uuid),
        placeName = nullableString(json, "placeName"),
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
    private fun queryParam(raw: String): String = URLEncoder.encode(raw, Charsets.UTF_8.name())
    private fun mutationHeaders(): Map<String, String> = mapOf("Idempotency-Key" to UUID.randomUUID().toString())

    private fun requireValidMapWindow(fromEpochMillis: Long, toEpochMillis: Long) {
        require(fromEpochMillis < toEpochMillis)
        val duration = runCatching { Math.subtractExact(toEpochMillis, fromEpochMillis) }.getOrElse { -1L }
        require(duration in 1..MAX_MAP_WINDOW_MS)
    }

    private suspend fun request(
        method: String,
        path: String,
        body: JSONObject?,
        authenticated: Boolean,
        headers: Map<String, String> = emptyMap(),
    ): JSONObject? = withContext(Dispatchers.IO) {
        val maxAttempts = if (method == "GET") MAX_GET_ATTEMPTS else 1
        var lastError: Exception? = null
        for (attempt in 1..maxAttempts) {
            try {
                return@withContext requestOnce(method, path, body, authenticated, headers)
            } catch (error: CancellationException) {
                throw error
            } catch (error: Exception) {
                lastError = error
                if (attempt >= maxAttempts || !shouldRetryGet(method, error)) throw error
                delay(GET_RETRY_DELAY_MS)
            }
        }
        throw lastError ?: IOException("request failed")
    }

    private fun requestOnce(
        method: String,
        path: String,
        body: JSONObject?,
        authenticated: Boolean,
        headers: Map<String, String>,
    ): JSONObject? {
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
            if (status == HttpURLConnection.HTTP_NO_CONTENT) return null
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
            return json ?: throw ApiException(status, "protocol_error", "Invalid server response", requestId)
        } finally {
            connection.disconnect()
        }
    }
}
