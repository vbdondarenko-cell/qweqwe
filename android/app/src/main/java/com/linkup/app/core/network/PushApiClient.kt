package com.linkup.app.core.network

import com.linkup.app.core.session.SecureSessionStore
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URL
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONObject

class PushApiClient(
    baseUrl: String,
    private val sessions: SecureSessionStore,
) {
    private val root = validatedApiRoot(baseUrl)

    suspend fun registerAndroid(installationId: String, token: String, appVersion: String) = withContext(Dispatchers.IO) {
        val session = sessions.load() ?: return@withContext
        val body = JSONObject()
            .put("installationId", installationId)
            .put("token", token)
            .put("appVersion", appVersion)
            .toString()
            .toByteArray(Charsets.UTF_8)
        val connection = (URL("$root/v1/me/push/android").openConnection() as HttpURLConnection).apply {
            requestMethod = "PUT"
            connectTimeout = 8_000
            readTimeout = 8_000
            doOutput = true
            setRequestProperty("Authorization", "Bearer ${session.token}")
            setRequestProperty("Content-Type", "application/json; charset=utf-8")
            setFixedLengthStreamingMode(body.size)
        }
        try {
            connection.outputStream.use { it.write(body) }
            val status = connection.responseCode
            if (status == HttpURLConnection.HTTP_NO_CONTENT) return@withContext
            val text = runCatching {
                val stream = if (status >= 400) connection.errorStream else connection.inputStream
                stream?.use { readUtf8Bounded(it, 64 * 1024) }.orEmpty()
            }.getOrDefault("")
            throw IOException("push registration failed: HTTP $status ${text.take(256)}")
        } finally {
            connection.disconnect()
        }
    }
}
