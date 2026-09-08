import MapKit
import SwiftUI

@MainActor
struct MapView: View {
    @StateObject private var coordinator: MapCoordinator
    @ObservedObject private var social: SocialCoordinator
    @ObservedObject private var cityContext: CityContextCoordinator
    @State private var cameraPosition: MapCameraPosition
    @State private var currentRegion: MKCoordinateRegion
    @State private var selectedCluster: MapCluster?
    @State private var showingPlaceSlots = false
    @State private var lastFocusedLocalityID: UUID?

    private let api: LinkUpAPI
    private let session: SessionCoordinator

    private static let initialRegion = MKCoordinateRegion(
        center: CLLocationCoordinate2D(latitude: 0, longitude: 0),
        span: MKCoordinateSpan(latitudeDelta: 120, longitudeDelta: 300)
    )

    init(
        api: LinkUpAPI,
        session: SessionCoordinator,
        social: SocialCoordinator,
        cityContext: CityContextCoordinator
    ) {
        self.api = api
        self.session = session
        _social = ObservedObject(wrappedValue: social)
        _cityContext = ObservedObject(wrappedValue: cityContext)
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
                selectedCluster = nil
                showingPlaceSlots = false
                Task { await coordinator.load(region: context.region) }
            }
            .task {
                if let context = cityContext.context { await focus(on: context) }
            }
            .onChange(of: cityContext.context?.locality.id) { _, _ in
                guard let context = cityContext.context else { return }
                Task { await focus(on: context) }
            }
            .onDisappear { coordinator.dispose() }
            .onChange(of: social.discoveryRevision) { _, _ in
                guard cityContext.context != nil else { return }
                Task {
                    await coordinator.load(region: currentRegion)
                    if showingPlaceSlots, let placeID = selectedCluster?.placeId {
                        await coordinator.refreshPlaceSlots(placeID: placeID)
                    }
                }
            }

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
                        Task { await resolveCityAndFocus() }
                    } label: {
                        if cityContext.isRefreshing || cityContext.phase == .loading {
                            ProgressView()
                                .tint(LinkUpPalette.red)
                                .frame(width: 40, height: 40)
                                .background(.ultraThinMaterial)
                                .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
                                .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
                        } else {
                            mapControl("scope")
                        }
                    }
                    .buttonStyle(.plain)
                    .disabled(cityContext.isRefreshing || cityContext.phase == .loading)
                    .accessibilityLabel(cityContext.context == nil ? "Set city context" : "Refresh and center city context")
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
        .sheet(isPresented: $showingPlaceSlots, onDismiss: {
            coordinator.clearPlaceSlots()
        }) {
            if let cluster = selectedCluster, let placeID = cluster.placeId {
                MapPlaceSlotsView(
                    cluster: cluster,
                    placeID: placeID,
                    mapCoordinator: coordinator,
                    social: social,
                    cityContext: cityContext,
                    api: api,
                    session: session
                )
            }
        }
    }


    private func resolveCityAndFocus() async {
        let previousLocalityID = cityContext.context?.locality.id
        await cityContext.resolveFromDevice()
        guard let context = cityContext.context else { return }
        if context.locality.id == previousLocalityID {
            await focus(on: context, force: true)
        }
    }

    private func focus(on context: CityContextModel, force: Bool = false) async {
        guard force || lastFocusedLocalityID != context.locality.id else { return }
        lastFocusedLocalityID = context.locality.id
        let region = MKCoordinateRegion(
            center: CLLocationCoordinate2D(
                latitude: context.locality.latitude,
                longitude: context.locality.longitude
            ),
            latitudinalMeters: 30_000,
            longitudinalMeters: 30_000
        )
        selectedCluster = nil
        showingPlaceSlots = false
        currentRegion = region
        cameraPosition = .region(region)
        await coordinator.load(region: region)
    }

    private func clusterMarker(_ cluster: MapCluster) -> some View {
        Button {
            select(cluster)
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

    private func select(_ cluster: MapCluster) {
        selectedCluster = cluster
        coordinator.clearPlaceSlots()
        guard let placeID = cluster.placeId else { return }
        showingPlaceSlots = true
        Task { await coordinator.loadPlaceSlots(placeID: placeID) }
    }

    private var areaPill: some View {
        HStack(spacing: 8) {
            Circle().fill(LinkUpPalette.red).frame(width: 8, height: 8)
            Text(cityContext.context.map { "\($0.locality.name) · server city lock" } ?? "Server-authorized map")
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
        VStack(spacing: 10) {
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
                    showingPlaceSlots = false
                    coordinator.clearPlaceSlots()
                } label: {
                    Image(systemName: "xmark")
                        .font(.system(size: 12, weight: .bold))
                        .foregroundStyle(LinkUpPalette.textMuted)
                        .frame(width: 32, height: 32)
                }
                .buttonStyle(.plain)
            }

            if let placeID = cluster.placeId {
                Button {
                    showingPlaceSlots = true
                    Task { await coordinator.loadPlaceSlots(placeID: placeID) }
                } label: {
                    HStack {
                        Text("View LinkUps at this place")
                        Spacer()
                        Image(systemName: "chevron.right")
                    }
                    .font(LinkUpTypography.body(12, weight: .semibold))
                    .foregroundStyle(LinkUpPalette.red)
                }
                .buttonStyle(.plain)
            } else {
                Text("Zoom in to resolve this aggregate into a specific place.")
                    .font(LinkUpTypography.body(11))
                    .foregroundStyle(LinkUpPalette.textMuted)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }
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
        if cityContext.context == nil {
            if cityContext.phase == .loading { return "Loading city context…" }
            return "Set city context to load nearby LinkUps"
        }
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

@MainActor
private struct MapPlaceSlotsView: View {
    @Environment(\.dismiss) private var dismiss

    let cluster: MapCluster
    let placeID: UUID
    let api: LinkUpAPI
    let session: SessionCoordinator

    @ObservedObject var mapCoordinator: MapCoordinator
    @ObservedObject var social: SocialCoordinator
    @ObservedObject var cityContext: CityContextCoordinator
    @State private var selectedSlot: SlotModel?

    var body: some View {
        NavigationStack {
            Group {
                switch mapCoordinator.placeSlotsPhase {
                case .idle, .loading:
                    VStack(spacing: 12) {
                        ProgressView().tint(LinkUpPalette.red)
                        Text("Loading LinkUps…")
                            .font(LinkUpTypography.body(12))
                            .foregroundStyle(LinkUpPalette.textDimmed)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                case .empty:
                    LinkUpEmptyState(
                        title: "No LinkUps here",
                        message: "There are no server-authorized LinkUps for this place in the current map window.",
                        systemImage: "mappin.slash"
                    )
                case .failed(let message):
                    LinkUpErrorState(message: message) {
                        Task { await mapCoordinator.refreshPlaceSlots(placeID: placeID) }
                    }
                    .padding(20)
                case .content:
                    ScrollView {
                        LazyVStack(spacing: 12) {
                            ForEach(mapCoordinator.placeSlots) { slot in
                                SlotCardView(
                                    slot: slot,
                                    cityTimeScope: activeCityTimeScope,
                                    isMutating: social.mutationControlsDisabled,
                                    open: { selectedSlot = slot },
                                    primaryAction: { primaryAction(slot) }
                                )
                            }
                        }
                        .padding(16)
                    }
                    .scrollIndicators(.hidden)
                    .refreshable {
                        await mapCoordinator.refreshPlaceSlots(placeID: placeID)
                    }
                }
            }
            .background(LinkUpPalette.background.ignoresSafeArea())
            .navigationTitle(cluster.placeName ?? "Place LinkUps")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Done") { dismiss() }
                        .foregroundStyle(LinkUpPalette.red)
                }
            }
            .sheet(item: $selectedSlot, onDismiss: {
                Task { await mapCoordinator.refreshPlaceSlots(placeID: placeID) }
            }) { slot in
                SlotDetailView(
                    slot: slot,
                    coordinator: social,
                    cityContext: cityContext,
                    api: api,
                    session: session
                )
            }
        }
        .preferredColorScheme(.dark)
    }

    private var activeCityTimeScope: CityTimeScope? {
        guard let context = cityContext.context, context.isFresh() else { return nil }
        return context.timeScope
    }

    private func primaryAction(_ slot: SlotModel) {
        switch slot.viewerState {
        case .none where slot.canRequestToJoin:
            Task {
                do {
                    _ = try await social.request(slot)
                    await mapCoordinator.refreshPlaceSlots(placeID: placeID)
                } catch is CancellationError {
                    return
                } catch {
                    return
                }
            }
        case .pending:
            return
        default:
            selectedSlot = slot
        }
    }
}
