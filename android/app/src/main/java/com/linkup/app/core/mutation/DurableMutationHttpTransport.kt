package com.linkup.app.core.mutation

import com.linkup.app.core.network.MAX_API_RESPONSE_BYTES
import com.linkup.app.core.network.ResponseTooLargeException
import com.linkup.app.core.network.readUtf8Bounded
import com.linkup.app.core.network.validatedApiRoot
import com.linkup.app.core.session.SecureSessionStore
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URL
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONObject

internal fun durableDispositionForHttpStatus(status: Int): DurableAttemptDisposition = when {
    status in 200..299 -> DurableAttemptDisposition.ACKNOWLEDGED
    status == 408 || status == 429 || status >= 500 -> DurableAttemptDisposition.AMBIGUOUS_FAILURE
    status in 400..499 -> DurableAttemptDisposition.DEFINITIVE_FAILURE
    else -> DurableAttemptDisposition.AMBIGUOUS_FAILURE
}

class DurableMutationHttpTransport(
    baseUrl: String,
    private val sessions: SecureSessionStore,
) : DurableMutationTransport {
    private val root = validatedApiRoot(baseUrl)

    override suspend fun execute(command: DurableMutationCommand): DurableAttemptResult = withContext(Dispatchers.IO) {
        val stored = sessions.load()
            ?: return@withContext DurableAttemptResult(
                disposition = DurableAttemptDisposition.DEFINITIVE_FAILURE,
                errorCode = "unauthorized",
                httpStatus = HttpURLConnection.HTTP_UNAUTHORIZED,
            )
        try {
            executeOnce(command, stored.token)
        } catch (error: CancellationException) {
            throw error
        } catch (_: IOException) {
            DurableAttemptResult(
                disposition = DurableAttemptDisposition.AMBIGUOUS_FAILURE,
                errorCode = "network_error",
            )
        }
    }

    private fun executeOnce(command: DurableMutationCommand, bearerToken: String): DurableAttemptResult {
        val connection = URL(root + command.path).openConnection() as HttpURLConnection
        try {
            connection.instanceFollowRedirects = false
            connection.useCaches = false
            connection.requestMethod = command.method
            connection.connectTimeout = 10_000
            connection.readTimeout = 15_000
            connection.setRequestProperty("Accept", "application/json")
            connection.setRequestProperty("Authorization", "Bearer $bearerToken")
            connection.setRequestProperty("Idempotency-Key", command.idempotencyKey)
            if (command.bodyJson != null) {
                connection.doOutput = true
                connection.setRequestProperty("Content-Type", "application/json; charset=utf-8")
                connection.outputStream.use { it.write(command.bodyJson.toByteArray(Charsets.UTF_8)) }
            }

            val status = connection.responseCode
            val disposition = durableDispositionForHttpStatus(status)
            val stream = if (status in 200..299) connection.inputStream else connection.errorStream
            val responseText = try {
                stream?.use { readUtf8Bounded(it, MAX_API_RESPONSE_BYTES) }.orEmpty()
            } catch (_: ResponseTooLargeException) {
                return DurableAttemptResult(
                    disposition = disposition,
                    errorCode = "response_too_large",
                    httpStatus = status,
                )
            }

            if (status == HttpURLConnection.HTTP_UNAUTHORIZED) sessions.clear()
            val errorCode = if (disposition == DurableAttemptDisposition.ACKNOWLEDGED) {
                null
            } else {
                runCatching { JSONObject(responseText).optString("code") }
                    .getOrNull()
                    ?.takeIf { it.isNotBlank() }
                    ?: "http_$status"
            }
            return DurableAttemptResult(
                disposition = disposition,
                responseJson = responseText.takeIf { it.isNotBlank() },
                errorCode = errorCode,
                httpStatus = status,
            )
        } finally {
            connection.disconnect()
        }
    }
}
