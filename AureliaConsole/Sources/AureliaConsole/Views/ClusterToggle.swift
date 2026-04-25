import SwiftUI

struct ClusterToggle: View {
    let isCluster: Bool
    let action: () -> Void
    @State private var isHovered = false

    var body: some View {
        Button(action: action) {
            HStack(spacing: 4) {
                Image(systemName: isCluster ? "network" : "desktopcomputer")
                    .font(.system(size: 9))
                Text(isCluster ? "CLUSTER" : "LOCAL")
                    .font(.system(size: 8, weight: .bold, design: .monospaced))
                    .tracking(1)
            }
            .foregroundStyle(isCluster ? LaminaTheme.accent : LaminaTheme.muted)
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(
                RoundedRectangle(cornerRadius: 3)
                    .fill(isCluster ? LaminaTheme.accent.opacity(0.1) : LaminaTheme.panelBg)
            )
            .overlay(
                RoundedRectangle(cornerRadius: 3)
                    .stroke(isCluster ? LaminaTheme.accent.opacity(0.3) : LaminaTheme.border, lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
        .onHover { isHovered = $0 }
    }
}
