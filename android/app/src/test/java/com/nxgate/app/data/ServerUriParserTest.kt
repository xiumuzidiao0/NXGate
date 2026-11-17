package com.nxgate.app.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class ServerUriParserTest {

    @Test
    fun parsesCompleteNXGateUri() {
        val profile = ServerUriParser.parse(
            "nxgate://server?host=203.0.113.10&port=9443&path=secret&user=ops&pass=p%40ss%2B1&name=Tokyo+Node&tls=1&insecure=1",
            idProvider = { "fixed-id" }
        )

        assertNotNull(profile)
        assertEquals("fixed-id", profile!!.id)
        assertEquals("Tokyo Node", profile.name)
        assertEquals("203.0.113.10", profile.host)
        assertEquals(9443, profile.port)
        assertEquals("secret", profile.path)
        assertEquals("ops", profile.username)
        assertEquals("p@ss+1", profile.password)
        assertTrue(profile.isTls)
        assertTrue(profile.allowInsecureTls)
    }

    @Test
    fun parsesLegacyAimiliSchemeAndDefaultsWhenOptionalParamsOmitted() {
        val profile = ServerUriParser.parse(
            "  aimili://server?host=edge.example.com  ",
            idProvider = { "legacy-id" }
        )

        assertNotNull(profile)
        assertEquals("NXGate (edge.example.com)", profile!!.name)
        assertEquals("edge.example.com", profile.host)
        assertEquals(8787, profile.port)
        assertEquals("enter", profile.path)
        assertEquals("admin", profile.username)
        assertEquals("", profile.password)
        assertFalse(profile.isTls)
        assertFalse(profile.allowInsecureTls)
    }

    @Test
    fun preservesExplicitEmptyPathDistinctFromOmittedPath() {
        val withEmptyPath = ServerUriParser.parse("nxgate://server?host=10.0.0.2&path=&port=8787")
        val withOmittedPath = ServerUriParser.parse("nxgate://server?host=10.0.0.2&port=8787")

        assertNotNull(withEmptyPath)
        assertNotNull(withOmittedPath)
        assertEquals("", withEmptyPath!!.path)
        assertEquals("enter", withOmittedPath!!.path)
    }

    @Test
    fun returnsNullOnInvalidSchemeHostPortOrEncoding() {
        assertNull(ServerUriParser.parse(""))
        assertNull(ServerUriParser.parse("https://server?host=1.2.3.4"))
        assertNull(ServerUriParser.parse("nxgate://other?host=1.2.3.4"))
        assertNull(ServerUriParser.parse("nxgate://server?port=8787"))
        assertNull(ServerUriParser.parse("nxgate://server?host="))
        assertNull(ServerUriParser.parse("nxgate://server?host=1.2.3.4&port=0"))
        assertNull(ServerUriParser.parse("nxgate://server?host=1.2.3.4&port=70000"))
        assertNull(ServerUriParser.parse("nxgate://server?host=1.2.3.4&port=abc"))
        assertNull(ServerUriParser.parse("nxgate://server?host=1.2.3.4&name=%ZZ"))
    }
}
