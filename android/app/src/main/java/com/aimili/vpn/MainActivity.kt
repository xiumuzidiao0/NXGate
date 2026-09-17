package com.aimili.vpn

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
import com.aimili.vpn.data.ApiClient
import com.aimili.vpn.theme.AimiliTheme
import com.aimili.vpn.ui.components.BiometricLockOverlay
import com.aimili.vpn.ui.navigation.MainAppScaffold

class MainActivity : FragmentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        instance = this
        enableEdgeToEdge()

        // Handle incoming intent aimili://server
        intent?.data?.let { uri ->
            ApiClient.parseAimiliUri(uri.toString())?.let { server ->
                AimiliApplication.instance.serverStore.addServer(server)
            }
        }

        val serverStore = AimiliApplication.instance.serverStore
        if (serverStore.biometricEnabled.value) {
            serverStore.setAppLocked(true)
            showBiometricPrompt(
                onSuccess = { serverStore.setAppLocked(false) },
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

            AimiliTheme(
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
                                onSuccess = { serverStore.setAppLocked(false) },
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
        val serverStore = AimiliApplication.instance.serverStore
        if (serverStore.biometricEnabled.value) {
            serverStore.setAppLocked(true)
        }
    }

    override fun onStart() {
        super.onStart()
        val serverStore = AimiliApplication.instance.serverStore
        if (serverStore.biometricEnabled.value && serverStore.isAppLocked.value) {
            showBiometricPrompt(
                onSuccess = { serverStore.setAppLocked(false) },
                onError = {}
            )
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
    }
}
