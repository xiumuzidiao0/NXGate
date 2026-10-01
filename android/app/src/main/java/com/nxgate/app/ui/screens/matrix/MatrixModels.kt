package com.nxgate.app.ui.screens.matrix

import com.nxgate.app.model.DynamicGroupCard
import com.nxgate.app.model.PortRuleItem

// Dropdown option data models
data class PortPolicyOption(val key: String, val label: String)
data class PortIntervalOption(val seconds: Int, val label: String)
data class PortAuthOption(val key: String, val label: String)

data class GroupCountryOption(val code: String, val label: String)
data class GroupIpTypeOption(val key: String, val label: String)
data class GroupUnlockOption(val key: String, val label: String)
data class GroupSortOption(val key: String, val label: String)
data class GroupTargetCountOption(val count: Int, val label: String)
data class GroupIntervalOption(val minutes: Int, val label: String)
data class GroupFallbackOption(val key: String, val label: String)

data class PipelineLeaf(
    val devName: String,
    val locText: String,
    val ipWithPort: String,
    val latency: Int,
    val openai: String = "unknown",
    val claude: String = "unknown",
    val gemini: String = "unknown",
    val netflix: String = "unknown"
)

data class PipelineStream(
    val port: Int,
    val proto: String = "SOCKS5",
    val authText: String,
    val isDefault: Boolean,
    val groupTitle: String,
    val groupSub: String,
    val policyLabel: String = "",
    val concurrencyText: String = "",
    val isFallback: Boolean,
    val leaves: List<PipelineLeaf>,
    val countryHint: String = "",
    val rule: PortRuleItem? = null,
    val group: DynamicGroupCard? = null
)
