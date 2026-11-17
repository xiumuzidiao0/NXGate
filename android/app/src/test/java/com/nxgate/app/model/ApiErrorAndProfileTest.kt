package com.nxgate.app.model

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import java.net.ConnectException
import java.net.SocketTimeoutException
import javax.net.ssl.SSLException

class ApiErrorAndProfileTest {
    @Test
    fun serverProfileBaseUrlNormalizesSchemeHostPortAndPath() {
        val plain = ServerProfile(
            id = "1",
            name = "Test",
            host = "198.51.100.10",
            port = 8787,
            path = "/enter/",
            username = "admin",
            password = "secret",
            isTls = false
        )
        assertEquals("http://198.51.100.10:8787/enter", plain.baseUrl)

        val overridden = ServerProfile(
            id = "2",
            name = "TLS Override",
            host = "https://panel.example.com:9443/",
            port = 8787,
            path = "",
            username = "admin",
            password = "secret",
            isTls = false
        )
        assertEquals("https://panel.example.com:9443", overridden.baseUrl)
    }

    @Test
    fun throwableMapsToStructuredApiErrorWithGuidance() {
        val sampleServer = ServerProfile(
            id = "1",
            name = "Node",
            host = "panel.example.com",
            port = 8787,
            path = "secret",
            username = "ops",
            password = "pw",
            isTls = true
        )

        val timeoutErr = SocketTimeoutException("read timed out").toApiError(sampleServer)
        assertTrue(timeoutErr is ApiError.NetworkTimeout)
        assertTrue(timeoutErr.userFriendlyMessage(isEn = true).contains("timed out"))

        val unreachableErr = ConnectException("Connection refused").toApiError(sampleServer)
        assertTrue(unreachableErr is ApiError.ServerUnreachable)
        assertTrue(unreachableErr.userFriendlyMessage(isEn = true).contains("panel.example.com:8787"))

        val tlsErr = SSLException("Wrong version number").toApiError(sampleServer)
        assertTrue(tlsErr is ApiError.TlsMismatch)
        assertTrue(tlsErr.suggestedAction(isEn = false).contains("TLS"))

        val authErr = Exception("HTTP 401: Unauthorized").toApiError(sampleServer)
        assertTrue(authErr is ApiError.Unauthorized)

        val pathErr = Exception("HTTP 404: Not Found").toApiError(sampleServer)
        assertTrue(pathErr is ApiError.SecretPathInvalid)
        assertTrue(pathErr.userFriendlyMessage(isEn = false).contains("secret"))
    }
}
