package com.nxgate.app.util

import com.nxgate.app.model.DynamicGroupCard
import com.nxgate.app.model.InboundProtocolItem
import com.nxgate.app.model.PortRuleItem
import org.junit.Assert.*
import org.junit.Test

class LocalizedDescriptionsTest {
    @Test fun appCopyUsesSelectedLanguage() {
        assertEquals("English", AppStringsEn.localized("中文", "English"))
        assertEquals("中文", AppStringsZh.localized("中文", "English"))
        assertEquals("Start Tunnel", AppStringsEn.startTunnel)
    }

    @Test fun routingLabelsPreserveChineseAndUseEnglish() {
        val rule = PortRuleItem(port = 7928, policy = "interval", intervalSeconds = 45, boundGroupIds = listOf("user-group"))
        assertEquals(rule.policyDisplay, AppStringsZh.policyLabel(rule))
        assertEquals(rule.bindingDisplay, AppStringsZh.bindingLabel(rule))
        assertEquals("Rotate every 45s", AppStringsEn.policyLabel(rule))
        assertTrue(AppStringsEn.bindingLabel(rule).contains("egress groups"))
        assertEquals("Bound to all healthy tunnels", AppStringsEn.bindingLabel(PortRuleItem(port = 7928)))
    }

    @Test fun inboundLabelsDoNotTranslateUserContent() {
        val inbound = InboundProtocolItem("node", "自定义名称", "VLESS", 443, "direct")
        assertEquals(inbound.title, AppStringsZh.inboundTitle(inbound))
        assertEquals("Inbound: VLESS (port 443)", AppStringsEn.inboundTitle(inbound))
        assertTrue(AppStringsEn.inboundSubtitle(inbound).contains("Direct (VPS network)"))
        assertTrue(AppStringsEn.inboundSubtitle(inbound.copy(outboundLabel = "用户出口")).contains("用户出口"))
    }

    @Test fun groupDescriptionsTranslateGeneratedCopyOnly() {
        val group = DynamicGroupCard("group", "用户组名", country = "FAVORITES", targetCount = 3)
        assertEquals(group.description, AppStringsZh.groupDescription(group))
        val english = AppStringsEn.groupDescription(group)
        assertTrue(english.contains("Favorites"))
        assertTrue(english.contains("Target: 3 tunnels"))
        assertFalse(Regex("[\u3400-\u9fff]").containsMatchIn(english))
    }
}
