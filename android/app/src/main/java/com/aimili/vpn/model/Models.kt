package com.aimili.vpn.model

data class ServerProfile(
    val id: String,
    val name: String,
    val host: String,
    val port: Int,
    val path: String,
    val username: String,
    val password: String,
    val isTls: Boolean = false,
    val latencyMs: Int = 38,
    val isOnline: Boolean = true,
    val exitIp: String = "114.119.18.2",
    val ipType: String = "原生家宽",
    val ispName: String = "中华电信骨干",
    val unlockStatus: String = "全通过",
    val downSpeedStr: String = "8.4 兆每秒",
    val totalTrafficStr: String = "12.1 吉字节",
    val activeConns: Int = 38,
    val orderIndex: Int = 0
) {
    val baseUrl: String
        get() {
            var rawHost = host.trim()
            var scheme = if (isTls) "https" else "http"
            if (rawHost.startsWith("http://", ignoreCase = true)) {
                scheme = "http"
                rawHost = rawHost.substring(7)
            } else if (rawHost.startsWith("https://", ignoreCase = true)) {
                scheme = "https"
                rawHost = rawHost.substring(8)
            }
            rawHost = rawHost.trim('/')
            var finalPort = port
            if (rawHost.contains(":")) {
                val parts = rawHost.split(":")
                rawHost = parts[0]
                val p = parts.getOrNull(1)?.toIntOrNull()
                if (p != null && p > 0) {
                    finalPort = p
                }
            }
            val cleanPath = path.trim().trim('/')
            return if (cleanPath.isNotEmpty()) {
                "$scheme://$rawHost:$finalPort/$cleanPath"
            } else {
                "$scheme://$rawHost:$finalPort"
            }
        }
}

data class ClusterSummary(
    val onlineCount: Int = 0,
    val offlineCount: Int = 0,
    val downSpeedStr: String = "0 B/s",
    val upSpeedStr: String = "0 B/s",
    val todayTrafficStr: String = "0 B"
)

data class MasterGatewayInfo(
    val devName: String = "tun0",
    val nodeName: String = "未连接",
    val nodeIp: String = "",
    val country: String = "",
    val uptimeStr: String = "0秒",
    val status: String = "未连接",
    val isConnected: Boolean = false
)

data class AvailableOutbound(
    val port: Int,
    val addr: String,
    val label: String,
    val isDefault: Boolean = false
)

data class SingBoxOverviewData(
    val nodes: List<InboundProtocolItem>,
    val availableOutbounds: List<AvailableOutbound>
)

data class TunnelItem(
    val id: String,
    val devName: String,
    val devIndex: Int = 0,
    val status: String = "connected",
    val latencyMs: Int = 0,
    val nodeIp: String = "",
    val nodePort: Int = 443,
    val country: String = "",
    val uptimeSeconds: Long = 0,
    val throughputBps: Long = 0,
    val throughputPassed: Boolean = true,
    val openai: String = "unknown",
    val claude: String = "unknown",
    val gemini: String = "unknown",
    val netflix: String = "unknown"
) {
    val isMaster: Boolean
        get() = devIndex == 0 || devName == "tun0"

    val title: String
        get() = "$devName ${if (isMaster) "(系统主网关出口)" else "(并发独立出口)"} • 延迟${if (latencyMs > 0) "${latencyMs}ms" else "就绪"}"

    val subtitle: String
        get() {
            val c = if (country.isNotEmpty()) "$country 出口 " else ""
            val tp = if (throughputBps > 0) "• 吞吐 %.1f MB/s".format(throughputBps / 1000000.0) else ""
            val hours = uptimeSeconds / 3600
            val mins = (uptimeSeconds % 3600) / 60
            val up = if (hours > 0) "${hours}h${mins}m" else "${mins}m"
            return "$c($nodeIp) • 运行 $up $tp"
        }
}

