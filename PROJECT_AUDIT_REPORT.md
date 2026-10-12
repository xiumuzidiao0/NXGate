# NXGate (自适应多出口智能路由网关) 项目综合审查与安全审计报告

**报告生成时间**：2026-10-12  
**项目版本**：`v2.6.5.9`  
**代码仓库**：`aimili-vpngate-go (NXGate)`  
**审查范围**：系统全景架构、核心网络栈实现、Web 与 API 安全、凭据与密钥管理、部署与构建流水线、依赖与配置健壮性  

---

## 目录

- [一、 审查概览与执行摘要](#一-审查概览与执行摘要)
  - [1.1 项目基本定位](#11-项目基本定位)
  - [1.2 综合健康度评分](#12-综合健康度评分)
  - [1.3 关键审计发现摘要表](#13-关键审计发现摘要表)
- [二、 核心架构与系统工程设计评估](#二-核心架构与系统工程设计评估)
  - [2.1 架构亮点与设计优势](#21-架构亮点与设计优势)
  - [2.2 核心子系统架构深剖](#22-核心子系统架构深剖)
  - [2.3 潜在架构隐患与系统局限性](#23-潜在架构隐患与系统局限性)
- [三、 安全漏洞与风险深度分析 (Security Audit)](#三-安全漏洞与风险深度分析-security-audit)
  - [3.1 [严重/高危] 在线自更新供应链安全隐患与未签名镜像信任](#31-严重高危-在线自更新供应链安全隐患与未签名镜像信任)
  - [3.2 [高危] install.sh 脚本中的 eval 动态命令注入漏洞](#32-高危-installsh-脚本中的-eval-动态命令注入漏洞)
  - [3.3 [高危] Web 配置保存接口的环境变量注入盲区 (CRLF / Env Injection)](#33-高危-web-配置保存接口的环境变量注入盲区-crlf--env-injection)
  - [3.4 [中高危] Basic Auth 凭据允许在 URL Query 参数中传输泄漏](#34-中高危-basic-auth-凭据允许在-url-query-参数中传输泄漏)
  - [3.5 [中危] 缺少 CSRF 跨站请求伪造防御机制与全局 CORS 风险](#35-中危-缺少-csrf-跨站请求伪造防御机制与全局-cors-风险)
  - [3.6 [中危] install.sh 随机密码生成算法中的熵池截断缺陷](#36-中危-installsh-随机密码生成算法中的熵池截断缺陷)
  - [3.7 [中危] Go 标准库上游已知 CVE 漏洞提示](#37-中危-go-标准库上游已知-cve-漏洞提示)
  - [3.8 [低中危] Web 控制台明文 HTTP 暴露与缺少内建 TLS](#38-低中危-web-控制台明文-http-暴露与缺少内建-tls)
- [四、 代码质量、健壮性与运行可靠性审查](#四-代码质量健壮性与运行可靠性审查)
  - [4.1 [严重] config.env.example 示例环境变量名与代码实际读取严重不匹配](#41-严重-configenvexample-示例环境变量名与代码实际读取严重不匹配)
  - [4.2 [重大] Docker 容器模式与底层网络驱动模式脱节](#42-重大-docker-容器模式与底层网络驱动模式脱节)
  - [4.3 [中危] Userland SSTP (Zero-TUN) 协议栈缺少 TCP 发送流控与重传队列](#43-中危-userland-sstp-zero-tun-协议栈缺少-tcp-发送流控与重传队列)
  - [4.4 [中危] SSTP Read 循环内频繁构造 time.AfterFunc 带来的 GC 压力](#44-中危-sstp-read-循环内频繁构造-timeafterfunc-带来的-gc-压力)
  - [4.5 [中危] 单元测试硬编码外部网络依赖导致测试耗时过长 (400s+)](#45-中危-单元测试硬编码外部网络依赖导致测试耗时过长-400s)
  - [4.6 [低危] 核心包 pkg/vpn 单元测试覆盖率为 0%](#46-低危-核心包-pkgvpn-单元测试覆盖率为-0)
- [五、 文档、构建工具链与 CI/CD 审查](#五-文档构建工具链与-cicd-审查)
  - [5.1 README.md 文档命令与 package.json 脚本不匹配](#51-readmemd-文档命令与-packagejson-脚本不匹配)
  - [5.2 多架构交叉编译与版本一致性检查工具链评价](#52-多架构交叉编译与版本一致性检查工具链评价)
  - [5.3 CI/CD 工作流的不足与改进建议](#53-cicd-工作流的不足与改进建议)
- [六、 修复建议与整改代码方案 (Actionable Remediation)](#六-修复建议与整改代码方案-actionable-remediation)
  - [6.1 修复 config.env.example 环境变量命名](#61-修复-configenvexample-环境变量命名)
  - [6.2 修复 install.sh 中的 eval 命令注入与熵截断](#62-修复-installsh-中的-eval-命令注入与熵截断)
  - [6.3 修复 UpdateSettings 的换行符注入盲区](#63-修复-updatesettings-的换行符注入盲区)
  - [6.4 修复自更新供应链签名验签机制](#64-修复自更新供应链签名验签机制)
  - [6.5 修复 Docker 默认驱动模式与自动降级逻辑](#65-修复-docker-默认驱动模式与自动降级逻辑)
  - [6.6 修复 Telegram 单元测试的 Mock 注入](#66-修复-telegram-单元测试的-mock-注入)
  - [6.7 补齐 web/package.json 中的 test 脚本指令](#67-补齐-webpackagejson-中的-test-脚本指令)
- [七、 总结与后续演进建议](#七-总结与后续演进建议)

---

## 一、 审查概览与执行摘要

### 1.1 项目基本定位

NXGate (`aimili-vpngate-go`) 是一款基于 Go 语言原生并发模型与 Linux 内核策略路由构建的高性能自适应多出口智能路由网关。其核心定位是：面向海量公网志愿节点进行动态发现、清洗与甄别（特别是识别原生住宅宽带 IP），并利用 Linux 底层策略路由 (`ip rule` + `table 100+N`) 与 `SO_BINDTODEVICE` 技术构建相互隔离的虚拟出口矩阵，同时向上游提供 SOCKS5 (RFC 1928 UDP Associate) 和 HTTP 代理协议，并与 sing-box 边缘抗审查协议入站进行链式分流。

### 1.2 综合健康度评分

| 维度 | 得分 (百分制) | 等级 | 简述 |
| :--- | :--- | :--- | :--- |
| **系统架构设计** | **90 / 100** | **优秀 (A)** | 内核策略路由防失联、纯用户态免 TUN (SSTP) 与 sing-box 边缘调度设计非常成熟 |
| **代码工程质量** | **82 / 100** | **良好 (B+)** | 代码结构整洁清晰，大量复用 `sync.Pool` 与 RingLog，但部分自定义网络栈缺乏完整流控 |
| **安全与防御机制** | **72 / 100** | **关注 (C+)** | 存在供应链未签名更新、Shell 脚本 `eval` 注入风险及环境变量 CRLF 盲区 |
| **配置与运维一致性** | **75 / 100** | **良好 (B)** | 示例配置文件与实际环境变量存在多处键名脱节；Docker 容器默认驱动模式缺少自适应 |
| **测试与 CI/CD 准入** | **78 / 100** | **良好 (B)** | CI 自动化流程完整，但测试中包含外部公网请求导致耗时异常；个别核心包覆盖率为空 |

### 1.3 关键审计发现摘要表

| 编号 | 严重级别 | 类别 | 缺陷摘要 | 影响文件 |
| :--- | :--- | :--- | :--- | :--- |
| **SEC-01** | **Critical** | 安全风险 | 自更新从第三方未验证镜像下载校验清单且无数字签名校验 | `pkg/server/update.go` |
| **SEC-02** | **High** | 安全漏洞 | `install.sh` 脚本在 `prompt_input` 中使用 `eval` 导致潜在命令注入 | `install.sh` |
| **SEC-03** | **High** | 安全漏洞 | Web 配置保存接口未对 `AgePublicKey` 等字段过滤换行符，存在环境变量注入 | `pkg/config/config.go` |
| **SEC-04** | **Medium-High** | 隐私/安全 | 支持在 URL Query 中明文传递 Basic Auth 账密，易在日志与历史中泄漏 | `pkg/server/middleware.go` |
| **BUG-01** | **Critical** | 功能缺陷 | `config.env.example` 环境变量命名与代码实际读取键名严重不一致 | `config.env.example` |
| **BUG-02** | **Major** | 容器缺陷 | Docker 镜像声明为纯用户态免 TUN，但未装 OpenVPN 且默认驱动为 Kernel | `Dockerfile`, `main.go` |
| **BUG-03** | **Medium** | 测试缺陷 | Telegram 单元测试硬编码公网 `api.telegram.org` 导致测试耗时 400s+ | `pkg/notify/telegram_test.go` |
| **DOC-01** | **Minor** | 文档/配置 | README 提及的 `cd web && npm test` 缺少 `package.json` 中的 script 别名 | `README.md`, `web/package.json` |

---

## 二、 核心架构与系统工程设计评估

### 2.1 架构亮点与设计优势

1. **Linux 内核策略路由强隔离与 SSH 防失联保护**：
   - 彻底解决了 OpenVPN 劫持宿主机全局网络的顽疾。启动参数注入了完整的过滤链：
     `route-noexec`, `route-nopull` 以及全量 `pull-filter ignore` 规则，禁止 OpenVPN 执行 `ip route` 修改主路由表或写入 `/etc/resolv.conf`。
   - 所有虚拟网卡绑定独占数字路由表 (`Table 100 + devIndex`)，默认路由严格限定在自身表中。
   - 启动即自动解析 `/etc/ssh/sshd_config` 及 `.d/` 下的所有 SSH 端口，并在最高优先级 `priority 50` 注入针对 SSH 源端口与目标端口走 `main` 表的策略规则，无论 VPN 发生何种网络异常，物理网卡 SSH 均绝不失联。

2. **纯用户态免 TUN (Zero-TUN SSTP) 协议栈集成**：
   - 自主实现了 SSTP 协议解析器与 IPv4/TCP/UDP 报文生成器，允许系统在没有 `/dev/net/tun` 设备或无 `CAP_NET_ADMIN` 特权的受限容器环境下通过用户态 Socket 直连节点。

3. **四阶段全生命周期节点甄选与清洗流水线**：
   - 聚合 10 个以上官方与第三方镜像源，采用竞速机制与 SingleFlight 防抖；
   - Phase 1 并发端口敲门 (Port Knock) 秒级剔除死节点；
   - Phase 2 真实吞吐切片采样（淘汰低于 70 KB/s 的节点）；
   - Phase 3 物理实测端点校验（同时校验 OpenAI、Claude、Gemini）；
   - Phase 4 结合 ASN 黑白名单、CDN 段过滤与 rDNS 反查的高置信家宽甄别。

4. **单二进制全内置静态打包与 age 加密订阅**：
   - SPA 前端资源与 sing-box 边缘脚本完全通过 Go 1.16+ `go:embed` 嵌入到单一 ELF 文件中，运行时无需外部静态资源服务器。
   - 订阅接口原生支持 RFC 5234 / RFC 7405 的 age 端到端加密（支持标准 X25519 及后量子抗算力的 MLKEM768-X25519 混合公钥规范），下发标准 ASCII Armored 密文，防止订阅在中间网络节点泄露。

### 2.2 核心子系统架构深剖

```text
┌────────────────────────────────────────────────────────────────────────┐
│                        外部客户端请求 (HTTP/SOCKS5)                     │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                      pkg/proxy: MultiPortManager                       │
│  - PortListener (单端口 HTTP CONNECT / SOCKS5 TCP / SOCKS5 UDP 嗅探)    │
│  - RFC 1928 UDP Associate 全链路代理 (socks5_udp.go)                   │
│  - Authenticator (支持随机密码、自定义账密、免密模式)                     │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
               M:N 调度器 (Round-Robin / Random / Interval / Sticky)
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                      pkg/tunnel & pkg/vpn 出口矩阵                      │
│                                                                        │
│   ┌──────────────────────────────────┐ ┌───────────────────────────┐   │
│   │   内核模式 (Kernel Mode)          │ │ 用户态模式 (Zero-TUN)     │   │
│   │   - OpenVPN 子进程 (tun0..tunN)  │ │ - SSTP 443 协议栈 (sstp/) │   │
│   │   - SO_BINDTODEVICE 套接字绑定   │ │ - 用户态合成 IP/TCP/UDP   │   │
│   │   - Table 100+N 隔离策略路由表   │ │ - 无需系统特权与虚拟网卡  │   │
│   └──────────────────────────────────┘ └───────────────────────────┘   │
└────────────────────────────────────────────────────────────────────────┘
```

### 2.3 潜在架构隐患与系统局限性

1. **Userland SSTP 模式缺乏完整的 TCP 状态机与可靠性流控**：
   - 详见后文 4.3 节。目前用户态实现为“尽力交付（Best-Effort）”伪 TCP，在丢包网络下缺乏重传保证。
2. **多端口并发代理的全局最大连接数并未全局控制**：
   - 规则层面的连接限制是按端口单独初始化的 `sem := make(chan struct{}, maxConn)`，若用户开启了 10 个代理端口且每个端口设置为 512，VPS 理论瞬时并发连接上限为 5120，可能耗尽宿主机底层 `ulimit -n` 文件描述符或内存。

---

## 三、 安全漏洞与风险深度分析 (Security Audit)

### 3.1 [严重/高危] 在线自更新供应链安全隐患与未签名镜像信任

- **位置**：`pkg/server/update.go:431-460` (`fetchExpectedSHA256`) 与 `PerformSelfUpdate`
- **问题分析**：
  在系统自更新流程中，程序通过公网拉取 `SHA256SUMS.txt` 校验和清单与预编译 ELF 可执行文件：
  ```go
  urls := []string{
      fmt.Sprintf("https://github.com/xiumuzidiao0/NXGate/releases/download/v%s/SHA256SUMS.txt", targetVer),
      fmt.Sprintf("https://ghproxy.net/https://github.com/xiumuzidiao0/NXGate/releases/download/v%s/SHA256SUMS.txt", targetVer),
      fmt.Sprintf("https://mirror.ghproxy.com/https://github.com/xiumuzidiao0/NXGate/releases/download/v%s/SHA256SUMS.txt", targetVer),
      ...
  }
  ```
  校验清单本身和二进制文件同时允许通过第三方非官方代理镜像源（如 `ghproxy.net`, `mirror.ghproxy.com`）下载。
  **漏洞隐患**：
  1. 如果第三方镜像源遭受投毒、DNS 劫持、或者恶意镜像节点返回被篡改的二进制文件，同时提供一个被篡改的 `SHA256SUMS.txt`，NXGate 会判定哈希匹配通过！
  2. 随后该恶意二进制直接以 root 特权原子替换 `/opt/nxgate/nxgate` 并重启 systemd 服务，造成严重的**供应链远程代码执行（RCE）**。
  3. 此外，更新完成后后台还静默下载 `install.sh` 并赋予 0755 可执行权限，同样未经过任何数字签名或哈希校验。
- **修复方案**：
  必须引入基于非对称密码学的签名校验（例如 Minisign、Cosign 或嵌入公钥的 Ed25519 签名）。在代码内硬编码开发者发布的 Release 公钥，下载更新时必须校验 `SHA256SUMS.txt.minisig` 签名，签名验证未通过前严禁执行替换。

### 3.2 [高危] install.sh 脚本中的 eval 动态命令注入漏洞

- **位置**：`install.sh:280-298` (`prompt_input` 函数)
- **源码片段**：
  ```bash
  prompt_input() {
      local prompt_msg="$1"
      local default_val="$2"
      local result_var="$3"
      local input_val=""
      ...
      if [ -z "$input_val" ]; then
          eval "$result_var=\"$default_val\""
      else
          eval "$result_var=\"$input_val\""
      fi
  }
  ```
- **漏洞隐患**：
  在 Bash 脚本中动态给变量赋值时使用了 `eval`。如果运维人员在终端输入内容、或者通过管道传递非交互式输入时，内容包含特殊字符（例如 `admin"; id; #` 或包含反引号、`$()` 等），`eval` 会将其作为原生 Shell 命令执行，导致任意代码注入。
- **修复方案**：
  使用 Bash 原生内置安全的变量赋值语法 `printf -v` 代替 `eval`：
  ```bash
  if [ -z "$input_val" ]; then
      printf -v "$result_var" '%s' "$default_val"
  else
      printf -v "$result_var" '%s' "$input_val"
  fi
  ```

### 3.3 [高危] Web 配置保存接口的环境变量注入盲区 (CRLF / Env Injection)

- **位置**：`pkg/config/config.go:214-230` (`UpdateSettings` 函数)
- **问题分析**：
  在处理管理员通过 Web API 提交的配置变更时，代码针对部分字符串字段做了换行符校验：
  ```go
  for name, value := range map[string]string{
      "ui_path":            dto.UIPath,
      "ui_username":        dto.UIUsername,
      "ui_password":        dto.UIPassword,
      "proxy_user":         dto.ProxyUser,
      "proxy_pass":         dto.ProxyPass,
      "telegram_bot_token": dto.TelegramBotToken,
      "telegram_chat_id":   dto.TelegramChatID,
  } {
      if strings.ContainsAny(value, "\r\n") {
          return fmt.Errorf("%s 不能包含换行符", name)
      }
  }
  ```
  **漏洞盲区**：
  上述检查遗漏了以下关键字段：
  - `dto.AgePublicKey`
  - `dto.AutoRotateIPType`
  - `dto.DiscoveryCountries`
  后续这些字段被直接通过 `fmt.Sprintf` 拼接写入到系统级文件 `/opt/nxgate/config.env` 中：
  ```go
  AGE_PUBLIC_KEY=%s
  AUTO_ROTATE_IP_TYPE=%s
  DISCOVERY_COUNTRIES=%s
  ```
  若攻击者提交的 `AgePublicKey` 包含 `\nNEW_VAR=malicious_val\n`，该恶意环境变量会在 systemd 启动网关时生效，构成环境变量注入漏洞。
- **修复方案**：
  将 `dto.AgePublicKey`、`dto.AutoRotateIPType` 以及 `dto.DiscoveryCountries` 的每个元素均加入 `\r\n` 强校验规则。

### 3.4 [中高危] Basic Auth 凭据允许在 URL Query 参数中传输泄漏

- **位置**：`pkg/server/middleware.go:73-82` (`BasicAuth` 中间件)
- **源码片段**：
  ```go
  user, pass, ok := r.BasicAuth()
  if !ok {
      qUser := r.URL.Query().Get("user")
      qPass := r.URL.Query().Get("pass")
      if qUser != "" && qPass != "" {
          user = qUser
          pass = qPass
          ok = true
      }
  }
  ```
- **风险分析**：
  为了方便部分无法配置 HTTP Basic Auth 请求头的客户端，代码放宽了认证逻辑，允许通过 URL 查询参数 `?user=xxx&pass=yyy` 完成认证。
  **安全风险**：
  1. URL 查询参数会被反向代理服务器（如 Nginx/Caddy/CDN）、系统访问日志以及网络监控系统以明文记录；
  2. 浏览器地址栏和历史记录会保存包含敏感密码的 URL；
  3. 若页面发生外部跳转，密码可能随 HTTP `Referer` 头泄漏给第三方站点。
- **修复建议**：
  仅针对移动端专用接口（如 `api/app/...`）使用短期 Token 或专用的安全 Header（如 `X-Auth-Token`），弃用在 GET/POST URL 查询参数中传递主账号明文密码的逻辑；或者至少在生产环境下输出严重安全告警。

### 3.5 [中危] 缺少 CSRF 跨站请求伪造防御机制与全局 CORS 风险

- **位置**：`pkg/server/middleware.go:34-45` (`SecurityHeaders`)
- **源码片段**：
  ```go
  if strings.HasPrefix(r.URL.Path, "/api/") || strings.Contains(r.URL.Path, "/api/") {
      w.Header().Set("Access-Control-Allow-Origin", "*")
      w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, X-Requested-With")
      w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
  ...
  ```
- **风险分析**：
  系统在 `/api/` 路由上无条件配置了 `Access-Control-Allow-Origin: *`。
  如果管理员在同一个浏览器中登录了 NXGate 管理面板（浏览器记住了 HTTP Basic Auth 认证凭据），随后访问了恶意的钓鱼网站，该钓鱼网站可以使用 `fetch()` 携带 Basic Auth 凭据对网关关键写接口（如 `/api/connect`、`/api/settings`、`/api/system/backup/import`、`/api/update/trigger`）发起跨站请求，执行未授权的配置更改甚至触发更新。
- **修复方案**：
  - 将 `Access-Control-Allow-Origin` 限定为同源或由配置指定的受信任域名；
  - 对所有变更状态的 POST/DELETE 接口强制校验请求的 `Origin` 或 `Referer` 标头；
  - 引入专用的 CSRF Token 或自定义请求头（如 `X-Requested-With: NXGate`）。

### 3.6 [中危] install.sh 随机密码生成算法中的熵池截断缺陷

- **位置**：`install.sh:269-277`
- **源码片段**：
  ```bash
  rand_pass() {
      local len="${1:-12}"
      head -c 32 /dev/urandom | tr -dc 'A-Za-z0-9' | head -c "$len" || echo "nxgate123"
  }
  ```
- **风险分析**：
  `/dev/urandom` 输出的每个字节为 0~255（共 256 种可能），而 `A-Za-z0-9` 仅包含 62 个字符，匹配率约为 `62 / 256 ≈ 24.2%`。
  当脚本仅读取 `head -c 32` 时，经过 `tr -dc` 过滤后的期望字符数仅为 `32 × 24.2% ≈ 7.7` 个字符。
  **后果**：
  当调用 `rand_pass 12` 期望生成 12 位密码时，由于经过过滤的可用字符平均只有 7~8 个，最终生成的密码长度严重不足（通常只有 7~8 位），削弱了控制台密码的抗爆破强度。
- **修复方案**：
  从 `/dev/urandom` 流式过滤而非预先截断输入字节：
  ```bash
  rand_pass() {
      local len="${1:-12}"
      LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c "$len" || echo "nxgate123"
  }
  ```

### 3.7 [中危] Go 标准库上游已知 CVE 漏洞提示

- **位置**：`go.mod` (指定 `go 1.25.13`)
- **审计发现**：
  使用官方安全审计工具 `govulncheck ./...` 扫描发现 10 处与 Go 1.25.13 标准库（`net/http`、`crypto/tls`、`net/textproto`）相关的已知上游漏洞：
  - `GO-2026-6611` / `GO-2026-6610`: HTTP/2 传输与窗口异常导致 CPU 消耗；
  - `GO-2026-6609`: `Range` 标头解析缺少大小限制导致内存开销；
  - `GO-2026-6607`: `crypto/tls` 中 ECH 扩展引用格式异常处理缺陷；
  - `GO-2026-6603`: HTTP/2 Trailer 标头内存消耗漏洞。
- **修复方案**：
  升级底层编译构建环境至 Go `1.26.9+`，确保在发布二进制产物时包含最新的安全修复补丁。

### 3.8 [低中危] Web 控制台明文 HTTP 暴露与缺少内建 TLS

- **位置**：`pkg/server/server.go:194`
- **问题分析**：
  Web 控制台与 API 接口原生仅支持明文 HTTP 监听。当用户在 VPS 上暴露公网 IP（`UI_HOST=0.0.0.0` 或 `::`）且未前置 Nginx/Caddy 等 TLS 反向代理时，管理员登录凭据、订阅 Token 及所有传输数据在公网均以明文传输，容易遭受中间人窃听与会话劫持。
- **建议**：
  在生产部署手册中强调必须配置反向代理，或在网关内原生集成基于 Let's Encrypt 的 ACME 自动化证书管理与自签名 TLS 降级支持。

---

## 四、 代码质量、健壮性与运行可靠性审查

### 4.1 [严重] config.env.example 示例环境变量名与代码实际读取严重不匹配

- **位置**：`config.env.example` vs `pkg/config/config.go`
- **问题分析**：
  工程根目录下的 `config.env.example` 是新用户和运维人员部署该网关的首要参考文件。然而，比对示例文件与 `pkg/config/config.go` 中的实际环境变量键名发现存在严重脱节：

| 示例文件 `config.env.example` | Go 代码实际读取 `pkg/config/config.go` | 默认值 | 导致后果 |
| :--- | :--- | :--- | :--- |
| `PROXY_HOST` | `LOCAL_PROXY_HOST` | `127.0.0.1` | 用户配置的代理监听地址被忽略 |
| `PROXY_PORT` | `LOCAL_PROXY_PORT` | `7928` | 用户自定义代理端口被忽略，强制为 7928 |
| `PROXY_USER` | `LOCAL_PROXY_USER` | *(随机)* | 用户自定义代理用户名失效，自动生成随机值 |
| `PROXY_PASS` | `LOCAL_PROXY_PASS` | *(随机)* | 用户自定义代理密码失效，自动生成随机值 |
| `PROXY_MAX_CONNECTIONS` | `LOCAL_PROXY_MAX_CONNECTIONS` | `512` | 并发连接上限设置失效 |
| `FETCH_INTERVAL_MINUTES` | `FETCH_INTERVAL_SECONDS` | `900s` | 节点抓取周期参数被忽略，固定为 15 分钟 |
| `OPENVPN_COMMAND` | `OPENVPN_CMD` | `openvpn` | 自定义 OpenVPN 二进制路径失效 |

- **危害程度**：
  严重！当外部用户复制 `config.env.example` 到生产环境使用时，所有自定义的本地代理端口、认证账密、并发限制和拉取间隔全部**静默失效**，回退为硬编码默认值或每次生成不同的随机凭据，极大破坏可用性。
- **修复方案**：
  必须立即同步 `config.env.example` 中的键名与 `pkg/config/config.go` 完全一致，或者在 `config.go` 中加入对旧键名/短键名的兼容回退查找。

### 4.2 [重大] Docker 容器模式与底层网络驱动模式脱节

- **位置**：`Dockerfile`, `docker-compose.yml`, `cmd/nxgate/main.go`
- **问题分析**：
  1. `docker-compose.yml` 明确注释：“纯用户态免 TUN 模式（无需任何 `--privileged`、无需 `CAP_NET_ADMIN`、无需 `/dev/net/tun`）”。
  2. 但检查 `Dockerfile` 发现：基础镜像为 `alpine:3.20`，只安装了 `ca-certificates tzdata`，**未安装 `openvpn`，未安装 `iproute2`**。
  3. 检查 `cmd/nxgate/main.go:145-151` 发现系统初始化流程：
     ```go
     best := nodePool.SelectBest()
     if best != nil {
         _ = vpnMgr.Connect(best) // 内部默认调用 ConnectWithMode(target, DriverKernel)
     }
     ```
  4. 检查 `pkg/vpn/manager.go` 发现：`DriverMode` 默认硬编码为 `DriverKernel`，并且在没有 TUN 或缺少 OpenVPN 时，`Connect` 失败后**没有任何自动降级为 `DriverUserland` 的自适应逻辑**！
- **后果**：
  按照官方 `docker-compose.yml` 启动容器后，网关在初次启动自动连接时，必定会尝试执行 `openvpn` 命令并报错退出；即使在 Web 控制台点击连接，除非用户手动在 API 显式传递 `mode=userland`，否则默认仍会尝试内核模式，导致 Docker 容器部署体验处于损坏状态。
- **修复方案**：
  1. 在 `main.go` 启动时自动检测当前系统是否存在 `/dev/net/tun` 以及 `openvpn` 可执行文件；
  2. 若检测到缺失，自动将全局默认模式切换为 `DriverUserland`；
  3. 支持通过环境变量 `DRIVER_MODE=userland` 允许显式指定驱动模式。

### 4.3 [中危] Userland SSTP (Zero-TUN) 协议栈缺少 TCP 发送流控与重传队列

- **位置**：`pkg/sstp/conn.go:95-138` (`Write` 方法)
- **问题分析**：
  在 `TCPConn.Write` 中：
  ```go
  for offset < total {
      chunkSize := total - offset
      if chunkSize > DefaultTCPMSS {
          chunkSize = DefaultTCPMSS
      }
      chunk := b[offset : offset+chunkSize]
      pkt := BuildIPv4TCPPacket(...)
      if err := c.tunnel.sendRaw(pkt); err != nil {
          return offset, err
      }
      c.seq += uint32(chunkSize)
      offset += chunkSize
  }
  ```
  **技术缺陷**：
  1. 该实现只是一个将字节切片直接打包并递增 `c.seq` 的发送器，完全没有维护**未确认报文队列（Unacknowledged Packets Queue）**与**超时重传定时器（RTO Timer）**；
  2. 没有考虑对端的接收通告窗口（Advertised Window / `c.rcvWnd`），如果在高并发大流量下，对端 TCP 窗口已满或发生丢包，本端无法获知哪些序列号需要重传，会导致上层长连接（如 HTTPS 下载）发生严重卡顿或连接超时中断。
- **建议**：
  目前 Zero-TUN 适合用于轻量级、低丢包的连通性探测和简单网页浏览。对于高吞吐生产环境，应在文档中明确推荐使用内核 TUN 模式；并在未来版本中完善轻量级滑动窗口重传队列。

### 4.4 [中危] SSTP Read 循环内频繁构造 time.AfterFunc 带来的 GC 压力

- **位置**：`pkg/sstp/conn.go:73-85`
- **源码片段**：
  ```go
  if !c.readDeadline.IsZero() {
      remain := time.Until(c.readDeadline)
      if remain <= 0 {
          return 0, ErrTimeout
      }
      t := time.AfterFunc(remain, func() {
          c.mu.Lock()
          c.readCond.Broadcast()
          c.mu.Unlock()
      })
      c.readCond.Wait()
      t.Stop()
  }
  ```
- **问题分析**：
  在 `for c.readBuf.Len() == 0` 循环内部，每次条件不满足等待时都会调用 `time.AfterFunc` 创建一个新的 goroutine 和 timer 结构体。如果并发连接数较多或数据到达呈现微小切片，会产生大量短暂的 timer 对象，增加 Go 运行时的 GC 压力与锁开销。
- **优化建议**：
  复用单个 `time.Timer`，在循环外通过 `timer.Reset()` 控制，或通过 channel select 替代 `sync.Cond`。

### 4.5 [中危] 单元测试硬编码外部网络依赖导致测试耗时过长 (400s+)

- **位置**：`pkg/notify/telegram_test.go:42` vs `pkg/notify/telegram.go:48`
- **问题分析**：
  运行 `go test ./...` 时，`pkg/notify` 测试耗时长达 **400 秒**！
  深入排查原因：
  `pkg/notify/telegram_test.go` 中虽然创建了一个本地测试服务器 `ts := httptest.NewServer(...)` 并注入了 `notifier.client = ts.Client()`，但是查看 `pkg/notify/telegram.go:SendMessage` 发现：
  ```go
  apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
  ```
  `SendMessage` 内部**硬编码**了官方 `https://api.telegram.org` 域名，完全没有使用 `ts.URL`！
  **后果**：
  当在无法直接访问 Telegram 官方 API 的网络环境（如普通境内服务器或隔离 CI 容器）下执行单元测试时，测试用例会发起真实的公网 HTTPS 请求并发生多次 35 秒超时重试，导致整个测试套件耗时超过 6 分钟！
- **修复方案**：
  在 `TelegramNotifier` 结构体中增加 `apiBaseURL` 字段（默认为 `https://api.telegram.org`），并在单元测试中将其重定向至 `ts.URL`，彻底解除单元测试对外网的硬依赖。

### 4.6 [低危] 核心包 pkg/vpn 单元测试覆盖率为 0%

- **位置**：`pkg/vpn/`
- **问题分析**：
  `pkg/vpn` 目录包含了系统的顶层控制器 `Manager`（负责故障转移、自动轮换、健康探活及驱动模式切换），但该目录下没有任何 `*_test.go` 文件，测试覆盖率为 `0.0%`。
- **建议**：
  为 `pkg/vpn` 补充针对状态快照、自动故障转移算法、驱动模式切换的单元测试，提升核心逻辑的回归测试保障。

---

## 五、 文档、构建工具链与 CI/CD 审查

### 5.1 README.md 文档命令与 package.json 脚本不匹配

- **位置**：`README.md:297` vs `web/package.json:4-6`
- **问题分析**：
  `README.md` 的“自动化质量准入审计”章节指示开发者运行：
  ```bash
  cd web && npm test
  ```
  但在 `web` 目录下执行该命令时会直接报错：
  `npm error Missing script: "test"`。
  查看 `web/package.json`：
  ```json
  "scripts": {
    "build": "esbuild src/app.js --bundle ...",
    "test:ui": "node tests/ui-smoke.mjs"
  }
  ```
  实际提供的脚本名称为 `test:ui`，缺少标准的 `test` 别名。
- **修复方案**：
  在 `web/package.json` 的 `scripts` 中增加 `"test": "npm run test:ui"`。

### 5.2 多架构交叉编译与版本一致性检查工具链评价

- **优点**：
  1. `scripts/build.sh` 设计优雅，原生关闭 CGO（`CGO_ENABLED=0`），可在秒级一次性产出 Linux `amd64`, `arm64`, `386`, `arm` 四种架构的纯静态无依赖二进制包，并生成配套的 `SHA256SUMS.txt`；
  2. `scripts/check-version.sh` 实现了对 `VERSION`、`pkg/config/version.go`、`install.sh`、`web/dist/index.html` 以及 `android/app/build.gradle.kts` 全链路 5 处核心版本定义的自动化比对与一键同步功能，有效避免了多组件版本脱节。

### 5.3 CI/CD 工作流的不足与改进建议

- **审计发现**：
  1. `.github/workflows/ci.yml` 覆盖了 Go 静态检查、漏洞扫描、单元测试与 Playwright 自动化 UI 冒烟测试，覆盖度较高；
  2. `.github/workflows/release.yml` 在生成发布产物时未对发布包执行数字签名（如利用 GitHub OIDC 颁发 Cosign 签名证书）；结合前述 3.1 节自更新漏洞，建议在 Release 工作流中加入 Cosign / Minisign 签名步骤，并在客户端加入验签逻辑。

---

## 六、 修复建议与整改代码方案 (Actionable Remediation)

### 6.1 修复 config.env.example 环境变量命名

修改 `config.env.example`，将过时的变量名纠正为与 `pkg/config/config.go` 完全一致：

```bash
# ----------------- 2. 本地代理网关配置 -----------------
# 代理监听地址与默认端口 (HTTP / HTTPS CONNECT / SOCKS5 / SOCKS5 UDP)
LOCAL_PROXY_HOST=0.0.0.0
LOCAL_PROXY_PORT=7928

# 代理账密 (留空表示自动生成高强度随机凭据)
LOCAL_PROXY_USER=
LOCAL_PROXY_PASS=

# 最大并发代理连接数限制 (默认 512，范围 16-4096)
LOCAL_PROXY_MAX_CONNECTIONS=512

# ----------------- 3. 节点池与发现参数 -----------------
DISCOVERY_COUNTRIES=JP,US
TARGET_VALID_NODES=5
MAX_SCAN_ROWS=1000

# 节点更新拉取与健康探测周期 (单位: 秒)
FETCH_INTERVAL_SECONDS=900
CHECK_INTERVAL_SECONDS=20

# ----------------- 5. 系统路径与底层进程 -----------------
DATA_DIR=data
OPENVPN_CMD=openvpn
OPENVPN_AUTH_USER=vpn
OPENVPN_AUTH_PASS=vpn
```

### 6.2 修复 install.sh 中的 eval 命令注入与熵截断

在 `install.sh` 中修改 `prompt_input` 和随机函数：

```bash
# 修复前:
# eval "$result_var=\"$input_val\""

# 修复后 (使用 printf -v 安全动态变量赋值):
prompt_input() {
    local prompt_msg="$1"
    local default_val="$2"
    local result_var="$3"
    local input_val=""

    if [ -t 0 ]; then
        read -p "$prompt_msg" input_val
    elif [ -c /dev/tty ]; then
        read -p "$prompt_msg" input_val </dev/tty
    fi

    if [ -z "$input_val" ]; then
        printf -v "$result_var" '%s' "$default_val"
    else
        printf -v "$result_var" '%s' "$input_val"
    fi
}

# 修复前:
# head -c 32 /dev/urandom | tr -dc 'A-Za-z0-9' | head -c "$len"

# 修复后 (连续流式读取，杜绝熵截断):
rand_str() {
    local len="${1:-8}"
    LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c "$len" || echo "aimili"
}

rand_pass() {
    local len="${1:-12}"
    LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c "$len" || echo "nxgate123"
}
```

### 6.3 修复 UpdateSettings 的换行符注入盲区

在 `pkg/config/config.go` 的 `UpdateSettings` 函数头部补充过滤：

```go
func (c *Config) UpdateSettings(dto SettingsDTO) error {
	checkFields := map[string]string{
		"ui_path":               dto.UIPath,
		"ui_username":           dto.UIUsername,
		"ui_password":           dto.UIPassword,
		"proxy_user":            dto.ProxyUser,
		"proxy_pass":            dto.ProxyPass,
		"telegram_bot_token":    dto.TelegramBotToken,
		"telegram_chat_id":      dto.TelegramChatID,
		"age_public_key":        dto.AgePublicKey,
		"auto_rotate_ip_type":   dto.AutoRotateIPType,
	}
	for name, value := range checkFields {
		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("%s 不能包含换行符", name)
		}
	}
	for _, country := range dto.DiscoveryCountries {
		if strings.ContainsAny(country, "\r\n") {
			return fmt.Errorf("国家代码不能包含换行符")
		}
	}
	// ... 后续逻辑保持不变
```

### 6.4 修复自更新供应链签名验签机制

在 `pkg/server/update.go` 中引入嵌入式公钥，并在下载二进制后要求校验数字签名：

```go
// 开发者内嵌固定 Minisign / Ed25519 验签公钥
const releasePublicKey = "RWQxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

func VerifyReleaseSignature(data []byte, sigData []byte) error {
    // 校验签名真实性，防止镜像被篡改或中间人攻击
    // 未通过验签时直接阻断更新
    return nil
}
```
同时在发布阶段（`release.yml`）自动运行 `minisign -S -s release.key -m dist/SHA256SUMS.txt` 生成签名文件。

### 6.5 修复 Docker 默认驱动模式与自动降级逻辑

在 `pkg/vpn/manager.go` 初始化或连接前检测系统能力：

```go
func (m *Manager) autoDetectDriverMode() DriverMode {
    if runtime.GOOS != "linux" {
        return DriverUserland
    }
    // 检查 /dev/net/tun
    if err := CheckTUNDevice(); err != nil {
        return DriverUserland
    }
    // 检查 openvpn 是否可用
    if _, err := exec.LookPath(m.cfg.OpenVPNCommand); err != nil {
        return DriverUserland
    }
    return DriverKernel
}
```
当系统缺少 OpenVPN 或 TUN 网卡时，`Manager` 自动切换为 `DriverUserland` 纯用户态模式，实现开箱即用的无特权 Docker 运行体验。

### 6.6 修复 Telegram 单元测试的 Mock 注入

在 `pkg/notify/telegram.go` 增加 `apiBaseURL` 属性：

```go
type TelegramNotifier struct {
	cfg        *config.Config
	client     *http.Client
	apiBaseURL string // 默认为 "https://api.telegram.org"
	lastID     int64
}

func NewTelegramNotifier(cfg *config.Config) *TelegramNotifier {
	return &TelegramNotifier{
		cfg:        cfg,
		client:     &http.Client{Timeout: 35 * time.Second},
		apiBaseURL: "https://api.telegram.org",
	}
}
```
并在 `pkg/notify/telegram_test.go` 中注入测试地址：
```go
notifier := NewTelegramNotifier(cfg)
notifier.client = ts.Client()
notifier.apiBaseURL = ts.URL // 重定向至本地 Mock 服务器，消除外部请求超时
```

### 6.7 补齐 web/package.json 中的 test 脚本指令

在 `web/package.json` 的 `scripts` 节点中增加：

```json
"scripts": {
  "build": "esbuild src/app.js --bundle --minify-whitespace --minify-syntax --format=iife --target=es2020 --outfile=dist/app.js",
  "test": "npm run test:ui",
  "test:ui": "node tests/ui-smoke.mjs"
}
```

---

## 七、 总结与后续演进建议

NXGate 是一个工程完成度极高、定位精准、架构设计富有创新的专业级多出口路由网关系统。其在**内核策略路由防失联模型**、**SOCKS5 UDP Associate 穿透**、**四阶段节点清洗**、**age 端到端加密**以及**单二进制自包容交付**方面展现了高水准的系统工程能力。

本次审查识别出的核心问题主要集中在：
1. **外部供应链与更新防篡改安全**（自更新缺乏公钥数字签名、Shell `eval` 命令注入）；
2. **环境配置与文档的一致性**（`config.env.example` 命名脱节、Docker 模式缺乏自适应降级）；
3. **测试隔离性**（单元测试依赖外网服务）。

在实施本报告第六章提出的修复方案后，系统将彻底消除上述高危隐患，并在高可用、高安全性与工程规范性上达到企业生产就绪（Production-Ready）的标准。
