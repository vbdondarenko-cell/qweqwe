package com.linkup.app.core.network

data class MonetizationPlan(
    val id: String,
    val billingPeriod: String,
    val priceUahMinor: Int,
    val effectiveMonthlyUahMinor: Int,
    val total12MonthsUahMinor: Int,
)

data class RewardedPolicyModel(
    val videoIntervalSeconds: Long,
    val videosRequired: Int,
    val nominalCompletionHours: Int,
    val rewardSeconds: Long,
    val claimCooldownSeconds: Long,
)

data class ReferralMilestoneModel(
    val qualifiedReferrals: Int,
    val inviterRewardDays: Int,
    val inviteeRewardDays: Int,
    val badge: Boolean,
)

data class MonetizationCatalog(
    val currency: String,
    val plans: List<MonetizationPlan>,
    val annualSavingsUahMinor: Int,
    val annualSavingsPercent: Double,
    val rewarded: RewardedPolicyModel,
    val referralDeadlineDays: Int,
    val referralMilestones: List<ReferralMilestoneModel>,
)

data class RewardedProgressModel(
    val videosWatchedCount: Int,
    val nextVideoAtEpochMillis: Long?,
    val lastFreePremiumClaimedAtEpochMillis: Long?,
    val nextFreePremiumClaimAtEpochMillis: Long?,
)

data class ReferralStatusModel(
    val qualifiedReferrals: Int,
    val nextMilestone: ReferralMilestoneModel?,
)

data class MonetizationStatusModel(
    val premiumActive: Boolean,
    val premiumUntilEpochMillis: Long?,
    val rewarded: RewardedProgressModel,
    val referral: ReferralStatusModel,
)

data class MonetizationCapabilities(
    val paidVerification: Boolean,
    val rewardedVerification: Boolean,
    val referralQualification: Boolean,
)

data class MonetizationSnapshotModel(
    val catalog: MonetizationCatalog,
    val status: MonetizationStatusModel,
    val capabilities: MonetizationCapabilities,
)
