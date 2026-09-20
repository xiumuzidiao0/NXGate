package com.nxgate.app.theme

import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Shapes
import androidx.compose.ui.unit.dp

val NXGateShapes = Shapes(
    extraSmall = RoundedCornerShape(4.dp),
    small = RoundedCornerShape(8.dp),
    medium = RoundedCornerShape(16.dp),
    large = RoundedCornerShape(20.dp), // M3 Expressive Cards: 20dp
    extraLarge = RoundedCornerShape(28.dp) // M3 Expressive Dialogs & Sheets: 28dp
)

val AimiliShapes = NXGateShapes

// Capsule shape for Buttons
val CapsuleShape = CircleShape

// Helper for connected button groups in M3 Expressive:
// First item has capsule outer (left), 8dp inner (right)
// Middle items have 8dp on both sides
// Last item has 8dp inner (left), capsule outer (right)
fun connectedButtonShape(index: Int, total: Int): RoundedCornerShape {
    if (total <= 1) return RoundedCornerShape(28.dp)
    return when (index) {
        0 -> RoundedCornerShape(topStart = 28.dp, bottomStart = 28.dp, topEnd = 8.dp, bottomEnd = 8.dp)
        total - 1 -> RoundedCornerShape(topStart = 8.dp, bottomStart = 8.dp, topEnd = 28.dp, bottomEnd = 28.dp)
        else -> RoundedCornerShape(8.dp)
    }
}

// Helper for connected chip groups in M3 Expressive:
// 3dp gap, inner 4dp, outer 8dp
fun connectedChipShape(index: Int, total: Int): RoundedCornerShape {
    if (total <= 1) return RoundedCornerShape(8.dp)
    return when (index) {
        0 -> RoundedCornerShape(topStart = 8.dp, bottomStart = 8.dp, topEnd = 4.dp, bottomEnd = 4.dp)
        total - 1 -> RoundedCornerShape(topStart = 4.dp, bottomStart = 4.dp, topEnd = 8.dp, bottomEnd = 8.dp)
        else -> RoundedCornerShape(4.dp)
    }
}

// Helper for connected list items in M3 Expressive:
// 3dp gap, outer 28dp, inner 8dp
fun connectedListShape(index: Int, total: Int): RoundedCornerShape {
    if (total <= 1) return RoundedCornerShape(28.dp)
    return when (index) {
        0 -> RoundedCornerShape(topStart = 28.dp, topEnd = 28.dp, bottomStart = 8.dp, bottomEnd = 8.dp)
        total - 1 -> RoundedCornerShape(topStart = 8.dp, topEnd = 8.dp, bottomStart = 28.dp, bottomEnd = 28.dp)
        else -> RoundedCornerShape(8.dp)
    }
}
