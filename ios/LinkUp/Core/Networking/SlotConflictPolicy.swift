import Foundation

extension APIError {
    var isSlotVersionConflict: Bool {
        guard case .http(let status, let code, _, _) = self else { return false }
        return status == 409 && code == "slot_version_conflict"
    }
}
