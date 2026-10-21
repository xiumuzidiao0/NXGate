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

fun AppStrings.inboundSubtitle(inbound: InboundProtocolItem): String {
    if (this != AppStringsEn) return inbound.subtitle
    val egress = inbound.outboundLabel.ifEmpty {
        if (inbound.outbound == "direct") "Direct (VPS network)" else inbound.outbound
    }
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
