package com.nxgate.app.util

import com.nxgate.app.model.DynamicGroupCard
import com.nxgate.app.model.AvailableOutbound
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

private fun pluralUnit(count: Int, singular: String, plural: String): String =
    "$count ${if (count == 1) singular else plural}"

fun localizeOutboundLabel(ob: AvailableOutbound, isEn: Boolean): String = localizeOutboundLabel(
    label = ob.label,
    outbound = ob.addr,
    port = ob.port,
    isEn = isEn,
    isDefault = ob.isDefault,
    authEnabled = ob.authEnabled,
    tunnelCount = ob.tunnelCount,
    groupCount = ob.groupCount
)

fun localizeOutboundLabel(
    label: String,
    outbound: String = "direct",
    port: Int = 0,
    isEn: Boolean,
    isDefault: Boolean = false,
    authEnabled: Boolean? = null,
    tunnelCount: Int? = null,
    groupCount: Int? = null
): String {
    if (!isEn) {
        return label.ifEmpty { if (outbound == "direct") "直连出口 (VPS 本机原生网络)" else "出口指向: $outbound" }
    }
    if (outbound == "direct" && label.isEmpty()) {
        return "Direct (VPS network)"
    }
    if (label.startsWith("直连出口") || outbound == "direct") {
        return if (label.startsWith("直连出口")) "Direct Exit (VPS native network, no chain)" else label
    }
    val hasAuth = authEnabled ?: (label.contains("密码保护") || outbound.contains("@"))
    val auth = if (hasAuth) "Auth" else "No Auth"
    val parsedPort = Regex("""PORT\s+(\d+)""").find(label)?.groupValues?.get(1) ?: (if (port > 0) "$port" else "")

    if (isDefault || label.startsWith("NXGate 默认出口")) {
        val p = parsedPort.ifEmpty { "7928" }
        return "NXGate Default Exit (PORT $p - SOCKS5 TCP+UDP $auth)"
    }
    val tCount = tunnelCount ?: Regex("""(\d+)\s*隧道""").find(label)?.groupValues?.get(1)?.toIntOrNull()
    val gCount = groupCount ?: Regex("""(\d+)\s*动态组""").find(label)?.groupValues?.get(1)?.toIntOrNull()
    if (label.startsWith("多端口出口") || tCount != null || gCount != null) {
        val extras = listOfNotNull(
            tCount?.takeIf { it > 0 }?.let { pluralUnit(it, "tunnel", "tunnels") },
            gCount?.takeIf { it > 0 }?.let { pluralUnit(it, "group", "groups") }
        ).joinToString(", ")
        val suffix = if (extras.isNotEmpty()) " - $extras" else ""
        return "Multi-Port Exit (PORT $parsedPort - SOCKS5 TCP+UDP $auth$suffix)"
    }
    return label.ifEmpty { outbound }
}

fun AppStrings.inboundSubtitle(inbound: InboundProtocolItem): String {
    if (this != AppStringsEn) return inbound.subtitle
    val egress = localizeOutboundLabel(
        label = inbound.outboundLabel,
        outbound = inbound.outbound,
        port = inbound.outboundPort,
        isEn = true,
        authEnabled = inbound.outboundAuthEnabled,
        tunnelCount = inbound.outboundTunnelCount,
        groupCount = inbound.outboundGroupCount
    )
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
