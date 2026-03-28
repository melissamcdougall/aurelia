import SwiftUI

struct DependencyGraphView: View {
    let graphStore: GraphStore

    var body: some View {
        VStack(spacing: 0) {
            // Toolbar
            HStack(spacing: 12) {
                Text("DEPENDENCY GRAPH")
                    .font(.system(size: 10, weight: .bold, design: .monospaced))
                    .foregroundStyle(LaminaTheme.fg)
                    .tracking(2)

                Spacer()

                // Node selector
                ForEach(graphStore.availableNodes, id: \.self) { node in
                    Button {
                        graphStore.selectedNode = node
                        graphStore.highlightedService = nil
                    } label: {
                        Text(node.uppercased())
                            .font(.system(size: 8, weight: .bold, design: .monospaced))
                            .tracking(1)
                            .foregroundStyle(
                                graphStore.selectedNode == node ? LaminaTheme.accent : LaminaTheme.muted
                            )
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                            .background(
                                RoundedRectangle(cornerRadius: 3)
                                    .fill(graphStore.selectedNode == node
                                          ? LaminaTheme.accent.opacity(0.1)
                                          : LaminaTheme.panelBg)
                            )
                            .overlay(
                                RoundedRectangle(cornerRadius: 3)
                                    .stroke(graphStore.selectedNode == node
                                            ? LaminaTheme.accent.opacity(0.3)
                                            : LaminaTheme.border, lineWidth: 1)
                            )
                    }
                    .buttonStyle(.plain)
                }

                // Filter toggle
                Button {
                    graphStore.showAllServices.toggle()
                    graphStore.highlightedService = nil
                } label: {
                    Text(graphStore.showAllServices ? "ALL" : "DEPS")
                        .font(.system(size: 8, weight: .bold, design: .monospaced))
                        .tracking(1)
                        .foregroundStyle(LaminaTheme.muted)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(RoundedRectangle(cornerRadius: 3).fill(LaminaTheme.panelBg))
                        .overlay(RoundedRectangle(cornerRadius: 3).stroke(LaminaTheme.border, lineWidth: 1))
                }
                .buttonStyle(.plain)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 10)

            Rectangle()
                .fill(LaminaTheme.border)
                .frame(height: 1)

            // Graph
            GraphCanvasView(graphStore: graphStore)
                .frame(maxWidth: .infinity, maxHeight: .infinity)

            Rectangle()
                .fill(LaminaTheme.border)
                .frame(height: 1)

            // Legend
            HStack(spacing: 20) {
                legendItem(label: "REQUIRES", solid: true)
                legendItem(label: "AFTER", solid: false)
                Spacer()
                Text("click a service to trace dependencies")
                    .font(.system(size: 9, design: .monospaced))
                    .foregroundStyle(LaminaTheme.dim)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 8)
        }
        .background(LaminaTheme.bg)
    }

    private func legendItem(label: String, solid: Bool) -> some View {
        HStack(spacing: 6) {
            Canvas { context, size in
                var path = Path()
                path.move(to: CGPoint(x: 0, y: size.height / 2))
                path.addLine(to: CGPoint(x: size.width, y: size.height / 2))
                let style = solid
                    ? StrokeStyle(lineWidth: 1.5, lineCap: .round)
                    : StrokeStyle(lineWidth: 1, lineCap: .round, dash: [4, 3])
                let color = solid ? LaminaTheme.fg.opacity(0.5) : LaminaTheme.muted.opacity(0.5)
                context.stroke(path, with: .color(color), style: style)
            }
            .frame(width: 24, height: 10)

            Text(label)
                .font(.system(size: 8, weight: .bold, design: .monospaced))
                .foregroundStyle(LaminaTheme.muted)
                .tracking(1)
        }
    }
}
