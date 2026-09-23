package com.nxgate.app.ui.navigation

import android.content.res.Configuration
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.gestures.Orientation
import androidx.compose.foundation.gestures.draggable
import androidx.compose.foundation.gestures.rememberDraggableState
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.AltRoute
import androidx.compose.material.icons.rounded.Dashboard
import androidx.compose.material.icons.rounded.Public
import androidx.compose.material.icons.rounded.ShowChart
import androidx.compose.material.icons.rounded.Tune
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.NavigationRail
import androidx.compose.material3.NavigationRailItem
import androidx.compose.material3.NavigationRailItemDefaults
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
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.unit.dp
import com.nxgate.app.NXGateApplication
import com.nxgate.app.model.ClusterSummary
import com.nxgate.app.theme.NXGateMotion
import com.nxgate.app.ui.screens.ClusterHubScreen
import com.nxgate.app.ui.screens.NodeSquareScreen
import com.nxgate.app.ui.screens.RoutingMatrixScreen
import com.nxgate.app.ui.screens.ServerConsoleScreen
import com.nxgate.app.ui.screens.ServerSystemScreen
import com.nxgate.app.ui.screens.SettingsSecurityScreen

enum class AppNavDestination(
    val title: String,
    val icon: ImageVector
) {
    Dashboard("概览", Icons.Rounded.Dashboard),
    Monitoring("监控", Icons.Rounded.ShowChart),
    Routing("调度", Icons.Rounded.AltRoute),
    Nodes("节点", Icons.Rounded.Public),
    System("系统", Icons.Rounded.Tune)
}

@Composable
fun MainAppScaffold() {
    val serverStore = NXGateApplication.instance.serverStore
    val servers by serverStore.servers.collectAsState()
    val activeServer by serverStore.activeServer.collectAsState()

    var currentDestination by remember { mutableStateOf(AppNavDestination.Dashboard) }
    var isSettingsOpen by remember { mutableStateOf(false) }

    val configuration = LocalConfiguration.current
    val isLandscape = configuration.orientation == Configuration.ORIENTATION_LANDSCAPE
    val isTablet = configuration.screenWidthDp >= 600
    val useNavRail = isTablet && isLandscape

    // Dynamic cluster summary
    val clusterSummary = remember(servers, activeServer) {
        val currServer = activeServer ?: servers.firstOrNull()
        ClusterSummary(
            onlineCount = servers.count { it.isOnline },
            offlineCount = servers.count { !it.isOnline },
            downSpeedStr = currServer?.downSpeedStr ?: "0.0 Mb/s",
            upSpeedStr = "0.0 Mb/s",
            todayTrafficStr = currServer?.totalTrafficStr ?: "0.0 Gb"
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
                AppNavDestination.System -> currentDestination = AppNavDestination.Nodes
                AppNavDestination.Dashboard -> {}
            }
        } else if (dragOffset < -dragThreshold) {
            dragOffset = 0f
            // Swipe left
            when (currentDestination) {
                AppNavDestination.Dashboard -> currentDestination = AppNavDestination.Monitoring
                AppNavDestination.Monitoring -> currentDestination = AppNavDestination.Routing
                AppNavDestination.Routing -> currentDestination = AppNavDestination.Nodes
                AppNavDestination.Nodes -> currentDestination = AppNavDestination.System
                AppNavDestination.System -> {}
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
            if (!isSettingsOpen && !useNavRail) {
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
                                unselectedIconColor = MaterialTheme.colorScheme.onSurface,
                                selectedTextColor = MaterialTheme.colorScheme.onSurface,
                                unselectedTextColor = MaterialTheme.colorScheme.onSurface,
                                indicatorColor = MaterialTheme.colorScheme.secondaryContainer
                            )
                        )
                    }
                }
            }
        },
        containerColor = MaterialTheme.colorScheme.surface
    ) { innerPadding ->
        Row(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
        ) {
            // Adaptive Navigation Rail for Tablet in Landscape mode
            if (!isSettingsOpen && useNavRail) {
                NavigationRail(
                    modifier = Modifier
                        .width(80.dp)
                        .fillMaxHeight(),
                    containerColor = MaterialTheme.colorScheme.surfaceContainer
                ) {
                    AppNavDestination.entries.forEach { destination ->
                        val isSelected = currentDestination == destination
                        NavigationRailItem(
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
                            colors = NavigationRailItemDefaults.colors(
                                selectedIconColor = MaterialTheme.colorScheme.onSecondaryContainer,
                                unselectedIconColor = MaterialTheme.colorScheme.onSurface,
                                selectedTextColor = MaterialTheme.colorScheme.onSurface,
                                unselectedTextColor = MaterialTheme.colorScheme.onSurface,
                                indicatorColor = MaterialTheme.colorScheme.secondaryContainer
                            )
                        )
                    }
                }
            }

            Box(
                modifier = Modifier
                    .weight(1f)
                    .fillMaxHeight()
            ) {
                // Main views or Settings screen with M3 Expressive animated transition
                AnimatedContent(
                    targetState = isSettingsOpen to currentDestination,
                    transitionSpec = {
                        if (targetState.first != initialState.first) {
                            if (targetState.first) {
                                slideInHorizontally(animationSpec = NXGateMotion.expressiveOffset) { it } togetherWith
                                        slideOutHorizontally(animationSpec = NXGateMotion.expressiveOffset) { -it / 3 }
                            } else {
                                slideInHorizontally(animationSpec = NXGateMotion.expressiveOffset) { -it / 3 } togetherWith
                                        slideOutHorizontally(animationSpec = NXGateMotion.expressiveOffset) { it }
                            }
                        } else {
                            fadeIn(animationSpec = NXGateMotion.expressiveFloat) togetherWith
                                    fadeOut(animationSpec = NXGateMotion.expressiveFloat)
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
                            AppNavDestination.System -> {
                                ServerSystemScreen(
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
}
