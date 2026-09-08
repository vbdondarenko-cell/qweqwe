import Foundation

struct CityTimeScope: Equatable {
    let timeZone: TimeZone

    init?(identifier: String) {
        guard let value = TimeZone(identifier: identifier) else { return nil }
        timeZone = value
    }

    var identifier: String { timeZone.identifier }

    func calendar() -> Calendar {
        var value = Calendar(identifier: .gregorian)
        value.timeZone = timeZone
        return value
    }

    func isToday(_ date: Date, relativeTo now: Date = Date()) -> Bool {
        calendar().isDate(date, inSameDayAs: now)
    }

    func isTomorrow(_ date: Date, relativeTo now: Date = Date()) -> Bool {
        let value = calendar()
        guard let tomorrow = value.date(byAdding: .day, value: 1, to: value.startOfDay(for: now)) else {
            return false
        }
        return value.isDate(date, inSameDayAs: tomorrow)
    }

    func localHour(for date: Date) -> Int {
        calendar().component(.hour, from: date)
    }

    func shortTimeString(for date: Date) -> String {
        let formatter = DateFormatter()
        formatter.locale = .autoupdatingCurrent
        formatter.timeZone = timeZone
        formatter.dateStyle = .none
        formatter.timeStyle = .short
        let abbreviation = timeZone.abbreviation(for: date) ?? identifier
        return "\(formatter.string(from: date)) · \(abbreviation)"
    }

    func displayString(for date: Date) -> String {
        let formatter = DateFormatter()
        formatter.locale = .autoupdatingCurrent
        formatter.timeZone = timeZone
        formatter.dateStyle = .medium
        formatter.timeStyle = .short
        let abbreviation = timeZone.abbreviation(for: date) ?? identifier
        return "\(formatter.string(from: date)) · \(abbreviation)"
    }
}

extension CityLocality {
    var timeScope: CityTimeScope? { CityTimeScope(identifier: timezone) }
}

extension CityContextModel {
    var timeScope: CityTimeScope? { locality.timeScope }
}
