package com.nxgate.app.data

import com.nxgate.app.model.ServerProfile
import java.net.URI
import java.net.URLDecoder

object ServerUriParser {
    fun parse(
        rawUri: String,
        idProvider: () -> String = { "server-${System.currentTimeMillis()}" }
    ): ServerProfile? {
        val trimmed = rawUri.trim()
        if (trimmed.isEmpty()) return null

        val uri = try {
            URI(trimmed)
        } catch (_: Exception) {
            return null
        }

        val scheme = uri.scheme?.lowercase() ?: return null
        if (scheme != "nxgate" && scheme != "aimili") return null
        if (!uri.host.equals("server", ignoreCase = true)) return null

        val params = parseQueryParams(uri.rawQuery) ?: return null
        val host = params["host"]?.takeIf { it.isNotBlank() } ?: return null
        val port = when (val rawPort = params["port"]) {
            null -> 8787
            else -> rawPort.toIntOrNull()?.takeIf { it in 1..65535 } ?: return null
        }
        val path = params["path"] ?: "enter"
        val user = params["user"] ?: "admin"
        val pass = params["pass"] ?: ""
        val name = params["name"]?.takeIf { it.isNotEmpty() } ?: "NXGate ($host)"
        val isTls = params["tls"] == "1"
        val allowInsecureTls = params["insecure"] == "1"

        return ServerProfile(
            id = idProvider(),
            name = name,
            host = host,
            port = port,
            path = path,
            username = user,
            password = pass,
            isTls = isTls,
            allowInsecureTls = allowInsecureTls
        )
    }

    private fun parseQueryParams(rawQuery: String?): Map<String, String>? {
        if (rawQuery.isNullOrEmpty()) return emptyMap()
        val result = linkedMapOf<String, String>()
        for (pair in rawQuery.split('&')) {
            if (pair.isEmpty()) continue
            val idx = pair.indexOf('=')
            val rawKey = if (idx >= 0) pair.substring(0, idx) else pair
            val rawVal = if (idx >= 0) pair.substring(idx + 1) else ""
            val key = decodeComponent(rawKey) ?: return null
            if (key.isEmpty()) continue
            val value = decodeComponent(rawVal) ?: return null
            result.putIfAbsent(key, value)
        }
        return result
    }

    private fun decodeComponent(raw: String): String? = try {
        URLDecoder.decode(raw, Charsets.UTF_8.name())
    } catch (_: IllegalArgumentException) {
        null
    }
}
