package com.nxgate.app

import android.Manifest
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import android.widget.Toast
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.biometric.BiometricManager
import androidx.biometric.BiometricPrompt
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.core.content.ContextCompat
import androidx.fragment.app.FragmentActivity
import com.nxgate.app.data.ApiClient
import com.nxgate.app.theme.NXGateTheme
import com.nxgate.app.ui.components.BiometricLockOverlay
import com.nxgate.app.ui.navigation.MainAppScaffold

class MainActivity : FragmentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        instance = this
        enableEdgeToEdge()

        // Handle incoming intent aimili://server
        intent?.data?.let { uri ->
            ApiClient.parseAimiliUri(uri.toString())?.let { server ->
                NXGateApplication.instance.serverStore.addServer(server)
            }
        }

        // Request Notification permission on Android 13+
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            if (ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED) {
                requestPermissions(arrayOf(Manifest.permission.POST_NOTIFICATIONS), 101)
            }
        }

        val serverStore = NXGateApplication.instance.serverStore
        if (savedInstanceState == null && serverStore.biometricEnabled.value) {
            serverStore.setAppLocked(true)
            showBiometricPrompt(
                onSuccess = {
                    serverStore.setAppLocked(false)
                    lastBackgroundTimestamp = 0L
                },
                onError = { err ->
                    Toast.makeText(this, err, Toast.LENGTH_SHORT).show()
                }
            )
        }

        setContent {
            val themeMode by serverStore.themeMode.collectAsState()
            val themePalette by serverStore.themePalette.collectAsState()
            val themeAccent by serverStore.themeAccent.collectAsState()
            val themeBase by serverStore.themeBase.collectAsState()
            val isAppLocked by serverStore.isAppLocked.collectAsState()

            val darkTheme = when (themeMode) {
                "light" -> false
                "dark" -> true
                else -> isSystemInDarkTheme()
            }

            NXGateTheme(
                darkTheme = darkTheme,
                paletteId = themePalette,
                accentId = themeAccent,
                baseId = themeBase
            ) {
                Box(modifier = Modifier.fillMaxSize()) {
                    MainAppScaffold()
                    BiometricLockOverlay(
                        isLocked = isAppLocked,
                        onUnlockRequested = {
                            showBiometricPrompt(
                                onSuccess = {
                                    serverStore.setAppLocked(false)
                                    lastBackgroundTimestamp = 0L
                                },
                                onError = { err ->
                                    Toast.makeText(this@MainActivity, err, Toast.LENGTH_SHORT).show()
                                }
                            )
                        }
                    )
                }
            }
        }
    }

    override fun onStop() {
        super.onStop()
        lastBackgroundTimestamp = System.currentTimeMillis()
    }

    override fun onStart() {
        super.onStart()
        val serverStore = NXGateApplication.instance.serverStore
        if (serverStore.biometricEnabled.value) {
            val now = System.currentTimeMillis()
            // 切出后台超过 60 秒 (60,000ms) 时触发锁屏
            if (lastBackgroundTimestamp > 0 && (now - lastBackgroundTimestamp) >= 60_000L) {
                serverStore.setAppLocked(true)
            }
            if (serverStore.isAppLocked.value) {
                showBiometricPrompt(
                    onSuccess = {
                        serverStore.setAppLocked(false)
                        lastBackgroundTimestamp = 0L
                    },
                    onError = {}
                )
            }
        }
    }

    override fun onDestroy() {
        super.onDestroy()
        if (instance == this) instance = null
    }

    fun showBiometricPrompt(onSuccess: () -> Unit, onError: (String) -> Unit = {}) {
        val executor = ContextCompat.getMainExecutor(this)
        val promptInfo = BiometricPrompt.PromptInfo.Builder()
            .setTitle("生物识别安全验证")
            .setSubtitle("请验证指纹、面容或系统安全锁以解锁 NXGate")
            .setAllowedAuthenticators(
                BiometricManager.Authenticators.BIOMETRIC_STRONG or
                        BiometricManager.Authenticators.BIOMETRIC_WEAK or
                        BiometricManager.Authenticators.DEVICE_CREDENTIAL
            )
            .build()

        val biometricPrompt = BiometricPrompt(this, executor, object : BiometricPrompt.AuthenticationCallback() {
            override fun onAuthenticationSucceeded(result: BiometricPrompt.AuthenticationResult) {
                super.onAuthenticationSucceeded(result)
                onSuccess()
            }

            override fun onAuthenticationError(errorCode: Int, errString: CharSequence) {
                super.onAuthenticationError(errorCode, errString)
                onError(errString.toString())
            }

            override fun onAuthenticationFailed() {
                super.onAuthenticationFailed()
                onError("指纹/面容不匹配，请重试")
            }
        })

        try {
            biometricPrompt.authenticate(promptInfo)
        } catch (e: Exception) {
            onError(e.message ?: "生物识别模块调用失败")
        }
    }

    companion object {
        var instance: MainActivity? = null
            private set
        private var lastBackgroundTimestamp: Long = 0L
    }
}
