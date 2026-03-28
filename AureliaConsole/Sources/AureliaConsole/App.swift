import SwiftUI

@main
struct AureliaConsoleApp: App {
    var body: some Scene {
        MenuBarExtra("Aurelia", systemImage: "circle.fill") {
            Text("AureliaConsole")
                .padding()
        }
        .menuBarExtraStyle(.window)
    }
}
