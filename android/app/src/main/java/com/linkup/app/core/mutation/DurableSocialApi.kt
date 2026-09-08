package com.linkup.app.core.mutation

import com.linkup.app.core.network.ApiException
import com.linkup.app.core.network.ChatAuthor
import com.linkup.app.core.network.ChatMessage
import com.linkup.app.core.network.CreateSlotInput
import com.linkup.app.core.network.EditSlotInput
import com.linkup.app.core.network.MySlotsView
import com.linkup.app.core.network.PendingSlotRequest
import com.linkup.app.core.network.SlotAccessMode
import com.linkup.app.core.network.SlotModel
import com.linkup.app.core.network.SlotOrganizer
import com.linkup.app.core.network.SlotState
import com.linkup.app.core.network.SlotViewerState
import com.linkup.app.core.network.SlotVisibility
import com.linkup.app.core.network.SocialApi
import com.linkup.app.core.session.SecureSessionStore
import java.io.IOException
import java.time.Instant
import java.util.UUID
import org.json.JSONException
import org.json.JSONObject

class DurableMutationQueuedException : IOException("Mutation is queued for safe retry")

class DurableSocialApi(
    private val delegate: SocialApi,
    private val sessions: SecureSessionStore,
    private val runner: DurableMutationRunner,
    private val transport: DurableMutationTransport,
    private val onUnauthorized: () -> Unit = {},
) : SocialApi {
    override suspend fun mySlots(view: MySlotsView): List<SlotModel> = delegate.mySlots(view)
    override suspend fun pulse(): List<SlotModel> = delegate.pulse()
    override suspend fun getSlot(slotId: String): SlotModel = delegate.getSlot(slotId)
    override suspend fun acceptedParticipants(slotId: String): List<SlotOrganizer> = delegate.acceptedParticipants(slotId)
    override suspend fun pendingRequests(slotId: String): List<PendingSlotRequest> = delegate.pendingRequests(slotId)
    override suspend fun chatMessages(slotId: String, limit: Int): List<ChatMessage> = delegate.chatMessages(slotId, limit)

    override suspend fun createSlot(input: CreateSlotInput): SlotModel {
        val body = JSONObject()
            .put("title", input.title)
            .put("activity", input.activity)
            .put("placeText", input.placeText)
            .put("capacity", input.capacity)
        input.details?.let { body.put("details", it) }
        input.zoneText?.let { body.put("zoneText", it) }
        input.canonicalPlaceId?.let { body.put("canonicalPlaceId", uuid(it)) }
        input.startAtEpochMillis?.let { body.put("startAt", Instant.ofEpochMilli(it).toString()) }
        return executeSlot("POST", "/v1/slots", body)
    }

    override suspend fun editSlot(slotId: String, input: EditSlotInput): SlotModel {
        require(input.expectedVersion > 0)
        require(input.canonicalPlaceId == null || !input.clearCanonicalPlaceId)
        val normalizedSlotId = uuid(slotId)
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
        return executeSlot("PATCH", "/v1/slots/$normalizedSlotId", body)
    }

    override suspend fun cancelSlot(slotId: String, expectedVersion: Long): SlotModel {
        require(expectedVersion > 0)
        return executeSlot(
            "POST",
            "/v1/slots/${uuid(slotId)}/cancel",
            JSONObject().put("expectedVersion", expectedVersion),
        )
    }

    override suspend fun requestSlot(slotId: String): SlotModel =
        executeSlot("POST", "/v1/slots/${uuid(slotId)}/request", null)

    override suspend fun leaveSlot(slotId: String): SlotModel =
        executeSlot("POST", "/v1/slots/${uuid(slotId)}/leave", null)

    override suspend fun removeParticipant(slotId: String, userId: String, expectedVersion: Long): SlotModel {
        require(expectedVersion > 0)
        return executeSlot(
            "POST",
            "/v1/slots/${uuid(slotId)}/members/${uuid(userId)}/remove",
            JSONObject().put("expectedVersion", expectedVersion),
        )
    }

    override suspend fun approveRequest(slotId: String, userId: String): SlotModel =
        executeSlot("POST", "/v1/slots/${uuid(slotId)}/requests/${uuid(userId)}/approve", null)

    override suspend fun rejectRequest(slotId: String, userId: String): SlotModel =
        executeSlot("POST", "/v1/slots/${uuid(slotId)}/requests/${uuid(userId)}/reject", null)

    override suspend fun startSlot(slotId: String): SlotModel =
        executeSlot("POST", "/v1/slots/${uuid(slotId)}/start", null)

    override suspend fun completeSlot(slotId: String): SlotModel =
        executeSlot("POST", "/v1/slots/${uuid(slotId)}/complete", null)

    override suspend fun sendChatMessage(slotId: String, text: String): ChatMessage {
        val normalized = text.trim()
        require(normalized.isNotEmpty())
        val json = execute(
            method = "POST",
            path = "/v1/slots/${uuid(slotId)}/chat/messages",
            body = JSONObject().put("text", normalized),
            responseKind = DurableResponseKind.CHAT,
        )
        return parseChatMessage(json)
    }

    suspend fun replayPending(): DurableReplayReport? {
        val stored = sessions.load() ?: return null
        val report = runner.replay(mutationOwnerFingerprint(stored.token), transport)
        if (sessions.load() == null) onUnauthorized()
        return report
    }

    private suspend fun executeSlot(method: String, path: String, body: JSONObject?): SlotModel =
        parseSlot(execute(method, path, body, DurableResponseKind.SLOT))

    private suspend fun execute(
        method: String,
        path: String,
        body: JSONObject?,
        responseKind: DurableResponseKind,
    ): JSONObject {
        val stored = sessions.load() ?: run {
            onUnauthorized()
            throw ApiException(401, "unauthorized", "Authentication required")
        }
        val command = DurableMutationCommand(
            idempotencyKey = UUID.randomUUID().toString(),
            ownerFingerprint = mutationOwnerFingerprint(stored.token),
            method = method,
            path = path,
            bodyJson = body?.toString(),
            responseKind = responseKind,
            createdAtEpochMillis = System.currentTimeMillis().coerceAtLeast(1L),
        )
        return when (val result = runner.submit(command, transport)) {
            is DurableAttemptResult -> when (result.disposition) {
                DurableAttemptDisposition.ACKNOWLEDGED -> parseAcknowledged(result)
                DurableAttemptDisposition.DEFINITIVE_FAILURE -> {
                    if (result.httpStatus == 401) onUnauthorized()
                    throw ApiException(
                        status = result.httpStatus ?: 400,
                        code = result.errorCode ?: "request_failed",
                        message = "Mutation was rejected by the server",
                    )
                }
                DurableAttemptDisposition.AMBIGUOUS_FAILURE -> throw DurableMutationQueuedException()
            }
        }
    }

    private fun parseAcknowledged(result: DurableAttemptResult): JSONObject {
        val raw = result.responseJson
            ?: throw ApiException(result.httpStatus ?: 200, "protocol_error", "Mutation was acknowledged without a response body")
        return try {
            JSONObject(raw)
        } catch (_: JSONException) {
            throw ApiException(result.httpStatus ?: 200, "protocol_error", "Invalid mutation response")
        }
    }

    private fun parseSlot(json: JSONObject): SlotModel = SlotModel(
        id = json.getString("id"),
        organizer = parseOrganizer(json.getJSONObject("organizer")),
        title = json.getString("title"),
        activity = json.getString("activity"),
        details = nullableString(json, "details"),
        placeText = json.getString("placeText"),
        zoneText = nullableString(json, "zoneText"),
        canonicalPlaceId = nullableString(json, "canonicalPlaceId"),
        startAtEpochMillis = nullableString(json, "startAt")?.let { Instant.parse(it).toEpochMilli() },
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

    private fun parseOrganizer(json: JSONObject): SlotOrganizer = SlotOrganizer(
        id = json.getString("id"),
        username = json.getString("username"),
        displayName = json.getString("displayName"),
        avatarUrl = nullableString(json, "avatarUrl"),
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

    private fun uuid(raw: String): String = UUID.fromString(raw).toString()
}
