package com.nxgate.app.data

import com.nxgate.app.model.MasterGatewayInfo
import com.nxgate.app.model.TunnelItem
import org.json.JSONObject

internal fun parseTunnelItem(obj: JSONObject, index: Int): TunnelItem {
    val id = obj.optString("id", "tun-$index")
    val devName = obj.optString("dev_name", "tun$index")
    val devIndex = obj.optInt("dev_index", index)
    val tStatus = obj.optString("status", "connected")
    val uptime = obj.optLong("uptime", 0)
    val nodeObj = obj.optJSONObject("node")
    val ip = nodeObj?.optString("ip", "") ?: ""
    val port = nodeObj?.optInt("port", 443) ?: 443
    val cShort = nodeObj?.optString("country_short", "") ?: ""
    val cLong = nodeObj?.optString("country_long", "") ?: ""
    val country = if (cShort.isNotEmpty()) cShort else (if (cLong.isNotEmpty()) cLong else "JP")

    val latFromObj = if (obj.has("latency_ms")) obj.optInt("latency_ms", 0) else 0
    val latFromNode = if (nodeObj != null && nodeObj.has("latency_ms")) nodeObj.optInt("latency_ms", 0) else 0
    val pingFromNode = nodeObj?.optInt("ping", 0) ?: 0
    val latency = when {
        latFromObj > 0 -> latFromObj
        latFromNode > 0 -> latFromNode
        pingFromNode > 0 -> pingFromNode
        else -> 0
    }

    val unlockObj = obj.optJSONObject("unlock")
    val throughputBps = unlockObj?.optLong("throughput_bps", 0) ?: 0
    val throughputPassed = unlockObj?.optBoolean("throughput_passed", true) ?: true

    val isUserland = obj.optBoolean("is_userland", false) || devName.startsWith("sstp")
    val isResidential = nodeObj?.optBoolean("is_residential", false) ?: false
    val isHosting = nodeObj?.optBoolean("is_hosting", false) ?: (nodeObj?.optString("ip_type") == "hosting" || nodeObj?.optString("ip_type") == "datacenter")
    val riskScore = nodeObj?.optDouble("risk_score", -1.0)?.toFloat() ?: -1.0f
    val riskLevel = nodeObj?.optString("risk_level", "unknown") ?: "unknown"
    val riskLabel = nodeObj?.optString("risk_label", "") ?: ""

    return TunnelItem(
        id = id,
        devName = devName,
        devIndex = devIndex,
        status = tStatus,
        uptimeSeconds = uptime,
        latencyMs = latency,
        nodeIp = ip,
        nodePort = port,
        country = country,
        countryLong = cLong,
        throughputBps = throughputBps,
        throughputPassed = throughputPassed,
        isUserland = isUserland,
        isResidential = isResidential,
        isHosting = isHosting,
        riskScore = riskScore,
        riskLevel = riskLevel,
        riskLabel = riskLabel,
        openai = unlockObj?.optString("openai", "unknown") ?: "unknown",
        claude = unlockObj?.optString("claude", "unknown") ?: "unknown",
        gemini = unlockObj?.optString("gemini", "unknown") ?: "unknown",
        netflix = unlockObj?.optString("netflix", "unknown") ?: "unknown"
    )
}

internal fun parseMasterGateway(vpnObj: JSONObject?, includeNodeDetails: Boolean, isEn: Boolean): MasterGatewayInfo {
    val activeNode = vpnObj?.optString("active_node_id", "") ?: ""
    val activeNodeObj = if (includeNodeDetails) vpnObj?.optJSONObject("active_node") else null
    val masterIp = if (includeNodeDetails) (activeNodeObj?.optString("ip", activeNode) ?: activeNode) else ""
    val masterCShort = activeNodeObj?.optString("country_short", "") ?: ""
    val masterCLong = activeNodeObj?.optString("country_long", "") ?: ""
    val masterLat = if (activeNodeObj != null && activeNodeObj.has("latency_ms") && activeNodeObj.optInt("latency_ms") > 0) {
        activeNodeObj.optInt("latency_ms")
    } else {
        activeNodeObj?.optInt("ping", 0) ?: 0
    }
    val status = vpnObj?.optString("status", "disconnected") ?: "disconnected"
    val uptimeSec = vpnObj?.optLong("uptime_seconds", 0) ?: 0
    val hours = uptimeSec / 3600
    val mins = (uptimeSec % 3600) / 60
    val uptimeStr = if (isEn) {
        if (hours > 0) "${hours}h ${mins}m" else "${mins}m"
    } else {
        if (hours > 0) "${hours}小时${mins}分" else "${mins}分钟"
    }
    val isConnected = status == "connected"
    val isZeroTun = vpnObj?.optBoolean("zero_tun", false) ?: false
    val driverMode = vpnObj?.optString("driver_mode", if (isZeroTun) "userland" else "kernel") ?: if (isZeroTun) "userland" else "kernel"
    val devLabel = when {
        isZeroTun && isEn -> "Zero-TUN Gateway"
        isZeroTun -> "主网关 (免TUN)"
        isEn -> "Primary NIC (tun0)"
        else -> "主网卡零号 (tun0)"
    }
    val disconnectedLabel = if (isEn) "Disconnected" else "未连接"
    return MasterGatewayInfo(
        devName = devLabel,
        nodeName = activeNode.ifEmpty { disconnectedLabel },
        nodeIp = masterIp,
        country = masterCShort,
        countryLong = masterCLong,
        latencyMs = masterLat,
        uptimeStr = uptimeStr,
        status = if (isConnected) (if (isEn) "Healthy" else "断流检测通过") else disconnectedLabel,
        isConnected = isConnected,
        isZeroTun = isZeroTun,
        driverMode = driverMode
    )
}
