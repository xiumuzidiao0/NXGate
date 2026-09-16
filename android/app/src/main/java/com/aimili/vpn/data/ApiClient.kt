package com.aimili.vpn.data

import android.net.Uri
import android.util.Log
import com.aimili.vpn.model.AvailableOutbound
import com.aimili.vpn.model.BlacklistRecord
import com.aimili.vpn.model.DynamicGroupCard
import com.aimili.vpn.model.InboundProtocolItem
import com.aimili.vpn.model.LiveTrafficInfo
import com.aimili.vpn.model.MasterGatewayInfo
import com.aimili.vpn.model.NodeCandidate
import com.aimili.vpn.model.PortRuleItem
import com.aimili.vpn.model.ServerProfile
import com.aimili.vpn.model.ServerStatusData
import com.aimili.vpn.model.SingBoxOverviewData
import com.aimili.vpn.model.SystemLogEntry
import com.aimili.vpn.model.TunnelItem
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.Credentials
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import org.json.JSONArray
import org.json.JSONObject
import java.util.concurrent.TimeUnit

class ApiClient {
    private val client = OkHttpClient.Builder()
        .connectTimeout(15, TimeUnit.SECONDS)
        .readTimeout(45, TimeUnit.SECONDS)
        .writeTimeout(30, TimeUnit.SECONDS)
        .build()

    private val jsonMediaType = "application/json; charset=utf-8".toMediaType()

