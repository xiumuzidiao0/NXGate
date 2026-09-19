# NXGate (自适应多出口智能路由网关)

<div align="center">

**面向 Linux VPS 的多出口流量调度、全球原生住宅宽带智能发现与边缘抗封锁单端口代理网关系统**

[![Release](https://img.shields.io/github/v/release/xiumuzidiao0/NXGate?style=flat-square&label=Release&color=16a34a)](https://github.com/xiumuzidiao0/NXGate/releases/latest)
[![Go Version](https://img.shields.io/badge/Go-1.25.13+-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-amd64%20%7C%20arm64%20%7C%20386%20%7C%20arm-6366f1?style=flat-square)](https://github.com/xiumuzidiao0/NXGate/releases/latest)
[![License](https://img.shields.io/badge/License-GPL--3.0-334155?style=flat-square)](LICENSE)

</div>

NXGate 是一款采用 **Go 1.25.13+ 原生高并发模型与 Linux 内核策略路由** 构建的高性能自适应多出口智能路由网关系统。系统被构建为单一静态可执行二进制文件（内置 5 视图响应式 SPA Web 管理控制台，并提供 Material 3 原生 Android 配套客户端），常驻内存小于 15MB，专为 Linux VPS 与边缘节点设计，具备高并发高吞吐、零路由污染与底层硬件级策略路由强隔离能力。

---

## 目录

- [系统全景架构](#系统全景架构)
- [核心子系统工程设计](#核心子系统工程设计)
  - [1. L3/L4 策略路由与宿主机安全内核模型](#1-l3l4-策略路由与宿主机安全内核模型)
  - [2. 多协议代理中继与 SOCKS5 UDP Associate 穿透](#2-多协议代理中继与-socks5-udp-associate-穿透)
  - [3. M:N 多端口分流调度与动态自适应出口矩阵](#3-mn-多端口分流调度与动态自适应出口矩阵)
  - [4. 四阶段节点全生命周期发现与过滤流水线](#4-四阶段节点全生命周期发现与过滤流水线)
  - [5. 边缘抗审查网关与 7×24h 守护看门狗](#5-边缘抗审查网关与-724h-守护看门狗)
  - [6. 客户端矩阵 (SPA 控制台与 Android 原生应用)](#6-客户端矩阵-spa-控制台与-android-原生应用)
- [快速部署与运维管理](#快速部署与运维管理)
  - [一键安装与部署](#一键安装与部署)
  - [CLI 快捷指令参考](#cli-快捷指令参考)
  - [Systemd 服务生命周期](#systemd-服务生命周期)
- [配置参数规范](#配置参数规范)
- [源码构建与质量准入](#源码构建与质量准入)
  - [本地编译运行](#本地编译运行)
  - [全架构静态交叉编译](#全架构静态交叉编译)
  - [自动化质量准入审计](#自动化质量准入审计)
- [开源协议](#开源协议)

---

## 系统全景架构

NXGate 构建了从公网边缘抗封锁入站到全球原生住宅家宽出口的多层解耦流水线：

```text
                                       ┌──────────────────────────────────────────────────────────┐
                                       │                    Client Applications                   │
                                       │   Android Client / Clash Meta / Web / Headless Scraper   │
                                       └────────────────────────────┬─────────────────────────────┘
                                                                    │
                                                 VLESS / Hysteria2 / TUIC / Direct SOCKS5
                                                                    ▼
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ NXGate Ingress & Core Gateway                                                                                   │
│                                                                                                                 │
│   ┌────────────────────────────────────────┐       ┌────────────────────────────────────────────────────────┐   │
│   │ Edge Anti-Censorship Ingress (sing-box)│       │ Unified Proxy Demuxer (RFC 1928)                       │   │
│   │ 22 Inbound Protocols (Reality/Hy2/TUIC)├──────►│ Dual-Stack Sniffer: HTTP CONNECT / SOCKS5 TCP / UDP  │   │
│   │ 30s Heartbeat Watchdog Supervisor      │       │ Isolated Auth Provider (Random Base64 Credential)      │   │
│   └────────────────────────────────────────┘       └──────────────────────────┬─────────────────────────────┘   │
│                                                                               │                                 │
│                                    M:N Port Scheduler (Round-Robin / Random / Interval / Sticky)                │
│                                                                               │                                 │
│                                                                               ▼                                 │
│   ┌─────────────────────────────────────────────────────────────────────────────────────────────────────────┐   │
│   │ Dynamic Egress Pool & Circuit Breaker                                                                   │   │
│   │   • System Primary Group (tun0, Priority System Egress)                                                 │   │
│   │   • Dynamic Groups (tun1..tunN, Auto-Balanced by Country / Residential / Physical AI Unlock)           │   │
│   │   • Sliding-Window Failure Counter & 15s Sub-Second Failover Engine                                     │   │
│   └───────────────────────────────────────────────────┬─────────────────────────────────────────────────────┘   │
└───────────────────────────────────────────────────────┼─────────────────────────────────────────────────────────┘
                                                        │
                         Linux Policy Routing (SO_BINDTODEVICE + Isolated Table IDs)
                                                        │
┌───────────────────────────────────────────────────────▼─────────────────────────────────────────────────────────┐
│ Linux Kernel & Network Stack                                                                                    │
│                                                                                                                 │
│   [Priority 50]  sport/dport 22 ───────────────────► Table main (Host eth0 Default Gateway, 100% Locked)       │
│   [Priority 1000] oif tun0      ───────────────────► Table 100  (Default via tun0 dev, route-noexec)           │
│   [Priority 1000] oif tun1      ───────────────────► Table 101  (Default via tun1 dev, route-noexec)           │
│   [Priority 1000] oif tunN      ───────────────────► Table 100+N(Default via tunN dev, route-noexec)           │
└───────────────────────────────────────────────────────┬─────────────────────────────────────────────────────────┘
                                                        │
                                                        ▼
                                       ┌──────────────────────────────────┐
                                       │     Global VPNGate Exit Matrix   │
                                       │  Residential Broadband & Datacenter│
                                       └──────────────────────────────────┘
```

---

## 核心子系统工程设计

### 1. L3/L4 策略路由与宿主机安全内核模型

为彻底杜绝 OpenVPN 隧道接管宿主机全局网络导致 SSH 失联这一行业通病，NXGate 构筑了内核级防线：

- **路由完全禁执 (`route-noexec` & `route-nopull`)**：
  OpenVPN 进程以底层标志位启动，物理级剥夺其执行 `ip route` 或修改系统路由表的权限；配合全量过滤链：
  ```text
  route-noexec
  route-nopull
  pull-filter ignore "redirect-gateway"
  pull-filter ignore "redirect-private"
  pull-filter ignore "route-gateway"
  pull-filter ignore "route"
  pull-filter ignore "route-ipv6"
  pull-filter ignore "dhcp-option"
  pull-filter ignore "topology"
  pull-filter ignore "block-outside-dns"
  pull-filter ignore "register-dns"
  pull-filter ignore "ip-win32"
  script-security 1
  ```
  远端节点推送的任何默认网关、子网路由与 DNS 均被原地丢弃，杜绝系统 `/etc/resolv.conf` 污染。
- **独占隔离策略路由表 (`Table 100 + devIndex`)**：
  每个虚拟网卡（`tun0`, `tun1`...）仅在其专属的数字路由表中配置默认出站；宿主机 `main` 路由表永不注入 VPN 路由。
- **SSH 内核级策略锁定 (`Priority 50`)**：
  网关启动及运行时自动检索 `/etc/ssh/sshd_config` 及其 `.d` 目录下的所有监听端口，在内核策略路由最高优先级（`priority 50`）注入：
  ```bash
  ip rule add sport <SSH_PORT> table main priority 50
  ip rule add dport <SSH_PORT> table main priority 50
  ```
  确保不论 VPN 接口发生任何翻转、换线或宕机，主机 SSH 流量无条件由物理网卡 `eth0` 承载。
- **反向路径过滤自愈 (`rp_filter = 2`)**：
  动态开启松散反向路径过滤，在隧道释放时自动恢复内核初始 sysctl 状态。

---

### 2. 多协议代理中继与 SOCKS5 UDP Associate 穿透

NXGate 实现了一套单端口双栈协议嗅探与中继引擎：

- **自适应协议分流 (Protocol Sniffing)**：
  单监听端口同时接纳 HTTP/1.1、HTTP CONNECT 隧道握手与 RFC 1928 SOCKS5 握手，根据首包特征动态切换状态机。
- **全链路 RFC 1928 UDP 穿透**：
  原生实现 SOCKS5 UDP Associate 规范。客户端请求创建 UDP 中继时，网关动态分配独立中继端口，维护 TCP 状态机生命周期关联；上游 Socket 通过 `SO_BINDTODEVICE` 绑定特定 `tunX` 虚拟网卡，完整支持 DNS over UDP、QUIC/HTTP3、VoIP 通话及实时游戏包出海。
- **高吞吐网络参数调优**：
  全链路启用 `TCP_NODELAY` 禁用 Nagle 算法，配置 512KB BDP Socket 发送/接收缓冲区，降低大规模流媒体与大文件传输时的上下文切换开销。

---

### 3. M:N 多端口分流调度与动态自适应出口矩阵

支持将 VPS 构建为多端口分布式出海集群：

- **解耦的 M:N 端口规则**：
  管理员可开启任意数量的本地代理端口（如 `7928`、`7929`、`7930`...），并将端口绑定到任意自适应出口组或特定静态隧道。
- **四种出口调度策略**：
  - `round_robin`：请求级轮询切换出口，实现 IP 负载均衡；
  - `random`：权重随机分发；
  - `interval`：按设定周期（如每 10 分钟）平滑轮替主用出口；
  - `sticky`：基于源 IP 维持长连接会话亲和性。
- **主网关保留规范 (`system-primary`)**：
  底层严格锁定 `devIndex = 0`（`tun0`）专属于系统主网关出口，并发动态组统一从 `tun1` 向上单调递增，消除网卡资源竞态。
- **滑动窗口熔断器 (Circuit Breaker) 与零宕机兜底**：
  自适应组以 15 秒为周期对纳管隧道执行端到端有效性探活。遇出口异常（如志愿节点下线），熔断器在毫秒级将请求透明回退至全局健康出口，杜绝向上游抛出 HTTP 502。

---

### 4. 四阶段节点全生命周期发现与过滤流水线

面向全球海量公网志愿节点，NXGate 引入四阶段严格过滤管道，仅准入高可用优质连接：

```text
  Raw VPNGate Mirror Feed (~1,000 Nodes)
                    │
                    ▼
  [Phase 1] 48-Worker Concurrent TCP Port Knock (2.5s Timeout)
            淘汰离线端点与端口阻断，节约 90% 拨号开销
                    │
                    ▼
  [Phase 2] L4 Throughput Check (5s Sampling Download)
            强制要求物理下行速率 >= 70 KB/s，剔除假死与限速节点
                    │
                    ▼
  [Phase 3] L7 Triple-AI Physical Endpoint Verification
            物理接口实测通过 OpenAI + Claude + Gemini 官方端点校验
                    │
                    ▼
  [Phase 4] 6-Signal Heuristic Residential Broadband Classifier
            Cloudflare CDN 黑名单 + ASN 黑名单 + 运营商白名单 + rDNS 逆向分析
                    │
                    ▼
  Active Candidate Pool for Dynamic Routing (~50-100 High-Quality Exits)
```

1. **Phase 1: 毫秒级端口敲门 (TCP Port Knock)**：并发 48 goroutine 对全量镜像执行轻量 SYN 探测，2.5 秒内排查死端。
2. **Phase 2: 真实吞吐采样 (Throughput Probing)**：拨号就绪后，经专属出口向测速端点执行切片下载，未达到连续 70 KB/s 阈值者直接淘汰。
3. **Phase 3: 三大 AI 物理端点核验**：
   - OpenAI：验证 `ios.chat.openai.com` 鉴权与 `cdn-cgi/trace` 归属；
   - Claude：验证 `api.anthropic.com` 消息路由鉴权响应与 Cloudflare 边缘阻断；
   - Gemini：验证 `gemini.google.com` 服务可用性。
4. **Phase 4: 住宅宽带甄别引擎**：瀑布流评估 ISP、ASN 属性，准确标定原生住宅家宽（Residential）与数据中心（Datacenter）。

---

### 5. 边缘抗审查网关与 7×24h 守护看门狗

- **22 种原生入站协议矩阵**：
  深度纳管 sing-box 内核，支持 VLESS-REALITY（借用亚马逊/苹果权威 TLS 指纹）、Hysteria2（极速 UDP 拥塞控制）、TUIC v5、Shadowsocks 2022、AnyTLS 等抗审查入站协议，并链式分流至 NXGate 住宅出口。
- **7×24h 自愈看门狗 (Watchdog)**：
  独立协程以 30 秒为周期监控进程健康度，遇宿主机 OOM 或意外崩溃自动执行冷启动自愈。
- **纯 Go 零依赖 Clash Meta / Mihomo YAML 订阅生成**：
  标准生成 `🚀 节点选择`、`♻️ 自动选择 (URL-Test)`、`⚡ 故障转移 (Fallback)` 与分流规则链，支持安全路径免密下发。

---

### 6. 客户端矩阵 (SPA 控制台与 Android 原生应用)

- **深空 5 视图响应式 SPA 控制台**：
  内置于 Go 二进制文件（`go:embed`），采用 7 级深空 Surface 色阶。包含仪表盘（Dashboard）、边缘入站（sing-box）、多端口矩阵（Matrix）、节点广场（Nodes）及安全配置（Settings）。覆盖 390px 至 1440px+ 视口。
- **Material 3 原生 Android 配套客户端 (`com.nxgate.app`)**：
  - 全流程 Material 3 规范与 Android 12+ 莫奈壁纸动态取色；
  - 手机端原生贴底 `NavigationBar`，平板横屏自适应左侧 `NavigationRail`；
  - CameraX + ZXing 离线安全扫码添加服务器；
  - Android 生物识别硬件锁（BiometricPrompt 指纹/面容/凭据）。

---

## 快速部署与运维管理

### 一键安装与部署

在目标 Linux VPS（支持 Debian / Ubuntu / CentOS / RHEL / Rocky / AlmaLinux / Alpine）以 `root` 权限执行：

```bash
curl -sSL https://raw.githubusercontent.com/xiumuzidiao0/NXGate/main/install.sh | bash
```

安装脚本将自动执行架构匹配、拉取预编译可执行文件、配置网络守护进程服务并写入全局管理指令。

### CLI 快捷指令参考

在服务器任意终端位置，使用 `nx` 进行管理：

```bash
nx               # 唤出终端交互式可视化控制中心
nx status        # 打印当前网关运行状态、sing-box 状态与入口地址
nx update        # 检查并拉取最新 Release 版本执行热更新与自愈
nx restart       # 平滑重启网关服务
nx start         # 启动网关服务
nx stop          # 停止网关服务
nx logs          # 查看实时运行日志 (journalctl -u aimilivpn -f)
```

### Systemd 服务生命周期

```bash
# 检查守护进程运行状态
systemctl status aimilivpn

# 重启网关核心服务
systemctl restart aimilivpn

# 查看开机启动项
systemctl is-enabled aimilivpn
```

---

## 配置参数规范

配置文件路径位于 `/opt/aimilivpn/config.env`（环境变量覆盖优先级高于文件）：

| 环境变量名 | 默认值 | 允许范围 / 格式 | 功能说明 |
| :--- | :--- | :--- | :--- |
| `UI_HOST` | `::` | 字符串 (IP) | Web 管理控制台监听绑定地址 |
| `UI_PORT` | `8787` | `1-65535` | Web 管理控制台对外服务端口 |
| `UI_PATH` | `enter` | 纯字母数字字符串 | 控制台访问安全路径前缀（防扫描，如 `/enter/`） |
| `UI_USERNAME` | `admin` | 字符串 | Web 管理后台登录账号 |
| `UI_PASSWORD` | *(随机生成)* | 字符串 | Web 管理后台登录强密码 |
| `LOCAL_PROXY_HOST`| `127.0.0.1` | 字符串 (IP) | 本地默认代理监听地址 |
| `LOCAL_PROXY_PORT`| `7928` | `1-65535` | 本地默认代理监听端口 (HTTP/SOCKS5 单端口自适应) |
| `LOCAL_PROXY_MAX_CONNECTIONS` | `512` | `16-4096` | 单代理端口最大允许并发连接数 |
| `FETCH_INTERVAL_SECONDS` | `900` | `60-86400` | VPNGate 节点镜像全量拉取刷新周期（秒） |
| `CHECK_INTERVAL_SECONDS` | `20` | `5-300` | 在线主连接与活跃出口健康探测心跳间隔（秒） |
| `TARGET_VALID_NODES` | `5` | `1-50` | 内存池最小维持的经过端口预检的优质候选节点数 |
| `MAX_SCAN_ROWS` | `1000` | `10-5000` | 单次从镜像 CSV 中解析的最大行数 |
| `DISCOVERY_COUNTRIES` | *(空)* | 逗号分隔 ISO 代码 (如 `JP,US`) | 节点发现首选国家过滤白名单（留空为全球） |
| `DATA_DIR` | `/opt/aimilivpn/data`| 绝对路径 | 运行时证书、路由配置与日志存储目录 |

---

## 源码构建与质量准入

### 本地编译运行

构建环境需满足 Go 1.25.13+：

```bash
# 1. 克隆代码仓库
git clone https://github.com/xiumuzidiao0/NXGate.git
cd NXGate

# 2. 编译当前平台二进制文件
CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/aimilivpn ./cmd/aimilivpn

# 3. 运行网关服务 (需 root 权限以管理虚拟网卡)
sudo ./bin/aimilivpn
```

### 全架构静态交叉编译

内置发行编译脚本可一次性输出 4 种 CPU 架构的无依赖静态程序包：

```bash
chmod +x scripts/build.sh
./scripts/build.sh
```

输出文件位于 `dist/` 目录：
- `aimilivpn_linux_amd64` (x86_64 服务器通用)
- `aimilivpn_linux_arm64` (aarch64 树莓派 / 鲲鹏 / 飞腾 / 甲骨文 ARM)
- `aimilivpn_linux_386` (32位 x86)
- `aimilivpn_linux_arm` (32位 ARMv7)
- `SHA256SUMS.txt` (全产物哈希校验清单)

### 自动化质量准入审计

代码库受本地 Git Pre-Push 钩子与 GitHub Actions CI/CD 双重准入约束：

```bash
# 1. 静态代码分析与 CVE 漏洞扫描
bash scripts/audit.sh

# 2. 全量单元与集成测试 (覆盖全部 11 个子包)
go test -count=1 -v ./...

# 3. 代码库全链路版本一致性核验
bash scripts/check-version.sh

# 4. 前端视口回归测试
cd web && npm test
```

---

## 开源协议

本项目采用 [GNU General Public License v3.0 (GPL-3.0)](LICENSE) 开源协议授权。
引用或二次分发请遵循开源许可证要求保留原作者信息。
