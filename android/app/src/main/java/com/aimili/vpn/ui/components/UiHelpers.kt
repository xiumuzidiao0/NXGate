package com.aimili.vpn.ui.components

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

/**
 * Universal Unicode Country Flag Generator.
 * Converts ANY 2-letter ISO 3166-1 alpha-2 code into the official national flag emoji.
 * Supports all 249 countries and territories globally.
 */
fun countryFlag(code: String): String {
    val clean = code.trim().uppercase()
    val finalCode = when (clean) {
        "UK" -> "GB"
        else -> clean
    }
    if (finalCode.length == 2 && finalCode[0] in 'A'..'Z' && finalCode[1] in 'A'..'Z') {
        val firstChar = Character.toChars(0x1F1E6 + (finalCode[0] - 'A'))
        val secondChar = Character.toChars(0x1F1E6 + (finalCode[1] - 'A'))
        return String(firstChar) + String(secondChar)
    }
    return ""
}

val COUNTRY_CHINESE_NAMES = mapOf(
    "JP" to "日本", "US" to "美国", "KR" to "韩国", "TW" to "台湾", "HK" to "香港",
    "SG" to "新加坡", "GB" to "英国", "DE" to "德国", "FR" to "法国", "CA" to "加拿大",
    "AU" to "澳大利亚", "VN" to "越南", "TH" to "泰国", "MY" to "马来西亚", "IN" to "印度",
    "RU" to "俄罗斯", "NL" to "荷兰", "BR" to "巴西", "PH" to "菲律宾", "ID" to "印尼",
    "IT" to "意大利", "ES" to "西班牙", "SE" to "瑞典", "CH" to "瑞士", "NZ" to "新西兰",
    "PL" to "波兰", "UA" to "乌克兰", "TR" to "土耳其", "ZA" to "南非", "AR" to "阿根廷",
    "CL" to "智利", "CO" to "哥伦比亚", "MX" to "墨西哥", "NO" to "挪威", "FI" to "芬兰",
    "DK" to "丹麦", "IE" to "爱尔兰", "AT" to "奥地利", "BE" to "比利时", "CZ" to "捷克",
    "RO" to "罗马尼亚", "IL" to "以色列", "AE" to "阿联酋", "SA" to "沙特阿拉伯", "CN" to "中国"
)

fun countryChineseName(code: String): String {
    return COUNTRY_CHINESE_NAMES[code.trim().uppercase()] ?: code
}

@Composable
fun UnlockPill(label: String, status: String) {
    val isOk = status == "unlocked"
    val borderColor = if (isOk) MaterialTheme.colorScheme.primary.copy(alpha = 0.5f) else MaterialTheme.colorScheme.outline
    Surface(
        shape = RoundedCornerShape(4.dp),
        color = if (isOk) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surfaceContainerHigh,
        border = BorderStroke(0.5.dp, borderColor)
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 4.dp, vertical = 2.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = label,
                style = MaterialTheme.typography.labelSmall,
                fontWeight = if (isOk) FontWeight.Bold else FontWeight.Medium,
                color = if (isOk) MaterialTheme.colorScheme.onPrimaryContainer else MaterialTheme.colorScheme.onSurface
            )
            Spacer(Modifier.width(2.dp))
            Icon(
                imageVector = if (isOk) Icons.Rounded.Check else Icons.Rounded.Close,
                contentDescription = null,
                tint = if (isOk) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface,
                modifier = Modifier.size(11.dp)
            )
        }
    }
}
