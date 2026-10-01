package com.nxgate.app.model

/**
 * Structured domain error hierarchy for NXGate API interactions.
 * Enables actionable and self-healing UI guidance for different failure modes.
 */
sealed interface ApiError {
    /** Network connection timed out (server unreachable or network congestion). */
    data class NetworkTimeout(val timeoutMs: Long = 10000L, val cause: Throwable? = null) : ApiError

    /** Authentication failed: HTTP 401 Unauthorized (wrong username or password). */
    data class Unauthorized(val username: String = "", val message: String = "") : ApiError

    /** Path invalid: HTTP 404 (wrong secret path or endpoint not found). */
    data class SecretPathInvalid(val attemptedPath: String = "", val message: String = "") : ApiError

    /** Protocol mismatch: HTTP vs HTTPS/TLS misconfiguration. */
    data class TlsMismatch(val expectedTls: Boolean, val message: String = "") : ApiError

    /** Server offline or connection refused by host. */
    data class ServerUnreachable(val host: String = "", val port: Int = 0, val cause: Throwable? = null) : ApiError

    /** HTTP error with non-2xx status code. */
    data class HttpStatus(val code: Int, val message: String) : ApiError

    /** Response JSON deserialization or payload validation failure. */
    data class ParseFailure(val rawMessage: String, val cause: Throwable? = null) : ApiError

    /** General unclassified error with human-readable description. */
    data class General(val message: String, val cause: Throwable? = null) : ApiError

    fun userFriendlyMessage(isEn: Boolean = false): String {
        return when (this) {
            is NetworkTimeout -> if (isEn) "Connection timed out (${timeoutMs}ms)" else "网络连接超时 (${timeoutMs}ms)"
            is Unauthorized -> if (isEn) "Authentication failed: invalid credentials" else "认证失败：账号或密码错误"
            is SecretPathInvalid -> if (isEn) "Path error: secret path not found ($attemptedPath)" else "路径错误：隐蔽访问路径不匹配 ($attemptedPath)"
            is TlsMismatch -> if (isEn) "Protocol error: check HTTP/HTTPS setting" else "协议错误：请检查 HTTP/HTTPS 配置"
            is ServerUnreachable -> if (isEn) "Server unreachable ($host:$port)" else "无法连接服务器 ($host:$port)"
            is HttpStatus -> if (isEn) "Server returned HTTP $code: $message" else "服务器返回错误 HTTP $code: $message"
            is ParseFailure -> if (isEn) "Data parsing error: $rawMessage" else "数据解析异常: $rawMessage"
            is General -> message
        }
    }
}
