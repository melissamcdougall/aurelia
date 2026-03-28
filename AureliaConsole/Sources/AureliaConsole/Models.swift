import Foundation

enum ServiceState: String, Codable, Sendable {
    case stopped, starting, running, stopping, failed
}

enum HealthStatus: String, Codable, Sendable {
    case unknown, healthy, unhealthy
}

struct ServiceInfo: Codable, Identifiable, Sendable {
    var id: String {
        if let node { return "\(node)/\(name)" }
        return name
    }
    let name: String
    let type: String
    let state: ServiceState
    let health: HealthStatus
    let pid: Int?
    let port: Int?
    let uptime: String?
    let restartCount: Int
    let lastExitCode: Int?
    let lastError: String?
    let node: String?

    enum CodingKeys: String, CodingKey {
        case name, type, state, health, pid, port, uptime, node
        case restartCount = "restart_count"
        case lastExitCode = "last_exit_code"
        case lastError = "last_error"
    }
}

struct LogResponse: Codable, Sendable {
    let lines: [String]?
}

struct ErrorResponse: Codable, Sendable {
    let error: String
}

struct ClusterServicesResponse: Codable, Sendable {
    let services: [ServiceInfo]
    let peers: [String: String]
}

struct ClusterGraphResponse: Codable, Sendable {
    let nodes: [GraphNode]
    let peers: [String: String]  // node name -> "ok" | "timeout" | "error" | "unreachable"
}

// MARK: - Graph Layout

enum EdgeKind: Sendable {
    case requires  // hard dependency — solid line
    case after     // soft ordering — dashed line
}

struct LayoutNode: Sendable {
    let name: String
    let type: String
    let state: ServiceState
    let health: HealthStatus
    let layer: Int
    let positionInLayer: Int
    var center: CGPoint = .zero
    let frame: CGRect

    init(name: String, type: String, state: ServiceState, health: HealthStatus, layer: Int, positionInLayer: Int) {
        self.name = name
        self.type = type
        self.state = state
        self.health = health
        self.layer = layer
        self.positionInLayer = positionInLayer
        let x = CGFloat(layer) * GraphLayout.columnSpacing + GraphLayout.margin
        let y = CGFloat(positionInLayer) * GraphLayout.rowSpacing + GraphLayout.margin
        self.frame = CGRect(x: x, y: y, width: GraphLayout.nodeWidth, height: GraphLayout.nodeHeight)
        self.center = CGPoint(x: frame.midX, y: frame.midY)
    }
}

struct LayoutEdge: Sendable {
    let from: String
    let to: String
    let kind: EdgeKind
    var fromPoint: CGPoint = .zero
    var toPoint: CGPoint = .zero
}

struct GraphLayout: Sendable {
    static let nodeWidth: CGFloat = 140
    static let nodeHeight: CGFloat = 50
    static let columnSpacing: CGFloat = 200
    static let rowSpacing: CGFloat = 70
    static let margin: CGFloat = 40

    let nodes: [LayoutNode]
    let edges: [LayoutEdge]
    let canvasSize: CGSize
}

// MARK: - API Types

struct GraphNode: Codable, Sendable {
    let name: String
    let type: String
    let state: ServiceState
    let health: HealthStatus
    let port: Int?
    let uptime: String?
    let restartCount: Int
    let after: [String]?
    let requires: [String]?
    let node: String?

    enum CodingKeys: String, CodingKey {
        case name, type, state, health, port, uptime, after, requires, node
        case restartCount = "restart_count"
    }
}
