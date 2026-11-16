package com.nxgate.app.util

import com.nxgate.app.model.DynamicGroupCard
import com.nxgate.app.model.InboundProtocolItem
import com.nxgate.app.model.PortRuleItem

fun AppStrings.policyLabel(rule: PortRuleItem): String = if (this != AppStringsEn) rule.policyDisplay else when (rule.policy) {
    "round_robin" -> "Round robin"
    "least_rtt" -> "Lowest latency"
    "interval" -> "Rotate every ${rule.intervalSeconds}s"
    "random" -> "Random"
    else -> rule.policy
}

fun AppStrings.bindingLabel(rule: PortRuleItem): String = if (this != AppStringsEn) rule.bindingDisplay else when {
    rule.boundGroupIds.isNotEmpty() -> "Bound to ${rule.boundGroupIds.size} egress groups"
    rule.boundTunnelIds.isNotEmpty() -> "Bound to ${rule.boundTunnelIds.size} specific tunnels"
    else -> "Bound to all healthy tunnels"
}

fun AppStrings.inboundTitle(inbound: InboundProtocolItem): String =
    if (this == AppStringsEn) "Inbound: ${inbound.protocol} (port ${inbound.port})" else inbound.title

fun localizeOutboundLabel(label: String, outbound: String = "direct", port: Int = 0, isEn: Boolean): String {
    if (!isEn) {
        return label.ifEmpty { if (outbound == "direct") "直连出口 (VPS 本机原生网络)" else "出口指向: $outbound" }
    }
    if (label.isEmpty()) {
        return if (outbound == "direct") "Direct (VPS network)" else outbound
    }
    if (label.startsWith("直连出口")) {
        return "Direct Exit (VPS native network, no chain)"
    }
    if (label.startsWith("NXGate 默认出口")) {
        val p = Regex("""PORT\s+(\d+)""").find(label)?.groupValues?.get(1) ?: (if (port > 0) "$port" else "7928")
        val auth = if (label.contains("密码保护")) "Auth" else "No Auth"
        return "NXGate Default Exit (PORT $p - SOCKS5 TCP+UDP $auth)"
    }
    if (label.startsWith("多端口出口")) {
        val p = Regex("""PORT\s+(\d+)""").find(label)?.groupValues?.get(1) ?: (if (port > 0) "$port" else "")
        val auth = if (label.contains("密码保护")) "Auth" else "No Auth"
        val tCount = Regex("""(\d+)\s*隧道""").find(label)?.groupValues?.get(1)
        val gCount = Regex("""(\d+)\s*动态组""").find(label)?.groupValues?.get(1)
        val extras = listOfNotNull(
            tCount?.let { "$it tunnels" },
            gCount?.let { "$it groups" }
        ).joinToString(", ")
        val suffix = if (extras.isNotEmpty()) " - $extras" else ""
        return "Multi-Port Exit (PORT $p - SOCKS5 TCP+UDP $auth$suffix)"
    }
    return label
}

fun AppStrings.inboundSubtitle(inbound: InboundProtocolItem): String {
    if (this != AppStringsEn) return inbound.subtitle
    val egress = localizeOutboundLabel(inbound.outboundLabel, inbound.outbound, inbound.outboundPort, isEn = true)
    return "Port ${inbound.port} • $egress"
}

fun AppStrings.groupDescription(group: DynamicGroupCard): String {
    if (this != AppStringsEn) return group.description
    val country = when {
        group.country.isEmpty() -> "All countries"
        group.country.equals("FAVORITES", true) -> "Favorites"
        else -> group.country
    }
    val network = when (group.ipType) {
        "residential" -> "Residential"
        "hosting" -> "Datacenter"
        else -> "All networks"
    }
    val risk = when (group.purityLevel) {
        "pure_residential" -> "Residential (risk ≤15)"
        "clean" -> "Low risk (≤35)"
        "datacenter" -> "Datacenter"
        else -> "Any risk level"
    }
    val mode = when (group.connectMode) {
        "zero_tun" -> "User-space SSTP"
        "kernel" -> "Kernel TUN"
        else -> "Automatic mode"
    }
    val access = when (group.unlockFilter) {
        "ai" -> "AI services"
        "connectivity" -> "HTTP 204 connectivity"
        "streaming" -> "Streaming services"
        "all" -> "AI and streaming services"
        else -> "Any"
    }
    val fallback = when (group.fallbackPolicy) {
        "favorites" -> "Favorites"
        "auto_low_latency" -> "Lowest latency"
        "auto_speed" -> "Highest bandwidth"
        "none" -> "Off"
        else -> group.fallbackPolicy
    }
    return "Country: $country\nNetwork and risk: $network • $risk • $mode\nService access: $access; fallback: $fallback\nTarget: ${group.targetCount} tunnels; evaluate every ${group.intervalMinutes} minutes"
}
