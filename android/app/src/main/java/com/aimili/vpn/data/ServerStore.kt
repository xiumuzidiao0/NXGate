package com.aimili.vpn.data

import android.content.Context
import android.content.SharedPreferences
import com.aimili.vpn.model.ServerProfile
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import org.json.JSONArray
import org.json.JSONObject
import java.util.UUID

class ServerStore(context: Context) {
    private val prefs: SharedPreferences = context.getSharedPreferences("aimili_server_prefs", Context.MODE_PRIVATE)

    private val _servers = MutableStateFlow<List<ServerProfile>>(emptyList())
    val servers: StateFlow<List<ServerProfile>> = _servers.asStateFlow()

    private val _activeServer = MutableStateFlow<ServerProfile?>(null)
    val activeServer: StateFlow<ServerProfile?> = _activeServer.asStateFlow()

    private val _biometricEnabled = MutableStateFlow(true)
    val biometricEnabled: StateFlow<Boolean> = _biometricEnabled.asStateFlow()

    private val _cleartextWarningEnabled = MutableStateFlow(true)
    val cleartextWarningEnabled: StateFlow<Boolean> = _cleartextWarningEnabled.asStateFlow()

    private val _themeMode = MutableStateFlow("system") // "system", "light", "dark"
    val themeMode: StateFlow<String> = _themeMode.asStateFlow()

    private val _themePalette = MutableStateFlow("teal") // "monet", "teal", "ocean", "emerald", "purple", "amber", "rose"
    val themePalette: StateFlow<String> = _themePalette.asStateFlow()

    init {
        loadData()
    }