data class PortRuleItem(
    val port: Int,
    val enabled: Boolean = true,
    val policy: String = "round_robin", // "round_robin", "interval", "random"
    val intervalSeconds: Int = 300,
    val authMode: String = "random", // "none", "random", "custom"
    val authUser: String = "",
    val authPass: String = "",
    val boundTunnelIds: List<String> = emptyList(),
    val boundGroupIds: List<String> = emptyList()
) {
    val policyDisplay: String
        get() = when (policy) {
            "round_robin" -> "轮询"
            "interval" -> "定时轮换 ${intervalSeconds}秒"
            "random" -> "随机"
            else -> policy
        }

    val authDisplay: String
        get() = when (authMode) {
            "random" -> "随机账密"
            "custom" -> "自定义账密"
            "none" -> "免密直连"
            else -> authMode
        }

    val bindingDisplay: String
        get() = when {
            boundGroupIds.isNotEmpty() -> "绑定 ${boundGroupIds.size} 个自适应组"
            boundTunnelIds.isNotEmpty() -> "绑定 ${boundTunnelIds.size} 个指定隧道"
            else -> "绑定全部健康隧道"
        }

    val title: String
        get() = "端口 $port ${if (enabled) "已启用" else "已停用"}"

    val subtitle: String
        get() = "策略：$policyDisplay；鉴权：$authDisplay；$bindingDisplay"
}

data class DynamicGroupCard(
    val id: String,
    val name: String,
    val enabled: Boolean = true,
    val isSystem: Boolean = false,
    val country: String = "",
    val ipType: String = "all", // "all", "residential", "hosting"
    val unlockFilter: String = "none", // "none", "ai", "streaming", "all"
    val sortBy: String = "latency", // "latency", "speed", "score"
    val targetCount: Int = 1,
    val intervalMinutes: Int = 30,
    val activeTunnelIds: List<String> = emptyList(),
    val statusText: String = ""
) {
    val title: String
        get() = name

    val description: String
        get() {
            val c = if (country.isNotEmpty()) country else "全部国家"
            val ip = when (ipType) {
                "residential" -> "住宅宽带"
                "hosting" -> "机房托管"
                else -> "全部网络"
            }
            val ul = when (unlockFilter) {
                "ai" -> "智能服务全通过"
                "streaming" -> "流媒体全通过"
                "all" -> "智能与流媒体全解"
                else -> "不限"
            }
            return "国家目标：$c\n网络类型：$ip\n解锁要求：$ul\n维持数量：$targetCount 个并发，评估周期：$intervalMinutes 分钟"
        }
}

data class InboundProtocolItem(
    val id: String,
    val name: String,
    val protocol: String,
    val port: Int,
    val outbound: String,
    val outboundPort: Int = 0,
    val outboundLabel: String = "",
    val uuid: String = "",
    val password: String = "",
    val sni: String = "",
    val shareUrl: String = ""
) {
    val title: String
        get() = "入站协议: $protocol (外部端口 $port)"

    val subtitle: String
        get() {
            val ob = if (outboundLabel.isNotEmpty()) outboundLabel else (if (outbound == "direct") "直连出口 (VPS 本机原生网络)" else "出口指向: $outbound")
            return "外部端口 $port • $ob"
        }
}

data class NodeCandidate(
    val id: String,
    val ip: String,
    val port: Int,
    val countryShort: String,
    val countryLong: String = "",
    val latencyMs: Int = 0,
    val isp: String = "",
    val score: Int = 0,
    val speedBps: Long = 0,
    val ipType: String = "unknown",
    val isFavorite: Boolean = false,
    val openai: String = "unknown",
    val claude: String = "unknown",
    val gemini: String = "unknown",
    val netflix: String = "unknown"
) {
    val title: String
        get() = "${countryLong}节点 $ip 延迟${if (latencyMs > 0) "${latencyMs}毫秒" else "待测"}"

    val speedMbStr: String
        get() = "%.1f 兆每秒".format(speedBps / 1000000.0)

    val ipTypeDisplay: String
        get() = when (ipType) {
            "residential" -> "原生家宽"
            "hosting" -> "机房托管"
            else -> "普通网络"
        }

    val unlockDisplay: String
        get() {
            val aiOk = openai == "unlocked" && claude == "unlocked" && gemini == "unlocked"
            return if (aiOk && netflix == "unlocked") {
                "智能服务与流媒体全通过"
            } else if (aiOk) {
                "三大智能服务全通过"
            } else if (openai == "unlocked") {
                "通义与双子通过，克劳德阻断"
            } else {
                "部分受阻"
            }
        }

    val description: String
        get() = "运营商：${isp.ifEmpty { "骨干运营商" }}，信誉评分 $score\n属性：$ipTypeDisplay，速度 $speedMbStr\n实测解锁：$unlockDisplay"
}

data class BlacklistRecord(
    val nodeId: String,
    val ip: String,
    val country: String,
    val reason: String,
    val blacklistedAt: String = "",
    val expiresAt: String = ""
)
