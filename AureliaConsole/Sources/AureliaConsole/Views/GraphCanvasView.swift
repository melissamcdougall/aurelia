import SwiftUI

struct GraphCanvasView: View {
    let graphStore: GraphStore

    var body: some View {
        let layout = graphStore.layout

        if layout.nodes.isEmpty {
            VStack(spacing: 12) {
                Image(systemName: "point.3.connected.trianglepath.dotted")
                    .font(.system(size: 28, weight: .thin))
                    .foregroundStyle(LaminaTheme.dim)
                Text("NO DEPENDENCIES")
                    .font(LaminaTheme.label)
                    .foregroundStyle(LaminaTheme.dim)
                    .tracking(2)
                if !graphStore.showAllServices {
                    Text("try showing all services")
                        .font(LaminaTheme.monoTiny)
                        .foregroundStyle(LaminaTheme.dim)
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else {
            ScrollView([.horizontal, .vertical]) {
                ZStack(alignment: .topLeading) {
                    // Edges layer
                    Canvas { context, _ in
                        for edge in layout.edges {
                            let highlighted = graphStore.highlightedService
                            let edgeDimmed = highlighted != nil
                                && edge.from != highlighted
                                && edge.to != highlighted
                            drawEdge(context: &context, edge: edge, dimmed: edgeDimmed)
                        }
                    }
                    .frame(width: layout.canvasSize.width, height: layout.canvasSize.height)

                    // Nodes layer
                    ForEach(layout.nodes, id: \.name) { node in
                        let highlighted = graphStore.highlightedService
                        let isHighlighted = node.name == highlighted
                            || graphStore.transitiveDeps(of: highlighted ?? "").contains(node.name)
                        let isDimmed = highlighted != nil && !isHighlighted && node.name != highlighted

                        GraphNodeView(
                            node: node,
                            isHighlighted: isHighlighted,
                            isDimmed: isDimmed,
                            onTap: {
                                withAnimation(.easeInOut(duration: 0.15)) {
                                    graphStore.highlightedService =
                                        graphStore.highlightedService == node.name ? nil : node.name
                                }
                            }
                        )
                        .position(x: node.center.x, y: node.center.y)
                    }
                }
                .frame(width: layout.canvasSize.width, height: layout.canvasSize.height)
                .contentShape(Rectangle())
                .onTapGesture {
                    withAnimation(.easeInOut(duration: 0.15)) {
                        graphStore.highlightedService = nil
                    }
                }
            }
        }
    }

    private func drawEdge(context: inout GraphicsContext, edge: LayoutEdge, dimmed: Bool) {
        let from = edge.fromPoint
        let to = edge.toPoint
        let cpOffset = (to.x - from.x) * 0.4

        var path = Path()
        path.move(to: from)
        path.addCurve(
            to: to,
            control1: CGPoint(x: from.x + cpOffset, y: from.y),
            control2: CGPoint(x: to.x - cpOffset, y: to.y)
        )

        let opacity = dimmed ? 0.08 : (edge.kind == .requires ? 0.5 : 0.3)

        switch edge.kind {
        case .requires:
            context.stroke(
                path,
                with: .color(Color(LaminaTheme.fg).opacity(opacity)),
                style: StrokeStyle(lineWidth: 1.5, lineCap: .round)
            )
        case .after:
            context.stroke(
                path,
                with: .color(Color(LaminaTheme.muted).opacity(opacity)),
                style: StrokeStyle(lineWidth: 1, lineCap: .round, dash: [6, 4])
            )
        }
    }
}
