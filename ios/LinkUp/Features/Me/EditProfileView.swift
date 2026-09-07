import SwiftUI

@MainActor
struct EditProfileView: View {
    @Environment(\.dismiss) private var dismiss
    @ObservedObject var session: SessionCoordinator

    private let user: UserProfile

    @State private var displayName: String
    @State private var avatarUrl: String
    @State private var visibility: String
    @State private var language: String
    @State private var isSaving = false
    @State private var errorMessage: String?

    init(user: UserProfile, session: SessionCoordinator) {
        self.user = user
        self.session = session
        _displayName = State(initialValue: user.displayName)
        _avatarUrl = State(initialValue: user.avatarUrl ?? "")
        _visibility = State(initialValue: user.profileVisibility)
        _language = State(initialValue: user.language)
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 18) {
                    identityPreview
                    field("Display name", text: $displayName)
                    field("Avatar URL", text: $avatarUrl)

                    selectionSection(
                        title: "Profile visibility",
                        values: [("PUBLIC", "Public"), ("HIDDEN", "Hidden")],
                        selection: $visibility
                    )
                    selectionSection(
                        title: "Language",
                        values: [("uk", "Українська"), ("en", "English")],
                        selection: $language
                    )

                    if let errorMessage {
                        errorBanner(errorMessage)
                    }

                    LinkUpButton(
                        title: isSaving ? "Saving…" : "Save profile",
                        disabled: isSaving || !canSave
                    ) {
                        save()
                    }
                }
                .padding(20)
            }
            .background(LinkUpPalette.background)
            .navigationTitle("Edit profile")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Cancel") { dismiss() }
                        .foregroundStyle(LinkUpPalette.textDimmed)
                        .disabled(isSaving)
                }
            }
        }
        .preferredColorScheme(.dark)
        .interactiveDismissDisabled(isSaving)
    }

    private var identityPreview: some View {
        HStack(spacing: 14) {
            LinkUpAvatar(initials: initials, size: .lg)
            VStack(alignment: .leading, spacing: 3) {
                Text(displayName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? user.displayName : displayName)
                    .font(LinkUpTypography.display(17))
                    .foregroundStyle(LinkUpPalette.textPrimary)
                Text("@\(user.username)")
                    .font(LinkUpTypography.body(12))
                    .foregroundStyle(LinkUpPalette.textMuted)
            }
            Spacer()
        }
    }

    private var canSave: Bool {
        guard InputContracts.validProfileDisplayName(displayName) else { return false }
        guard InputContracts.validAvatarURLPayload(avatarUrl) else { return false }
        guard visibility == "PUBLIC" || visibility == "HIDDEN" else { return false }
        return language == "uk" || language == "en"
    }

    private func save() {
        guard canSave, !isSaving else { return }
        isSaving = true
        errorMessage = nil
        let normalizedName = displayName.trimmingCharacters(in: .whitespacesAndNewlines)
        let normalizedAvatar = avatarUrl.trimmingCharacters(in: .whitespacesAndNewlines)

        Task {
            defer { isSaving = false }
            do {
                try await session.updateProfile(
                    displayName: normalizedName,
                    avatarUrl: normalizedAvatar,
                    profileVisibility: visibility,
                    language: language
                )
                dismiss()
            } catch is CancellationError {
                return
            } catch {
                errorMessage = error.localizedDescription
            }
        }
    }

    private func field(_ label: String, text: Binding<String>) -> some View {
        let isAvatar = label == "Avatar URL"
        let count = isAvatar ? InputContracts.trimmed(text.wrappedValue).utf8.count : InputContracts.scalarCount(InputContracts.trimmed(text.wrappedValue))
        let limit = isAvatar ? InputContracts.avatarURLMaxUTF8Bytes : InputContracts.profileDisplayNameMaxScalars
        return VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text(label)
                    .font(LinkUpTypography.body(12, weight: .semibold))
                Spacer()
                Text("\(count)/\(limit)")
                    .font(LinkUpTypography.mono(9))
                    .foregroundStyle(count > limit ? LinkUpPalette.critical : LinkUpPalette.textMuted)
            }
            .foregroundStyle(LinkUpPalette.textDimmed)
            TextField(label, text: text)
                .textInputAutocapitalization(isAvatar ? .never : .words)
                .autocorrectionDisabled(isAvatar)
                .font(LinkUpTypography.body(14))
                .padding(.horizontal, 12)
                .frame(height: 46)
                .background(LinkUpPalette.elevated)
                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
        }
    }

    private func selectionSection(
        title: String,
        values: [(String, String)],
        selection: Binding<String>
    ) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(LinkUpTypography.body(12, weight: .semibold))
                .foregroundStyle(LinkUpPalette.textDimmed)
            HStack(spacing: 8) {
                ForEach(values, id: \.0) { value in
                    LinkUpChip(title: value.1, active: selection.wrappedValue == value.0) {
                        selection.wrappedValue = value.0
                    }
                }
            }
        }
    }

    private func errorBanner(_ message: String) -> some View {
        HStack(alignment: .top, spacing: 9) {
            Image(systemName: "exclamationmark.triangle.fill")
                .foregroundStyle(LinkUpPalette.critical)
            Text(message)
                .font(LinkUpTypography.body(12))
                .foregroundStyle(LinkUpPalette.textDimmed)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding(12)
        .background(LinkUpPalette.critical.opacity(0.09))
        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
    }

    private var initials: String {
        let name = displayName.trimmingCharacters(in: .whitespacesAndNewlines)
        let value = (name.isEmpty ? user.displayName : name)
            .split(separator: " ")
            .prefix(2)
            .compactMap(\.first)
            .map(String.init)
            .joined()
        return value.isEmpty ? "LU" : value.uppercased()
    }
}
