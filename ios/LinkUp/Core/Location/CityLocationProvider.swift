@preconcurrency import CoreLocation
import Foundation

private enum CityLocationProviderError: Error, LocalizedError {
    case busy
    case permissionDenied
    case unavailable
    case invalidObservation
    case simulated

    var errorDescription: String? {
        switch self {
        case .busy: "A location request is already in progress."
        case .permissionDenied: "Location permission is required to set your city."
        case .unavailable: "A current location could not be obtained."
        case .invalidObservation: "The current location is not accurate enough for city resolution."
        case .simulated: "Simulated locations cannot be used for City Context."
        }
    }
}

@MainActor
final class CityLocationProvider: NSObject, CLLocationManagerDelegate {
    private let manager = CLLocationManager()
    private var continuation: CheckedContinuation<CityLocationObservation, Error>?

    override init() {
        super.init()
        manager.delegate = self
        manager.desiredAccuracy = kCLLocationAccuracyHundredMeters
        manager.distanceFilter = kCLDistanceFilterNone
    }

    func currentObservation() async throws -> CityLocationObservation {
        guard continuation == nil else { throw CityLocationProviderError.busy }
        return try await withTaskCancellationHandler(operation: {
            try await withCheckedThrowingContinuation { continuation in
                self.continuation = continuation
                continueAuthorizationFlow()
            }
        }, onCancel: {
            Task { @MainActor [weak self] in
                self?.finish(.failure(CancellationError()))
            }
        })
    }

    func permissionClass() -> CityPermissionClass? {
        switch manager.authorizationStatus {
        case .authorizedAlways, .authorizedWhenInUse:
            manager.accuracyAuthorization == .fullAccuracy ? .precise : .approximate
        default:
            nil
        }
    }

    func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        guard continuation != nil else { return }
        continueAuthorizationFlow()
    }

    func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        guard continuation != nil, let location = locations.last else { return }
        do {
            finish(.success(try observation(from: location)))
        } catch {
            finish(.failure(error))
        }
    }

    func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {
        guard continuation != nil else { return }
        finish(.failure(CityLocationProviderError.unavailable))
    }

    private func continueAuthorizationFlow() {
        switch manager.authorizationStatus {
        case .notDetermined:
            manager.requestWhenInUseAuthorization()
        case .authorizedAlways, .authorizedWhenInUse:
            manager.requestLocation()
        case .denied, .restricted:
            finish(.failure(CityLocationProviderError.permissionDenied))
        @unknown default:
            finish(.failure(CityLocationProviderError.permissionDenied))
        }
    }

    private func observation(from location: CLLocation) throws -> CityLocationObservation {
        guard CLLocationCoordinate2DIsValid(location.coordinate),
              location.horizontalAccuracy.isFinite,
              location.horizontalAccuracy > 0,
              location.timestamp.timeIntervalSince1970 > 0 else {
            throw CityLocationProviderError.invalidObservation
        }
        if location.sourceInformation?.isSimulatedBySoftware == true {
            throw CityLocationProviderError.simulated
        }
        let accuracy = Int(location.horizontalAccuracy.rounded(.up))
        guard (1...10_000).contains(accuracy), let permission = permissionClass() else {
            throw CityLocationProviderError.invalidObservation
        }
        return CityLocationObservation(
            latitudeE6: Int((location.coordinate.latitude * 1_000_000).rounded()),
            longitudeE6: Int((location.coordinate.longitude * 1_000_000).rounded()),
            accuracyM: accuracy,
            permissionClass: permission,
            capturedAt: location.timestamp,
            mocked: false
        )
    }

    private func finish(_ result: Result<CityLocationObservation, Error>) {
        guard let continuation else { return }
        self.continuation = nil
        manager.stopUpdatingLocation()
        continuation.resume(with: result)
    }
}
