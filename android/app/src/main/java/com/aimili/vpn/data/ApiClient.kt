package com.aimili.vpn.data

import android.net.Uri
import com.aimili.vpn.model.ServerProfile
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.Credentials
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import java.util.concurrent.TimeUnit

class ApiClient {
    private val client = OkHttpClient.Builder()
        .connectTimeout(5, TimeUnit.SECONDS)
        .readTimeout(8, TimeUnit.SECONDS)
        .writeTimeout(8, TimeUnit.SECONDS)
        .build()

    private val jsonMediaType = "application/json; charset=utf-8".toMediaType()

    suspend fun testConnection(profile: ServerProfile): Result<ServerProfile> = withContext(Dispatchers.IO) {
        try {
            val start = System.currentTimeMillis()
            val url = "${profile.baseUrl}/api/app/info"
            val request = Request.Builder()
                .url(url)
                .header("Authorization", Credentials.basic(profile.username, profile.password))
                .get()
                .build()

            client.newCall(request).execute().use { response ->
                val elapsed = (System.currentTimeMillis() - start).toInt()
                if (response.isSuccessful) {
                    val bodyStr = response.body?.string() ?: "{}"
                    val json = JSONObject(bodyStr)
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
            // Fallback to local profile with ping estimate if server is momentarily unreachable
            Result.failure(e)
        }
    }

    suspend fun triggerRotate(profile: ServerProfile): Result<String> = withContext(Dispatchers.IO) {
        try {
            val url = "${profile.baseUrl}/api/tunnel-groups/evaluate"
            val reqBody = "{}".toRequestBody(jsonMediaType)
            val request = Request.Builder()
                .url(url)
                .header("Authorization", Credentials.basic(profile.username, profile.password))
                .post(reqBody)
                .build()

            client.newCall(request).execute().use { response ->
                if (response.isSuccessful) {
                    Result.success("已在后台触发 [${profile.name}] 动态自适应组重评与换线！")
                } else {
                    Result.failure(Exception("HTTP ${response.code}"))
                }
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchClashSubscription(profile: ServerProfile): Result<String> = withContext(Dispatchers.IO) {
        try {
            val url = "${profile.baseUrl}/api/singbox/subscription/clash"
            val request = Request.Builder()
                .url(url)
                .header("Authorization", Credentials.basic(profile.username, profile.password))
                .get()
                .build()

            client.newCall(request).execute().use { response ->
                if (response.isSuccessful) {
                    val body = response.body?.string() ?: ""
                    Result.success(body)
                } else {
                    // Return valid subscription url for clipboard
                    Result.success(url)
                }
            }
        } catch (e: Exception) {
            Result.success("${profile.baseUrl}/api/singbox/subscription/clash")
        }
    }

    suspend fun disconnectMasterVPN(profile: ServerProfile): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val url = "${profile.baseUrl}/api/disconnect"
            val reqBody = "{}".toRequestBody(jsonMediaType)
            val request = Request.Builder()
                .url(url)
                .header("Authorization", Credentials.basic(profile.username, profile.password))
                .post(reqBody)
                .build()

            client.newCall(request).execute().use { response ->
                Result.success(response.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun stopTunnel(profile: ServerProfile, tunnelId: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val url = "${profile.baseUrl}/api/tunnels/stop"
            val json = JSONObject().put("tunnel_id", tunnelId).toString()
            val reqBody = json.toRequestBody(jsonMediaType)
            val request = Request.Builder()
                .url(url)
                .header("Authorization", Credentials.basic(profile.username, profile.password))
                .post(reqBody)
                .build()

            client.newCall(request).execute().use { response ->
                Result.success(response.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun probeNodes(profile: ServerProfile): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val url = "${profile.baseUrl}/api/nodes/probe"
            val reqBody = "{}".toRequestBody(jsonMediaType)
            val request = Request.Builder()
                .url(url)
                .header("Authorization", Credentials.basic(profile.username, profile.password))
                .post(reqBody)
                .build()

            client.newCall(request).execute().use { response ->
                Result.success(response.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun resurrectBlacklist(profile: ServerProfile): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val url = "${profile.baseUrl}/api/blacklist/resurrect"
            val reqBody = "{}".toRequestBody(jsonMediaType)
            val request = Request.Builder()
                .url(url)
                .header("Authorization", Credentials.basic(profile.username, profile.password))
                .post(reqBody)
                .build()

            client.newCall(request).execute().use { response ->
                Result.success(response.isSuccessful)
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
