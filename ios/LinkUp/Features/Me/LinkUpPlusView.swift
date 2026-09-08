import SwiftUI

@MainActor
struct LinkUpPlusView: View {
    @Environment(\.dismiss) private var dismiss
    @ObservedObject var coordinator: MeCoordinator

    @State private var referralCode = ""
    @State private var referralMessage: String?

    var body: some View {
        NavigationStack {
            Group {
                switch coordinator.monetizationPhase {
                case .idle, .loading:
                    VStack(spacing: 12) {
                        ProgressView().tint(LinkUpPalette.red)
                        Text("Loading LinkUp+…")
                            .font(LinkUpTypography.body(12))
                            .foregroundStyle(LinkUpPalette.textDimmed)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                case .failed(let message):
                    LinkUpErrorState(message: message) {
                        Task { await coordinator.loadMonetization() }
                    }
                    .padding(20)
                case .content:
                    if let snapshot = coordinator.monetization {
                        content(snapshot)
                    } else {
                        LinkUpErrorState(message: "LinkUp+ state is unavailable.") {
                            Task { await coordinator.loadMonetization() }
                        }
                        .padding(20)
                    }
                }
            }
            .background(LinkUpPalette.background.ignoresSafeArea())
            .navigationTitle("LinkUp+")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Done") { dismiss() }
                        .foregroundStyle(LinkUpPalette.red)
                }
            }
        }
        .preferredColorScheme(.dark)
        .task {
            if coordinator.monetizationPhase == .idle {
                await coordinator.loadMonetization()
            }
        }
    }

    private func content(_ snapshot: MonetizationSnapshot) -> some View {
        ScrollView {
            VStack(spacing: 16) {
                premiumStatus(snapshot)
                plans(snapshot)
                rewarded(snapshot)
                referral(snapshot)
                capabilities(snapshot)
            }
            .padding(20)
        }
        .scrollIndicators(.hidden)
        .refreshable { await coordinator.loadMonetization() }
    }

    private func premiumStatus(_ snapshot: MonetizationSnapshot) -> some View {
        LinkUpCard {
            HStack(spacing: 12) {
                Image(systemName: snapshot.status.premiumActive ? "checkmark.seal.fill" : "seal")
                    .font(.system(size: 24))
                    .foregroundStyle(snapshot.status.premiumActive ? LinkUpPalette.success : LinkUpPalette.textMuted)
                    .frame(width: 48, height: 48)
                    .background(LinkUpPalette.zone)
                    .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                VStack(alignment: .leading, spacing: 3) {
                    Text(snapshot.status.premiumActive ? "LinkUp+ active" : "LinkUp+ inactive")
                        .font(LinkUpTypography.display(16))
                        .foregroundStyle(LinkUpPalette.textPrimary)
                    if let until = snapshot.status.premiumUntil {
                        Text("Until \(until.formatted(date: .abbreviated, time: .shortened))")
                            .font(LinkUpTypography.body(11))
                            .foregroundStyle(LinkUpPalette.textDimmed)
                    } else {
                        Text("Server entitlement is not active.")
                            .font(LinkUpTypography.body(11))
                            .foregroundStyle(LinkUpPalette.textMuted)
                    }
                }
                Spacer()
            }
        }
    }

    private func plans(_ snapshot: MonetizationSnapshot) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            sectionTitle("Plans")
            ForEach(snapshot.catalog.plans) { plan in
                LinkUpCard {
                    HStack {
                        VStack(alignment: .leading, spacing: 3) {
                            Text(plan.billingPeriod == "YEAR" ? "Annual" : "Monthly")
                                .font(LinkUpTypography.body(14, weight: .semibold))
                                .foregroundStyle(LinkUpPalette.textPrimary)
                            Text(price(plan.priceUahMinor, currency: snapshot.catalog.currency))
                                .font(LinkUpTypography.mono(13, weight: .bold))
                                .foregroundStyle(LinkUpPalette.red)
                            if plan.billingPeriod == "YEAR" {
                                Text("Effective \(price(plan.effectiveMonthlyUahMinor, currency: snapshot.catalog.currency))/month")
                                    .font(LinkUpTypography.body(10))
                                    .foregroundStyle(LinkUpPalette.textMuted)
                            }
                        }
                        Spacer()
                        Image(systemName: "lock.fill")
                            .foregroundStyle(LinkUpPalette.textMuted)
                    }
                }
            }
            Text("Purchases are not enabled in the native iOS client until App Store server verification is available.")
                .font(LinkUpTypography.body(10))
                .foregroundStyle(LinkUpPalette.textMuted)
        }
    }

    private func rewarded(_ snapshot: MonetizationSnapshot) -> some View {
        let status = snapshot.status.rewarded
        let policy = snapshot.catalog.rewarded
        return VStack(alignment: .leading, spacing: 8) {
            sectionTitle("Rewarded access")
            LinkUpCard {
                VStack(alignment: .leading, spacing: 10) {
                    HStack {
                        Text("Progress")
                            .font(LinkUpTypography.body(13, weight: .semibold))
                        Spacer()
                        Text("\(status.videosWatchedCount)/\(policy.videosRequired)")
                            .font(LinkUpTypography.mono(12, weight: .bold))
                    }
                    LinkUpProgress(
                        value: min(status.videosWatchedCount, policy.videosRequired),
                        maximum: max(1, policy.videosRequired),
                        showsLabel: false
                    )
                    if let next = status.nextVideoAt {
                        Text("Next server-authorized video: \(next.formatted(date: .abbreviated, time: .shortened))")
                            .font(LinkUpTypography.body(10))
                            .foregroundStyle(LinkUpPalette.textMuted)
                    }
                    if let nextClaim = status.nextFreePremiumClaimAt {
                        Text("Next claim window: \(nextClaim.formatted(date: .abbreviated, time: .shortened))")
                            .font(LinkUpTypography.body(10))
                            .foregroundStyle(LinkUpPalette.textMuted)
                    }
                    if !snapshot.capabilities.rewardedVerification {
                        Text("Rewarded verification is currently disabled by the server.")
                            .font(LinkUpTypography.body(10, weight: .semibold))
                            .foregroundStyle(LinkUpPalette.warning)
                    }
                }
                .foregroundStyle(LinkUpPalette.textPrimary)
            }
        }
    }

    private func referral(_ snapshot: MonetizationSnapshot) -> some View {
        let status = snapshot.status.referral
        return VStack(alignment: .leading, spacing: 8) {
            sectionTitle("Referrals")
            LinkUpCard {
                VStack(alignment: .leading, spacing: 10) {
                    HStack {
                        VStack(alignment: .leading, spacing: 2) {
                            Text("Your code")
                                .font(LinkUpTypography.body(11))
                                .foregroundStyle(LinkUpPalette.textMuted)
                            Text(status.referralCode)
                                .font(LinkUpTypography.mono(17, weight: .bold))
                                .foregroundStyle(LinkUpPalette.textPrimary)
                                .textSelection(.enabled)
                        }
                        Spacer()
                        ShareLink(item: status.referralCode) {
                            Image(systemName: "square.and.arrow.up")
                                .foregroundStyle(LinkUpPalette.red)
                                .frame(width: 44, height: 44)
                                .background(LinkUpPalette.red.opacity(0.1))
                                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                        }
                        .accessibilityLabel(L10n.text("Share referral code"))
                    }

                    HStack {
                        Text("Qualified referrals")
                            .font(LinkUpTypography.body(11))
                        Spacer()
                        Text("\(status.qualifiedReferrals)")
                            .font(LinkUpTypography.mono(12, weight: .bold))
                    }
                    .foregroundStyle(LinkUpPalette.textDimmed)

                    if let milestone = status.nextMilestone {
                        Text("Next milestone: \(milestone.qualifiedReferrals) qualified · +\(milestone.inviterRewardDays)d")
                            .font(LinkUpTypography.body(10))
                            .foregroundStyle(LinkUpPalette.textMuted)
                    }

                    if let bound = status.boundReferralCode {
                        HStack {
                            Text("Bound code")
                            Spacer()
                            Text(bound).font(LinkUpTypography.mono(11, weight: .semibold))
                        }
                        .font(LinkUpTypography.body(11))
                        .foregroundStyle(LinkUpPalette.success)
                    } else {
                        referralBinding(snapshot)
                    }

                    if let deadline = status.qualifyingDeadline {
                        Text("Qualification deadline: \(deadline.formatted(date: .abbreviated, time: .shortened))")
                            .font(LinkUpTypography.body(10))
                            .foregroundStyle(LinkUpPalette.textMuted)
                    }

                    if !snapshot.capabilities.referralQualification {
                        Text("Referral qualification is currently disabled by the server; binding a code does not fabricate a reward.")
                            .font(LinkUpTypography.body(10, weight: .semibold))
                            .foregroundStyle(LinkUpPalette.warning)
                    }
                }
            }
        }
    }

    private func referralBinding(_ snapshot: MonetizationSnapshot) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            TextField("Referral code", text: $referralCode)
                .textInputAutocapitalization(.characters)
                .autocorrectionDisabled()
                .font(LinkUpTypography.mono(13))
                .padding(12)
                .background(LinkUpPalette.zone)
                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
                .onChange(of: referralCode) { _, value in
                    referralCode = ReferralCodeContract.filteredInput(value)
                    referralMessage = nil
                    coordinator.clearMutationError()
                }

            LinkUpButton(
                title: coordinator.isMutating ? "Binding…" : "Bind referral code",
                disabled: coordinator.isMutating || referralCode.count < 6
            ) {
                Task {
                    let success = await coordinator.bindReferral(referralCode)
                    if success {
                        referralCode = ""
                        referralMessage = "Referral code bound by the server."
                    }
                }
            }

            if let message = referralMessage {
                Text(message)
                    .font(LinkUpTypography.body(10, weight: .semibold))
                    .foregroundStyle(LinkUpPalette.success)
            }
            if let error = coordinator.mutationError {
                Text(error)
                    .font(LinkUpTypography.body(10))
                    .foregroundStyle(LinkUpPalette.critical)
            }
        }
    }

    private func capabilities(_ snapshot: MonetizationSnapshot) -> some View {
        LinkUpCard {
            VStack(alignment: .leading, spacing: 8) {
                Text("Server capabilities")
                    .font(LinkUpTypography.body(12, weight: .semibold))
                capability("Paid verification", snapshot.capabilities.paidVerification)
                capability("Rewarded verification", snapshot.capabilities.rewardedVerification)
                capability("Referral qualification", snapshot.capabilities.referralQualification)
            }
            .foregroundStyle(LinkUpPalette.textPrimary)
        }
    }

    private func capability(_ title: String, _ enabled: Bool) -> some View {
        HStack {
            Text(title).font(LinkUpTypography.body(11))
            Spacer()
            Text(enabled ? "READY" : "OFF")
                .font(LinkUpTypography.mono(9, weight: .semibold))
                .foregroundStyle(enabled ? LinkUpPalette.success : LinkUpPalette.textMuted)
        }
    }

    private func sectionTitle(_ title: String) -> some View {
        Text(title.uppercased())
            .font(LinkUpTypography.mono(10, weight: .semibold))
            .foregroundStyle(LinkUpPalette.textMuted)
            .padding(.horizontal, 4)
    }

    private func price(_ minor: Int, currency: String) -> String {
        (Double(minor) / 100).formatted(.currency(code: currency))
    }
}
