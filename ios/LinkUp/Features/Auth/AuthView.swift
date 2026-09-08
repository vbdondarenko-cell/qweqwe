import SwiftUI
import UIKit

@MainActor
struct AuthView: View {
    private enum Mode: String, CaseIterable {
        case login = "Log in"
        case register = "Register"
        case recovery = "Recover"
    }

    @ObservedObject var session: SessionCoordinator

    @State private var mode: Mode = .login
    @State private var identifier = ""
    @State private var email = ""
    @State private var username = ""
    @State private var displayName = ""
    @State private var password = ""
    @State private var confirmation = ""
    @State private var isBusy = false
    @State private var errorMessage: String?
    @State private var successMessage: String?
    @State private var showingReset = false

    var body: some View {
        ScrollView {
            VStack(spacing: 24) {
                brand
                modePicker
                form
            }
            .padding(.horizontal, 24)
            .padding(.top, 56)
            .padding(.bottom, 32)
        }
        .scrollDismissesKeyboard(.interactively)
        .background(LinkUpPalette.background.ignoresSafeArea())
        .foregroundStyle(LinkUpPalette.textPrimary)
        .sheet(isPresented: $showingReset) {
            PasswordResetView(session: session)
        }
    }

    private var brand: some View {
        VStack(spacing: 10) {
            ZStack {
                Circle().fill(LinkUpPalette.red).frame(width: 72, height: 72)
                Text("LINK")
                    .font(LinkUpTypography.display(17, weight: .black))
                    .foregroundStyle(.white)
            }
            Text("LinkUp")
                .font(LinkUpTypography.display(30, weight: .black))
            Text("Real people. Real plans. Right now.")
                .font(LinkUpTypography.body(14))
                .foregroundStyle(LinkUpPalette.textDimmed)
        }
    }

    private var modePicker: some View {
        HStack(spacing: 8) {
            ForEach(Mode.allCases, id: \.self) { item in
                LinkUpChip(title: item.rawValue, active: mode == item) {
                    guard !isBusy else { return }
                    mode = item
                    errorMessage = nil
                    successMessage = nil
                    password = ""
                    confirmation = ""
                }
            }
        }
    }

    @ViewBuilder private var form: some View {
        VStack(spacing: 14) {
            if mode == .login {
                authField("Email or username", text: $identifier, contentType: .username)
                secureField("Password", text: $password)
            } else if mode == .register {
                authField("Email", text: $email, contentType: .emailAddress)
                authField("Username", text: $username, contentType: .username)
                authField("Display name", text: $displayName, contentType: .name)
                secureField("Password", text: $password)
                secureField("Confirm password", text: $confirmation)
            } else {
                authField("Email", text: $email, contentType: .emailAddress)
                Text("We'll request a password-reset message from the LinkUp server. Account existence is never disclosed here.")
                    .font(LinkUpTypography.body(12))
                    .foregroundStyle(LinkUpPalette.textMuted)
                    .frame(maxWidth: .infinity, alignment: .leading)
                Button {
                    showingReset = true
                } label: {
                    HStack {
                        Image(systemName: "key.fill")
                        Text("I have a reset link or code")
                    }
                    .font(LinkUpTypography.body(12, weight: .semibold))
                    .foregroundStyle(LinkUpPalette.red)
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
                .buttonStyle(.plain)
            }

            if let errorMessage {
                messageCard(errorMessage, color: LinkUpPalette.critical, symbol: "exclamationmark.triangle.fill")
            }
            if let successMessage {
                messageCard(successMessage, color: LinkUpPalette.success, symbol: "checkmark.circle.fill")
            }

            LinkUpButton(title: actionTitle, disabled: isBusy || !canSubmit) { submit() }

            if isBusy {
                ProgressView().tint(LinkUpPalette.red)
            }
        }
        .padding(18)
        .background(LinkUpPalette.elevated)
        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.card))
        .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.card).stroke(LinkUpPalette.border) }
    }

    private var actionTitle: String {
        switch mode {
        case .login: "Log in"
        case .register: "Create account"
        case .recovery: "Send reset instructions"
        }
    }

    private var canSubmit: Bool {
        switch mode {
        case .login:
            InputContracts.validLoginIdentifierShape(identifier) &&
                InputContracts.validPasswordPayload(password)
        case .register:
            InputContracts.validAccountEmail(email) &&
                InputContracts.validAccountUsername(username) &&
                InputContracts.validProfileDisplayName(displayName) &&
                InputContracts.validPasswordPayload(password) &&
                password == confirmation
        case .recovery:
            InputContracts.validAccountEmail(email)
        }
    }

    private func submit() {
        guard canSubmit, !isBusy else { return }
        isBusy = true
        errorMessage = nil
        successMessage = nil

        Task {
            defer { isBusy = false }
            do {
                switch mode {
                case .login:
                    try await session.login(
                        identifier: identifier.trimmingCharacters(in: .whitespacesAndNewlines),
                        password: password,
                        deviceLabel: "LinkUp iOS"
                    )
                case .register:
                    let language = Locale.preferredLanguages.first?.lowercased().hasPrefix("uk") == true ? "uk" : "en"
                    try await session.register(
                        email: email.trimmingCharacters(in: .whitespacesAndNewlines),
                        username: username.trimmingCharacters(in: .whitespacesAndNewlines),
                        displayName: displayName.trimmingCharacters(in: .whitespacesAndNewlines),
                        password: password,
                        language: language,
                        deviceLabel: "LinkUp iOS"
                    )
                case .recovery:
                    try await session.requestPasswordRecovery(email: email.trimmingCharacters(in: .whitespacesAndNewlines))
                    successMessage = "If that account exists, reset instructions have been requested."
                }
            } catch is CancellationError {
                return
            } catch {
                errorMessage = error.localizedDescription
            }
        }
    }

    private func authField(
        _ placeholder: String,
        text: Binding<String>,
        contentType: UITextContentType
    ) -> some View {
        TextField(placeholder, text: text)
            .textContentType(contentType)
            .textInputAutocapitalization(.never)
            .autocorrectionDisabled()
            .font(LinkUpTypography.body(14))
            .padding(.horizontal, 14)
            .frame(height: 48)
            .background(LinkUpPalette.zone)
            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
            .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
    }

    private func secureField(_ placeholder: String, text: Binding<String>) -> some View {
        SecureField(placeholder, text: text)
            .textContentType(mode == .register ? .newPassword : .password)
            .font(LinkUpTypography.body(14))
            .padding(.horizontal, 14)
            .frame(height: 48)
            .background(LinkUpPalette.zone)
            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
            .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
    }

    private func messageCard(_ text: String, color: Color, symbol: String) -> some View {
        HStack(alignment: .top, spacing: 10) {
            Image(systemName: symbol).foregroundStyle(color)
            Text(text)
                .font(LinkUpTypography.body(12))
                .foregroundStyle(LinkUpPalette.textDimmed)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding(12)
        .background(color.opacity(0.09))
        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
        .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(color.opacity(0.3)) }
    }
}
