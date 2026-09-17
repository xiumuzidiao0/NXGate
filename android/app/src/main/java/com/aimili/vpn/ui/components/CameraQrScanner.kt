package com.aimili.vpn.ui.components

import android.Manifest
import android.content.pm.PackageManager
import android.util.Size
import android.widget.Toast
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.camera.core.CameraSelector
import androidx.camera.core.ImageAnalysis
import androidx.camera.core.ImageProxy
import androidx.camera.core.Preview
import androidx.camera.lifecycle.ProcessCameraProvider
import androidx.camera.view.PreviewView
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.CameraAlt
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material.icons.rounded.ContentPaste
import androidx.compose.material.icons.rounded.QrCodeScanner
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.core.content.ContextCompat
import androidx.lifecycle.compose.LocalLifecycleOwner
import com.aimili.vpn.data.ApiClient
import com.aimili.vpn.model.ServerProfile
import com.google.zxing.BarcodeFormat
import com.google.zxing.BinaryBitmap
import com.google.zxing.DecodeHintType
import com.google.zxing.MultiFormatReader
import com.google.zxing.PlanarYUVLuminanceSource
import com.google.zxing.common.HybridBinarizer
import java.util.concurrent.Executors

class QrCodeAnalyzer(
    private val onQrCodeScanned: (String) -> Unit
) : ImageAnalysis.Analyzer {
    private val reader = MultiFormatReader().apply {
        val hints = mapOf(
            DecodeHintType.POSSIBLE_FORMATS to listOf(BarcodeFormat.QR_CODE),
            DecodeHintType.TRY_HARDER to true,
            DecodeHintType.CHARACTER_SET to "UTF-8"
        )
        setHints(hints)
    }
    private var isScanning = true

    override fun analyze(image: ImageProxy) {
        if (!isScanning) {
            image.close()
            return
        }

        val plane = image.planes[0]
        val buffer = plane.buffer
        val data = ByteArray(buffer.remaining())
        buffer.get(data)

        val width = image.width
        val height = image.height

        val source = PlanarYUVLuminanceSource(
            data,
            width,
            height,
            0,
            0,
            width,
            height,
            false
        )
        val binaryBitmap = BinaryBitmap(HybridBinarizer(source))

        try {
            val result = reader.decodeWithState(binaryBitmap)
            if (result != null && result.text.isNotEmpty()) {
                isScanning = false
                onQrCodeScanned(result.text)
            }
        } catch (_: Exception) {
            // QR code not in this frame
        } finally {
            reader.reset()
            image.close()
        }
    }
}

