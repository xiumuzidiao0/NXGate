# freesub 项目技术亮点与可借鉴之处

## 一、项目概述

**freesub** 是一个基于 GitHub Actions 的免费节点自动测活订阅池项目，核心特点：
- 🔄 **全自动化**：每 6 小时自动抓取、测活、分类、导出
- 🎯 **真实测活**：使用 sing-box v1.14.0 内核建立真实代理隧道进行物理验证
- 🏠 **住宅宽带识别**：六重信号甄别真实家宽 IP（housing/mobile/CDN/ASN/rDNS/欺诈分）
- 📦 **多格式导出**：同时生成 Clash、V2RayN、sing-box 三种格式订阅
- 🌍 **按国家分类**：基于真实出口 IP（非入口 IP）进行精准国家归类

---

## 二、核心技术亮点对比

### 2.1 真实测活与分层超时

**freesub 实现**：
```python
# 分层超时策略
PROBE_TIMEOUT = 12          # 首击探测 12s（容纳慢启动节点）
PROBE_RETRY_TIMEOUT = 4     # 重试探测 4s（快速放弃死节点）
PORT_KNOCK_TIMEOUT = 2.5    # 端口预检 2.5s
```

**优势**：
- ✅ 避免慢启动节点被误杀（首次给足 12 秒）
- ✅ 死节点快速淘汰（重试仅 4 秒，不浪费 CI 时间）
- ✅ 端口预检快速削减 90% 无效节点

**AimiliVPN 当前实现**：
- ✅ 已实现分层超时：`FirstAttemptTimeout: 12s` + `RetryTimeout: 4s`
- ✅ 已实现住宅宽带检测器（6 重信号）
- ⚠️ **可借鉴**：增加端口预检阶段（TCP/QUIC 快速握手），在 OpenVPN 拨号前就淘汰死端口

---

### 2.2 断流检测与 MITM 识别

**freesub 实现**：
```python
# 断流检测：Cloudflare 限时下载
SPEED_TEST_BYTES = 2_500_000          # 下载 2.5MB
SPEED_TEST_BUDGET = 5.0               # 时间预算 5 秒
SPEED_MIN_BYTES_PER_S = 70_000        # 吞吐 < 70KB/s 判定断流

# MITM 识别：TLS 证书校验
# cloudflare.com/cdn-cgi/trace 返回 tls=VERIFIED 才通过
```

**优势**：
- ✅ 拦截"连上但带宽趋零"的断流节点
- ✅ 识别证书劫持节点（中间人攻击）

**AimiliVPN 可借鉴**：
- 📥 **建议引入**：在 `ProbeTunnel` 三重探测后，增加第四重"吞吐量检测"
- 📥 **实现位置**：`pkg/tunnel/unlock.go` 的 `ProbeTunnel` 函数
- 📥 **具体方案**：
  ```go
  // 第四重：吞吐量检测（防断流节点）
  func checkThroughput(devName string, timeout time.Duration) (int64, error) {
      // 通过 devName 专属 HTTP 客户端下载 Cloudflare 测速文件
      // 计算实际吞吐量（bytes/s），< 70KB/s 判定为断流
  }
  ```

---

### 2.3 真实出口 IP 归类（非入口 IP）

**freesub 核心逻辑**：
```python
# 通过代理隧道访问 IP 识别服务，获取真实出口 IP
IP_ECHO_URLS = [
    "https://api.ip.sb/geoip",              # 返回 country_code/asn/isp
    "https://ipinfo.io/json",               # 返回 country/org
    "http://ip-api.com/json/",              # 返回 countryCode/isp
]

# 出口 IP 查不到时，回退查入口服务器 IP（兜底策略）
```

**AimiliVPN 当前实现**：
- ✅ 已实现出口 IP 获取：`pkg/tunnel/unlock.go` 的 `probePhysicalIP` 函数
- ✅ 已使用 `ipinfo.io` 和 `ifconfig.me` 双重验证
- ✅ 基于真实出口 IP 进行国家归类

