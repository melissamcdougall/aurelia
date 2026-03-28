import SwiftUI

struct ServiceListView: View {
    let store: ServiceStore
    @State private var expandedService: String?

    var body: some View {
        if !store.isConnected {
            DisconnectedView()
        } else if store.services.isEmpty {
            Text("NO SERVICES")
                .font(LaminaTheme.label)
                .foregroundStyle(LaminaTheme.dim)
                .tracking(2)
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .center)
        } else {
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 0) {
                    if store.clusterMode {
                        clusterView
                    } else {
                        localView
                    }
                }
            }
        }
    }

    @ViewBuilder
    private var localView: some View {
        ForEach(store.services) { service in
            serviceEntry(service)
        }
    }

    @ViewBuilder
    private var clusterView: some View {
        ForEach(store.nodeNames, id: \.self) { node in
            NodeHeaderView(
                name: node,
                status: store.peers[node] ?? "ok"
            )

            ForEach(store.services(forNode: node)) { service in
                serviceEntry(service)
            }
        }
    }

    @ViewBuilder
    private func serviceEntry(_ service: ServiceInfo) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            ServiceRowView(
                service: service,
                isExpanded: expandedService == service.id,
                onToggle: {
                    withAnimation(.easeInOut(duration: 0.15)) {
                        expandedService = expandedService == service.id ? nil : service.id
                    }
                },
                onAction: { action in
                    Task {
                        switch action {
                        case "start": await store.start(service: service)
                        case "stop": await store.stop(service: service)
                        case "restart": await store.restart(service: service)
                        default: break
                        }
                    }
                }
            )

            if expandedService == service.id {
                ServiceDetailView(service: service, store: store)
            }

            Rectangle()
                .fill(LaminaTheme.border)
                .frame(height: 1)
        }
    }
}

// MARK: - Node Header

struct NodeHeaderView: View {
    let name: String
    let status: String

    var body: some View {
        HStack(spacing: 8) {
            Circle()
                .fill(peerColor)
                .frame(width: 6, height: 6)

            Text(name.uppercased())
                .font(.system(size: 9, weight: .bold, design: .monospaced))
                .foregroundStyle(LaminaTheme.accent)
                .tracking(2)

            Rectangle()
                .fill(LaminaTheme.border)
                .frame(height: 1)

            if status != "ok" {
                Text(status.uppercased())
                    .font(.system(size: 8, weight: .bold, design: .monospaced))
                    .foregroundStyle(peerColor)
                    .tracking(1)
            }
        }
        .padding(.horizontal, 14)
        .padding(.top, 10)
        .padding(.bottom, 4)
    }

    private var peerColor: Color {
        switch status {
        case "ok": LaminaTheme.statusOk
        case "timeout": LaminaTheme.statusWarn
        default: LaminaTheme.statusError
        }
    }
}

// MARK: - Disconnected

struct DisconnectedView: View {
    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: "bolt.horizontal.circle")
                .font(.system(size: 28, weight: .thin))
                .foregroundStyle(LaminaTheme.dim)
            Text("AURELIA NOT RUNNING")
                .font(LaminaTheme.label)
                .foregroundStyle(LaminaTheme.dim)
                .tracking(2)
            Text("waiting for daemon")
                .font(LaminaTheme.monoTiny)
                .foregroundStyle(LaminaTheme.dim)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .center)
    }
}
