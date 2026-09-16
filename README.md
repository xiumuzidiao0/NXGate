# AimiliVPN (Go 高性能重构版)

<div align="center">

**面向 Linux VPS 的现代化 VPNGate 节点自适应管理、多出口流量调度与边缘抗封锁单端口代理网关**

[![正式版本](https://img.shields.io/github/v/release/xiumuzidiao0/aimili-vpngate-go?style=flat-square&label=正式版&color=16a34a)](https://github.com/xiumuzidiao0/aimili-vpngate-go/releases/latest)
[![Go Version](https://img.shields.io/badge/Go-1.25.13+-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/平台-amd64%20%7C%20arm64%20%7C%20386%20%7C%20arm-6366f1?style=flat-square)](https://github.com/xiumuzidiao0/aimili-vpngate-go/releases/latest)
[![License](https://img.shields.io/badge/License-GPL--3.0-334155?style=flat-square)](LICENSE)

</div>

采用 **Go 1.25.13+ 原生高并发模型与系统底层零拷贝技术** 对传统 VPN 代理系统进行工业级重构。编译后生成**单一可执行二进制文件**（内置 5 视图现代化深空响应式 SPA Web 控制台），内存常驻极低（< 15MB），专为资源受限的轻量级 Linux VPS（如 256MB / 512MB 内存机型）打造，同时具备高吞吐、零泄漏与物理级策略路由强隔离能力。

---

## 快速安装与终端管理

### 1. 一键极速安装与部署 (推荐)

使用 `root` 用户在受支持的 Linux VPS (Ubuntu / Debian / CentOS / Rocky / AlmaLinux / Alpine) 上执行：

```bash
curl -sSL https://raw.githubusercontent.com/xiumuzidiao0/aimili-vpngate-go/main/install.sh | bash
```

安装脚本将自动：
- 识别 CPU 架构，优先从官方加速源秒级拉取预编译静态二进制包（约 7.5MB）；
- 自动安装配置系统级网络依赖与 OpenVPN；
- 部署 `aimilivpn` 守护进程并注册 `systemd` 服务实现开机自愈自启；
- 创建全局快捷指令 `ml` 与 `aimili`。

### 2. 终端极速更新与命令行快捷操作

系统支持全自动免交互命令行操作，亦可随时唤出终端控制菜单：

```bash
ml update         # 从 GitHub Release 官方源秒级极速更新至最新发行版本并自动热重启
ml status         # 查看当前运行状态、Web 入口与管理账密
ml restart        # 安全平滑重启 AimiliVPN 服务
ml logs           # 查看实时 journalctl 运行日志流
ml menu           # 打开终端交互式可视化控制中心
```

---

## 核心系统级优化与高级架构 (v2.5.4)

### 1. 🌐 SOCKS5 UDP Associate 全链路穿透 (RFC 1928)
- **原生 UDP 代理中继**：在单端口统一嗅探（HTTP / SOCKS5）基础上，完整实现 RFC 1928 SOCKS5 UDP Associate 协议，客户端发起 UDP 请求（DNS over UDP、QUIC/HTTP3、VoIP 语音通话、在线游戏数据包）时实现全链路透明代理转发；
- **TCP 状态机强绑定与网卡穿透**：通过 Linux `SO_BINDTODEVICE`（`createBoundUDPSocket`）将上游 UDP Socket 直接绑定至具体的 `tunX` 虚拟网卡设备，严格受对应策略路由表隔离。

### 2. 🚪 节点端口快速预检机制 (Port Knock)
- **毫秒级过滤废弃节点**：在全量测速与拨号前，并发 48 goroutines 对候选节点执行 2.5 秒 TCP 快速握手敲门；
- **节约 90% 拨号开销**：在 OpenVPN 握手前快速淘汰离线与被墙死端口，告别过去逐个节点数十秒死等的痛点。

### 3. 🌊 带宽断流与假死检测 (Throughput Check)
- **物理吞吐采样**：新隧道握手成功后，通过虚拟网卡专属 HTTP 客户端向测速端点执行 5 秒分块采样下载；
- **淘汰断流节点**：严苛判定阈值（采样下载达到 50KB 且平均吞吐 >= 70 KB/s），彻底拦截“连上但带宽趋零”的假死志愿节点。

### 4. 🤖 三大 AI 严格物理出网校验 (OpenAI + Claude + Gemini)
- **真实物理网卡探针**：
  - OpenAI: `https://ios.chat.openai.com/public-api/mobile/server_status/v1` & `cdn-cgi/trace`
  - Claude: `https://api.anthropic.com/v1/messages` (鉴权响应拦截) & `cdn-cgi/trace`
  - Gemini: `https://gemini.google.com/app` 核心应用端点
- **零误判入池门禁**：当动态组或节点筛选配置为 `unlock: "ai"` 时，强制要求 **三大 AI 必须全部连通解锁** 方可通过准入。

### 5. 🏠 住宅宽带智能甄别引擎 (Residential Broadband Detector)
- **六重信号瀑布式分析**：基于 Cloudflare CDN 网段黑名单、主流云厂商 ASN 黑名单、全球主流运营商 ISP 白名单、机房/家宽关键词及 rDNS 反向解析；
- **智能置信度评估**：精准区隔机房托管（Datacenter）与真实原生住宅宽带（Residential），确保出海出口享有最高网络信誉。

### 6. 🔐 独立系统随机账密鉴权体系
- **与 Web 管理密码彻底解耦**：自动生成纯字母数字的安全随机凭据（`vpn_xxxx` / 16位高强度密码），杜绝 `@`、`%`、`:` 等特殊符号引发的 URL 编码转义与解析断层；
- **持久化隔离守护**：生成后以 `0600` 权限固化在 `data/proxy_auth.txt`，管理控制台修改 Web 登录密码绝不影响代理链路与 sing-box 出口。

### 7. 👑 系统主出口网关组 (`system-primary`，独占 `tun0`)
- **网卡强隔离保障**：底层严格保留 `devIndex = 0`（`tun0`）专属于系统主网关出口，并发多出口自适应池统一从 `tun1`、`tun2`... 向上单调递增，彻底杜绝自适应组与主网关争抢网卡的冲突；
- **自适应策略化接管**：将主连接升级为持久化系统特殊自适应组，支持按国家、网络类型、解锁能力与择优指标自动选拔与保活。

### 8. ⚡ 毫秒级滑动失败窗口熔断器 (Circuit Breaker) & 跨池逃生
- **15秒周期自愈探针**：自适应组每 15 秒主动向每个在线隧道发包探测，一旦节点因房东断网等原因出现丢包，在 15 秒内自动判定失效并触发替补；
- **毫秒级跨池逃生**：当某个端口绑定的组发生故障时，调度器在毫秒级内自动回退至全池健康出口（如 `tun0`），绝不向断网节点送死流量，杜绝 502 Bad Gateway。

### 9. 📦 客户端订阅多格式导出 (Clash Meta / Mihomo 专属 YAML 导出)
- **纯 Go 零依赖生成器**：实现 `GenerateClashYAML()`，将 sing-box 入站节点标准化生成完整的 Clash Meta / Mihomo 配置；
- **专业策略组装配**：包含 `🚀 节点选择`、`♻️ 自动选择 (URL-Test)`、`⚡ 故障转移 (Fallback)` 与 `🐟 漏网之鱼`，内置国内外 GEOIP 分流规则；
- **免密安全更新**：客户端通过已验证的安全管理路径或 Token 即可直接更新配置，无需弹出 Basic Auth 认证框；Web 控制台支持一键复制与直接下载 YAML 文件。

### 10. 🛡️ sing-box 边缘抗封锁入站 & 7×24h 后台自愈守护 (Watchdog)
- **入站矩阵管理**：支持可视化管理 VLESS-REALITY、Hysteria2、TUIC、Shadowsocks、AnyTLS 等 22 种抗审查入站协议；
- **自愈守护进程**：后台 30 秒独立心跳巡检，一旦 sing-box 发生意外终止，自愈守护程序在 30 秒内安全拉起，实现 7×24 小时无人值守。

### 11. 💻 现代化深空 5 视图 SPA Web 控制台
- 采用原生 Vanilla JS + 7 级深空 Surface 色阶构建响应式 SPA 架构，彻底消除杂乱堆叠：
  - **📊 运行概览 (Dashboard)**：实时上下行网速、活跃连接、主网关卡片、流式系统日志；
  - **🚀 边缘入站 (sing-box)**：抗封锁入站节点列表、链式代理出口选择、Clash 订阅管理；
  - **🔀 多端口分流 (Matrix)**：M:N 代理端口规则矩阵、动态自适应隧道组参数管理；
  - **🌐 优质节点 (Nodes)**：全量候选节点筛选、TCP 延迟测速、流媒体与 AI 解锁探测；
  - **⚙️ 系统设置 (Settings)**：Web 端口、管理账号密码、安全路径、代理端口配置；
- **多端响应式适配**：覆盖 390px、768px、1024px、1440px 视口，Playwright 端到端自动化回归测试 100% 通过。

---

## 核心特性架构对比

| 特性 | 原 Python 版本 | Go 重构版本 (v2.5.4) |
| :--- | :--- | :--- |
| **程序分发与体积** | 需 Python 3.10+、海量依赖脚本 | **单一静态二进制文件（约 7.5MB）**，零外部语言依赖 |
| **内存与 CPU 消耗** | 80MB ~ 150MB | **< 15MB 内存，CPU 占用 < 1%**，抗压能力大幅跃升 |
| **并发代理模型** | Thread + 全局解释器锁 (GIL) 瓶颈 | **Goroutine + Linux Epoll**，轻松承载数千长连接并发 |
| **UDP 协议支持** | 仅支持 TCP CONNECT | **完整实现 RFC 1928 SOCKS5 UDP Associate，QUIC/VOIP/DNS 全穿透** |
| **节点准入门禁** | 盲目拨号死等 | **Phase 1 端口预检 (2.5s) + Phase 2 带宽断流检测 (>=70KB/s)** |
| **AI 解锁甄别** | 仅依赖入口国家粗判 | **三大 AI (OpenAI + Claude + Gemini) 真实物理网卡端点实测** |
| **IP 属性识别** | 无区分 | **六重信号瀑布流分类原生住宅家宽（Residential）与机房 IP** |
| **主连接设备管理** | 设备跳跃无序 | **`tun0` 专享独占保留**，封装为系统自适应组 |
| **多出口并发调度** | 仅支持单一主连接 | **支持 `tun1..tun63` 多出口并发，M:N 端口调度 (轮询/随机/定时)** |
| **动态自适应维护** | 无 | **自动按国家/家宽属性/AI解锁能力维持 Top N 节点在线** |
| **故障熔断自愈** | 无，长达数十秒死等超时 | **毫秒级滑动失败窗口熔断器 + 跨池自动兜底容灾** |
| **屏蔽库管理** | 盲目拉黑 24 小时 | **精准定性防误杀 + 3小时自动探活复活 + Web 一键自愈** |
| **边缘抗封锁入站** | 无 | **深度集成 sing-box (Reality / Hy2 / TUIC / SS / AnyTLS) 链式代理** |
| **出站认证解耦** | 无 | **专有系统随机账密鉴权，彻底与 Web 管理密码解耦防脱节** |
| **客户端生态导出** | 仅单节点通用 URL | **一键导出 Clash Meta / Mihomo 格式全功能 YAML 订阅** |
| **守护与高可用** | 意外退出即断网 | **内置 7×24 小时 sing-box Watchdog 自愈守护协程** |
| **网络性能调优** | 系统默认小缓冲区 | **BDP TCP 512KB Socket 发送/接收缓冲区 + 全链路 NoDelay** |
| **Web 控制台设计** | 简陋单页 HTML 拼接 | **深空 7 级色阶 5 视图 SPA 架构，折叠侧栏与移动底栏自适应** |
| **宿主机安全性** | 曾有主路由被篡改风险 | **虚拟网卡严格限制在私有策略路由表 (Table 100+N)，主机 SSH 100% 隔离** |

---

## 目录结构说明

```text
aimili-vpngate-go/
├── .github/workflows/
│   ├── ci.yml                 # 持续集成质量准入流水线 (版本核验/安全审计/全包测试/多端UI测试)
│   ├── release.yml            # 自动化跨平台交叉编译与 GitHub Release 产物发布流水线
│   └── mirror.yml             # 每 6 小时自动拉取 VPNGate 镜像快照工作流
├── cmd/
│   ├── aimilivpn/             # 网关主程序入口 (main.go)
│   └── mirror/                # VPNGate 镜像抓取与源数据同步工具
├── docs/
│   ├── PROJECT_HANDOVER.md    # [核心维护文档] 系统全景架构设计、技术实现细节、生产运维手册与故障排查
│   ├── CICD_AND_TESTING_GUIDE.md # CI/CD 发布与测试流程标准化指南
│   └── FREESUB_COMPARISON.md  # 与 freesub 项目的技术特性深度对比与架构演进分析
├── pkg/
│   ├── config/                # 环境变量、持久化配置、独立随机代理凭据与版本号定义
│   ├── nodes/                 # 节点拉取、端口预检 (pool.go)、住宅IP分类 (residential.go)、信誉评分
│   ├── notify/                # Telegram 告警推送与交互机器人
│   ├── proxy/                 # HTTP/SOCKS5 嗅探代理、SOCKS5 UDP Associate 穿透、多端口分流管理器
│   ├── server/                # Web 控制台 HTTP 路由、Clash/singbox 订阅生成、出站解析
│   ├── singbox/               # 边缘抗封锁协议客户端 API 与 22 种原生协议元数据
│   ├── stats/                 # 实时上下行流量统计与内存环形日志总线
│   ├── tunnel/                # Linux 隧道池管理、带宽断流检测 (throughput.go)、三大 AI 解锁探测
│   └── vpn/                   # OpenVPN 进程监管、TUN 检测与主连接管理器
├── scripts/
│   ├── audit.sh               # 静态分析与 govulncheck 依赖安全漏洞扫描脚本
│   ├── build.sh               # 4 大架构交叉编译打包脚本
│   ├── check-version.sh       # 版本一致性核验与一键同步工具
│   ├── release.sh             # 一键自动化发布助手脚本
│   └── aimilivpn.service      # systemd 系统守护进程配置模板
├── web/
│   ├── dist/                  # 现代化响应式深空 5 视图 SPA 控制台前端源码 (index.html, styles.css, app.js)
│   ├── tests/                 # Playwright 4 端视口自动化回归测试 (ui-smoke.mjs)
│   ├── embed.go               # go:embed 静态资产打包
│   └── package.json
├── config.env.example         # 环境变量配置标准示例模板
├── install.sh                 # Linux 一键安装、服务部署与终端管理脚本 (ml)
├── Dockerfile                 # 多阶段极简容器构建镜像
├── VERSION                    # 全局单一版本信任源定义文件
└── go.mod
```

---

## 快速上手与本地编译

### 1. 本地直接编译运行

确保机器已安装 Go 1.25.13+ 环境：

```bash
# 编译当前架构二进制
go build -ldflags="-s -w" -o aimilivpn ./cmd/aimilivpn

# 启动运行 (需要 root 权限以管理虚拟网卡与策略路由)
sudo ./aimilivpn
```

### 2. 交叉编译全架构发行包

```bash
chmod +x scripts/build.sh
./scripts/build.sh
```

编译产物将输出在 `dist/` 目录中：
- `dist/aimilivpn_linux_amd64` (及其 `.gz` 压缩包)
- `dist/aimilivpn_linux_arm64` (及其 `.gz` 压缩包)
- `dist/aimilivpn_linux_386` (及其 `.gz` 压缩包)
- `dist/aimilivpn_linux_arm` (及其 `.gz` 压缩包)
- `dist/SHA256SUMS.txt` 校验和清单

### 3. 一键发布新版本 (CI/CD)

```bash
# 自动同步版本、运行静态安全审计与全量回归测试、打 Tag 并推送唤起 GitHub Actions
./scripts/release.sh <新版本号，例如 2.5.5>
```

---

## 常见问题排查与技术细节

### Q1: 为什么主连接与自适应组各出口互不干扰？
A: 系统采用 Linux 高级策略路由隔离体系。主连接独占 `tun0`（绑定路由表 `Table 100`），并发自适应组出口有序占用 `tun1`、`tun2`...（分别绑定 `Table 101`、`Table 102`...）。主路由表（`main table`）保留宿主机 `eth0` 默认网关，**宿主机 SSH 22 端口及网络 100% 隔离安全**。

### Q2: 遇到部分志愿节点握手成功但无法出网或断流怎么办？
A: 系统内置了 15 秒高频出网真实性探针、握手门禁测试及 **Throughput 带宽采样过滤机制**。任何握手成功但实际无法连通外网或带宽低于 70 KB/s 的节点，会被自动标记并隔离至屏蔽库（Blacklist），系统毫秒级从自适应池中选取下一个真实通网的候选节点替补，保障出口池时刻处于可用状态。

### Q3: 客户端如何导入 Clash Meta 订阅？
A: 在 Web 控制台「边缘入站」页面顶部，点击 **「📋 复制 Clash 订阅」** 即可获取专属链接，直接粘贴到 Clash Verge Rev、Mihomo Party、Clash Meta for Android 等客户端中即可一键更新使用。

### Q4: 代理端口鉴权为什么要与 Web 控制台管理员密码解耦？
A: 系统在 v2.5.4 中引入了**独立系统随机账密体系**。如果代理端口跟随 Web 登录密码，一旦用户在控制台修改密码，就会导致所有客户端与 sing-box 链式出口配置由于缓存脱节而瞬间失效。采用独立的字母数字随机凭据后，不仅能从根本上规避特殊符号 URL 转义问题，还可确保代理通道长效稳定不漂移。
