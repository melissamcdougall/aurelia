import Foundation

actor AureliaClient {
    private let socketPath: String

    init(socketPath: String? = nil) {
        if let socketPath {
            self.socketPath = socketPath
        } else {
            self.socketPath = FileManager.default.homeDirectoryForCurrentUser
                .appendingPathComponent(".aurelia/aurelia.sock").path
        }
    }

    func services() async throws -> [ServiceInfo] {
        let data = try await get("/v1/services")
        return try JSONDecoder().decode([ServiceInfo].self, from: data)
    }

    func logs(service: String, lines: Int = 50) async throws -> [String] {
        let data = try await get("/v1/services/\(service)/logs?n=\(lines)")
        let response = try JSONDecoder().decode(LogResponse.self, from: data)
        return response.lines
    }

    func action(service: String, action: String) async throws {
        let data = try await post("/v1/services/\(service)/\(action)")
        // Check for error response
        if let errorResponse = try? JSONDecoder().decode(ErrorResponse.self, from: data) {
            throw ClientError.apiError(errorResponse.error)
        }
    }

    func health() async throws -> Bool {
        let data = try await get("/v1/health")
        let json = try JSONDecoder().decode([String: String].self, from: data)
        return json["status"] == "ok"
    }

    // MARK: - HTTP via curl

    private func get(_ path: String) async throws -> Data {
        try await curl(method: "GET", path: path)
    }

    private func post(_ path: String) async throws -> Data {
        try await curl(method: "POST", path: path)
    }

    private func curl(method: String, path: String) async throws -> Data {
        let process = Process()
        process.executableURL = URL(fileURLWithPath: "/usr/bin/curl")
        process.arguments = [
            "--unix-socket", socketPath,
            "-s",
            "-X", method,
            "http://localhost\(path)",
        ]

        let stdout = Pipe()
        let stderr = Pipe()
        process.standardOutput = stdout
        process.standardError = stderr

        try process.run()
        process.waitUntilExit()

        let data = stdout.fileHandleForReading.readDataToEndOfFile()

        if process.terminationStatus != 0 {
            throw ClientError.curlFailed(Int(process.terminationStatus))
        }

        if data.isEmpty {
            throw ClientError.emptyResponse
        }

        return data
    }

    enum ClientError: Error, LocalizedError {
        case curlFailed(Int)
        case emptyResponse
        case apiError(String)

        var errorDescription: String? {
            switch self {
            case .curlFailed(let code): "curl exited with code \(code)"
            case .emptyResponse: "Empty response from daemon"
            case .apiError(let message): message
            }
        }
    }
}
