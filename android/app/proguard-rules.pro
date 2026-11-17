# Keep app data models and API/store classes
-keep class com.nxgate.app.model.** { *; }
-keep class com.nxgate.app.data.** { *; }

# Keep manifest entrypoints and system services
-keep public class com.nxgate.app.NXGateApplication
-keep public class com.nxgate.app.MainActivity
-keep public class com.nxgate.app.widget.NXGateWidgetProvider
-keep public class com.nxgate.app.tile.NXGateTileService

# Kotlin & Coroutines reflection / fast ServiceLoader
-keepattributes *Annotation*,Signature,InnerClasses,EnclosingMethod
-keep class kotlin.Metadata { *; }
-keepnames class kotlinx.coroutines.internal.MainDispatcherFactory {}
-keepnames class kotlinx.coroutines.CoroutineExceptionHandler {}
-dontwarn kotlinx.coroutines.flow.**

# OkHttp / OkHttp-SSE / Okio
-dontwarn okhttp3.**
-dontwarn okio.**
-dontwarn javax.annotation.**
-dontwarn org.conscrypt.**
-dontwarn org.bouncycastle.**
-dontwarn org.openjsse.**

# AndroidX Biometric & Security Crypto / Tink
-keep class androidx.biometric.** { *; }
-keep class androidx.security.crypto.** { *; }
-keep class com.google.crypto.tink.** { *; }
-dontwarn com.google.crypto.tink.**

# CameraX & ZXing QR scanning
-keep class androidx.camera.** { *; }
-keep class com.google.zxing.** { *; }
-dontwarn com.google.zxing.**
