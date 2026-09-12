package com.linkup.app.core.network

import java.io.IOException
import java.net.HttpURLConnection
import java.net.URL
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import org.json.JSONException
import org.json.JSONObject

private const val APP_UPDATE_GET_ATTEMPTS = 2
private const val APP_UPDATE_RETRY_DELAY_MS = 250L

/** GET /v1/app-update/latest's parsed response — see backend internal/appupdate. */
data class AppUpdateInfo(
    val versionCode: Int,
    val versionName: String,
    val sha256: String,
    val releaseNotes: String,
    val mandatory: Boolean,
    val apkUrl: String,
)

/**
 * Over-the-air update check. Deliberately unauthenticated on both ends
 * (matching the server's own doc comment): a signed-out client, or one
 * whose session already expired, must still be able to learn a fix exists.
 */
class AppUpdateApiClient(baseUrl: String) {
    private val root = validatedApiRoot(baseUrl)

    suspend fun latest(): AppUpdateInfo = withContext(Dispatchers.IO) {
        var lastError: Exception? = null
        repeat(APP_UPDATE_GET_ATTEMPTS) { attempt ->
            try {
                return@withContext requestOnce()
            } catch (error: CancellationException) {
                throw error
            } catch (error: Exception) {
                lastError = error
                val retryable = when (error) {
                    is ApiException -> error.status in setOf(408, 429, 502, 503, 504)
                    is IOException -> true
                    else -> false
                }
                if (attempt == APP_UPDATE_GET_ATTEMPTS - 1 || !retryable) throw error
                delay(APP_UPDATE_RETRY_DELAY_MS)
            }
        }
        throw lastError ?: IOException("app update check failed")
    }

    private fun requestOnce(): AppUpdateInfo {
        val connection = (URL(root + "/v1/app-update/latest").openConnection() as HttpURLConnection).apply {
            requestMethod = "GET"
            connectTimeout = 10_000
            readTimeout = 15_000
            instanceFollowRedirects = false
            useCaches = false
            setRequestProperty("Accept", "application/json")
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
                throw ApiException(502, "protocol_error", "invalid server response", requestId)
            }
            return AppUpdateInfo(
                versionCode = json.getInt("versionCode"),
                versionName = json.getString("versionName"),
                sha256 = json.getString("sha256"),
                releaseNotes = json.optString("releaseNotes", ""),
                mandatory = json.optBoolean("mandatory", false),
                apkUrl = json.getString("apkUrl"),
            )
        } finally {
            connection.disconnect()
        }
    }
}
