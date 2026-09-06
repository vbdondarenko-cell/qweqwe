package com.linkup.app.core.network

import com.linkup.app.core.session.SecureSessionStore
import java.net.HttpURLConnection
import java.net.URL
import java.time.Instant
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONObject

class LinkUpApiClient(
    baseUrl: String,
    private val sessions: SecureSessionStore,
) {
    private val root = baseUrl.trimEnd('/')

    init {
        require(root.startsWith("https://") || root.startsWith("http://10.0.2.2") || root.startsWith("http://127.0.0.1")) {
            "API base URL must use HTTPS outside local Android development"
        }
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

    suspend fun logout() {
        try {
            request("POST", "/v1/auth/logout", null, true)
        } finally {
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
        avatarUrl = if (json.isNull("avatarUrl")) null else json.optString("avatarUrl").ifBlank { null },
        profileVisibility = json.getString("profileVisibility"),
        language = json.getString("language"),
    )

    private suspend fun request(method: String, path: String, body: JSONObject?, authenticated: Boolean): JSONObject? =
        withContext(Dispatchers.IO) {
            val connection = URL(root + path).openConnection() as HttpURLConnection
            try {
                connection.requestMethod = method
                connection.connectTimeout = 10_000
                connection.readTimeout = 15_000
                connection.setRequestProperty("Accept", "application/json")
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
                val stream = if (status in 200..299) connection.inputStream else connection.errorStream
                val text = stream?.bufferedReader(Charsets.UTF_8)?.use { it.readText() }.orEmpty()
                val json = if (text.isBlank()) JSONObject() else JSONObject(text)
                if (status !in 200..299) {
                    throw ApiException(
                        status = status,
                        code = json.optString("code", "http_error"),
                        message = json.optString("message", "Request failed"),
                        requestId = json.optString("requestId").ifBlank { null },
                    )
                }
                json
            } finally {
                connection.disconnect()
            }
        }
}
