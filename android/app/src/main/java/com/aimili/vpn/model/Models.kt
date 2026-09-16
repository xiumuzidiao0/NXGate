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
            val scheme = if (isTls) "https" else "http"
            val cleanPath = path.trim().trim('/')
            return if (cleanPath.isNotEmpty()) {
                "$scheme://$host:$port/$cleanPath"
            } else {
                "$scheme://$host:$port"
            }
        }
}

data class ClusterSummary(
    val onlineCount: Int = 3,
    val offlineCount: Int = 0,
    val downSpeedStr: String = "10.2 兆每秒",
    val upSpeedStr: String = "1.4 兆每秒",
    val todayTrafficStr: String = "18.6 吉字节"
)

data class MasterGatewayInfo(
    val devName: String = "主网卡零号，策略表一百",
    val nodeName: String = "日本住宅节点十二号",
    val uptimeStr: String = "18小时24分",
    val status: String = "断流检测通过",
    val isConnected: Boolean = true
)

data class TunnelItem(
    val id: String,
    val title: String, // e.g. "并发隧道一号"
    val latencyMs: Int, // e.g. 42
    val subtitle: String // e.g. "日本出口，智能解锁全通过，吞吐 1.8 兆每秒"
)

data class PortRuleItem(
    val port: Int,
    val enabled: Boolean,
    val policy: String, // "轮询", "定时轮换 300 秒", "随机"
    val authMode: String, // "随机账密", "自定义账密", "免密直连"
    val binding: String // "绑定全部健康隧道", "绑定日本住宅组", "绑定美国备用组"
) {
    val title: String
        get() = "端口 $port ${if (enabled) "已启用" else "已停用"}"

    val subtitle: String
        get() = "策略：$policy；鉴权：$authMode；$binding"
}

data class DynamicGroupCard(
    val id: String,
    val title: String, // "日本前三住宅组"
    val country: String = "日本",
    val netType: String = "住宅宽带",
    val unlockReq: String = "智能服务全通过",
    val targetCount: Int = 3,
    val evalMinutes: Int = 15
) {
    val description: String
        get() = "国家目标：$country\n网络类型：$netType\n解锁要求：$unlockReq\n维持数量：$targetCount 个并发，评估周期：$evalMinutes 分钟"
}

data class InboundProtocolItem(
    val id: String,
    val title: String, // "入站协议：真实传输"
    val subtitle: String // "外部端口 443，出口指向端口 7928 日本住宅池"
)

data class NodeCandidate(
    val id: String,
    val title: String, // "日本节点 220.92.176.236"
    val latencyMs: Int,
    val isp: String,
    val score: Int,
    val ipType: String,
    val speedStr: String,
    val unlockStr: String,
    val isStarred: Boolean = false
) {
    val description: String
        get() = "运营商：$isp，信誉评分 $score\n属性：$ipType，速度 $speedStr\n实测解锁：$unlockStr"
}
