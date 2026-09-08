import SwiftUI

@MainActor
struct PasswordResetView: View {
    @Environment(\.dismiss) private var dismiss
    @ObservedObject var session: SessionCoordinator

    @State private var resetInput = ""
    @State private var routedToken: String?
    @State private var password = ""
    @State private var confirmation = ""
    @State private var isBusy = false
    @State private var errorMessage: String?
    @State private var completed = false

    init(session: SessionCoordinator, routedToken: String? = nil) {
        _session = ObservedObject(wrappedValue: session)
        let canonical = routedToken.flatMap { OpaqueTokenContract.canonical32ByteBase64URL($0) }
        _routedToken = State(initialValue: canonical)
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(routedToken == nil
                         ? "Paste the HTTPS reset link from your message or the reset code itself. The link is parsed locally and is never opened by the app."
                         : "The reset link was validated locally. The app will send only its reset token to the configured LinkUp API.")
                        .font(LinkUpTypography.body(13))
                        .foregroundStyle(LinkUpPalette.textDimmed)

                    if routedToken != nil {
                        messageBanner(
                            "Reset link verified. Enter a new password to continue.",
                            tint: LinkUpPalette.success,
                            symbol: "link"
                        )
                    } else {
                        TextField("Reset link or code", text: $resetInput, axis: .vertical)
                            .lineLimit(2...5)
                            .textInputAutocapitalization(.never)
                            .autocorrectionDisabled()
                            .font(LinkUpTypography.mono(11))
                            .padding(12)
                            .background(LinkUpPalette.elevated)
                            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                            .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
                    }

                    secureField("New password", text: $password)
                    secureField("Confirm new password", text: $confirmation)
                    Text("Password payload must be 8…1024 UTF-8 bytes.")
                        .font(LinkUpTypography.body(10))
                        .foregroundStyle(LinkUpPalette.textMuted)

                    if let errorMessage {
                        messageBanner(errorMessage, tint: LinkUpPalette.critical, symbol: "exclamationmark.triangle.fill")
                    }

                    if completed {
                        messageBanner(
                            "Password changed. You can now log in with the new password.",
                            tint: LinkUpPalette.success,
                            symbol: "checkmark.circle.fill"
                        )
                    }

                    LinkUpButton(
                        title: completed ? "Return to login" : (isBusy ? "Changing…" : "Change password"),
                        disabled: isBusy || (!completed && !canSubmit)
                    ) {
                        if completed { dismiss() } else { submit() }
                    }
                }
                .padding(20)
            }
            .background(LinkUpPalette.background)
            .navigationTitle("Reset password")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Cancel") { dismiss() }
                        .foregroundStyle(LinkUpPalette.textDimmed)
                        .disabled(isBusy)
                }
            }
        }
        .preferredColorScheme(.dark)
        .interactiveDismissDisabled(isBusy)
        .onDisappear { clearSensitiveFields() }
    }

    private var effectiveResetToken: String? {
        routedToken ?? passwordResetToken(from: resetInput)
    }

    private var canSubmit: Bool {
        effectiveResetToken != nil &&
            InputContracts.validPasswordPayload(password) &&
            password == confirmation
    }

    private func submit() {
        guard let token = effectiveResetToken, canSubmit, !isBusy else {
            errorMessage = "Enter a valid reset link/code and matching new passwords."
            return
        }
        isBusy = true
        errorMessage = nil
        Task {
            defer { isBusy = false }
            do {
                try await session.resetPassword(token: token, newPassword: password)
                clearSensitiveFields()
                completed = true
            } catch is CancellationError {
                return
            } catch {
                errorMessage = error.localizedDescription
            }
        }
    }

    private func secureField(_ placeholder: String, text: Binding<String>) -> some View {
        SecureField(placeholder, text: text)
            .textContentType(.newPassword)
            .font(LinkUpTypography.body(14))
            .padding(.horizontal, 14)
            .frame(height: 48)
            .background(LinkUpPalette.zone)
            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
            .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
    }

    private func messageBanner(_ text: String, tint: Color, symbol: String) -> some View {
        HStack(alignment: .top, spacing: 9) {
            Image(systemName: symbol).foregroundStyle(tint)
            Text(text)
                .font(LinkUpTypography.body(12))
                .foregroundStyle(LinkUpPalette.textDimmed)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding(12)
        .background(tint.opacity(0.09))
        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
    }

    private func clearSensitiveFields() {
        resetInput = ""
        routedToken = nil
        password = ""
        confirmation = ""
    }
}
