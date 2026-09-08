package com.linkup.app.core.mutation

import java.security.MessageDigest
import java.util.UUID

const val DURABLE_MUTATION_MAX_COMMANDS = 100
const val DURABLE_MUTATION_MAX_BODY_BYTES = 64 * 1024
const val DURABLE_MUTATION_REPLAY_WINDOW_MS = 20L * 60L * 60L * 1000L

enum class DurableResponseKind { SLOT, CHAT, NONE }

data class DurableMutationCommand(
    val idempotencyKey: String,
    val ownerFingerprint: String,
    val method: String,
    val path: String,
    val bodyJson: String?,
    val responseKind: DurableResponseKind,
    val createdAtEpochMillis: Long,
    val firstAttemptAtEpochMillis: Long? = null,
) {
    init {
        require(runCatching { UUID.fromString(idempotencyKey).toString() == idempotencyKey.lowercase() }.getOrDefault(false))
        require(ownerFingerprint.length == 64 && ownerFingerprint.all { it in '0'..'9' || it in 'a'..'f' })
        require(method in setOf("POST", "PATCH", "PUT", "DELETE"))
        require(path.startsWith("/v1/") && !path.contains("://") && !path.contains('\n') && !path.contains('\r'))
        require(bodyJson == null || bodyJson.toByteArray(Charsets.UTF_8).size <= DURABLE_MUTATION_MAX_BODY_BYTES)
        require(createdAtEpochMillis > 0)
        require(firstAttemptAtEpochMillis == null || firstAttemptAtEpochMillis >= createdAtEpochMillis)
    }

    fun canAutoReplay(nowEpochMillis: Long): Boolean {
        require(nowEpochMillis > 0)
        val attempted = firstAttemptAtEpochMillis ?: return true
        if (nowEpochMillis < attempted) return false
        return nowEpochMillis - attempted <= DURABLE_MUTATION_REPLAY_WINDOW_MS
    }

    fun markAttempt(nowEpochMillis: Long): DurableMutationCommand {
        require(nowEpochMillis >= createdAtEpochMillis)
        return if (firstAttemptAtEpochMillis == null) copy(firstAttemptAtEpochMillis = nowEpochMillis) else this
    }
}

interface MutationOutbox {
    fun enqueue(command: DurableMutationCommand)
    fun markAttempt(idempotencyKey: String, nowEpochMillis: Long): DurableMutationCommand?
    fun remove(idempotencyKey: String)
    fun pendingReplayable(ownerFingerprint: String, nowEpochMillis: Long): List<DurableMutationCommand>
    fun unsafeAmbiguousCount(ownerFingerprint: String, nowEpochMillis: Long): Int
    fun clearOwner(ownerFingerprint: String)
    fun clearAll()
}

fun mutationOwnerFingerprint(bearerToken: String): String {
    require(bearerToken.isNotBlank())
    val digest = MessageDigest.getInstance("SHA-256").digest(bearerToken.toByteArray(Charsets.UTF_8))
    val alphabet = "0123456789abcdef"
    return buildString(digest.size * 2) {
        digest.forEach { byte ->
            val value = byte.toInt() and 0xff
            append(alphabet[value ushr 4])
            append(alphabet[value and 0x0f])
        }
    }
}