@Composable
fun CameraQrScannerDialog(
    onDismiss: () -> Unit,
    onServerScanned: (ServerProfile) -> Unit
) {
    val context = LocalContext.current
    val lifecycleOwner = LocalLifecycleOwner.current

    var hasCameraPermission by remember {
        mutableStateOf(
            ContextCompat.checkSelfPermission(context, Manifest.permission.CAMERA) == PackageManager.PERMISSION_GRANTED
        )
    }

    var manualInput by remember { mutableStateOf("") }
    var showManualMode by remember { mutableStateOf(false) }

    val permissionLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.RequestPermission()
    ) { isGranted ->
        hasCameraPermission = isGranted
        if (!isGranted) {
            Toast.makeText(context, "相机权限未授予，可手动粘贴配置链接", Toast.LENGTH_SHORT).show()
            showManualMode = true
        }
    }

    // 自动向安卓系统发起权限请求，直接弹出系统授权窗口，无需额外造按钮拦截
    LaunchedEffect(Unit) {
        if (!hasCameraPermission) {
            permissionLauncher.launch(Manifest.permission.CAMERA)
        }
    }

    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false)
    ) {
        Card(
            modifier = Modifier
                .fillMaxWidth(0.92f)
                .clip(RoundedCornerShape(28.dp)),
            shape = RoundedCornerShape(28.dp),
            colors = CardDefaults.cardColors(
                containerColor = MaterialTheme.colorScheme.surfaceContainer
            )
        ) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(20.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(14.dp)
            ) {
                // Header
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Surface(
                            shape = RoundedCornerShape(12.dp),
                            color = MaterialTheme.colorScheme.primaryContainer,
                            modifier = Modifier.size(36.dp)
                        ) {
                            Box(contentAlignment = Alignment.Center) {
                                Icon(
                                    imageVector = Icons.Rounded.QrCodeScanner,
                                    contentDescription = null,
                                    tint = MaterialTheme.colorScheme.primary,
                                    modifier = Modifier.size(20.dp)
                                )
                            }
                        }
                        Spacer(Modifier.width(10.dp))
                        Text(
                            text = if (showManualMode) "手动导入服务端" else "相机扫码添加",
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold,
                            color = MaterialTheme.colorScheme.onSurface
                        )
                    }
                    IconButton(onClick = onDismiss) {
                        Icon(Icons.Rounded.Close, contentDescription = "关闭")
                    }
                }

                if (showManualMode || !hasCameraPermission) {
                    // Manual paste mode
                    Column(
                        modifier = Modifier.fillMaxWidth(),
                        verticalArrangement = Arrangement.spacedBy(12.dp)
                    ) {
                        Text(
                            text = "请粘贴来自 Web 控制台顶栏「手机 App 绑定」生成的 aimili:// 链接：",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )

                        OutlinedTextField(
                            value = manualInput,
                            onValueChange = { manualInput = it },
                            label = { Text("aimili://server 导入链接") },
                            placeholder = { Text("aimili://server?host=...&port=8787...") },
                            singleLine = false,
                            maxLines = 4,
                            shape = RoundedCornerShape(14.dp),
                            modifier = Modifier.fillMaxWidth()
                        )

                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.spacedBy(8.dp)
                        ) {
                            if (!hasCameraPermission) {
                                OutlinedButton(
                                    onClick = { permissionLauncher.launch(Manifest.permission.CAMERA) },
                                    modifier = Modifier.weight(1f),
                                    shape = RoundedCornerShape(50)
                                ) {
                                    Icon(Icons.Rounded.CameraAlt, contentDescription = null, modifier = Modifier.size(16.dp))
                                    Spacer(Modifier.width(4.dp))
                                    Text("授予相机")
                                }
                            } else {
                                OutlinedButton(
                                    onClick = { showManualMode = false },
                                    modifier = Modifier.weight(1f),
                                    shape = RoundedCornerShape(50)
                                ) {
                                    Icon(Icons.Rounded.CameraAlt, contentDescription = null, modifier = Modifier.size(16.dp))
                                    Spacer(Modifier.width(4.dp))
                                    Text("切换相机")
                                }
                            }

                            Button(
                                onClick = {
                                    val parsed = ApiClient.parseAimiliUri(manualInput.trim())
                                    if (parsed != null) {
                                        onServerScanned(parsed)
                                    } else {
                                        Toast.makeText(context, "无效的 aimili://server 导入协议", Toast.LENGTH_SHORT).show()
                                    }
                                },
                                modifier = Modifier.weight(1f),
                                shape = RoundedCornerShape(50),
                                colors = ButtonDefaults.buttonColors(
                                    containerColor = MaterialTheme.colorScheme.primary,
                                    contentColor = MaterialTheme.colorScheme.onPrimary
                                )
                            ) {
                                Icon(Icons.Rounded.ContentPaste, contentDescription = null, modifier = Modifier.size(16.dp))
                                Spacer(Modifier.width(4.dp))
                                Text("解析导入")
                            }
                        }
                    }
                } else {
                    // Real-time camera scanner view
                    val cameraExecutor = remember { Executors.newSingleThreadExecutor() }
                    DisposableEffect(Unit) {
                        onDispose {
                            cameraExecutor.shutdown()
                        }
                    }

                    Box(
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(280.dp)
                            .clip(RoundedCornerShape(20.dp))
                            .background(Color.Black),
                        contentAlignment = Alignment.Center
                    ) {
                        AndroidView(
                            factory = { ctx ->
                                val previewView = PreviewView(ctx)
                                val cameraProviderFuture = ProcessCameraProvider.getInstance(ctx)
                                cameraProviderFuture.addListener({
                                    val cameraProvider = cameraProviderFuture.get()
                                    val preview = Preview.Builder().build().also {
                                        it.surfaceProvider = previewView.surfaceProvider
                                    }

                                    val imageAnalysis = ImageAnalysis.Builder()
                                        .setTargetResolution(Size(1280, 720))
                                        .setBackpressureStrategy(ImageAnalysis.STRATEGY_KEEP_ONLY_LATEST)
                                        .build()

                                    imageAnalysis.setAnalyzer(
                                        cameraExecutor,
                                        QrCodeAnalyzer { rawText ->
                                            val profile = ApiClient.parseAimiliUri(rawText)
                                            if (profile != null) {
                                                previewView.post {
                                                    onServerScanned(profile)
                                                }
                                            }
                                        }
                                    )

                                    val cameraSelector = CameraSelector.DEFAULT_BACK_CAMERA

                                    try {
                                        cameraProvider.unbindAll()
                                        cameraProvider.bindToLifecycle(
                                            lifecycleOwner,
                                            cameraSelector,
                                            preview,
                                            imageAnalysis
                                        )
                                    } catch (e: Exception) {
                                        e.printStackTrace()
                                    }
                                }, ContextCompat.getMainExecutor(ctx))
                                previewView
                            },
                            modifier = Modifier.fillMaxSize()
                        )

                        // Visual scanning reticle
                        Box(
                            modifier = Modifier
                                .size(190.dp)
                                .border(BorderStroke(2.5.dp, MaterialTheme.colorScheme.primary), RoundedCornerShape(16.dp))
                        )

                        Text(
                            text = "将镜头对准 Web 控制台的二维码",
                            style = MaterialTheme.typography.labelSmall,
                            color = Color.White.copy(alpha = 0.85f),
                            modifier = Modifier
                                .align(Alignment.BottomCenter)
                                .padding(bottom = 12.dp)
                        )
                    }

                    TextButton(onClick = { showManualMode = true }) {
                        Text("无法识别？点击切换手动粘贴", style = MaterialTheme.typography.labelMedium)
                    }
                }
            }
        }
    }
}
