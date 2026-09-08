import SwiftUI

struct NotificationsPanelView: View {
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 14) {
                    Spacer(minLength: 36)
                    Image(systemName: "bell")
                        .font(.system(size: 24, weight: .semibold))
                        .foregroundStyle(LinkUpPalette.textMuted)
                        .frame(width: 64, height: 64)
                        .background(LinkUpPalette.zone)
                        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.card))
                        .overlay {
                            RoundedRectangle(cornerRadius: LinkUpRadius.card)
                                .stroke(LinkUpPalette.border)
                        }
                        .accessibilityHidden(true)

                    Text("No notification feed yet")
                        .font(LinkUpTypography.display(16, weight: .bold))
                        .foregroundStyle(LinkUpPalette.textPrimary)

                    Text("The current server contract does not expose an iOS notification feed. LinkUp will not fabricate notification rows or unread counts.")
                        .font(LinkUpTypography.body(13))
                        .foregroundStyle(LinkUpPalette.textDimmed)
                        .multilineTextAlignment(.center)
                        .frame(maxWidth: 280)

                    Spacer(minLength: 26)
                    Text("No engagement bait. Only events that matter.")
                        .font(LinkUpTypography.mono(10))
                        .foregroundStyle(LinkUpPalette.textMuted)
                        .multilineTextAlignment(.center)
                }
                .frame(maxWidth: .infinity)
                .padding(.horizontal, 20)
                .padding(.bottom, 20)
            }
            .scrollIndicators(.hidden)
            .background(LinkUpPalette.background)
            .navigationTitle("Notifications")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Done") { dismiss() }
                        .foregroundStyle(LinkUpPalette.red)
                }
            }
        }
        .preferredColorScheme(.dark)
    }
}
