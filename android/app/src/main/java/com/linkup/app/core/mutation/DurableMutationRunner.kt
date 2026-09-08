package com.linkup.app.core.mutation

enum class DurableAttemptDisposition {
    ACKNOWLEDGED,
    DEFINITIVE_FAILURE,
    AMBIGUOUS_FAILURE,
}

data class DurableAttemptResult(
    val disposition: DurableAttemptDisposition,
    val responseJson: String? = null,
    val errorCode: String? = null,
    val httpStatus: Int? = null,
)

fun interface DurableMutationTransport {
    suspend fun execute(command: DurableMutationCommand): DurableAttemptResult
}

data class DurableReplayReport(
    val acknowledgedKeys: List<String>,
    val definitiveFailureKey: String? = null,
    val ambiguousFailureKey: String? = null,
    val unsafeAmbiguousCount: Int = 0,
) {
    val completed: Boolean
        get() = definitiveFailureKey == null && ambiguousFailureKey == null && unsafeAmbiguousCount == 0
}

class DurableMutationRunner(
    private val outbox: MutationOutbox,
    private val nowEpochMillis: () -> Long = System::currentTimeMillis,
) {
    fun unsafeAmbiguousCount(ownerFingerprint: String): Int {
        requireValidOwner(ownerFingerprint)
        return outbox.unsafeAmbiguousCount(ownerFingerprint, safeNow(1L))
    }

    fun findReplayableEquivalent(
        ownerFingerprint: String,
        method: String,
        path: String,
        bodyJson: String?,
        responseKind: DurableResponseKind,
    ): DurableMutationCommand? {
        requireValidOwner(ownerFingerprint)
        val now = safeNow(1L)
        return outbox.pendingReplayable(ownerFingerprint, now).firstOrNull {
            it.method == method &&
                it.path == path &&
                it.bodyJson == bodyJson &&
                it.responseKind == responseKind
        }
    }

    suspend fun submit(
        command: DurableMutationCommand,
        transport: DurableMutationTransport,
    ): DurableAttemptResult {
        outbox.enqueue(command)
        val attempted = outbox.markAttempt(command.idempotencyKey, safeNow(command.createdAtEpochMillis))
            ?: error("durable mutation disappeared before transport attempt")
        val result = transport.execute(attempted)
        if (result.disposition != DurableAttemptDisposition.AMBIGUOUS_FAILURE) {
            outbox.remove(command.idempotencyKey)
        }
        return result
    }

    suspend fun replay(
        ownerFingerprint: String,
        transport: DurableMutationTransport,
    ): DurableReplayReport {
        requireValidOwner(ownerFingerprint)
        val now = safeNow(1L)
        val unsafe = outbox.unsafeAmbiguousCount(ownerFingerprint, now)
        if (unsafe > 0) {
            return DurableReplayReport(
                acknowledgedKeys = emptyList(),
                unsafeAmbiguousCount = unsafe,
            )
        }

        val acknowledged = mutableListOf<String>()
        for (command in outbox.pendingReplayable(ownerFingerprint, now)) {
            val attempted = outbox.markAttempt(command.idempotencyKey, safeNow(command.createdAtEpochMillis))
                ?: continue
            when (transport.execute(attempted).disposition) {
                DurableAttemptDisposition.ACKNOWLEDGED -> {
                    outbox.remove(command.idempotencyKey)
                    acknowledged += command.idempotencyKey
                }
                DurableAttemptDisposition.DEFINITIVE_FAILURE -> {
                    outbox.remove(command.idempotencyKey)
                    return DurableReplayReport(
                        acknowledgedKeys = acknowledged,
                        definitiveFailureKey = command.idempotencyKey,
                    )
                }
                DurableAttemptDisposition.AMBIGUOUS_FAILURE -> {
                    return DurableReplayReport(
                        acknowledgedKeys = acknowledged,
                        ambiguousFailureKey = command.idempotencyKey,
                    )
                }
            }
        }
        return DurableReplayReport(acknowledgedKeys = acknowledged)
    }

    private fun requireValidOwner(ownerFingerprint: String) {
        require(ownerFingerprint.length == 64 && ownerFingerprint.all { it in '0'..'9' || it in 'a'..'f' })
    }

    private fun safeNow(createdAtEpochMillis: Long): Long =
        maxOf(createdAtEpochMillis, nowEpochMillis().coerceAtLeast(1L))
}
