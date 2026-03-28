import Foundation

enum ServiceState: String, Codable, Sendable {
    case stopped, starting, running, stopping, failed
}

enum HealthStatus: String, Codable, Sendable {
    case unknown, healthy, unhealthy
}

struct ServiceInfo: Codable, Identifiable, Sendable {
    var id: String { name }
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
    let lines: [String]
}

struct ErrorResponse: Codable, Sendable {
    let error: String
}
