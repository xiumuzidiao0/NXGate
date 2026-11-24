package com.nxgate.app.model

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
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

    @Test
    fun nodePoolFieldsAndDefaultsArePreservedOnModels() {
        val active = NodeCandidate(id = "n1", ip = "198.51.100.1", port = 443, countryShort = "JP")
        assertTrue(active.inActivePool)

        val cold = active.copy(id = "n2", countryShort = "CH", inActivePool = false)
        assertFalse(cold.inActivePool)

        val defaults = ServerSettingsDTO()
        assertEquals(250, defaults.nodeHotPoolSize)
        assertEquals(600, defaults.nodeStoreMaxSize)
        assertEquals(3, defaults.nodeMinPerCountry)
        assertEquals(40, defaults.nodeMajorCountryCap)

        val updated = defaults.copy(
            nodeHotPoolSize = -1,
            nodeStoreMaxSize = 1200,
            nodeMinPerCountry = 5,
            nodeMajorCountryCap = 15
        )
        assertEquals(-1, updated.nodeHotPoolSize)
        assertEquals(1200, updated.nodeStoreMaxSize)
        assertEquals(5, updated.nodeMinPerCountry)
        assertEquals(15, updated.nodeMajorCountryCap)
    }
}