**结论**：**AimiliVPN 在这一点上已经完全对齐 freesub，甚至更优**（实时物理网卡探测）

---

### 2.4 住宅宽带识别（六重信号）

**freesub 六重信号**：
1. ① `ip-api.com` 的 `hosting` 字段（hosting=true → 机房）
2. ② `mobile` 字段（mobile=true → 移动网络）
3. ③ Cloudflare/CDN Anycast 网段比对
4. ④ MaxMind GeoLite2 ASN 白/黑名单
5. ⑤ rDNS/ISP 名称特征识别
6. ⑥ Scamalytics 欺诈分复核（fraud ≥75 降级、≥90 剔除）

**AimiliVPN 当前实现**：
- ✅ 已实现六重信号：`pkg/nodes/iptype.go` 的 `evaluateIPType` 函数
- ✅ 信号权重：hosting(-3) > residential(+3) > mobile(+2) > ISP(+1)
- ✅ 置信度阈值：score ≥ 2 判定为住宅宽带

**结论**：**AimiliVPN 已完全实现 freesub 的住宅宽带识别逻辑**

---

### 2.5 测前去重（凭据指纹）

**freesub 核心优化**：
```python
# 同 server+port+protocol+uuid/password 的节点只测一次
# 4968 个节点去重后 → 2927 个（削减 42%）
# CI 时间从 40 分钟 → 7.5 分钟
```

**AimiliVPN 可借鉴**：
- 📥 **当前状态**：VPNGate 数据源节点重复率低（志愿者节点 IP 唯一）
- 📥 **未来扩展**：如果接入多订阅源聚合，可引入凭据指纹去重

---

### 2.6 导出保真度回归测试

**freesub 质量保障**：
```python
# 导出后再解析回来，逐字段比对（roundtrip test）
# 任何字段丢失都在 CI 内拦截
# 本轮实测：24/24 样本零丢失（0% 有损率）
```

**AimiliVPN 可借鉴**：
- 📥 **建议引入**：Clash YAML 导出后的回归测试
- 📥 **实现位置**：`pkg/server/clash_test.go`
- 📥 **具体方案**：
  ```go
  func TestClashYAMLRoundtrip(t *testing.T) {
      // 1. 生成 Clash YAML
      // 2. 解析回 sing-box.Node 对象
      // 3. 比对关键字段（server/port/uuid/password/sni）
      // 4. 任何丢失即报错
  }
  ```

---

## 三、可直接复用的代码模块

### 3.1 端口预检函数（快速淘汰死节点）

**复用价值**：⭐⭐⭐⭐⭐  
**适用场景**：VPNGate 节点池刷新前的快速预检

```python
def port_knock_tcp(ip, port, timeout=2.5):
    """TCP 端口预检（快速握手）"""
    try:
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            s.settimeout(timeout)
            s.connect((ip, port))
            return True
    except:
        return False
```

**Go 语言实现建议**：
```go
// pkg/nodes/precheck.go
func PortKnockTCP(ip string, port int, timeout time.Duration) bool {
    conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), timeout)
    if err != nil {
        return false
    }
    conn.Close()
    return true
}
```

---

### 3.2 Cloudflare 网段检测（快速识别 CDN 任播）

**复用价值**：⭐⭐⭐⭐  
**适用场景**：住宅宽带识别器的第三重信号

```python
CLOUDFLARE_IP_NETWORKS = [ipaddress.ip_network(n) for n in (
    "173.245.48.0/20", "103.21.244.0/22", "104.16.0.0/13", ...
)]

def is_cloudflare_ip(ip_str):
    ip_obj = ipaddress.ip_address(ip_str)
    return any(ip_obj in net for net in CLOUDFLARE_IP_NETWORKS)
```

**Go 语言实现**：  
✅ **已在 AimiliVPN 中实现**：`pkg/nodes/iptype.go` 的 `isCloudflareIP` 函数

---

### 3.3 批量 IP 情报查询（ip-api.com）

