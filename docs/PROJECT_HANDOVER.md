# AimiliVPN (Go 高性能版) 项目交接与工程维护文档

**编写日期**：2026-09-16  
**当前版本**：`v2.5.1`  
**代码仓库**：`https://github.com/xiumuzidiao0/NXGate`  
**维护状态**：CI/CD 自动化全绿、线上运行正常、全功能经过端到端验证  

---

## 目录

- [一、 项目概况与核心使命](#一-项目概况与核心使命)
- [二、 系统全景架构设计](#二-系统全景架构设计)
- [三、 核心技术实现与代码索引](#三-核心技术实现与代码索引)
  - [1. SOCKS5 UDP Associate 全链路穿透 (RFC 1928)](#1-socks5-udp-associate-全链路穿透-rfc-1928)
  - [2. 节点端口快速预检机制 (Port Knock)](#2-节点端口快速预检机制-port-knock)
  - [3. 带宽断流检测机制 (Throughput Check)](#3-带宽断流检测机制-throughput-check)
  - [4. 三大 AI 解锁严格物理校验 (OpenAI + Claude + Gemini)](#4-三大-ai-解锁严格物理校验-openai--claude--gemini)
  - [5. 现代化深空 UI/UX 架构与组件设计](#5-现代化深空-uiux-架构与组件设计)
- [四、 线上服务器环境与运维管理手册](#四-线上服务器环境与运维管理手册)
  - [1. 生产服务器环境资产](#1-生产服务器环境资产)
  - [2. 常用运维与管理指令](#2-常用运维与管理指令)
- [五、 CI/CD 自动化流水线与发布流程](#五-cicd-自动化流水线与发布流程)
  - [1. 核心流水线列表](#1-核心流水线列表)
  - [2. 发布新版本极简 SOP](#2-发布新版本极简-sop)
- [六、 目录结构与核心源码索引](#六-目录结构与核心源码索引)
- [七、 常见问题排查 (Troubleshooting)](#七-常见问题排查-troubleshooting)
- [八、 后续技术演进建议与待办规划](#八-后续技术演进建议与待办规划)

---

## 一、 项目概况与核心使命

AimiliVPN 是基于 Go 语言重构的高性能 Linux 出口网关系统。其核心使命是：
1. **聚合与清洗海量 VPNGate 志愿者节点资源**，自动识别并优先调度日本、美国等全球**原生住宅家宽（Residential Broadband）IP**；
2. **构建多出口并发虚拟网卡池（`tun0` ~ `tunN`）**，利用 Linux 内核策略路由与 `SO_BINDTODEVICE` 实现严格的流量隔离与独立出网；
3. **提供统一本地代理网关（HTTP / HTTPS CONNECT / SOCKS5 / SOCKS5 UDP）**，支持多端口独立分流与自适应动态池轮换；
4. **无缝联动边缘抗封锁协议核心（sing-box）**，提供全套 22 种现代抗审查入站（VLESS-REALITY、Hysteria2、TUIC、Shadowsocks 2022 等），打通国内客户端到海外住宅出口的无缝穿透。

---

## 二、 系统全景架构设计

系统整体分为 5 大垂直分层：

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ 1. 边缘抗封锁入站层 (Edge Ingress)                                         │
│    sing-box 核心原生托管 22 种协议:                                        │
│    VLESS-REALITY / VLESS-H2 / Hysteria2 / TUIC v5 / SS 2022 / AnyTLS 等      │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │ 链式出站: socks5://127.0.0.1:7928 (TCP + UDP)
┌──────────────────────────────────────▼──────────────────────────────────────┐
│ 2. 统一代理调度与分流矩阵层 (Proxy Gateway & Matrix)                         │
│    - 自适应协议嗅探 (HTTP/SOCKS5/SOCKS5 UDP Associate)                      │
│    - 多端口独立分流规则 (7928, 7929...)                                     │
│    - 动态自适应隧道组 (system-primary 主出口组及自定义国家/家宽/AI 组)       │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │ 绑定 Linux 虚拟网卡设备: SO_BINDTODEVICE
┌──────────────────────────────────────▼──────────────────────────────────────┐
│ 3. 物理虚拟网卡隧道池 (Tunnel Pool)                                         │
│    - 并发独立 OpenVPN 进程集群: tun0 (主出口), tun1, tun2...                │
│    - 严格策略路由隔离 (IP Rule & Route Table)                               │
│    - 故障熔断器 (Circuit Breaker) 与槽位动态复用                            │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │ 自动化评估与探测驱动
┌──────────────────────────────────────▼──────────────────────────────────────┐
│ 4. 节点质量评估与健康检测引擎 (Evaluation & Probing)                        │
│    - 端口预检 (Port Knock): 2.5s TCP 快速握手，淘汰 90% 死端口               │
│    - 带宽断流检测 (Throughput): Cloudflare 端点测速，淘汰 < 70KB/s 假死节点 │
│    - 三大 AI 严格物理校验: OpenAI + Claude + Gemini 全部可用方可入 AI 组    │
│    - 住宅 IP 识别分类器: 基于 ASN / rDNS / ISP 多维度置信度智能分析        │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │ 交互控制与可视化展示
┌──────────────────────────────────────▼──────────────────────────────────────┐
│ 5. 控制台与交互层 (Management UI & CLI)                                     │
│    - 嵌入式现代 WebUI: 7 级深空 Surface 色阶、Geist 字体、Playwright 4端适配│
│    - 终端 TUI 控制脚本: nx 快速管理菜单、nx update 极速更新                 │
│    - Telegram 运维机器人: 告警推送与对话控制 (/status, /tunnels, /rotate)    │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 三、 核心技术实现与代码索引

### 1. SOCKS5 UDP Associate 全链路穿透 (RFC 1928)
- **实现位置**：`pkg/proxy/socks5.go` & `pkg/proxy/socks5_udp.go`
- **业务痛点**：过去仅支持 TCP `CONNECT`，客户端发起 UDP（DNS over UDP、QUIC/HTTP3、VoIP 语音通话、游戏数据包）时被直接阻断。
- **关键设计**：
  1. **控制面应答**：客户端发送 `CMD=0x03` (`cmdUdpAssociate`)，服务端在本地监听随机端口的 UDP 中继套接字，通过 TCP 应答返回 `BND.ADDR` 与 `BND.PORT`；
  2. **TCP 生命周期绑定**：后台协程阻塞读取 TCP 控制连接，当客户端 TCP 连接断开时立即释放对应的 UDP 中继与上游套接字；
  3. **物理网卡穿透**：通过 Linux `SO_BINDTODEVICE`（`createBoundUDPSocket`）将上游 UDP Socket 绑定至具体 `tunX` 设备，并使用隧道绑定的安全 DNS 解析目标；
  4. **sing-box 联动**：将 sing-box 链式出站升级为 `socks5://127.0.0.1:7928`，实现全链路原生无损 UDP 穿透。

### 2. 节点端口快速预检机制 (Port Knock)
- **实现位置**：`pkg/nodes/pool.go` (`FilterReachableNodes`, `PortKnockTCP`, `BatchPortKnock`)
- **业务痛点**：VPNGate 列表中存在大量已离线或被防火墙阻断的节点，过去直接发起 OpenVPN 拨号耗费大量等待时间（每次数十秒）。
- **关键设计**：
  - 在全量测速与拨号前，并发 48 goroutines 执行 2.5 秒超时的 TCP 握手敲门；
  - 快速剔除无法连通的死端口节点，节约 90% 以上的无效拨号等待时间。

### 3. 带宽断流检测机制 (Throughput Check)
- **实现位置**：`pkg/tunnel/throughput.go` (`CheckThroughputWithRetry`)
- **业务痛点**：部分节点虽然 TCP 可以连通甚至 OpenVPN 拨号成功，但实际物理带宽几乎为零（断流、限速、假死），导致用户连接后无法打开任何网页。
- **关键设计**：
  - 通过绑定虚拟网卡的 HTTP 客户端对 Cloudflare 测速端点执行 5 秒采样下载；
  - 设定最低阈值：下载达到 50KB 且平均速率 >= 70 KB/s 方为合格，低速假死节点自动淘汰并触发递补。

### 4. 三大 AI 解锁严格物理校验 (OpenAI + Claude + Gemini)
- **实现位置**：`pkg/tunnel/unlock.go` & `pkg/nodes/unlock.go`
- **业务痛点**：过去 AI 解锁测试存在盲区（例如未接入 Gemini 探测，Claude 403 误判等）。
- **关键设计**：
  - 引入完整的物理探测端点：
    - OpenAI: `https://chatgpt.com/cdn-cgi/trace`
    - Claude: `https://claude.ai/login`
    - Gemini: `https://gemini.google.com/app`
  - 当自适应组或节点筛选配置为 `unlock: "ai"` 时，强制要求**三大 AI 必须全部成功解锁**方可通过准入。

### 5. 现代化深空 UI/UX 架构与组件设计
- **实现位置**：`web/dist/styles.css` & `web/dist/index.html` & `web/dist/app.js`
- **关键设计**：
  - **色彩体系**：构建 7 级深空 Surface 色阶（`--surface-0` ~ `--surface-6`），主色调升级为科技青蓝（`#4dcadc`）配合微妙光晕；
  - **排版与动效**：流式等宽数值展示（防止跳动）、Spring 弹簧阻尼动画、毛玻璃抽屉面板；
  - **结构优化**：移除系统设置中冗余的“自动轮换策略”Tab，出口轮换全权归口至“多端口分流 -> 动态自适应组”统一管理；
  - **跨端适配**：覆盖 390px、768px、1024px、1440px，Playwright 端到端回归测试 100% PASS。

---

## 四、 线上服务器环境与运维管理手册

### 1. 生产服务器环境资产

| 资产类型 | 参数 / 地址 | 备注 / 鉴权信息 |
| :--- | :--- | :--- |
| **服务器公网 IP** | `47.238.2.197` | 阿里云 ECS (Debian 12, x86_64) |
| **SSH 远程登录** | `ssh root@47.238.2.197` (端口 22) | 密码：`Aa18979346882` |
| **Web 管理控制台** | `http://47.238.2.197:8787/enter` | 账号：`xmzd` / 密码：`a18979346882` |
| **本地自适应代理端口** | `127.0.0.1:7928` | HTTP/HTTPS CONNECT/SOCKS5/UDP |
| **安装与运行目录** | `/opt/aimilivpn/` | 包含二进制、配置、运行时 tun 证书等 |
| **终端快捷管理命令** | `nx` (软链接至 `/opt/aimilivpn/install.sh`) | 在服务器终端任意位置执行 |

### 2. 常用运维与管理指令

```bash
# 1. 极速更新至最新 Release 发行构建 (自动拉取未压缩 ELF 并比对 SHA-256)
nx update

# 2. 交互式 TUI 管理控制面板 (查看节点、启停服务、修改端口、配置 TG 等)
nx

# 3. 检查系统后台服务状态
systemctl status aimilivpn

# 4. 实时查看网关服务运行日志 (查看自动换流、测速与 AI 解锁情况)
journalctl -u aimilivpn -f

# 5. 重启网关服务
systemctl restart aimilivpn

# 6. 验证当前生效的程序版本
/opt/aimilivpn/aimilivpn --version
cat /opt/aimilivpn/VERSION

# 7. 查看当前活跃隧道与各隧道实测解锁状态
curl -s http://localhost:8964/api/tunnels | jq '.data[] | {id, ip: .node.ip, unlock}'
```

---

## 五、 CI/CD 自动化流水线与发布流程

项目已完全摆脱手动编译与手工上传 Release 附件的传统方式，实现了全流程自动化。

### 1. 核心流水线列表

1. **`.github/workflows/ci.yml` (代码质量准入)**：
   - 触发条件：对 `main` 分支的 push 与 pull request；
   - 执行阶段：
     - 版本单一信任源核验 (`scripts/check-version.sh`)
     - 代码与依赖已知 CVE 安全扫描 (`go vet` + `govulncheck`)
     - Go 后端全量子包单元与集成测试 (`go test -count=1 ./...`)
     - Playwright 4 端视口 UI 回归测试 (`npm --prefix web run test:ui`)
     - 多架构交叉编译完整性验证 (`scripts/build.sh`)
2. **`.github/workflows/release.yml` (自动发布流水线)**：
   - 触发条件：推送 `v*` 格式的 Git 标签；
   - 执行阶段：
     - 严格校验 Git Tag 与代码内部版本号是否一致；
     - 自动化交叉编译输出 4 大架构产物；
     - 自动上传 **9 大标准附件**至 GitHub Release。

### 2. 发布新版本极简 SOP

当您需要发布新版本（例如 `v2.5.2`）时，**仅需在本地运行一条辅助脚本**：

```bash
# 第一步：运行自动化发布助手
./scripts/release.sh 2.5.2
```

该脚本会自动执行：
1. 校验当前是否处于 `main` 分支；
2. 自动同步 `VERSION`、`pkg/config/version.go`、`install.sh`、`web/dist/index.html` 四处版本号；
3. 执行 `go vet` + `govulncheck` 安全审计；
4. 执行全量子包单元测试与 Playwright 4 视口回归测试；
5. 自动完成 commit 并打上 Git Tag `v2.5.2`；
6. 推送至 GitHub 并唤起 GitHub Actions 自动编译与附件上传。

发布完成后，线上任意服务器只需执行 `nx update` 即可完成秒级升级！

---

## 六、 目录结构与核心源码索引

```text
aimili-vpngate-go/
├── .github/workflows/
│   ├── ci.yml                 # GitHub Actions CI 测试与构建验证
│   └── release.yml            # GitHub Actions 自动化 Release 发布流水线
├── .githooks/
│   ├── commit-msg             # Conventional Commits 提交信息强制校验钩子
│   └── pre-push               # 推送前本地自动测试守门钩子
├── cmd/
│   └── aimilivpn/             # 网关主程序入口 (main.go)
├── config.env.example         # 环境变量配置标准示例模板
├── dist/                      # 交叉编译产物输出目录 (git 忽略)
├── docs/
│   ├── CICD_AND_TESTING_GUIDE.md   # CI/CD 发布与测试流程标准化指南
│   ├── FREESUB_COMPARISON.md       # 与 freesub 项目技术特性深度对比分析
│   └── PROJECT_HANDOVER.md         # [当前文件] 项目交接与工程维护文档 (系统全景/生产运维/故障排查)
├── install.sh                 # Linux 一键安装、服务部署与终端管理脚本 (nx)
├── pkg/
│   ├── config/                # 配置加载、版本定义 (version.go)
│   ├── nodes/                 # 节点拉取、端口预检 (pool.go)、住宅IP分类 (residential.go)
│   ├── notify/                # Telegram 告警推送与交互机器人
│   ├── proxy/                 # HTTP/SOCKS5 嗅探代理、SOCKS5 UDP Associate 转发
│   ├── server/                # Web 控制台 HTTP 路由、Clash/singbox 订阅生成
│   ├── singbox/               # 边缘抗封锁协议客户端 API 与 22 种原生协议元数据
│   ├── stats/                 # 实时上下行流量统计与速度滑动窗口跟踪器
│   └── tunnel/                # Linux 隧道池管理、带宽断流检测 (throughput.go)、AI解锁
├── scripts/
│   ├── audit.sh               # 静态分析与 govulncheck 依赖安全漏洞扫描
│   ├── build.sh               # 4 大架构交叉编译打包脚本
│   ├── check-version.sh       # 版本一致性核验与一键同步工具
│   └── release.sh             # 一键自动化发布助手脚本
├── web/
│   ├── dist/                  # 前端静态发布产物 (index.html, styles.css, app.js, 字体)
│   ├── tests/                 # Playwright 4 端视口自动化回归测试 (ui-smoke.mjs)
│   ├── embed.go               # Go embed 静态资产打包与防 XSS 安全测试
│   └── package.json
└── VERSION                    # 全局单一版本信任源定义文件
```

---

## 七、 常见问题排查 (Troubleshooting)

### 1. `nx update` 下载失败并回退到源码编译
- **原因**：Release 附件中缺少对应架构未压缩的 ELF 文件，或者 `SHA256SUMS.txt` 校验和未更新。
- **排查与解决**：
  - 检查 GitHub Release 页面是否包含未压缩的 `aimilivpn_linux_amd64` 等 9 个文件；
  - 确保使用 `./scripts/release.sh` 流程发布，避免手动上传时因网络中断导致附件进入 `state: starter` 不可用状态。

### 2. 某个出海节点频繁断开或连上后无网速
- **原因**：VPNGate 志愿者节点负载过高、被墙断流，或者属于假死节点。
- **自动处理机制**：系统已集成 Throughput 测速过滤与故障熔断器（Circuit Breaker），速率连续低于 70 KB/s 或握手超时的节点会被自动隔离至屏蔽库（Blacklist），并平滑递补可用节点。
- **手动恢复**：在 Web 控制台的“屏蔽库管理”弹窗中，可点击“探活复活节点”进行批量重新检测。

### 3. 本地 `git commit` 或 `git push` 被钩子拦截
- **Commit 被拦截**：提交信息必须带有 `feat:`, `fix:`, `docs:`, `refactor:`, `chore:` 等合法前缀。
- **Push 被拦截**：本地存在测试用例失败或代码中的版本定义不一致。可先运行 `./scripts/check-version.sh` 与 `go test ./...` 排查修复。
- **紧急跳过**：非生产代码紧急保存可附加 `--no-verify` 参数跳过本地钩子。

---

## 八、 后续技术演进建议与待办规划

1. **IPv6 双栈物理出海支持**：目前隧道与网关主要基于 IPv4 策略路由，未来可扩展对纯 IPv6 VPNGate 节点及出海隧道的支持；
2. **多节点聚合均衡 (Multi-WAN Bonding)**：在现有独立单连接轮询（Round-Robin）基础上，进一步引入基于多隧道的并发链路聚合加速；
3. **WebUI 实时拓扑链路图**：在控制台 Dashboard 增加客户端 -> 入站 sing-box -> 代理网关 -> VPNGate 住宅节点出口的动态可视化流量链路图。

---

**交接确认**：本项目所有模块均已通过严格的端到端测试与线上生产验证，系统处于高可用就绪状态。如有任何维护疑问，请参考 `docs/CICD_AND_TESTING_GUIDE.md` 及相关源码注释。
