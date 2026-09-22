package com.nxgate.app.data

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import com.nxgate.app.model.ServerProfile
import com.nxgate.app.service.NXGateWidgetProvider
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import org.json.JSONArray
import org.json.JSONObject
import java.util.UUID

class ServerStore(private val context: Context) {
    private val prefs: SharedPreferences = run {
        try {
            val masterKey = MasterKey.Builder(context)
                .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
                .build()
            val encPrefs = EncryptedSharedPreferences.create(
                context,
                "nxgate_server_secure_prefs",
                masterKey,
                EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
                EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
            )
            // 自动从旧版存储无缝平滑迁移
            val oldAimiliSecure = context.getSharedPreferences("aimili_server_secure_prefs", Context.MODE_PRIVATE)
            val oldPrefs = context.getSharedPreferences("aimili_server_prefs", Context.MODE_PRIVATE)
            val sourcePrefs = if (oldAimiliSecure.contains(KEY_SERVERS)) oldAimiliSecure else oldPrefs
            if (!encPrefs.contains(KEY_SERVERS) && sourcePrefs.contains(KEY_SERVERS)) {
                val oldJson = sourcePrefs.getString(KEY_SERVERS, null)
                val oldActive = sourcePrefs.getString(KEY_ACTIVE_SERVER_ID, null)
                val oldBio = sourcePrefs.getBoolean(KEY_BIOMETRIC, true)
                val oldBioTimeout = sourcePrefs.getInt(KEY_BIOMETRIC_TIMEOUT, 60)
                val oldClear = sourcePrefs.getBoolean(KEY_CLEARTEXT_WARN, true)
                val oldMode = sourcePrefs.getString(KEY_THEME_MODE, "system")
                val oldPalette = sourcePrefs.getString(KEY_THEME_PALETTE, "teal")
                val oldAccent = sourcePrefs.getString(KEY_THEME_ACCENT, "teal")
                val oldBase = sourcePrefs.getString(KEY_THEME_BASE, "neutral")

                encPrefs.edit().apply {
                    putString(KEY_SERVERS, oldJson)
                    putString(KEY_ACTIVE_SERVER_ID, oldActive)
                    putBoolean(KEY_BIOMETRIC, oldBio)
                    putInt(KEY_BIOMETRIC_TIMEOUT, oldBioTimeout)
                    putBoolean(KEY_CLEARTEXT_WARN, oldClear)
                    putString(KEY_THEME_MODE, oldMode)
                    putString(KEY_THEME_PALETTE, oldPalette)
                    putString(KEY_THEME_ACCENT, oldAccent)
                    putString(KEY_THEME_BASE, oldBase)
                    apply()
                }
            }
            encPrefs
        } catch (e: Exception) {
            e.printStackTrace()
            context.getSharedPreferences("nxgate_server_prefs", Context.MODE_PRIVATE)
        }
    }

    private val _servers = MutableStateFlow<List<ServerProfile>>(emptyList())
    val servers: StateFlow<List<ServerProfile>> = _servers.asStateFlow()

    private val _activeServer = MutableStateFlow<ServerProfile?>(null)
    val activeServer: StateFlow<ServerProfile?> = _activeServer.asStateFlow()

    private val _biometricEnabled = MutableStateFlow(true)
    val biometricEnabled: StateFlow<Boolean> = _biometricEnabled.asStateFlow()

    private val _biometricTimeoutSeconds = MutableStateFlow(60)
    val biometricTimeoutSeconds: StateFlow<Int> = _biometricTimeoutSeconds.asStateFlow()

    private val _cleartextWarningEnabled = MutableStateFlow(true)
    val cleartextWarningEnabled: StateFlow<Boolean> = _cleartextWarningEnabled.asStateFlow()

    private val _isAppLocked = MutableStateFlow(false)
    val isAppLocked: StateFlow<Boolean> = _isAppLocked.asStateFlow()

    fun setAppLocked(locked: Boolean) {
        _isAppLocked.value = locked
    }

    private val _themeMode = MutableStateFlow("system") // "system", "light", "dark"
    val themeMode: StateFlow<String> = _themeMode.asStateFlow()

    private val _themePalette = MutableStateFlow("pixel") // "monet", "pixel", "miuix", "teal", "ocean", etc.
    val themePalette: StateFlow<String> = _themePalette.asStateFlow()

    private val _themeAccent = MutableStateFlow("teal") // "teal", "bay_blue", "miuix_blue", "mint", "coral", "iris", "amber", "rose"
    val themeAccent: StateFlow<String> = _themeAccent.asStateFlow()

    private val _themeBase = MutableStateFlow("neutral") // "neutral", "slate", "sand", "oled"
    val themeBase: StateFlow<String> = _themeBase.asStateFlow()

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

        _servers.value = list
        val savedActiveId = prefs.getString(KEY_ACTIVE_SERVER_ID, list.firstOrNull()?.id)
        _activeServer.value = list.find { it.id == savedActiveId } ?: list.firstOrNull()

