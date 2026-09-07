package com.linkup.app.core.network

import com.linkup.app.core.session.SecureSessionStore
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URL
import java.time.Instant
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import org.json.JSONObject

class MonetizationApiClient(
    baseUrl: String,
    private val sessions: SecureSessionStore,
) {
    private val root = validatedApiRoot(baseUrl)

    suspend fun snapshot(): MonetizationSnapshotModel = withContext(Dispatchers.IO) {
        var lastError: Exception? = null
        repeat(2) { attempt ->
            try {
                return@withContext requestOnce("GET", "/v1/me/monetization", null)
            } catch (error: CancellationException) {
                throw error
            } catch (error: Exception) {
                lastError = error
                val retryable = when (error) {
                    is ApiException -> error.status in setOf(408, 429, 502, 503, 504)
                    is IOException -> true
                    else -> false
                }
                if (attempt == 1 || !retryable) throw error
                delay(250L)
            }
        }
        throw lastError ?: IOException("request failed")
    }

    suspend fun bindReferral(code: String): MonetizationSnapshotModel = withContext(Dispatchers.IO) {
        requestOnce(
            method = "POST",
            path = "/v1/me/referral",
            body = JSONObject().put("code", code),
        )
    }

    private fun requestOnce(method: String, path: String, body: JSONObject?): MonetizationSnapshotModel {
        val stored = sessions.load() ?: throw ApiException(401, "unauthorized", "authentication required")
        val connection = (URL("$root$path").openConnection() as HttpURLConnection).apply {
            requestMethod = method
            connectTimeout = 10_000
            readTimeout = 15_000
            setRequestProperty("Accept", "application/json")
            setRequestProperty("Authorization", "Bearer ${stored.token}")
            useCaches = false
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
            val input = if (status in 200..299) connection.inputStream else connection.errorStream
            val text = input?.use { readUtf8Bounded(it) }.orEmpty()
            if (status !in 200..299) {
                if (status == 401) sessions.clear()
                val problem = runCatching { JSONObject(text) }.getOrNull()
                throw ApiException(
                    status = status,
                    code = problem?.optString("code")?.takeIf { it.isNotBlank() } ?: "request_failed",
                    message = problem?.optString("message")?.takeIf { it.isNotBlank() } ?: "request failed",
                    requestId = problem?.optString("requestId")?.takeIf { it.isNotBlank() },
                )
            }
            return parseSnapshot(JSONObject(text))
        } finally {
            connection.disconnect()
        }
    }

    private fun parseSnapshot(json: JSONObject): MonetizationSnapshotModel {
        val catalogJson = json.getJSONObject("catalog")
        val plansJson = catalogJson.getJSONArray("plans")
        val plans = buildList {
            for (index in 0 until plansJson.length()) {
                val item = plansJson.getJSONObject(index)
                add(MonetizationPlan(
                    id = item.getString("id"),
                    billingPeriod = item.getString("billingPeriod"),
                    priceUahMinor = item.getInt("priceUahMinor"),
                    effectiveMonthlyUahMinor = item.getInt("effectiveMonthlyUahMinor"),
                    total12MonthsUahMinor = item.getInt("total12MonthsUahMinor"),
                ))
            }
        }
        val rewardedJson = catalogJson.getJSONObject("rewarded")
        val milestonesJson = catalogJson.getJSONArray("referralMilestones")
        val milestones = buildList {
            for (index in 0 until milestonesJson.length()) add(parseMilestone(milestonesJson.getJSONObject(index)))
        }
        val statusJson = json.getJSONObject("status")
        val rewardedStatus = statusJson.getJSONObject("rewarded")
        val referralStatus = statusJson.getJSONObject("referral")
        val capabilities = json.getJSONObject("capabilities")
        return MonetizationSnapshotModel(
            catalog = MonetizationCatalog(
                currency = catalogJson.getString("currency"),
                plans = plans,
                annualSavingsUahMinor = catalogJson.getInt("annualSavingsUahMinor"),
                annualSavingsPercent = catalogJson.getDouble("annualSavingsPercent"),
                rewarded = RewardedPolicyModel(
                    videoIntervalSeconds = rewardedJson.getLong("videoIntervalSeconds"),
                    videosRequired = rewardedJson.getInt("videosRequired"),
                    nominalCompletionHours = rewardedJson.getInt("nominalCompletionHours"),
                    rewardSeconds = rewardedJson.getLong("rewardSeconds"),
                    claimCooldownSeconds = rewardedJson.getLong("claimCooldownSeconds"),
                ),
                referralDeadlineDays = catalogJson.getInt("referralDeadlineDays"),
                referralMilestones = milestones,
            ),
            status = MonetizationStatusModel(
                premiumActive = statusJson.getBoolean("premiumActive"),
                premiumUntilEpochMillis = optionalInstant(statusJson, "premiumUntil"),
                rewarded = RewardedProgressModel(
                    videosWatchedCount = rewardedStatus.getInt("videosWatchedCount"),
                    nextVideoAtEpochMillis = optionalInstant(rewardedStatus, "nextVideoAt"),
                    lastFreePremiumClaimedAtEpochMillis = optionalInstant(rewardedStatus, "lastFreePremiumClaimedAt"),
                    nextFreePremiumClaimAtEpochMillis = optionalInstant(rewardedStatus, "nextFreePremiumClaimAt"),
                ),
                referral = ReferralStatusModel(
                    referralCode = referralStatus.getString("referralCode"),
                    boundReferralCode = optionalString(referralStatus, "boundReferralCode"),
                    qualifyingDeadlineEpochMillis = optionalInstant(referralStatus, "qualifyingDeadline"),
                    qualifiedReferrals = referralStatus.getInt("qualifiedReferrals"),
                    nextMilestone = if (referralStatus.has("nextMilestone") && !referralStatus.isNull("nextMilestone")) {
                        parseMilestone(referralStatus.getJSONObject("nextMilestone"))
                    } else null,
                ),
            ),
            capabilities = MonetizationCapabilities(
                paidVerification = capabilities.getBoolean("paidVerification"),
                rewardedVerification = capabilities.getBoolean("rewardedVerification"),
                referralQualification = capabilities.getBoolean("referralQualification"),
            ),
        )
    }

    private fun parseMilestone(json: JSONObject): ReferralMilestoneModel = ReferralMilestoneModel(
        qualifiedReferrals = json.getInt("qualifiedReferrals"),
        inviterRewardDays = json.getInt("inviterRewardDays"),
        inviteeRewardDays = json.getInt("inviteeRewardDays"),
        badge = json.optBoolean("badge", false),
    )

    private fun optionalString(json: JSONObject, key: String): String? =
        if (!json.has(key) || json.isNull(key)) null else json.getString(key)

    private fun optionalInstant(json: JSONObject, key: String): Long? =
        optionalString(json, key)?.let { Instant.parse(it).toEpochMilli() }
}
