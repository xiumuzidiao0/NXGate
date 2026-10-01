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

    fun suggestedAction(isEn: Boolean = false): String {
        return when (this) {
            is NetworkTimeout -> if (isEn) "Please check your network connectivity or server firewall status." else "请检查移动网络连接或服务器安全组/防火墙端口状态。"
            is Unauthorized -> if (isEn) "Verify username and password in Settings -> Server Profiles." else "请在「系统与安全」中核验该服务器的管理账号与密码。"
            is SecretPathInvalid -> if (isEn) "Verify the Secret Path configuration matches the server UI_PATH." else "请核对服务器配置中的安全路径是否与服务端 UI_PATH 一致。"
            is TlsMismatch -> if (isEn) "Toggle the HTTP/HTTPS (TLS) switch in Server Profile." else "请在服务器设置中切换 HTTP/HTTPS (TLS) 开关。"
            is ServerUnreachable -> if (isEn) "Confirm the VPS is running and port is listening." else "请确认云服务器是否开机且 NXGate 守护进程正常运行。"
            is HttpStatus -> if (isEn) "Server encountered an error processing the request." else "服务器处理请求时返回非正常状态码。"
            is ParseFailure -> if (isEn) "Ensure app version is compatible with server version." else "请确认 App 客户端与服务端版本是否兼容。"
            is General -> if (isEn) "Retry the operation or inspect server logs." else "请重试操作或在控制台查看详细运行日志。"
        }
    }
}

/**
 * Maps any generic Throwable to a structured domain [ApiError].
 */
fun Throwable.toApiError(server: ServerProfile? = null): ApiError {
    return when (this) {
        is java.net.SocketTimeoutException -> ApiError.NetworkTimeout(10000L, this)
        is java.net.ConnectException -> ApiError.ServerUnreachable(server?.host ?: "", server?.port ?: 0, this)
        is java.net.UnknownHostException -> ApiError.ServerUnreachable(server?.host ?: "", server?.port ?: 0, this)
        is javax.net.ssl.SSLException -> ApiError.TlsMismatch(server?.isTls ?: false, message ?: "")
        else -> {
            val msg = message ?: ""
            when {
                msg.contains("401") || msg.contains("Unauthorized", ignoreCase = true) ->
                    ApiError.Unauthorized(server?.username ?: "", msg)
                msg.contains("404") || msg.contains("Not Found", ignoreCase = true) ->
                    ApiError.SecretPathInvalid(server?.path ?: "", msg)
                msg.contains("timeout", ignoreCase = true) ->
                    ApiError.NetworkTimeout(10000L, this)
                else -> ApiError.General(msg, this)
            }
        }
    }
}
