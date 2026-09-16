package com.aimili.vpn.ui.navigation

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.Orientation
import androidx.compose.foundation.gestures.draggable
import androidx.compose.foundation.gestures.rememberDraggableState
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.AltRoute
import androidx.compose.material.icons.rounded.Dashboard
import androidx.compose.material.icons.rounded.Public
import androidx.compose.material.icons.rounded.ShowChart
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import com.aimili.vpn.AimiliApplication
import com.aimili.vpn.model.ClusterSummary
import com.aimili.vpn.theme.AimiliMotion
import com.aimili.vpn.ui.screens.ClusterHubScreen
import com.aimili.vpn.ui.screens.NodeSquareScreen
import com.aimili.vpn.ui.screens.RoutingMatrixScreen
import com.aimili.vpn.ui.screens.ServerConsoleScreen
import com.aimili.vpn.ui.screens.SettingsSecurityScreen

enum class AppNavDestination(
    val title: String,
    val icon: ImageVector
) {
    Dashboard("概览", Icons.Rounded.Dashboard),
    Monitoring("监控", Icons.Rounded.ShowChart),
    Routing("调度", Icons.Rounded.AltRoute),
    Nodes("节点", Icons.Rounded.Public)
}

@Composable
fun MainAppScaffold() {
    val serverStore = AimiliApplication.instance.serverStore
    val servers by serverStore.servers.collectAsState()
    val activeServer by serverStore.activeServer.collectAsState()

    var currentDestination by remember { mutableStateOf(AppNavDestination.Dashboard) }
    var isSettingsOpen by remember { mutableStateOf(false) }

    // Dynamic cluster summary
    val clusterSummary = remember(servers) {
        ClusterSummary(
            onlineCount = servers.count { it.isOnline },
            offlineCount = servers.count { !it.isOnline },
            downSpeedStr = "10.2 兆每秒",
            upSpeedStr = "1.4 兆每秒",
            todayTrafficStr = "18.6 吉字节"
        )
    }

    // Swipe gesture handling between screens
    var dragOffset by remember { mutableFloatStateOf(0f) }
    val dragThreshold = 90f

    val swipeState = rememberDraggableState { delta ->
        dragOffset += delta
        if (dragOffset > dragThreshold) {
            dragOffset = 0f
            // Swipe right
            when (currentDestination) {
                AppNavDestination.Monitoring -> currentDestination = AppNavDestination.Dashboard
                AppNavDestination.Routing -> currentDestination = AppNavDestination.Monitoring
                AppNavDestination.Nodes -> currentDestination = AppNavDestination.Routing
                AppNavDestination.Dashboard -> {}
            }
        } else if (dragOffset < -dragThreshold) {
            dragOffset = 0f
            // Swipe left
            when (currentDestination) {
                AppNavDestination.Dashboard -> currentDestination = AppNavDestination.Monitoring
                AppNavDestination.Monitoring -> currentDestination = AppNavDestination.Routing
                AppNavDestination.Routing -> currentDestination = AppNavDestination.Nodes
                AppNavDestination.Nodes -> {}
            }
        }
    }

    Scaffold(
        modifier = Modifier
            .fillMaxSize()
            .draggable(
                state = swipeState,
                orientation = Orientation.Horizontal,
                onDragStopped = { dragOffset = 0f }
            ),
        bottomBar = {
            if (!isSettingsOpen) {
                NavigationBar(
                    modifier = Modifier.height(80.dp),
                    containerColor = MaterialTheme.colorScheme.surfaceContainer
                ) {
                    AppNavDestination.entries.forEach { destination ->
                        val isSelected = currentDestination == destination
                        NavigationBarItem(
                            selected = isSelected,
                            onClick = { currentDestination = destination },
                            icon = {
                                Icon(
                                    imageVector = destination.icon,
                                    contentDescription = destination.title,
                                    modifier = Modifier.size(24.dp)
                                )
                            },
                            label = {
                                Text(
                                    text = destination.title,
                                    style = MaterialTheme.typography.labelMedium
                                )
                            },
                            colors = NavigationBarItemDefaults.colors(
                                selectedIconColor = MaterialTheme.colorScheme.onSecondaryContainer,
                                unselectedIconColor = MaterialTheme.colorScheme.onSurfaceVariant,
                                selectedTextColor = MaterialTheme.colorScheme.onSurface,
                                unselectedTextColor = MaterialTheme.colorScheme.onSurfaceVariant,
                                indicatorColor = MaterialTheme.colorScheme.secondaryContainer
                            )
                        )
                    }
                }
            }
        },
        containerColor = MaterialTheme.colorScheme.surface
    ) { innerPadding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
        ) {
            // Main views or Settings screen with M3 Expressive animated transition
            AnimatedContent(
                targetState = isSettingsOpen to currentDestination,
                transitionSpec = {
                    if (targetState.first != initialState.first) {
                        if (targetState.first) {
                            // Slide in settings from right
                            slideInHorizontally(animationSpec = AimiliMotion.expressiveOffset) { it } togetherWith
                                    slideOutHorizontally(animationSpec = AimiliMotion.expressiveOffset) { -it / 3 }
                        } else {
                            // Slide out settings to right (reverse)
                            slideInHorizontally(animationSpec = AimiliMotion.expressiveOffset) { -it / 3 } togetherWith
                                    slideOutHorizontally(animationSpec = AimiliMotion.expressiveOffset) { it }
                        }
                    } else {
                        // Fade between tabs
                        fadeIn(animationSpec = AimiliMotion.expressiveFloat) togetherWith
                                fadeOut(animationSpec = AimiliMotion.expressiveFloat)
                    }
                },
                label = "ScreenTransition"
            ) { (settingsOpen, destination) ->
                if (settingsOpen) {
                    SettingsSecurityScreen(
                        servers = servers,
                        onBack = { isSettingsOpen = false }
                    )
                } else {
                    when (destination) {
                        AppNavDestination.Dashboard -> {
                            ClusterHubScreen(
                                servers = servers,
                                activeServer = activeServer,
                                summary = clusterSummary,
                                onSelectServerAndOpenConsole = { server ->
                                    serverStore.setActiveServer(server.id)
                                    currentDestination = AppNavDestination.Monitoring
                                },
                                onOpenSettings = { isSettingsOpen = true }
                            )
                        }
                        AppNavDestination.Monitoring -> {
                            ServerConsoleScreen(
                                activeServer = activeServer,
                                allServers = servers,
                                onSelectServer = { serverStore.setActiveServer(it) },
                                onPreviousServer = { serverStore.selectPreviousServer() },
                                onNextServer = { serverStore.selectNextServer() }
                            )
                        }
                        AppNavDestination.Routing -> {
                            RoutingMatrixScreen(
                                activeServer = activeServer,
                                allServers = servers,
                                onSelectServer = { serverStore.setActiveServer(it) },
                                onPreviousServer = { serverStore.selectPreviousServer() },
                                onNextServer = { serverStore.selectNextServer() }
                            )
                        }
                        AppNavDestination.Nodes -> {
                            NodeSquareScreen(
                                activeServer = activeServer,
                                allServers = servers,
                                onSelectServer = { serverStore.setActiveServer(it) },
                                onPreviousServer = { serverStore.selectPreviousServer() },
                                onNextServer = { serverStore.selectNextServer() }
                            )
                        }
                    }
                }
            }
        }
    }
}
