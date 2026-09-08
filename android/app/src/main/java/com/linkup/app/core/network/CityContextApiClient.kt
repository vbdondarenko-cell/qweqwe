package com.linkup.app.core.network

import com.linkup.app.core.session.SecureSessionStore
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URL
import java.time.Instant
import java.time.ZoneId
import java.util.UUID
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import org.json.JSONException
import org.json.JSONObject

private const val CITY_CONTEXT_GET_ATTEMPTS = 2
private const val CITY_CONTEXT_RETRY_DELAY_MS = 250L

class CityContextApiClient(
    baseUrl: String,
    private val sessions: SecureSessionStore,
) : CityContextApi {
    private val root = validatedApiRoot(baseUrl)

    override suspend fun currentCityContext(): CityContextModel = withContext(Dispatchers.IO) {
        var lastError: Exception? = null
        repeat(CITY_CONTEXT_GET_ATTEMPTS) { attempt ->
            try {
                return@withContext requestOnce("GET", "/v1/city-context", null)
            } catch (error: CancellationException) {
                throw error
            } catch (error: Exception) {
                lastError = error
                val retryable = when (error) {
                    is ApiException -> error.status in setOf(408, 429, 502, 503, 504)
                    is IOException -> true
                    else -> false
                }
                if (attempt == CITY_CONTEXT_GET_ATTEMPTS - 1 || !retryable) throw error
                delay(CITY_CONTEXT_RETRY_DELAY_MS)
            }
        }
        throw lastError ?: IOException("city context request failed")
    }

    override suspend fun resolveCityContext(observation: CityLocationObservation): CityContextModel = withContext(Dispatchers.IO) {
        require(observation.latitudeE6 in -90_000_000..90_000_000)
        require(observation.longitudeE6 in -180_000_000..180_000_000)
        require(observation.accuracyM in 1..10_000)
        require(observation.capturedAtEpochMillis > 0)

        val body = JSONObject()
            .put("latitudeE6", observation.latitudeE6)
            .put("longitudeE6", observation.longitudeE6)
            .put("accuracyM", observation.accuracyM)
            .put("permissionClass", observation.permissionClass.name)
            .put("capturedAt", Instant.ofEpochMilli(observation.capturedAtEpochMillis).toString())
            .put("mocked", observation.mocked)

        // Deliberately no blind POST retry: the server applies City-Lock hysteresis.
        // A new observation may be submitted explicitly after transport recovery.
        requestOnce("POST", "/v1/city-context/resolve", body)
    }

    private fun requestOnce(method: String, path: String, body: JSONObject?): CityContextModel {
        val stored = sessions.load() ?: throw ApiException(401, "unauthorized", "authentication required")
        val connection = (URL(root + path).openConnection() as HttpURLConnection).apply {
            requestMethod = method
            connectTimeout = 10_000
            readTimeout = 15_000
            instanceFollowRedirects = false
            useCaches = false
            setRequestProperty("Accept", "application/json")
            setRequestProperty("Authorization", "Bearer ${stored.token}")
            if (body != null) {
                doOutput = true
                setRequestProperty("Content-Type", "application/json; charset=utf-8")
            }
        }
        try {
            if (body != null) {
                connection.outputStream.use { it.write(body.toString().toByteArray(Charsets.UTF_8)) }
            }
            val status = connection.responseCode
            val requestId = connection.getHeaderField("X-Request-ID")
            val input = if (status in 200..299) connection.inputStream else connection.errorStream
            val text = try {
                input?.use { readUtf8Bounded(it) }.orEmpty()
            } catch (_: ResponseTooLargeException) {
                throw ApiException(status, "response_too_large", "server response exceeded the client safety limit", requestId)
            }
            if (status !in 200..299) {
                if (status == HttpURLConnection.HTTP_UNAUTHORIZED) sessions.clear()
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
                throw protocolError(requestId)
            }
            return parseCityContext(json, requestId)
        } finally {
            connection.disconnect()
        }
    }

    private fun parseCityContext(json: JSONObject, requestId: String?): CityContextModel {
        val localityJson = json.optJSONObject("locality") ?: throw protocolError(requestId)
        val id = canonicalUuid(localityJson.optString("id")) ?: throw protocolError(requestId)
        val name = localityJson.optString("name").trim()
        val countryCode = localityJson.optString("countryCode").trim()
        val timezone = localityJson.optString("timezone").trim()
        if (name.isEmpty() || countryCode.length != 2 || countryCode != countryCode.uppercase()) {
            throw protocolError(requestId)
        }
        if (runCatching { ZoneId.of(timezone) }.isFailure) throw protocolError(requestId)

        val permission = runCatching { CityPermissionClass.valueOf(json.getString("permissionClass")) }
            .getOrElse { throw protocolError(requestId) }
        val accuracyM = json.optInt("accuracyM", -1)
        if (accuracyM !in 1..10_000) throw protocolError(requestId)
        val observedAt = parseInstant(json, "observedAt", requestId)
        val expiresAt = parseInstant(json, "expiresAt", requestId)
        if (expiresAt <= observedAt) throw protocolError(requestId)

        return CityContextModel(
            locality = CityLocality(id, name, countryCode, timezone),
            permissionClass = permission,
            accuracyM = accuracyM,
            observedAtEpochMillis = observedAt,
            expiresAtEpochMillis = expiresAt,
            switchPending = json.optBoolean("switchPending", false),
        )
    }

    private fun parseInstant(json: JSONObject, key: String, requestId: String?): Long =
        runCatching { Instant.parse(json.getString(key)).toEpochMilli() }.getOrElse { throw protocolError(requestId) }

    private fun canonicalUuid(raw: String): String? = runCatching {
        val parsed = UUID.fromString(raw).toString()
        parsed.takeIf { it == raw.lowercase() }
    }.getOrNull()

    private fun protocolError(requestId: String?) = ApiException(
        status = HttpURLConnection.HTTP_OK,
        code = "protocol_error",
        message = "invalid city context response",
        requestId = requestId,
    )
}
