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
import com.linkup.app.core.network.V11HostingApi
import java.io.IOException
import java.time.Instant
import java.util.UUID
import org.json.JSONException
import org.json.JSONObject

class DurableMutationQueuedException : IOException("Mutation is queued for safe retry")
class DurableMutationSafetyException : IOException("An older ambiguous mutation needs reconciliation before new writes")

class DurableSocialApi(
    private val delegate: SocialApi,
    private val currentBearerToken: () -> String?,
    private val runner: DurableMutationRunner,
    private val transport: DurableMutationTransport,
    private val onUnauthorized: () -> Unit = {},
    private val nowEpochMillis: () -> Long = System::currentTimeMillis,
) : SocialApi, V11HostingApi {
    override suspend fun mySlots(view: MySlotsView): List<SlotModel> = delegate.mySlots(view)
    override suspend fun pulse(): List<SlotModel> = delegate.pulse()
    override suspend fun getSlot(slotId: String): SlotModel = delegate.getSlot(slotId)
    override suspend fun acceptedParticipants(slotId: String): List<SlotOrganizer> = delegate.acceptedParticipants(slotId)
    override suspend fun pendingRequests(slotId: String): List<PendingSlotRequest> = delegate.pendingRequests(slotId)
    override suspend fun chatMessages(slotId: String, limit: Int): List<ChatMessage> = delegate.chatMessages(slotId, limit)

    override suspend fun createSlot(input: CreateSlotInput): SlotModel =
        executeSlot("POST", "/v1/slots", createBody(input))

    override suspend fun createDraft(input: CreateSlotInput): SlotModel =
        executeSlot("POST", "/v1/slots/drafts", createBody(input))

    override suspend fun publishDraft(slotId: String, expectedVersion: Long): SlotModel {
        require(expectedVersion > 0)
        return executeSlot(
            "POST",
            "/v1/slots/${uuid(slotId)}/publish",
            JSONObject().put("expectedVersion", expectedVersion),
        )
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
        input.accessMode?.let { body.put("accessMode", it.name) }
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
        val token = currentBearerToken() ?: return null
        val report = runner.replay(mutationOwnerFingerprint(token), transport)
        if (currentBearerToken() == null) onUnauthorized()
        return report
    }

    private fun createBody(input: CreateSlotInput): JSONObject {
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
        return body
    }

    private suspend fun executeSlot(method: String, path: String, body: JSONObject?): SlotModel =
        parseSlot(execute(method, path, body, DurableResponseKind.SLOT))

    private suspend fun execute(
        method: String,
        path: String,
        body: JSONObject?,
        responseKind: DurableResponseKind,
    ): JSONObject {
        val token = currentBearerToken() ?: run {
            onUnauthorized()
            throw ApiException(401, "unauthorized", "Authentication required")
        }
        val ownerFingerprint = mutationOwnerFingerprint(token)
        if (runner.unsafeAmbiguousCount(ownerFingerprint) > 0) {
            throw DurableMutationSafetyException()
        }

        val bodyJson = body?.toString()
        val command = runner.findReplayableEquivalent(
            ownerFingerprint = ownerFingerprint,
            method = method,
            path = path,
            bodyJson = bodyJson,
            responseKind = responseKind,
        ) ?: DurableMutationCommand(
            idempotencyKey = UUID.randomUUID().toString(),
            ownerFingerprint = ownerFingerprint,
            method = method,
            path = path,
            bodyJson = bodyJson,
            responseKind = responseKind,
            createdAtEpochMillis = nowEpochMillis().coerceAtLeast(1L),
        )

        val result = runner.submit(command, transport)
        return when (result.disposition) {
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