    private fun buildRequest(profile: ServerProfile, path: String, method: String = "GET", bodyJson: String? = null, bypassPath: Boolean = false): Request {
        val base = if (bypassPath) {
            val scheme = if (profile.isTls) "https" else "http"
            var rawHost = profile.host.trim().removePrefix("http://").removePrefix("https://").trim('/')
            var finalPort = profile.port
            if (rawHost.contains(":")) {
                val parts = rawHost.split(":")
                rawHost = parts[0]
                val p = parts.getOrNull(1)?.toIntOrNull()
                if (p != null && p > 0) finalPort = p
            }
            "$scheme://$rawHost:$finalPort"
        } else {
            profile.baseUrl
        }

        val url = "$base$path"
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

    private fun executeCall(profile: ServerProfile, path: String, method: String = "GET", bodyJson: String? = null): Response {
        val req1 = buildRequest(profile, path, method, bodyJson, bypassPath = false)
        val resp1 = client.newCall(req1).execute()
        if (resp1.code == 404 && profile.path.trim().trim('/').isNotEmpty()) {
            resp1.close()
            val req2 = buildRequest(profile, path, method, bodyJson, bypassPath = true)
            return client.newCall(req2).execute()
        }
        return resp1
    }

    suspend fun testConnection(profile: ServerProfile): Result<ServerProfile> = withContext(Dispatchers.IO) {
        try {
            val start = System.currentTimeMillis()
            executeCall(profile, "/api/app/info").use { response ->
                val elapsed = (System.currentTimeMillis() - start).toInt()
                if (response.isSuccessful) {
                    val json = JSONObject(response.body?.string() ?: "{}")
                    val downBps = json.optLong("download_speed_bps", 0)
                    val totalMb = json.optLong("total_download_mb", 0) + json.optLong("total_upload_mb", 0)
                    val downStr = if (downBps > 0) "%.2f Mb/s".format((downBps * 8.0) / 1_000_000.0) else profile.downSpeedStr
                    val totalStr = if (totalMb > 0) {
                        if (totalMb >= 1024) "%.2f Gb".format((totalMb * 8.0) / 1024.0) else "%.1f Mb".format(totalMb * 8.0)
                    } else profile.totalTrafficStr

                    val updated = profile.copy(
                        isOnline = true,
                        latencyMs = if (elapsed > 0) elapsed else 35,
                        exitIp = json.optString("active_node", profile.exitIp).ifEmpty { profile.exitIp },
                        activeConns = json.optInt("tunnels_count", profile.activeConns),
                        downSpeedStr = downStr,
                        totalTrafficStr = totalStr
                    )
                    Result.success(updated)
                } else {
                    Result.failure(Exception("HTTP ${response.code}: ${response.message}"))
                }
            }
        } catch (e: Exception) {
            Log.e("ApiClient", "testConnection failed for ${profile.host}", e)
            Result.failure(e)
        }
    }

    suspend fun fetchLogs(profile: ServerProfile): Result<List<SystemLogEntry>> = withContext(Dispatchers.IO) {
        try {
            executeCall(profile, "/api/logs").use { resp ->
                if (!resp.isSuccessful) return@withContext Result.failure(Exception("HTTP ${resp.code}: ${resp.message}"))
                val arr = JSONArray(resp.body?.string() ?: "[]")
                val list = mutableListOf<SystemLogEntry>()
                for (i in 0 until arr.length()) {
                    val obj = arr.getJSONObject(i)
                    list.add(
                        SystemLogEntry(
                            timestamp = obj.optString("timestamp", ""),
                            level = obj.optString("level", "INFO"),
                            module = obj.optString("module", "System"),
                            message = obj.optString("message", "")
                        )
                    )
                }
                Result.success(list)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchServerStatus(profile: ServerProfile): Result<ServerStatusData> = withContext(Dispatchers.IO) {
        try {
            executeCall(profile, "/api/status").use { resp ->
                if (!resp.isSuccessful) return@withContext Result.failure(Exception("HTTP ${resp.code}: ${resp.message}"))
                val json = JSONObject(resp.body?.string() ?: "{}")

                // 1. VPN Master info
                val vpnObj = json.optJSONObject("vpn")
                val activeNode = vpnObj?.optString("active_node_id", "") ?: ""
                val status = vpnObj?.optString("status", "disconnected") ?: "disconnected"
                val uptimeSec = vpnObj?.optLong("uptime_seconds", 0) ?: 0

                val hours = uptimeSec / 3600
                val mins = (uptimeSec % 3600) / 60
                val uptimeStr = if (hours > 0) "${hours}小时${mins}分" else "${mins}分钟"

                val isConnected = status == "connected"
                val masterInfo = MasterGatewayInfo(
                    devName = "主网卡零号 (tun0)",
                    nodeName = if (activeNode.isNotEmpty()) activeNode else "未连接",
                    uptimeStr = uptimeStr,
                    status = if (isConnected) "断流检测通过" else "未连接",
                    isConnected = isConnected
                )

                // 2. Real-time Traffic snapshot
                val trafficObj = json.optJSONObject("traffic")
                val traffic = if (trafficObj != null) {
                    LiveTrafficInfo(
                        downloadSpeedBps = trafficObj.optLong("download_speed_bps", 0),
                        uploadSpeedBps = trafficObj.optLong("upload_speed_bps", 0),
                        totalDownloadBytes = trafficObj.optLong("total_download_bytes", 0),
                        totalUploadBytes = trafficObj.optLong("total_upload_bytes", 0),
                        activeConnections = trafficObj.optInt("active_connections", 0)
                    )
                } else {
                    LiveTrafficInfo()
                }

                // 3. Tunnels
                val tunnelsArr = json.optJSONArray("tunnels")
                val tunnelsList = mutableListOf<TunnelItem>()
                if (tunnelsArr != null) {
                    for (i in 0 until tunnelsArr.length()) {
                        val obj = tunnelsArr.getJSONObject(i)
                        val id = obj.optString("id", "")
                        val devName = obj.optString("dev_name", "tun$i")
                        val devIndex = obj.optInt("dev_index", i)
                        val tStatus = obj.optString("status", "connected")
                        val uptime = obj.optLong("uptime", 0)
                        val latency = obj.optInt("latency_ms", 38)
                        val nodeObj = obj.optJSONObject("node")
                        val ip = nodeObj?.optString("ip", "") ?: ""
                        val port = nodeObj?.optInt("port", 443) ?: 443
                        val country = nodeObj?.optString("country_long", nodeObj.optString("country_short", "JP")) ?: "JP"

                        val unlockObj = obj.optJSONObject("unlock")
                        val throughputBps = unlockObj?.optLong("throughput_bps", 0) ?: 0
                        val throughputPassed = unlockObj?.optBoolean("throughput_passed", true) ?: true

                        tunnelsList.add(
                            TunnelItem(
                                id = id,
                                devName = devName,
                                devIndex = devIndex,
                                status = tStatus,
                                uptimeSeconds = uptime,
                                latencyMs = latency,
                                nodeIp = ip,
                                nodePort = port,
                                country = country,
                                throughputBps = throughputBps,
                                throughputPassed = throughputPassed,
                                openai = unlockObj?.optString("openai", "unknown") ?: "unknown",
                                claude = unlockObj?.optString("claude", "unknown") ?: "unknown",
                                gemini = unlockObj?.optString("gemini", "unknown") ?: "unknown",
                                netflix = unlockObj?.optString("netflix", "unknown") ?: "unknown"
                            )
                        )
                    }
                }

                Result.success(ServerStatusData(masterInfo, traffic, tunnelsList))
            }
        } catch (e: Exception) {
            Log.e("ApiClient", "fetchServerStatus failed", e)
            Result.failure(e)
        }
    }

    suspend fun fetchStatus(profile: ServerProfile): Result<MasterGatewayInfo> = withContext(Dispatchers.IO) {
        fetchServerStatus(profile).map { it.masterGateway }
    }

    suspend fun connectMaster(profile: ServerProfile, nodeId: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            executeCall(profile, "/api/connect", "POST", JSONObject().put("node_id", nodeId).toString()).use { resp ->
                Result.success(resp.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun disconnectMasterVPN(profile: ServerProfile): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            executeCall(profile, "/api/disconnect", "POST", "{}").use { resp ->
                Result.success(resp.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchTunnels(profile: ServerProfile): Result<List<TunnelItem>> = withContext(Dispatchers.IO) {
        try {
            executeCall(profile, "/api/tunnels").use { resp ->
                if (!resp.isSuccessful) return@withContext Result.failure(Exception("HTTP ${resp.code}: ${resp.message}"))
                val rawStr = resp.body?.string() ?: "[]"
                val arr = JSONArray(rawStr)
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
                    val port = nodeObj?.optInt("port", 443) ?: 443
                    val country = nodeObj?.optString("country_short", "JP") ?: "JP"
                    val latency = nodeObj?.optInt("latency_ms", 35) ?: 35

                    val unlockObj = obj.optJSONObject("unlock")
                    val openai = unlockObj?.optString("openai", "unknown") ?: "unknown"
                    val claude = unlockObj?.optString("claude", "unknown") ?: "unknown"
                    val gemini = unlockObj?.optString("gemini", "unknown") ?: "unknown"
                    val netflix = unlockObj?.optString("netflix", "unknown") ?: "unknown"
                    val throughputBps = unlockObj?.optLong("throughput_bytes_per_sec", 0) ?: 0
                    val throughputPassed = unlockObj?.optBoolean("throughput_passed", true) ?: true

                    list.add(
                        TunnelItem(
                            id = id,
                            devName = devName,
                            devIndex = devIndex,
                            status = status,
                            latencyMs = latency,
                            nodeIp = ip,
                            nodePort = port,
                            country = country,
                            uptimeSeconds = uptime,
                            throughputBps = throughputBps,
                            throughputPassed = throughputPassed,
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
            Log.e("ApiClient", "fetchTunnels failed", e)
            Result.failure(e)
        }
    }

    suspend fun startTunnel(profile: ServerProfile, nodeId: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            executeCall(profile, "/api/tunnels/start", "POST", JSONObject().put("node_id", nodeId).toString()).use { resp ->
                Result.success(resp.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun stopTunnel(profile: ServerProfile, tunnelId: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            executeCall(profile, "/api/tunnels/stop", "POST", JSONObject().put("tunnel_id", tunnelId).toString()).use { resp ->
                Result.success(resp.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun probeTunnelUnlock(profile: ServerProfile, tunnelId: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            executeCall(profile, "/api/unlock/probe", "POST", JSONObject().put("tunnel_id", tunnelId).toString()).use { resp ->
                Result.success(resp.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchPortRules(profile: ServerProfile): Result<List<PortRuleItem>> = withContext(Dispatchers.IO) {
        try {
            executeCall(profile, "/api/proxy/ports").use { resp ->
                if (!resp.isSuccessful) return@withContext Result.failure(Exception("HTTP ${resp.code}: ${resp.message}"))
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

                    val bgArr = obj.optJSONArray("bound_group_ids")
                    val boundGroups = mutableListOf<String>()
                    if (bgArr != null) {
                        for (j in 0 until bgArr.length()) boundGroups.add(bgArr.getString(j))
                    }

                    val btArr = obj.optJSONArray("bound_tunnel_ids")
                    val boundTuns = mutableListOf<String>()
                    if (btArr != null) {
                        for (j in 0 until btArr.length()) boundTuns.add(btArr.getString(j))
                    }

                    list.add(
                        PortRuleItem(
                            port = port,
                            enabled = enabled,
                            policy = policy,
                            intervalSeconds = intervalSec,
                            authMode = authMode,
                            authUser = authUser,
                            authPass = authPass,
                            boundGroupIds = boundGroups,
                            boundTunnelIds = boundTuns
                        )
                    )
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
            executeCall(profile, "/api/proxy/ports", "POST", jsonBody).use { resp ->
                Result.success(resp.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun triggerRotate(profile: ServerProfile, groupId: String? = null): Result<String> = withContext(Dispatchers.IO) {
        try {
            val body = if (groupId != null) JSONObject().put("group_id", groupId).toString() else "{}"
            executeCall(profile, "/api/tunnel-groups/evaluate", "POST", body).use { response ->
                if (response.isSuccessful) {
                    Result.success("已在后台触发 [${profile.name}] 动态重评与换线！")
                } else {
                    Result.failure(Exception("HTTP ${response.code}: ${response.message}"))
                }
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchDynamicGroups(profile: ServerProfile): Result<List<DynamicGroupCard>> = withContext(Dispatchers.IO) {
        try {
            executeCall(profile, "/api/tunnel-groups").use { resp ->
                if (!resp.isSuccessful) return@withContext Result.failure(Exception("HTTP ${resp.code}: ${resp.message}"))
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

    suspend fun saveDynamicGroup(profile: ServerProfile, group: DynamicGroupCard): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val json = JSONObject().apply {
                put("id", group.id)
                put("name", group.name)
                put("enabled", group.enabled)
                put("is_system", group.isSystem)
                put("country", group.country)
                put("ip_type", group.ipType)
                put("unlock_filter", group.unlockFilter)
                put("sort_by", group.sortBy)
                put("target_count", group.targetCount)
                put("interval_minutes", group.intervalMinutes)
            }
            executeCall(profile, "/api/tunnel-groups", "POST", json.toString()).use { resp ->
                Result.success(resp.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun deleteDynamicGroup(profile: ServerProfile, groupId: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            executeCall(profile, "/api/tunnel-groups?id=${Uri.encode(groupId)}", "DELETE").use { resp ->
                Result.success(resp.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchSingBoxOverview(profile: ServerProfile): Result<SingBoxOverviewData> = withContext(Dispatchers.IO) {
        try {
            executeCall(profile, "/api/singbox/overview").use { resp ->
                if (!resp.isSuccessful) return@withContext Result.failure(Exception("HTTP ${resp.code}: ${resp.message}"))
                val json = JSONObject(resp.body?.string() ?: "{}")
                val nodeArr = json.optJSONArray("nodes") ?: JSONArray()
                val obArr = json.optJSONArray("available_outbounds") ?: JSONArray()

                val outboundsList = mutableListOf<AvailableOutbound>()
                for (i in 0 until obArr.length()) {
                    val ob = obArr.getJSONObject(i)
                    outboundsList.add(
                        AvailableOutbound(
                            port = ob.optInt("port", 0),
                            addr = ob.optString("addr", "direct"),
                            label = ob.optString("label", "直连出口"),
                            isDefault = ob.optBoolean("is_default", false)
                        )
                    )
                }

                val nodesList = mutableListOf<InboundProtocolItem>()
                for (i in 0 until nodeArr.length()) {
                    val obj = nodeArr.getJSONObject(i)
                    val rawOutbound = obj.optString("outbound", "direct")
                    val matchedLabel = outboundsList.find { isOutboundMatch(it.addr, rawOutbound) }?.label ?: ""

                    nodesList.add(
                        InboundProtocolItem(
                            id = obj.optString("name", "inbound-$i"),
                            name = obj.optString("name", "inbound-$i"),
                            protocol = obj.optString("protocol", "VLESS-REALITY"),
                            port = obj.optInt("port", 443),
                            outbound = rawOutbound,
                            outboundPort = obj.optInt("outbound_port", 0),
                            outboundLabel = matchedLabel,
                            uuid = obj.optString("uuid", ""),
                            password = obj.optString("password", ""),
                            sni = obj.optString("sni", ""),
                            shareUrl = obj.optString("url", "")
                        )
                    )
                }
                Result.success(SingBoxOverviewData(nodesList, outboundsList))
            }
        } catch (e: Exception) {
            Log.e("ApiClient", "fetchSingBoxOverview failed", e)
            Result.failure(e)
        }
    }

    private fun isOutboundMatch(availableAddr: String, nodeOutbound: String): Boolean {
        if (availableAddr.equals(nodeOutbound, ignoreCase = true)) return true
        val cleanA = availableAddr.replace("socks5://", "").replace("socks://", "").trim('/')
        val cleanB = nodeOutbound.replace("socks5://", "").replace("socks://", "").trim('/')
        if (cleanA.equals(cleanB, ignoreCase = true)) return true
        val hostPortA = if (cleanA.contains("@")) cleanA.substringAfter("@") else cleanA
        val hostPortB = if (cleanB.contains("@")) cleanB.substringAfter("@") else cleanB
        return hostPortA.equals(hostPortB, ignoreCase = true)
    }

    suspend fun fetchSingBoxNodes(profile: ServerProfile): Result<List<InboundProtocolItem>> = withContext(Dispatchers.IO) {
        fetchSingBoxOverview(profile).map { it.nodes }
    }

    suspend fun addSingBoxNode(profile: ServerProfile, protocol: String, port: String, outbound: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val json = JSONObject().apply {
                put("protocol", protocol)
                put("port", port)
                put("outbound", outbound)
            }
            executeCall(profile, "/api/singbox/nodes", "POST", json.toString()).use { resp ->
                Result.success(resp.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun setSingBoxOutbound(profile: ServerProfile, target: String, outbound: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val body = JSONObject().put("target", target).put("outbound", outbound).toString()
            executeCall(profile, "/api/singbox/nodes/outbound", "POST", body).use { resp ->
                Result.success(resp.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun deleteSingBoxNode(profile: ServerProfile, target: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            executeCall(profile, "/api/singbox/nodes?target=${Uri.encode(target)}", "DELETE").use { resp ->
                Result.success(resp.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchClashSubscription(profile: ServerProfile): Result<String> = withContext(Dispatchers.IO) {
        try {
            executeCall(profile, "/api/singbox/subscription/clash").use { response ->
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
            executeCall(profile, "/api/nodes").use { resp ->
                if (!resp.isSuccessful) return@withContext Result.failure(Exception("HTTP ${resp.code}: ${resp.message}"))
                val rawStr = resp.body?.string() ?: "[]"
                val arr = JSONArray(rawStr)
                val list = ArrayList<NodeCandidate>(arr.length())
                for (i in 0 until arr.length()) {
                    val obj = arr.getJSONObject(i)
                    val id = obj.optString("id", "node-$i")
                    val ip = obj.optString("ip", "")
                    val port = obj.optInt("port", 1194)
                    val cShort = obj.optString("country_short", "JP")
                    val cLong = obj.optString("country_long", "日本")
                    val latency = obj.optInt("latency_ms", 0)
                    val isp = obj.optString("isp", "")
                    val score = obj.optLong("score", 80).toInt()
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
            Log.e("ApiClient", "fetchNodes failed", e)
            Result.failure(e)
        }
    }

    suspend fun toggleFavorite(profile: ServerProfile, nodeId: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            executeCall(profile, "/api/nodes/favorite", "POST", JSONObject().put("node_id", nodeId).toString()).use { resp ->
                Result.success(resp.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun probeNodes(profile: ServerProfile): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            executeCall(profile, "/api/nodes/probe", "POST", "{}").use { resp ->
                Result.success(resp.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchBlacklist(profile: ServerProfile): Result<List<BlacklistRecord>> = withContext(Dispatchers.IO) {
        try {
            executeCall(profile, "/api/blacklist").use { resp ->
                if (!resp.isSuccessful) return@withContext Result.failure(Exception("HTTP ${resp.code}: ${resp.message}"))
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

    suspend fun addBlacklist(profile: ServerProfile, nodeId: String, ip: String, country: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val body = JSONObject().put("node_id", nodeId).put("ip", ip).put("country", country).toString()
            executeCall(profile, "/api/blacklist", "POST", body).use { resp ->
                Result.success(resp.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun removeBlacklist(profile: ServerProfile, nodeId: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val body = JSONObject().put("node_id", nodeId).toString()
            executeCall(profile, "/api/blacklist/remove", "POST", body).use { resp ->
                Result.success(resp.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun resurrectBlacklist(profile: ServerProfile): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            executeCall(profile, "/api/blacklist/resurrect", "POST", "{}").use { resp ->
                Result.success(resp.isSuccessful)
            }
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
