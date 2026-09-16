package com.aimili.vpn

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import com.aimili.vpn.data.ApiClient
import com.aimili.vpn.theme.AimiliTheme
import com.aimili.vpn.ui.navigation.MainAppScaffold

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()

        // Handle incoming intent aimili://server
        intent?.data?.let { uri ->
            ApiClient.parseAimiliUri(uri.toString())?.let { server ->
                AimiliApplication.instance.serverStore.addServer(server)
            }
        }

        setContent {
            val serverStore = AimiliApplication.instance.serverStore
            val themeMode by serverStore.themeMode.collectAsState()
            val themePalette by serverStore.themePalette.collectAsState()

            val darkTheme = when (themeMode) {
                "light" -> false
                "dark" -> true
                else -> isSystemInDarkTheme()
            }

            AimiliTheme(
                darkTheme = darkTheme,
                paletteId = themePalette
            ) {
                MainAppScaffold()
            }
        }
    }
}
