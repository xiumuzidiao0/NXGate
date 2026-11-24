# NXGate

[简体中文](README.md) | [English](README_EN.md)

[![Release](https://img.shields.io/github/v/release/xiumuzidiao0/NXGate?style=flat-square&label=Release)](https://github.com/xiumuzidiao0/NXGate/releases/latest)
[![CI](https://img.shields.io/github/actions/workflow/status/xiumuzidiao0/NXGate/ci.yml?branch=main&style=flat-square&label=CI)](https://github.com/xiumuzidiao0/NXGate/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-334155?style=flat-square)](LICENSE)

NXGate 是一个面向 Linux 服务器的多出口代理网关，用于维护并发 VPN 隧道或外部代理，并将本地 HTTP / SOCKS5 代理端口按策略路由到筛选后的出口节点；支持可选集成 `sing-box` 入站与加密订阅分发，内置 Web 管理控制台并提供配套 Android 客户端。

---

## 核心特性

- **双连接驱动与路由隔离**
  - **内核 TUN（OpenVPN）**：通过 `route-noexec` / `route-nopull` 与策略路由表（`100 + devIndex`）隔离各隧道流量，同时保留宿主机 SSH 与默认路由。
  - **用户态 SSTP（Zero-TUN）**：无需 `/dev/net/tun` 设备与 `CAP_NET_ADMIN` 特权即可在常规容器或非 Linux 环境下建连出网。
- **节点筛选与动态出口组**
  - 支持从多源镜像、本地快照及自定义外部 HTTP/SOCKS5 代理中筛选可用节点，支持按国家/地区、IP 类型（住宅 / 机房）、风险评分及目标服务连通性排序与过滤。
  - 支持固定绑定或按策略自动维护主出口与并发动态组，内置健康探测与失败降权/隔离机制。
- **多端口代理中继**
  - 单端口兼容 HTTP CONNECT 与 SOCKS5（TCP + UDP Associate），可绑定全部健康隧道、指定隧道或动态出口组，支持 `round_robin`、`random`、`interval` 与 `least_rtt` 调度。
- **`sing-box` 入站与订阅分发（可选）**
  - 统一管理多协议入站并将其出站链式指向直连或任意本地代理端口。
  - 由 Web 端口直接提供通用订阅（Raw/Base64）与 Clash YAML，支持随机 `SUB_TOKEN` 隔离及 `age`（`X25519` / `MLKEM768-X25519`）端到端加密。
- **多端控制**
  - 单个 Go 静态二进制通过 `go:embed` 内嵌中/英双语 Web 控制台，支持 SSE 实时状态与配置持久化。
  - 配套 Kotlin / Jetpack Compose Android 客户端，支持扫码录入配置、生物识别锁及远程策略切换。

---

## 快速部署

### Linux（systemd）

在支持 `systemd` 的 Linux 主机（Debian / Ubuntu / CentOS / RHEL / Rocky / AlmaLinux / Fedora）上下载并运行安装脚本：

```bash
curl -fL https://raw.githubusercontent.com/xiumuzidiao0/NXGate/main/install.sh -o install.sh && bash install.sh
```

- 如需先审阅脚本内容再执行，可只运行前半段 `curl` 下载到本地后查看 `install.sh`，再运行 `bash install.sh`。
- 脚本需在交互终端（`/dev/tty`）下确认后才会安装依赖，默认跳过 `sing-box`，且不自动修改系统防火墙、SSH 规则或策略路由；旧版脚本此前写入的规则不会自动删除。

### Docker Compose

仓库提供的 [`docker-compose.yml`](docker-compose.yml) 默认以用户态模式运行（无需特权与 TUN 设备）：

```bash
docker compose up -d
```

默认将 Web 端口映射至 `0.0.0.0:8787`，代理端口仅绑定在 `127.0.0.1:7928`。若仅需本地访问控制台，可在 `.env` 中设置 `NXGATE_UI_PUBLISH_HOST=127.0.0.1` 后重启容器。

### 常用管理命令

安装器部署后会注册 `nx` 命令（以及 `nxgate` systemd 服务）：

```bash
nx               # 打开交互式终端菜单
nx status        # 查看服务状态、访问地址与监听信息
nx start         # 启动服务
nx stop          # 停止服务
nx restart       # 重启服务
nx update        # 检查并更新发行二进制
journalctl -u nxgate -f
```

---

## 配置说明

- **配置来源与优先级**：systemd 部署默认读取 `/opt/nxgate/config.env` 环境变量；通过网页或 API 保存的修改持久化在 `${DATA_DIR}/settings.json`，并在后续启动时优先于对应初始化环境变量。裸二进制直接运行仅读取当前进程环境变量。
- **安全默认值**：
  - 首次未指定 `UI_PATH` / `UI_PASSWORD` 时会生成随机管理路径（`${DATA_DIR}/admin_path.txt`）与随机密码（`${DATA_DIR}/ui_password.txt`），默认用户名 `admin`。
  - 允许在设置中显式清空 `ui_path` 或勾选关闭 Web 认证；若控制台处于 `0.0.0.0` 等通配监听且未开启认证，服务端日志与 Web 控制台会持续展示安全警告。公网部署建议前置 HTTPS 反向代理并限制来源 IP。
  - 代理端口默认绑定 `127.0.0.1`，不直接暴露公网。

| 环境变量 | 默认值 | 说明 |
| :--- | :--- | :--- |
| `UI_HOST` | `0.0.0.0` | Web 控制台监听地址 |
| `UI_PORT` | `8787` | Web 控制台端口（`1-65535`） |
| `UI_PATH` | 随机生成并持久化 | 控制台访问路径前缀 |
| `UI_USERNAME` / `UI_PASSWORD` | `admin` / 随机密码 | Web Basic Auth 凭据 |
| `SUB_TOKEN` | 随机生成并持久化 | 订阅链接访问令牌 |
| `AGE_ENCRYPT_ENABLED` | `false` | 是否开启订阅 `age` 加密（`true` / `false`） |
| `AGE_PUBLIC_KEY` | 空 | `age1...` 或 `age1pq...` 接收端公钥 |
| `LOCAL_PROXY_HOST` | `127.0.0.1` | 默认代理监听地址 |
| `LOCAL_PROXY_PORT` | `7928` | 默认代理端口（`1-65535`） |
| `LOCAL_PROXY_GLOBAL_MAX_CONNECTIONS` | `1024` | 全局代理总并发上限（`1-65536`） |
| `LOCAL_PROXY_MAX_CONNECTIONS` | `512` | 单端口并发上限（`16-4096`） |
| `DISCOVERY_COUNTRIES` | 空 | 逗号分隔的国家过滤列表（如 `JP,US`，留空为全球） |
| `DATA_DIR` | `data` | 运行时状态与配置持久化目录 |

---

## 源码构建与验证

前置依赖：Go `1.26+`、Node.js `20+`（仅修改 Web 前端时需要）、JDK `21` 与 Android SDK（仅构建或单测 Android 客户端时需要）。

```bash
# 1. 构建 Web 前端产物（如修改 web/src/*）
npm --prefix web install
npm --prefix web run build

# 2. 编译服务端二进制或四架构发行包
CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/nxgate ./cmd/nxgate
./scripts/build.sh

# 3. 运行验证测试
go vet ./...
go test -count=1 ./...
npm --prefix web run test:i18n
npm --prefix web run test:status
npm --prefix web run test:ui
./scripts/check-version.sh
```

Android 单元测试可在 `android/` 目录下执行 `./gradlew testDebugUnitTest --no-daemon`。

---

## 开源协议

基于 [MIT License](LICENSE) 开源。
