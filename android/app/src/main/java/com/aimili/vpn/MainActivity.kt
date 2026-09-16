package com.aimili.vpn

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
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
            AimiliTheme {
                MainAppScaffold()
            }
        }
    }
}
