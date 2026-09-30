plugins {
    id("com.android.application") version "8.7.3" apply false
    id("org.jetbrains.kotlin.plugin.compose") version "2.0.21" apply false
    id("org.jetbrains.kotlin.android") version "2.0.21" apply false
}

val aapt2Wrapper = rootDir.resolve("tools/aapt2_wrapper")
if (aapt2Wrapper.exists()) {
    System.setProperty("android.aapt2Override", aapt2Wrapper.absolutePath)
}
