package com.aimili.vpn.ui.components

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

fun countryFlag(code: String): String {
    return when (code.uppercase()) {
        "JP" -> "🇯🇵"
        "US" -> "🇺🇸"
        "KR" -> "🇰🇷"
        "SG" -> "🇸🇬"
        "HK" -> "🇭🇰"
        "TW" -> "🇹🇼"
        "DE" -> "🇩🇪"
        "GB", "UK" -> "🇬🇧"
        "CA" -> "🇨🇦"
        "AU" -> "🇦🇺"
        else -> "🌐"
    }
}

@Composable
fun UnlockPill(label: String, status: String) {
    val isOk = status == "unlocked"
    val color = if (isOk) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outline
    Surface(
        shape = RoundedCornerShape(4.dp),
        color = if (isOk) MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.6f) else MaterialTheme.colorScheme.surfaceContainerHigh,
        border = BorderStroke(0.5.dp, color.copy(alpha = 0.5f))
    ) {
        Text(
            text = "$label ${if (isOk) "✓" else "✕"}",
            style = MaterialTheme.typography.labelSmall,
            fontWeight = if (isOk) FontWeight.Bold else FontWeight.Normal,
            color = if (isOk) MaterialTheme.colorScheme.onPrimaryContainer else MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(horizontal = 4.dp, vertical = 1.dp)
        )
    }
}
