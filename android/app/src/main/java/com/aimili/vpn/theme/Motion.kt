package com.aimili.vpn.theme

import androidx.compose.animation.core.FiniteAnimationSpec
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.spring
import androidx.compose.ui.unit.IntOffset

/**
 * Material 3 Expressive motion specifications.
 * Transitions and state changes have a spring effect with slight bounce/overshoot.
 */
object AimiliMotion {
    val expressiveFloat: FiniteAnimationSpec<Float> = spring(
        dampingRatio = 0.82f,
        stiffness = Spring.StiffnessMedium
    )

    val expressiveOffset: FiniteAnimationSpec<IntOffset> = spring(
        dampingRatio = 0.82f,
        stiffness = Spring.StiffnessMedium
    )

    fun <T> expressive(): FiniteAnimationSpec<T> = spring(
        dampingRatio = 0.82f,
        stiffness = Spring.StiffnessMedium
    )
}