        _biometricEnabled.value = prefs.getBoolean(KEY_BIOMETRIC, true)
        _biometricTimeoutSeconds.value = prefs.getInt(KEY_BIOMETRIC_TIMEOUT, 60)
        _cleartextWarningEnabled.value = prefs.getBoolean(KEY_CLEARTEXT_WARN, true)
        _themeMode.value = prefs.getString(KEY_THEME_MODE, "system") ?: "system"
        val defaultPalette = if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.S) "monet" else "teal"
        _themePalette.value = prefs.getString(KEY_THEME_PALETTE, defaultPalette) ?: defaultPalette
        _themeAccent.value = prefs.getString(KEY_THEME_ACCENT, "teal") ?: "teal"
        _themeBase.value = prefs.getString(KEY_THEME_BASE, "neutral") ?: "neutral"
    }

    fun setActiveServer(id: String) {
        val target = _servers.value.find { it.id == id } ?: return
        _activeServer.value = target
        prefs.edit().putString(KEY_ACTIVE_SERVER_ID, id).apply()
        NXGateWidgetProvider.updateAllWidgets(context)
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
        } else {
            NXGateWidgetProvider.updateAllWidgets(context)
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
            NXGateWidgetProvider.updateAllWidgets(context)
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
        NXGateWidgetProvider.updateAllWidgets(context)
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

    fun setBiometricTimeoutSeconds(seconds: Int) {
        _biometricTimeoutSeconds.value = seconds
        prefs.edit().putInt(KEY_BIOMETRIC_TIMEOUT, seconds).apply()
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

    fun setThemeAccent(accent: String) {
        _themeAccent.value = accent
        prefs.edit().putString(KEY_THEME_ACCENT, accent).apply()
    }

    fun setThemeBase(base: String) {
        _themeBase.value = base
        prefs.edit().putString(KEY_THEME_BASE, base).apply()
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

    /**
     * 导出全量网关集群配置为标准 JSON 字符串
     */
    fun exportClusterJson(includePasswords: Boolean = true): String {
        val array = JSONArray()
        _servers.value.forEach { server ->
            val obj = serializeServer(server)
            if (!includePasswords) {
                obj.put("password", "")
            }
            array.put(obj)
        }
        val wrapper = JSONObject().apply {
            put("version", 1)
            put("app", "nxgate")
            put("exported_at", System.currentTimeMillis())
            put("servers_count", array.length())
            put("servers", array)
        }
        return wrapper.toString(2)
    }

    /**
     * 从 JSON 文本批量导入网关配置（重复网关根据 host:port 自动覆盖更新）
     */
    fun importClusterJson(jsonStr: String): Result<Int> {
        return try {
            val trimmed = jsonStr.trim()
            val serversArr = if (trimmed.startsWith("{")) {
                val root = JSONObject(trimmed)
                if (root.has("servers")) root.getJSONArray("servers") else return Result.failure(Exception("缺少 servers 节点"))
            } else if (trimmed.startsWith("[")) {
                JSONArray(trimmed)
            } else {
                return Result.failure(Exception("无法识别的备份 JSON 格式"))
            }

            var imported = 0
            val currentList = _servers.value.toMutableList()
            for (i in 0 until serversArr.length()) {
                val obj = serversArr.getJSONObject(i)
                val newProfile = deserializeServer(obj)
                val existingIndex = currentList.indexOfFirst { it.host == newProfile.host && it.port == newProfile.port }
                if (existingIndex >= 0) {
                    currentList[existingIndex] = newProfile
                } else {
                    currentList.add(newProfile)
                }
                imported++
            }
            _servers.value = currentList
            saveList(currentList)
            if (_activeServer.value == null) {
                _activeServer.value = currentList.firstOrNull()
            }
            Result.success(imported)
        } catch (e: Exception) {
            Result.failure(e)
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
            put("allowInsecureTls", p.allowInsecureTls)
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
        var downSpeed = obj.optString("downSpeedStr", "0.0 Mb/s")
        if (downSpeed.contains("兆每秒")) {
            downSpeed = downSpeed.replace("兆每秒", "Mb/s").trim()
        }
        var totalTraffic = obj.optString("totalTrafficStr", "0.0 Mb")
        if (totalTraffic.contains("吉字节")) {
            totalTraffic = totalTraffic.replace("吉字节", "Gb").trim()
        }

        return ServerProfile(
            id = obj.optString("id", UUID.randomUUID().toString()),
            name = obj.optString("name", "NXGate 网关"),
            host = obj.optString("host", "127.0.0.1"),
            port = obj.optInt("port", 8787),
            path = obj.optString("path", "enter"),
            username = obj.optString("username", "admin"),
            password = obj.optString("password", ""),
            isTls = obj.optBoolean("isTls", false),
            allowInsecureTls = obj.optBoolean("allowInsecureTls", false),
            latencyMs = obj.optInt("latencyMs", 0),
            isOnline = obj.optBoolean("isOnline", false),
            exitIp = obj.optString("exitIp", ""),
            ipType = obj.optString("ipType", ""),
            ispName = obj.optString("ispName", ""),
            unlockStatus = obj.optString("unlockStatus", ""),
            downSpeedStr = downSpeed,
            totalTrafficStr = totalTraffic,
            activeConns = obj.optInt("activeConns", 0),
            orderIndex = obj.optInt("orderIndex", 0)
        )
    }

    companion object {
        private const val KEY_SERVERS = "server_list_json"
        private const val KEY_ACTIVE_SERVER_ID = "active_server_id"
        private const val KEY_BIOMETRIC = "biometric_enabled"
        private const val KEY_BIOMETRIC_TIMEOUT = "biometric_timeout_seconds"
        private const val KEY_CLEARTEXT_WARN = "cleartext_warn_enabled"
        private const val KEY_THEME_MODE = "theme_mode_str"
        private const val KEY_THEME_PALETTE = "theme_palette_str"
        private const val KEY_THEME_ACCENT = "theme_accent_str"
        private const val KEY_THEME_BASE = "theme_base_str"
    }
}
