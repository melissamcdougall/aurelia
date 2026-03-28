import AppKit
import SwiftUI

@MainActor
final class GraphWindowController {
    private var window: NSPanel?
    private let graphStore: GraphStore

    init(graphStore: GraphStore) {
        self.graphStore = graphStore
    }

    func showWindow() {
        if let window, window.isVisible {
            window.makeKeyAndOrderFront(nil)
            return
        }

        let panel = NSPanel(
            contentRect: NSRect(x: 0, y: 0, width: 900, height: 550),
            styleMask: [.titled, .closable, .resizable, .utilityWindow],
            backing: .buffered,
            defer: false
        )
        panel.title = "Aurelia — Dependency Graph"
        panel.titlebarAppearsTransparent = true
        panel.backgroundColor = NSColor(LaminaTheme.bg)
        panel.isMovableByWindowBackground = true
        panel.minSize = NSSize(width: 500, height: 300)
        panel.isReleasedWhenClosed = false
        panel.isFloatingPanel = false

        let hostingView = NSHostingView(rootView: DependencyGraphView(graphStore: graphStore))
        panel.contentView = hostingView

        panel.center()
        panel.makeKeyAndOrderFront(nil)

        graphStore.startPolling()
        self.window = panel
    }

    func closeWindow() {
        window?.close()
    }
}
