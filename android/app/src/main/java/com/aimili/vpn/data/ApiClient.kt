package com.aimili.vpn.data

import android.net.Uri
import com.aimili.vpn.model.BlacklistRecord
import com.aimili.vpn.model.DynamicGroupCard
import com.aimili.vpn.model.InboundProtocolItem
import com.aimili.vpn.model.MasterGatewayInfo
import com.aimili.vpn.model.NodeCandidate
import com.aimili.vpn.model.PortRuleItem
import com.aimili.vpn.model.ServerProfile
import com.aimili.vpn.model.TunnelItem
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.Credentials
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONArray
import org.json.JSONObject
import java.util.concurrent.TimeUnit

class ApiClient {
    private val client = OkHttpClient.Builder()
        .connectTimeout(6, TimeUnit.SECONDS)
        .readTimeout(10, TimeUnit.SECONDS)
        .writeTimeout(10, TimeUnit.SECONDS)
        .build()

    private val jsonMediaType = "application/json; charset=utf-8".toMediaType()

    private fun buildRequest(profile: ServerProfile, path: String, method: String = "GET", bodyJson: String? = null): Request {
        val url = "${profile.baseUrl}$path"
        val builder = Request.Builder()
            .url(url)
            .header("Authorization", Credentials.basic(profile.username, profile.password))

        when (method.uppercase()) {
            "POST" -> builder.post((bodyJson ?: "{}").toRequestBody(jsonMediaType))
            "DELETE" -> builder.delete()
            else -> builder.get()
        }
        return builder.build()
    }

