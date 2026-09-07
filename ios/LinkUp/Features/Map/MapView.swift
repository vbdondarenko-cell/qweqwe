import SwiftUI

struct MapView: View {
    var body: some View {
        GeometryReader { proxy in
            ZStack {
                LinkUpPalette.background.ignoresSafeArea()
                mapGrid(size: proxy.size)
                VStack {
                    areaPill.padding(.top, 14)
                    Spacer()
                    summaryPanel.padding(.horizontal, 20).padding(.bottom, 14)
                }
                HStack {
                    Spacer()
                    VStack(spacing: 8) {
                        mapControl("line.3.horizontal.decrease")
                        mapControl("flame")
                        mapControl("gearshape")
                    }
                    .padding(.trailing, 16).padding(.top, 78)
                    Spacer().frame(width: 0)
                }
                .frame(maxHeight: .infinity, alignment: .top)
            }
        }
    }

    private func mapGrid(size: CGSize) -> some View {
        Canvas { context, canvas in
            var path = Path()
            stride(from: 0.0, through: canvas.width, by: 40).forEach { x in
                path.move(to: CGPoint(x: x, y: 0)); path.addLine(to: CGPoint(x: x, y: canvas.height))
            }
            stride(from: 0.0, through: canvas.height, by: 40).forEach { y in
                path.move(to: CGPoint(x: 0, y: y)); path.addLine(to: CGPoint(x: canvas.width, y: y))
            }
            context.stroke(path, with: .color(LinkUpPalette.textPrimary.opacity(0.07)), lineWidth: 0.5)
        }
        .background(Color(red: 0.03, green: 0.035, blue: 0.043))
    }

    private var areaPill: some View {
        HStack(spacing: 8) {
            Circle().fill(LinkUpPalette.red).frame(width: 8, height: 8)
            Text("City context unavailable")
                .font(LinkUpTypography.body(12, weight: .semibold))
                .foregroundStyle(LinkUpPalette.textPrimary)
        }
        .padding(.horizontal, 16).padding(.vertical, 10)
        .linkUpGlass()
        .clipShape(Capsule())
        .overlay { Capsule().stroke(LinkUpPalette.border) }
    }

    private func mapControl(_ symbol: String) -> some View {
        Image(systemName: symbol)
            .font(.system(size: 18))
            .foregroundStyle(LinkUpPalette.textDimmed)
            .frame(width: 40, height: 40)
            .linkUpGlass()
            .clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.control))
            .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.control).stroke(LinkUpPalette.border) }
    }

    private var summaryPanel: some View {
        HStack {
            VStack(alignment: .leading, spacing: 3) {
                HStack(spacing: 7) {
                    Circle().fill(LinkUpPalette.success).frame(width: 8, height: 8)
                    Text("MAP")
                        .font(LinkUpTypography.mono(11, weight: .semibold))
                        .foregroundStyle(LinkUpPalette.success)
                }
                Text("No active map data")
                    .font(LinkUpTypography.display(14))
                    .foregroundStyle(LinkUpPalette.textPrimary)
            }
            Spacer()
            Text("List view  ›")
                .font(LinkUpTypography.body(12, weight: .semibold))
                .foregroundStyle(LinkUpPalette.textDimmed)
        }
        .padding(14).linkUpGlass().clipShape(RoundedRectangle(cornerRadius: LinkUpRadius.card))
        .overlay { RoundedRectangle(cornerRadius: LinkUpRadius.card).stroke(LinkUpPalette.border) }
    }
}
