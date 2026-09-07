import MapKit
import SwiftUI

@MainActor
struct MapView: View {
    @StateObject private var coordinator: MapCoordinator
    @State private var cameraPosition: MapCameraPosition
    @State private var currentRegion: MKCoordinateRegion
    @State private var selectedCluster: MapCluster?

    private static let initialRegion = MKCoordinateRegion(
        center: CLLocationCoordinate2D(latitude: 0, longitude: 0),
        span: MKCoordinateSpan(latitudeDelta: 120, longitudeDelta: 300)
    )

    init(api: LinkUpAPI, session: SessionCoordinator) {
        _coordinator = StateObject(wrappedValue: MapCoordinator(api: api, session: session))
        _cameraPosition = State(initialValue: .region(Self.initialRegion))
        _currentRegion = State(initialValue: Self.initialRegion)
    }

    var body: some View {
        ZStack {
            Map(position: $cameraPosition) {
                ForEach(coordinator.clusters) { cluster in
                    Annotation(
                        cluster.placeName ?? "\(cluster.slotCount) LinkUps",
                        coordinate: cluster.coordinate,
                        anchor: .bottom
                    ) {
                        clusterMarker(cluster)
                    }
                }
            }
            .mapStyle(.standard)
            .onMapCameraChange(frequency: .onEnd) { context in
                currentRegion = context.region
                Task { await coordinator.load(region: context.region) }
            }
            .task {
                if coordinator.phase == .idle {
                    await coordinator.load(region: currentRegion)
                }
            }
            .onDisappear { coordinator.dispose() }

            VStack(spacing: 0) {
                areaPill
                    .padding(.top, 12)
                Spacer()
                if let selectedCluster {
                    selectedClusterPanel(selectedCluster)
                        .padding(.horizontal, 20)
                        .padding(.bottom, 10)
                }
                summaryPanel
                    .padding(.horizontal, 20)
                    .padding(.bottom, 14)
            }

            HStack {
                Spacer()
                VStack(spacing: 8) {
                    Button {
                        cameraPosition = .automatic
                    } label: {
                        mapControl("scope")
                    }
                    .buttonStyle(.plain)
                    disabledControl("line.3.horizontal.decrease")
                    disabledControl("flame")
                }
                .padding(.trailing, 16)
                .padding(.top, 76)
            }
            .frame(maxHeight: .infinity, alignment: .top)

            if coordinator.phase == .loading {
                ProgressView()
                    .tint(LinkUpPalette.red)
                    .padding(14)
                    .background(.ultraThinMaterial)
                    .clipShape(Circle())
            }

            if case .failed(let message) = coordinator.phase {
                LinkUpErrorState(message: message) {
                    Task { await coordinator.load(region: currentRegion) }
                }
                .padding(24)
                .background(LinkUpPalette.background.opacity(0.94))
            }
        }
        .background(LinkUpPalette.background)
    }

    private func clusterMarker(_ cluster: MapCluster) -> some View {
        Button {
            selectedCluster = cluster
        } label: {
            VStack(spacing: 2) {
                ZStack {
                    Circle()
                        .fill(LinkUpPalette.red.opacity(0.18))
                        .frame(width: 42, height: 42)
                    Circle()
                        .stroke(LinkUpPalette.red, lineWidth: 2)
                        .frame(width: 42, height: 42)
                    Text("\(cluster.slotCount)")
                        .font(LinkUpTypography.mono(11, weight: .bold))
                        .foregroundStyle(LinkUpPalette.textPrimary)
                }
                if cluster.placeCount > 1 {
                    Text("\(cluster.placeCount) places")
                        .font(LinkUpTypography.mono(8, weight: .semibold))
                        .foregroundStyle(LinkUpPalette.textPrimary)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 3)
                        .background(LinkUpPalette.surface.opacity(0.9))
                        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.badge))
                }
            }
        }
        .buttonStyle(.plain)
        .accessibilityLabel("\(cluster.slotCount) LinkUps in this map cluster")
    }

    private var areaPill: some View {
        HStack(spacing: 8) {
            Circle().fill(LinkUpPalette.red).frame(width: 8, height: 8)
            Text("Server-authorized map")
                .font(LinkUpTypography.body(12, weight: .semibold))
                .foregroundStyle(LinkUpPalette.textPrimary)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
        .background(.ultraThinMaterial)
        .clipShape(Capsule())
        .overlay { Capsule().stroke(LinkUpPalette.border) }
    }

    private func selectedClusterPanel(_ cluster: MapCluster) -> some View {
        HStack(spacing: 12) {
            Image(systemName: cluster.placeId == nil ? "square.3.layers.3d" : "mappin.circle.fill")
                .font(.system(size: 20))
                .foregroundStyle(LinkUpPalette.red)
                .frame(width: 40, height: 40)
                .background(LinkUpPalette.red.opacity(0.12))
                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
            VStack(alignment: .leading, spacing: 3) {
                Text(cluster.placeName ?? "Map cluster")
                    .font(LinkUpTypography.display(14))
                    .foregroundStyle(LinkUpPalette.textPrimary)
                Text("\(cluster.slotCount) LinkUps · \(cluster.placeCount) places")
                    .font(LinkUpTypography.body(11))
                    .foregroundStyle(LinkUpPalette.textDimmed)
            }
            Spacer()
            Button {
                selectedCluster = nil
            } label: {
                Image(systemName: "xmark")
                    .font(.system(size: 12, weight: .bold))
                    .foregroundStyle(LinkUpPalette.textMuted)
                    .frame(width: 32, height: 32)
            }
            .buttonStyle(.plain)
        }
        .padding(12)
        .background(.ultraThinMaterial)
        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.card))
        .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.card).stroke(LinkUpPalette.border) }
    }

    private var summaryPanel: some View {
        HStack {
            VStack(alignment: .leading, spacing: 3) {
                HStack(spacing: 7) {
                    Circle()
                        .fill(coordinator.clusters.isEmpty ? LinkUpPalette.textMuted : LinkUpPalette.success)
                        .frame(width: 8, height: 8)
                    Text("MAP")
                        .font(LinkUpTypography.mono(11, weight: .semibold))
                        .foregroundStyle(coordinator.clusters.isEmpty ? LinkUpPalette.textMuted : LinkUpPalette.success)
                }
                Text(summaryText)
                    .font(LinkUpTypography.display(14))
                    .foregroundStyle(LinkUpPalette.textPrimary)
            }
            Spacer()
            Text("24h viewport")
                .font(LinkUpTypography.mono(10))
                .foregroundStyle(LinkUpPalette.textMuted)
        }
        .padding(14)
        .background(.ultraThinMaterial)
        .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.card))
        .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.card).stroke(LinkUpPalette.border) }
    }

    private var summaryText: String {
        let count = coordinator.clusters.reduce(0) { $0 + $1.slotCount }
        if coordinator.phase == .loading { return "Loading viewport…" }
        if count == 0 { return "No scheduled LinkUps in viewport" }
        return "\(count) LinkUps in viewport"
    }

    private func mapControl(_ symbol: String) -> some View {
        Image(systemName: symbol)
            .font(.system(size: 18))
            .foregroundStyle(LinkUpPalette.textDimmed)
            .frame(width: 40, height: 40)
            .background(.ultraThinMaterial)
            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
            .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
    }

    private func disabledControl(_ symbol: String) -> some View {
        mapControl(symbol)
            .opacity(0.55)
            .accessibilityHidden(true)
    }
}