    private fun loadData() {
        val rawJson = prefs.getString(KEY_SERVERS, null)
        val list = mutableListOf<ServerProfile>()
        if (rawJson != null) {
            try {
                val array = JSONArray(rawJson)
                for (i in 0 until array.length()) {
                    val obj = array.getJSONObject(i)
                    list.add(deserializeServer(obj))
                }
            } catch (e: Exception) {
                e.printStackTrace()
            }
        }

        // Seed with the required real server profiles if empty
        if (list.isEmpty()) {
            list.add(
                ServerProfile(
                    id = "tokyo-residential",
                    name = "东京住宅网关",
                    host = "47.238.2.197",
                    port = 8787,
                    path = "enter",
                    username = "xmzd",
                    password = "a18979346882",
                    isTls = false,
                    latencyMs = 38,
                    isOnline = true,
                    exitIp = "114.119.18.2",
                    ipType = "原生家宽",
                    ispName = "中华电信骨干",
                    unlockStatus = "全通过",
                    downSpeedStr = "8.4 Mb/s",
                    totalTrafficStr = "12.1 Gb",
                    activeConns = 38,
                    orderIndex = 0
                )
            )
            list.add(
                ServerProfile(
                    id = "silicon-valley-ai",
                    name = "硅谷智能专属池",
                    host = "104.28.0.8",
                    port = 8787,
                    path = "enter",
                    username = "admin",
                    password = "aimilivpn",
                    isTls = true,
                    latencyMs = 142,
                    isOnline = true,
                    exitIp = "23.94.102.8",
                    ipType = "机房托管",
                    ispName = "Comcast Business",
                    unlockStatus = "通义与双子通过，克劳德阻断",
                    downSpeedStr = "2.1 Mb/s",
                    totalTrafficStr = "6.5 Gb",
                    activeConns = 12,
                    orderIndex = 1
                )
            )
            saveList(list)
        }

        _servers.value = list
        val savedActiveId = prefs.getString(KEY_ACTIVE_SERVER_ID, list.firstOrNull()?.id)
        _activeServer.value = list.find { it.id == savedActiveId } ?: list.firstOrNull()

        _biometricEnabled.value = prefs.getBoolean(KEY_BIOMETRIC, true)
        _cleartextWarningEnabled.value = prefs.getBoolean(KEY_CLEARTEXT_WARN, true)
        _themeMode.value = prefs.getString(KEY_THEME_MODE, "system") ?: "system"
        val defaultPalette = if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.S) "monet" else "teal"
        _themePalette.value = prefs.getString(KEY_THEME_PALETTE, defaultPalette) ?: defaultPalette
    }

    fun setActiveServer(id: String) {
        val target = _servers.value.find { it.id == id } ?: return
        _activeServer.value = target
        prefs.edit().putString(KEY_ACTIVE_SERVER_ID, id).apply()
    }

    fun selectPreviousServer() {
        val currentList = _servers.value
        if (currentList.size <= 1) return
        val currentIdx = currentList.indexOfFirst { it.id == _activeServer.value?.id }
        val prevIdx = if (currentIdx <= 0) currentList.size - 1 else currentIdx - 1
        setActiveServer(currentList[prevIdx].id)
    }

    fun selectNextServer() {
        val currentList = _servers.value
        if (currentList.size <= 1) return
        val currentIdx = currentList.indexOfFirst { it.id == _activeServer.value?.id }
        val nextIdx = (currentIdx + 1) % currentList.size
        setActiveServer(currentList[nextIdx].id)
    }

    fun addServer(profile: ServerProfile) {
        val current = _servers.value.toMutableList()
        current.add(profile)
        _servers.value = current
        saveList(current)
        if (_activeServer.value == null) {
            setActiveServer(profile.id)
        }
    }

    fun updateServer(profile: ServerProfile) {
        val current = _servers.value.toMutableList()
        val idx = current.indexOfFirst { it.id == profile.id }
        if (idx >= 0) {
            current[idx] = profile
            _servers.value = current
            saveList(current)
            if (_activeServer.value?.id == profile.id) {
                _activeServer.value = profile
            }
        }
    }

    fun deleteServer(id: String) {
        val current = _servers.value.toMutableList()
        current.removeAll { it.id == id }
        _servers.value = current
        saveList(current)
        if (_activeServer.value?.id == id) {
            _activeServer.value = current.firstOrNull()
            prefs.edit().putString(KEY_ACTIVE_SERVER_ID, _activeServer.value?.id).apply()
        }
    }

    fun reorderServers(fromIndex: Int, toIndex: Int) {
        val current = _servers.value.toMutableList()
        if (fromIndex in current.indices && toIndex in current.indices) {
            val item = current.removeAt(fromIndex)
            current.add(toIndex, item)
            _servers.value = current
            saveList(current)
        }
    }

    fun setBiometricEnabled(enabled: Boolean) {
        _biometricEnabled.value = enabled
        prefs.edit().putBoolean(KEY_BIOMETRIC, enabled).apply()
    }

    fun setCleartextWarningEnabled(enabled: Boolean) {
        _cleartextWarningEnabled.value = enabled
        prefs.edit().putBoolean(KEY_CLEARTEXT_WARN, enabled).apply()
    }

    fun setThemeMode(mode: String) {
        _themeMode.value = mode
        prefs.edit().putString(KEY_THEME_MODE, mode).apply()
    }

    fun setThemePalette(palette: String) {
        _themePalette.value = palette
        prefs.edit().putString(KEY_THEME_PALETTE, palette).apply()
    }

    fun updateServerTraffic(serverId: String, downSpeedStr: String, upSpeedStr: String, totalTrafficStr: String, activeConns: Int) {
        val list = _servers.value.map {
            if (it.id == serverId) {
                it.copy(
                    downSpeedStr = downSpeedStr,
                    totalTrafficStr = totalTrafficStr,
                    activeConns = activeConns
                )
            } else it
        }
        _servers.value = list
        if (_activeServer.value?.id == serverId) {
            _activeServer.value = list.find { it.id == serverId }
        }
    }

    private fun saveList(list: List<ServerProfile>) {
        val array = JSONArray()
        list.forEach { array.put(serializeServer(it)) }
        prefs.edit().putString(KEY_SERVERS, array.toString()).apply()
    }

    private fun serializeServer(p: ServerProfile): JSONObject {
        return JSONObject().apply {
            put("id", p.id)
            put("name", p.name)
            put("host", p.host)
            put("port", p.port)
            put("path", p.path)
            put("username", p.username)
            put("password", p.password)
            put("isTls", p.isTls)
            put("latencyMs", p.latencyMs)
            put("isOnline", p.isOnline)
            put("exitIp", p.exitIp)
            put("ipType", p.ipType)
            put("ispName", p.ispName)
            put("unlockStatus", p.unlockStatus)
            put("downSpeedStr", p.downSpeedStr)
            put("totalTrafficStr", p.totalTrafficStr)
            put("activeConns", p.activeConns)
            put("orderIndex", p.orderIndex)
        }
    }

    private fun deserializeServer(obj: JSONObject): ServerProfile {
        var downSpeed = obj.optString("downSpeedStr", "8.4 Mb/s")
        if (downSpeed.contains("兆每秒")) {
            downSpeed = downSpeed.replace("兆每秒", "Mb/s").trim()
        }
        var totalTraffic = obj.optString("totalTrafficStr", "12.1 Gb")
        if (totalTraffic.contains("吉字节")) {
            totalTraffic = totalTraffic.replace("吉字节", "Gb").trim()
        }

        return ServerProfile(
            id = obj.optString("id", UUID.randomUUID().toString()),
            name = obj.optString("name", "AimiliVPN 网关"),
            host = obj.optString("host", "127.0.0.1"),
            port = obj.optInt("port", 8787),
            path = obj.optString("path", "enter"),
            username = obj.optString("username", "admin"),
            password = obj.optString("password", ""),
            isTls = obj.optBoolean("isTls", false),
            latencyMs = obj.optInt("latencyMs", 38),
            isOnline = obj.optBoolean("isOnline", true),
            exitIp = obj.optString("exitIp", "114.119.18.2"),
            ipType = obj.optString("ipType", "原生家宽"),
            ispName = obj.optString("ispName", "中华电信骨干"),
            unlockStatus = obj.optString("unlockStatus", "全通过"),
            downSpeedStr = downSpeed,
            totalTrafficStr = totalTraffic,
            activeConns = obj.optInt("activeConns", 38),
            orderIndex = obj.optInt("orderIndex", 0)
        )
    }

    companion object {
        private const val KEY_SERVERS = "server_list_json"
        private const val KEY_ACTIVE_SERVER_ID = "active_server_id"
        private const val KEY_BIOMETRIC = "biometric_enabled"
        private const val KEY_CLEARTEXT_WARN = "cleartext_warn_enabled"
        private const val KEY_THEME_MODE = "theme_mode_str"
        private const val KEY_THEME_PALETTE = "theme_palette_str"
    }
}
