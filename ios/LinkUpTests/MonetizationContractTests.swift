import XCTest
@testable import LinkUp

final class MonetizationContractTests: XCTestCase {
    func testReferralCodeNormalizationMatchesBackendContract() {
        XCTAssertEqual(ReferralCodeContract.normalize("  ab12cd  "), "AB12CD")
        XCTAssertEqual(ReferralCodeContract.normalize("ABCDEF1234567890ABCD"), "ABCDEF1234567890ABCD")
        XCTAssertNil(ReferralCodeContract.normalize("ABCDE"))
        XCTAssertNil(ReferralCodeContract.normalize("ABC-123"))
        XCTAssertNil(ReferralCodeContract.normalize("АBC123"))
        XCTAssertNil(ReferralCodeContract.normalize("ABCDEF1234567890ABCDE"))
    }

    func testReferralInputFilterKeepsOnlyBackendSafeAlphabet() {
        XCTAssertEqual(ReferralCodeContract.filteredInput(" ab-12_cd "), "AB12CD")
        XCTAssertEqual(ReferralCodeContract.filteredInput("абвABC123"), "ABC123")
        XCTAssertEqual(ReferralCodeContract.filteredInput(String(repeating: "a", count: 30)), String(repeating: "A", count: 20))
    }

    func testMonetizationSnapshotDecodesServerDatesAndCapabilities() throws {
        let data = Data(#"{
          "catalog": {
            "currency": "UAH",
            "plans": [{
              "id": "monthly",
              "billingPeriod": "MONTH",
              "priceUahMinor": 14999,
              "effectiveMonthlyUahMinor": 14999,
              "total12MonthsUahMinor": 179988
            }],
            "annualSavingsUahMinor": 60000,
            "annualSavingsPercent": 33.3,
            "rewarded": {
              "videoIntervalSeconds": 14400,
              "videosRequired": 5,
              "nominalCompletionHours": 20,
              "rewardSeconds": 86400,
              "claimCooldownSeconds": 604800
            },
            "referralDeadlineDays": 14,
            "referralMilestones": [{
              "qualifiedReferrals": 1,
              "inviterRewardDays": 1,
              "inviteeRewardDays": 1,
              "badge": false
            }]
          },
          "status": {
            "premiumActive": true,
            "premiumUntil": "2026-09-10T12:00:00Z",
            "rewarded": {
              "videosWatchedCount": 2,
              "nextVideoAt": "2026-09-08T14:00:00Z"
            },
            "referral": {
              "referralCode": "ABCDEF12",
              "qualifiedReferrals": 0,
              "nextMilestone": {
                "qualifiedReferrals": 1,
                "inviterRewardDays": 1,
                "inviteeRewardDays": 1,
                "badge": false
              }
            }
          },
          "capabilities": {
            "paidVerification": false,
            "rewardedVerification": false,
            "referralQualification": false
          }
        }"#.utf8)

        let snapshot = try APICoding.decoder().decode(MonetizationSnapshot.self, from: data)
        XCTAssertTrue(snapshot.status.premiumActive)
        XCTAssertEqual(snapshot.catalog.currency, "UAH")
        XCTAssertEqual(snapshot.catalog.plans.first?.priceUahMinor, 14999)
        XCTAssertEqual(snapshot.status.rewarded.videosWatchedCount, 2)
        XCTAssertEqual(snapshot.status.referral.referralCode, "ABCDEF12")
        XCTAssertFalse(snapshot.capabilities.paidVerification)
        XCTAssertNotNil(snapshot.status.premiumUntil)
        XCTAssertNotNil(snapshot.status.rewarded.nextVideoAt)
    }
}