**复用价值**：⭐⭐⭐  
**适用场景**：节点池刷新时批量查询住宅宽带属性

```python
IP_API_BATCH_URL = "http://ip-api.com/batch"
IP_API_BATCH_SIZE = 100  # 每批最多 100 个 IP
IP_API_BATCH_RPS_INTERVAL = 4.2  # 每 4.2 秒一批（免费限额 15 req/min）

def batch_query_ip_api(ip_list):
    """批量查询 IP 情报（hosting/mobile/isp/asn）"""
    results = []
    for i in range(0, len(ip_list), IP_API_BATCH_SIZE):
        batch = ip_list[i:i+IP_API_BATCH_SIZE]
        resp = requests.post(IP_API_BATCH_URL, json=batch, timeout=10)
        results.extend(resp.json())
        time.sleep(IP_API_BATCH_RPS_INTERVAL)
    return results
```

**AimiliVPN 优化建议**：
- 📥 当前逐个查询 `ip-api.com`，可改为批量查询（提速 ~100 倍）
- 📥 实现位置：`pkg/nodes/iptype.go` 的 `FetchIPInfo` 函数

---

## 四、架构对比与建议

| 维度 | freesub | AimiliVPN | 建议 |
|:---|:---|:---|:---|
| **节点来源** | 多订阅源聚合 | VPNGate 志愿者节点 | ✅ 当前架构更优（官方数据源） |
| **测活引擎** | sing-box v1.14.0 | OpenVPN 实际拨号 | ✅ 当前架构更优（真实隧道） |
| **分层超时** | 12s/4s/2.5s | 12s/4s | 📥 增加端口预检 2.5s |
| **断流检测** | Cloudflare 5MB@70KB/s | 无 | 📥 建议引入吞吐量检测 |
| **MITM 识别** | TLS 证书校验 | 无 | 📥 建议引入证书校验 |
| **出口 IP** | 真实出口（兜底入口） | 真实出口 | ✅ 已完全对齐 |
| **住宅宽带** | 六重信号 | 六重信号 | ✅ 已完全对齐 |
| **去重策略** | 凭据指纹 | IP 唯一 | ✅ 当前架构已足够 |
| **导出格式** | Clash/V2Ray/sing-box | Clash/sing-box | ✅ 已覆盖主流客户端 |
| **回归测试** | 导出后解析比对 | 无 | 📥 建议增加 Clash 回归测试 |

---

## 五、建议优先级排序

### 🔴 高优先级（立即引入）
1. **端口预检**（2.5 秒快速淘汰死节点，节省 OpenVPN 拨号时间）
2. **断流检测**（Cloudflare 吞吐量测试，拦截"连上但无速度"的节点）

### 🟡 中优先级（择机引入）
3. **MITM 识别**（TLS 证书校验，识别中间人劫持）
4. **Clash YAML 回归测试**（防止导出丢失关键字段）

### 🟢 低优先级（未来扩展）
5. **批量 IP 查询**（仅在节点池超过 100 个时优化）
6. **凭据指纹去重**（仅在接入多订阅源时引入）

---

## 六、结论

**freesub 项目最大的技术价值**：
1. ✅ **分层超时 + 端口预检**：在不误杀慢节点的前提下，大幅缩短 CI 时间
2. ✅ **断流检测 + MITM 识别**：彻底消灭"假通畅"节点
3. ✅ **导出保真度回归**：确保客户端解析零失败

**AimiliVPN 当前优势**：
1. ✅ 真实 OpenVPN 隧道测活（比 sing-box 代理更接近生产环境）
2. ✅ 住宅宽带识别已完全对齐（六重信号 + 置信度）
3. ✅ 三大 AI 物理探测（freesub 未实现的差异化能力）

**最终建议**：
- 🎯 **立即引入**：端口预检 + 断流检测（投入产出比最高）
- 🎯 **择机引入**：MITM 识别 + Clash 回归测试
- 🎯 **保持优势**：OpenVPN 真实隧道 + 三大 AI 探测
