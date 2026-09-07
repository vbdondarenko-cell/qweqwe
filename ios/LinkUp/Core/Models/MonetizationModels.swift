import Foundation

struct MonetizationPlan: Codable, Equatable, Sendable, Identifiable {
    let id: String
    let billingPeriod: String
    let priceUahMinor: Int
    let effectiveMonthlyUahMinor: Int
    let total12MonthsUahMinor: Int
}

struct RewardedPolicy: Codable, Equatable, Sendable {
    let videoIntervalSeconds: Int64
    let videosRequired: Int
    let nominalCompletionHours: Int
    let rewardSeconds: Int64
    let claimCooldownSeconds: Int64
}

struct ReferralMilestone: Codable, Equatable, Sendable {
    let qualifiedReferrals: Int
    let inviterRewardDays: Int
    let inviteeRewardDays: Int
    let badge: Bool
}

struct MonetizationCatalog: Codable, Equatable, Sendable {
    let currency: String
    let plans: [MonetizationPlan]
    let annualSavingsUahMinor: Int
    let annualSavingsPercent: Double
    let rewarded: RewardedPolicy
    let referralDeadlineDays: Int
    let referralMilestones: [ReferralMilestone]
}

struct RewardedStatus: Codable, Equatable, Sendable {
    let videosWatchedCount: Int
    let nextVideoAt: Date?
    let lastFreePremiumClaimedAt: Date?
    let nextFreePremiumClaimAt: Date?
}

struct ReferralStatus: Codable, Equatable, Sendable {
    let referralCode: String
    let boundReferralCode: String?
    let qualifyingDeadline: Date?
    let qualifiedReferrals: Int
    let nextMilestone: ReferralMilestone?
}

struct MonetizationStatus: Codable, Equatable, Sendable {
    let premiumActive: Bool
    let premiumUntil: Date?
    let rewarded: RewardedStatus
    let referral: ReferralStatus
}

struct MonetizationCapabilities: Codable, Equatable, Sendable {
    let paidVerification: Bool
    let rewardedVerification: Bool
    let referralQualification: Bool
}

struct MonetizationSnapshot: Codable, Equatable, Sendable {
    let catalog: MonetizationCatalog
    let status: MonetizationStatus
    let capabilities: MonetizationCapabilities
}

struct BindReferralBody: Encodable, Sendable {
    let code: String
}
