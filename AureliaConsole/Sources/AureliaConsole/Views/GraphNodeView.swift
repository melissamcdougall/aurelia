import SwiftUI

struct GraphNodeView: View {
    let node: LayoutNode
    let isHighlighted: Bool
    let isDimmed: Bool
    let onTap: () -> Void
    @State private var isHovered = false

    var body: some View {
        HStack(spacing: 6) {
            Circle()
                .fill(statusColor)
                .frame(width: 7, height: 7)
                .shadow(color: statusColor.opacity(0.5), radius: node.state == .running ? 2 : 0)

            Text(node.name)
                .font(.system(size: 11, weight: .medium, design: .monospaced))
                .foregroundStyle(LaminaTheme.fg)
                .lineLimit(1)

            Spacer()

            Text(node.type.uppercased())
                .font(.system(size: 7, weight: .bold, design: .monospaced))
                .foregroundStyle(LaminaTheme.accent.opacity(0.7))
                .tracking(1)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 6)
        .frame(width: GraphLayout.nodeWidth, height: GraphLayout.nodeHeight)
        .background(
            RoundedRectangle(cornerRadius: 4)
                .fill(isHovered ? LaminaTheme.panelBgHover : LaminaTheme.panelBg)
        )
        .overlay(
            RoundedRectangle(cornerRadius: 4)
                .stroke(isHighlighted ? LaminaTheme.accent : LaminaTheme.border, lineWidth: isHighlighted ? 1.5 : 1)
        )
        .opacity(isDimmed ? 0.25 : 1.0)
        .onTapGesture(perform: onTap)
        .onHover { isHovered = $0 }
    }

    private var statusColor: Color {
        switch (node.state, node.health) {
        case (.failed, _): LaminaTheme.statusError
        case (.running, .healthy): LaminaTheme.statusOk
        case (.running, .unhealthy): LaminaTheme.statusWarn
        case (.starting, _), (.stopping, _): LaminaTheme.statusWarn
        case (.stopped, _): LaminaTheme.statusOff
        default: LaminaTheme.statusOff
        }
    }
}