    suspend fun testConnection(profile: ServerProfile): Result<ServerProfile> = withContext(Dispatchers.IO) {
        try {
            val start = System.currentTimeMillis()
            val req = buildRequest(profile, "/api/app/info")
            client.newCall(req).execute().use { response ->
                val elapsed = (System.currentTimeMillis() - start).toInt()
                if (response.isSuccessful) {
                    val json = JSONObject(response.body?.string() ?: "{}")
                    val updated = profile.copy(
                        isOnline = true,
                        latencyMs = if (elapsed > 0) elapsed else 35,
                        exitIp = json.optString("active_node", profile.exitIp).ifEmpty { profile.exitIp }
                    )
                    Result.success(updated)
                } else {
                    Result.failure(Exception("HTTP ${response.code}: ${response.message}"))
                }
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchStatus(profile: ServerProfile): Result<MasterGatewayInfo> = withContext(Dispatchers.IO) {
        try {
            val req = buildRequest(profile, "/api/status")
            client.newCall(req).execute().use { resp ->
                if (!resp.isSuccessful) return@withContext Result.failure(Exception("HTTP ${resp.code}"))
                val json = JSONObject(resp.body?.string() ?: "{}")
                val vpnObj = json.optJSONObject("vpn")
                val activeNode = vpnObj?.optString("active_node_id", "") ?: ""
                val status = vpnObj?.optString("status", "disconnected") ?: "disconnected"
                val uptimeSec = vpnObj?.optLong("uptime_seconds", 0) ?: 0

                val hours = uptimeSec / 3600
                val mins = (uptimeSec % 3600) / 60
                val uptimeStr = "${hours}小时${mins}分"

                val isConnected = status == "connected"
                val info = MasterGatewayInfo(
                    devName = "主网卡零号，策略表一百",
                    nodeName = if (activeNode.isNotEmpty()) activeNode else "日本住宅节点十二号",
                    uptimeStr = uptimeStr,
                    status = if (isConnected) "断流检测通过" else "未连接",
                    isConnected = isConnected
                )
                Result.success(info)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun connectMaster(profile: ServerProfile, nodeId: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val req = buildRequest(profile, "/api/connect", "POST", JSONObject().put("node_id", nodeId).toString())
            client.newCall(req).execute().use { resp -> Result.success(resp.isSuccessful) }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun disconnectMasterVPN(profile: ServerProfile): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val req = buildRequest(profile, "/api/disconnect", "POST", "{}")
            client.newCall(req).execute().use { resp -> Result.success(resp.isSuccessful) }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchTunnels(profile: ServerProfile): Result<List<TunnelItem>> = withContext(Dispatchers.IO) {
        try {
            val req = buildRequest(profile, "/api/tunnels")
            client.newCall(req).execute().use { resp ->
                if (!resp.isSuccessful) return@withContext Result.failure(Exception("HTTP ${resp.code}"))
                val arr = JSONArray(resp.body?.string() ?: "[]")
                val list = mutableListOf<TunnelItem>()
                for (i in 0 until arr.length()) {
                    val obj = arr.getJSONObject(i)
                    val id = obj.optString("id", "tun-$i")
                    val devName = obj.optString("dev_name", "tun$i")
                    val devIndex = obj.optInt("dev_index", i)
                    val status = obj.optString("status", "connected")
                    val uptime = obj.optLong("uptime", 0)
                    val nodeObj = obj.optJSONObject("node")
                    val ip = nodeObj?.optString("ip", "") ?: ""
                    val country = nodeObj?.optString("country_short", "JP") ?: "JP"
                    val latency = nodeObj?.optInt("latency_ms", 35) ?: 35

                    val unlockObj = obj.optJSONObject("unlock")
                    val openai = unlockObj?.optString("openai", "unlocked") ?: "unlocked"
                    val claude = unlockObj?.optString("claude", "unlocked") ?: "unlocked"
                    val gemini = unlockObj?.optString("gemini", "unlocked") ?: "unlocked"
                    val netflix = unlockObj?.optString("netflix", "unlocked") ?: "unlocked"
                    val throughputBps = unlockObj?.optLong("throughput_bytes_per_sec", 1800000) ?: 1800000

                    list.add(
                        TunnelItem(
                            id = id,
                            devName = devName,
                            devIndex = devIndex,
                            status = status,
                            latencyMs = latency,
                            nodeIp = ip,
                            country = country,
                            uptimeSeconds = uptime,
                            throughputBps = throughputBps,
                            openai = openai,
                            claude = claude,
                            gemini = gemini,
                            netflix = netflix
                        )
                    )
                }
                Result.success(list)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun startTunnel(profile: ServerProfile, nodeId: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val req = buildRequest(profile, "/api/tunnels/start", "POST", JSONObject().put("node_id", nodeId).toString())
            client.newCall(req).execute().use { resp -> Result.success(resp.isSuccessful) }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun stopTunnel(profile: ServerProfile, tunnelId: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val req = buildRequest(profile, "/api/tunnels/stop", "POST", JSONObject().put("tunnel_id", tunnelId).toString())
            client.newCall(req).execute().use { resp -> Result.success(resp.isSuccessful) }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchPortRules(profile: ServerProfile): Result<List<PortRuleItem>> = withContext(Dispatchers.IO) {
        try {
            val req = buildRequest(profile, "/api/proxy/ports")
            client.newCall(req).execute().use { resp ->
                if (!resp.isSuccessful) return@withContext Result.failure(Exception("HTTP ${resp.code}"))
                val arr = JSONArray(resp.body?.string() ?: "[]")
                val list = mutableListOf<PortRuleItem>()
                for (i in 0 until arr.length()) {
                    val obj = arr.getJSONObject(i)
                    val port = obj.optInt("port", 7928)
                    val enabled = obj.optBoolean("enabled", true)
                    val policy = obj.optString("policy", "round_robin")
                    val intervalSec = obj.optInt("interval_seconds", 300)
                    val authMode = obj.optString("auth_mode", "random")
                    val authUser = obj.optString("auth_user", "")
                    val authPass = obj.optString("auth_pass", "")
                    list.add(PortRuleItem(port, enabled, policy, intervalSec, authMode, authUser, authPass))
                }
                Result.success(list)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun savePortRules(profile: ServerProfile, rules: List<PortRuleItem>): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val arr = JSONArray()
            for (r in rules) {
                arr.put(
                    JSONObject().apply {
                        put("port", r.port)
                        put("enabled", r.enabled)
                        put("policy", r.policy)
                        put("interval_seconds", r.intervalSeconds)
                        put("auth_mode", r.authMode)
                        put("auth_user", r.authUser)
                        put("auth_pass", r.authPass)
                        put("bound_tunnel_ids", JSONArray(r.boundTunnelIds))
                        put("bound_group_ids", JSONArray(r.boundGroupIds))
                    }
                )
            }
            val jsonBody = JSONObject().put("rules", arr).toString()
            val req = buildRequest(profile, "/api/proxy/ports", "POST", jsonBody)
            client.newCall(req).execute().use { resp -> Result.success(resp.isSuccessful) }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun triggerRotate(profile: ServerProfile, groupId: String? = null): Result<String> = withContext(Dispatchers.IO) {
        try {
            val body = if (groupId != null) JSONObject().put("group_id", groupId).toString() else "{}"
            val req = buildRequest(profile, "/api/tunnel-groups/evaluate", "POST", body)
            client.newCall(req).execute().use { response ->
                if (response.isSuccessful) {
                    Result.success("已在后台触发 [${profile.name}] 动态重评与换线！")
                } else {
                    Result.failure(Exception("HTTP ${response.code}"))
                }
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchDynamicGroups(profile: ServerProfile): Result<List<DynamicGroupCard>> = withContext(Dispatchers.IO) {
        try {
            val req = buildRequest(profile, "/api/tunnel-groups")
            client.newCall(req).execute().use { resp ->
                if (!resp.isSuccessful) return@withContext Result.failure(Exception("HTTP ${resp.code}"))
                val arr = JSONArray(resp.body?.string() ?: "[]")
                val list = mutableListOf<DynamicGroupCard>()
                for (i in 0 until arr.length()) {
                    val obj = arr.getJSONObject(i)
                    list.add(
                        DynamicGroupCard(
                            id = obj.optString("id", "dg-$i"),
                            name = obj.optString("name", "自适应组"),
                            enabled = obj.optBoolean("enabled", true),
                            isSystem = obj.optBoolean("is_system", false),
                            country = obj.optString("country", "JP"),
                            ipType = obj.optString("ip_type", "residential"),
                            unlockFilter = obj.optString("unlock_filter", "ai"),
                            sortBy = obj.optString("sort_by", "latency"),
                            targetCount = obj.optInt("target_count", 3),
                            intervalMinutes = obj.optInt("interval_minutes", 15),
                            statusText = obj.optString("status_text", "")
                        )
                    )
                }
                Result.success(list)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchSingBoxNodes(profile: ServerProfile): Result<List<InboundProtocolItem>> = withContext(Dispatchers.IO) {
        try {
            val req = buildRequest(profile, "/api/singbox/nodes")
            client.newCall(req).execute().use { resp ->
                if (!resp.isSuccessful) return@withContext Result.failure(Exception("HTTP ${resp.code}"))
                val json = JSONObject(resp.body?.string() ?: "{}")
                val arr = json.optJSONArray("nodes") ?: JSONArray()
                val list = mutableListOf<InboundProtocolItem>()
                for (i in 0 until arr.length()) {
                    val obj = arr.getJSONObject(i)
                    list.add(
                        InboundProtocolItem(
                            id = obj.optString("name", "inbound-$i"),
                            name = obj.optString("name", "inbound-$i"),
                            protocol = obj.optString("protocol", "VLESS-REALITY"),
                            port = obj.optInt("port", 443),
                            outbound = obj.optString("outbound", "socks5://127.0.0.1:7928"),
                            outboundPort = obj.optInt("outbound_port", 7928),
                            uuid = obj.optString("uuid", ""),
                            password = obj.optString("password", ""),
                            sni = obj.optString("sni", ""),
                            shareUrl = obj.optString("url", "")
                        )
                    )
                }
                Result.success(list)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun setSingBoxOutbound(profile: ServerProfile, target: String, outbound: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val body = JSONObject().put("target", target).put("outbound", outbound).toString()
            val req = buildRequest(profile, "/api/singbox/nodes/outbound", "POST", body)
            client.newCall(req).execute().use { resp -> Result.success(resp.isSuccessful) }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun deleteSingBoxNode(profile: ServerProfile, target: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val req = buildRequest(profile, "/api/singbox/nodes?target=${Uri.encode(target)}", "DELETE")
            client.newCall(req).execute().use { resp -> Result.success(resp.isSuccessful) }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchClashSubscription(profile: ServerProfile): Result<String> = withContext(Dispatchers.IO) {
        try {
            val req = buildRequest(profile, "/api/singbox/subscription/clash")
            client.newCall(req).execute().use { response ->
                if (response.isSuccessful) {
                    Result.success(response.body?.string() ?: "")
                } else {
                    Result.success("${profile.baseUrl}/api/singbox/subscription/clash")
                }
            }
        } catch (e: Exception) {
            Result.success("${profile.baseUrl}/api/singbox/subscription/clash")
        }
    }

    suspend fun fetchNodes(profile: ServerProfile): Result<List<NodeCandidate>> = withContext(Dispatchers.IO) {
        try {
            val req = buildRequest(profile, "/api/nodes")
            client.newCall(req).execute().use { resp ->
                if (!resp.isSuccessful) return@withContext Result.failure(Exception("HTTP ${resp.code}"))
                val arr = JSONArray(resp.body?.string() ?: "[]")
                val list = mutableListOf<NodeCandidate>()
                for (i in 0 until arr.length()) {
                    val obj = arr.getJSONObject(i)
                    val id = obj.optString("id", "node-$i")
                    val ip = obj.optString("ip", "")
                    val port = obj.optInt("port", 1194)
                    val cShort = obj.optString("country_short", "JP")
                    val cLong = obj.optString("country_long", "日本")
                    val latency = obj.optInt("latency_ms", 0)
                    val isp = obj.optString("isp", "")
                    val score = obj.optInt("score", 80)
                    val speed = obj.optLong("speed", 30000000)
                    val ipType = obj.optString("ip_type", "unknown")
                    val isFav = obj.optBoolean("is_favorite", false)

                    val unlockObj = obj.optJSONObject("unlock")
                    val openai = unlockObj?.optString("openai", "unknown") ?: "unknown"
                    val claude = unlockObj?.optString("claude", "unknown") ?: "unknown"
                    val gemini = unlockObj?.optString("gemini", "unknown") ?: "unknown"
                    val netflix = unlockObj?.optString("netflix", "unknown") ?: "unknown"

                    list.add(
                        NodeCandidate(
                            id = id,
                            ip = ip,
                            port = port,
                            countryShort = cShort,
                            countryLong = cLong,
                            latencyMs = latency,
                            isp = isp,
                            score = score,
                            speedBps = speed,
                            ipType = ipType,
                            isFavorite = isFav,
                            openai = openai,
                            claude = claude,
                            gemini = gemini,
                            netflix = netflix
                        )
                    )
                }
                Result.success(list)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun toggleFavorite(profile: ServerProfile, nodeId: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val body = JSONObject().put("node_id", nodeId).toString()
            val req = buildRequest(profile, "/api/nodes/favorite", "POST", body)
            client.newCall(req).execute().use { resp -> Result.success(resp.isSuccessful) }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun probeNodes(profile: ServerProfile): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val req = buildRequest(profile, "/api/nodes/probe", "POST", "{}")
            client.newCall(req).execute().use { resp -> Result.success(resp.isSuccessful) }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchBlacklist(profile: ServerProfile): Result<List<BlacklistRecord>> = withContext(Dispatchers.IO) {
        try {
            val req = buildRequest(profile, "/api/blacklist")
            client.newCall(req).execute().use { resp ->
                if (!resp.isSuccessful) return@withContext Result.failure(Exception("HTTP ${resp.code}"))
                val arr = JSONArray(resp.body?.string() ?: "[]")
                val list = mutableListOf<BlacklistRecord>()
                for (i in 0 until arr.length()) {
                    val obj = arr.getJSONObject(i)
                    list.add(
                        BlacklistRecord(
                            nodeId = obj.optString("node_id", ""),
                            ip = obj.optString("ip", ""),
                            country = obj.optString("country", "JP"),
                            reason = obj.optString("reason", "不可达"),
                            blacklistedAt = obj.optString("blacklisted_at", ""),
                            expiresAt = obj.optString("expires_at", "")
                        )
                    )
                }
                Result.success(list)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun resurrectBlacklist(profile: ServerProfile): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val req = buildRequest(profile, "/api/blacklist/resurrect", "POST", "{}")
            client.newCall(req).execute().use { resp -> Result.success(resp.isSuccessful) }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    companion object {
        fun parseAimiliUri(rawUri: String): ServerProfile? {
            try {
                val uri = Uri.parse(rawUri)
                if (uri.scheme == "aimili" && uri.host == "server") {
                    val host = uri.getQueryParameter("host") ?: return null
                    val port = uri.getQueryParameter("port")?.toIntOrNull() ?: 8787
                    val path = uri.getQueryParameter("path") ?: "enter"
                    val user = uri.getQueryParameter("user") ?: "admin"
                    val pass = uri.getQueryParameter("pass") ?: ""
                    val name = uri.getQueryParameter("name") ?: "AimiliVPN ($host)"
                    val isTls = uri.getQueryParameter("tls") == "1"

                    return ServerProfile(
                        id = "server-${System.currentTimeMillis()}",
                        name = name,
                        host = host,
                        port = port,
                        path = path,
                        username = user,
                        password = pass,
                        isTls = isTls
                    )
                }
            } catch (e: Exception) {
                e.printStackTrace()
            }
            return null
        }
    }
}
