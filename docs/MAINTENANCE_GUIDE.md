# NXGate 长期工程维护与运维手册 (Long-Term Maintenance Guide)

**适用版本**：`v2.6.1.1` 及以上  
**维护基线**：Go 1.23+、Android SDK 35 (Compose M3)、Web 原生现代化深空架构、sing-box 1.10+  
**最后修订**：2026-09-30  

---

## 目录

1. [系统架构与关键端口职责矩阵](#一-系统架构与关键端口职责矩阵)
2. [日常巡检与核心运维指令 (Daily Runbook)](#二-日常巡检与核心运维指令-daily-runbook)
3. [版本迭代与标准化发版流程 (Release SOP)](#三-版本迭代与标准化发版流程-release-sop)
4. [Git 仓库防膨胀与数据分支维护规范](#四-git-仓库防膨胀与数据分支维护规范)
5. [生产环境安全配置与加固基线](#五-生产环境安全配置与加固基线)
6. [常见突发故障与应急急救手册 (Emergency Runbook)](#六-常见突发故障与应急急救手册-emergency-runbook)
7. [组件升级与技术栈周期维护表](#七-组件升级与技术栈周期维护表)

---

## 一、 系统架构与关键端口职责矩阵

### 1. 核心端口与业务严格隔离规范

系统在架构设计上对**管理控制端口**与**数据代理端口**实行了绝对物理隔离：

| 端口/范围 | 协议/用途 | 鉴权机制 | 安全与架构约束 |
| :--- | :--- | :--- | :--- |
| **8787** (默认) | Web 控制台、RESTful API、SSE 实时监控流 | HTTP Basic Auth + 隐蔽路径 (`UIPath`) | **严禁参与任何代理流量转发**！拓扑流向图及代理调度中绝不可出现此端口。 |
| **7928** (默认) | 统一主出海代理网关 (SOCKS5 / HTTP / CONNECT) | 可选自定义独立账密 (NoAuth / UserPass) | 默认绑定系统主出海隧道 (`tun0`)，具备全局保底与热备自动重连能力。 |
| **7929 ~ 7940** | 多端口动态分流监听端口 | 独立端口级账密 | 按端口分流规则绑定指定国家、住宅出口组或特定静态隧道，独立承载业务流量。 |
| **外部动态端口** | sing-box 入站协议 (VLESS, Hy2, TUIC 等) | 协议原生认证 (UUID / Password / PBK) | 负责从客户端接收流量，内部链式路由至 `127.0.0.1:7928+` 本地网关。 |

### 2. 核心源码分层映射

```
cmd/nxgate/main.go            ──▶ 系统守护进程入口、信号捕获与优雅退出
  ├── pkg/config/             ──▶ 配置文件、环境变量读取与安全防注入持久化
  ├── pkg/nodes/              ──▶ 节点池聚合清洗、家宽识别(ISP/ASN/rDNS)、双层黑名单、CIDR子网屏蔽
  ├── pkg/tunnel/             ──▶ 虚拟网卡池(tun0..tunN)、独立策略路由隔离、动态出口组自适应评估
  ├── pkg/proxy/              ──▶ 本地统一代理网关、SOCKS5 UDP Associate、单连接/定时多策略调度
  ├── pkg/server/             ──▶ RESTful API、SSE 推送、订阅生成(age加密/标准化命名)、灾备导入导出
  ├── pkg/singbox/            ──▶ sing-box 进程管理、22种抗审查协议配置与实时状态监听
  ├── web/                    ──▶ 现代化深空 Bento 控制台 (原生纯 JS + CSS, 无庞大打包依赖)
  └── android/                ──▶ Android 原生远程管理端 (Jetpack Compose + Material 3, 纯管理控制)
```

---

## 二、 日常巡检与核心运维指令 (Daily Runbook)

### 1. 服务控制与状态查看

```bash
# 1. 查看 NXGate 核心服务运行状态
systemctl status nxgate

# 2. 实时跟踪核心运行日志与网络探测
journalctl -u nxgate -f -n 100

# 3. 使用命令行内置客户端查看快速状态 (包含各 tun 网卡连通状态)
nx status
# 或
/opt/nxgate/nxgate --status

# 4. 重启 / 优雅重载服务
systemctl restart nxgate
```

### 2. Linux 内核策略路由与网络接口巡检

当多出口并发运行（如同时开启 3 个出口网卡）时，Linux 内核会创建多个策略路由表：

```bash
# 检查当前虚拟网卡接口
ip addr show | grep -E 'tun[0-9]+'

# 检查当前所有策略路由规则 (应看到针对各 tunX 的 priority 1000 规则)
ip rule show

# 检查各分流路由表的内容 (以 table 100 为例)
ip route show table 100

# 测试指定虚拟网卡是否具备真实外网出海能力
curl --interface tun0 -m 5 https://api.ipify.org
curl --interface tun1 -m 5 https://api.ipify.org
```

### 3. 定期灾备与配置备份

- **WebUI 导出**：进入控制台「系统维护」设置项，点击「下载灾备配置文件」；
- **命令行自动备份**：
  ```bash
  # 配置文件默认存储于 /opt/nxgate/data/
  tar -czvf /backup/nxgate-config-$(date +%Y%m%d).tar.gz /opt/nxgate/data/
  ```

---

## 三、 版本迭代与标准化发版流程 (Release SOP)

为保证整个系统（二进制、脚本、前端、Android）绝对一致，发版严格遵循 **5 处核心版本源联动校验机制**。

### 1. 版本一致性 5 处核心源

1. `VERSION`（根目录版本标记文件）
2. `pkg/config/version.go`（Go 二进制内置编译常量）
3. `install.sh`（一键安装脚本内的 `DEFAULT_VERSION`）
4. `web/dist/index.html`（Web 控制台顶栏与侧栏版本徽章）
5. `android/app/build.gradle.kts`（Android 客户端的 `versionName`）

### 2. 自动化发版步骤 (5 步走)

```bash
# 第一步：确保所有功能开发与测试已通过，本地工作区干净
git status

# 第二步：使用专用同步工具一键更新 5 处版本号 (例如升级为 2.6.2)
./scripts/check-version.sh --set 2.6.2

# 第三步：编译打包并生成 Android Release APK (保持本地签名)
cd android && ./gradlew assembleRelease --no-daemon && cd ..
cp -f android/app/build/outputs/apk/release/app-release.apk nxgate-release.apk

# 第四步：执行全套质量准入审计与单元测试 (需 100% PASS)
npm --prefix web run test:ui
go test ./...

# 第五步：提交代码并打标签推送到远程 (触发 GitHub Actions 自动编译与 Release 发布)
git add -A && git commit -m "chore: bump version to v2.6.2"
git tag -a "v2.6.2" -m "Release v2.6.2"
git push origin main
git push origin v2.6.2
```

---

## 四、 Git 仓库防膨胀与数据分支维护规范

### 1. 核心架构原则：代码分支与数据分支彻底分离

由于 VPNGate 节点列表每 6 小时自动聚合更新一次（单文件约 16MB），**绝对不可将其提交至 `main` 主分支**！

- **`main` 分支**：仅用于存放 Go 源码、Web 前端代码、Android 客户端、文档与安装脚本，**永远保持轻量，供开发者与服务器快速 `git clone`**；
- **`data` 独立分支**：专门用于承载高频更新的 `mirror/vpngate.csv`、`mirror/vpngate.csv.gz` 与元数据；
  - 由 `.github/workflows/mirror.yml` 自动维护；
  - 采用 **孤儿分支 + 单 Commit 覆盖推（`git push -f origin data`）**；
  - 该分支在 GitHub 上的历史记录恒定保持为 1 条，杜绝膨胀。

### 2. 本地 Git 仓库垃圾回收 (定期巡检)

若本地 `.git` 体积出现异常增大，执行以下标准深度打包指令即可快速瘦身：

```bash
# 查看松散对象与打包占用
git count-objects -vH

# 执行深度回收与紧密重打包 (实测可立减 70% 以上磁盘占用)
git gc --aggressive --prune=now
```

---

## 五、 生产环境安全配置与加固基线

### 1. 网络与访问安全
- **隐蔽路径 (`UI_PATH`)**：不要将 Web 控制台直接暴露在 `/` 根目录。初始化时务必设置随机路径（如 `UI_PATH=gate-x89k`），未经授权访问根目录或 API 将静默返回伪装 404；
- **安全反向代理**：对外暴露时，强烈建议前置 Caddy 或 Nginx 开启全链路 HTTPS (TLS 1.3)，关闭外部直接 HTTP 明文访问；
- **隔离代理端口与管理端口**：防火墙仅对外暴露指定的代理端口（如 7928+）与 sing-box 监听端口，8787 控制端口仅限管理 IP 或内网访问。

### 2. 凭证与订阅安全
- **订阅 Token 独立化**：订阅鉴权只使用系统生成的安全随机 Token（`sub_token`）或 `age` 密文，**严禁将 WebUI 管理员密码用作订阅参数**；
- **Android 安全防护**：
  - `AndroidManifest.xml` 中已强制锁定 `android:allowBackup="false"`，杜绝通过调试导出凭证；
  - `network_security_config.xml` 中用户根证书严格限制在 `<debug-overrides>`，生产包绝不信任用户自签 CA，免疫中间人流量窥探。

---

## 六、 常见突发故障与应急急救手册 (Emergency Runbook)

### 故障 1：Web 控制台打不开 (8787 端口拒绝连接或无响应)
1. **排查进程是否存活**：
   ```bash
   pgrep -a nxgate
   ```
   若进程不在，检查日志确认是否因端口冲突退出：`journalctl -u nxgate -e`。
2. **排查监听绑定地址**：
   检查 `config.env` 中的 `UI_HOST`。若配置为 `127.0.0.1` 则外部无法直接访问，需配置为 `::` 或 `0.0.0.0`。
3. **排查隐蔽路径**：
   访问时必须带上完整的隐蔽路径：`http://服务器IP:8787/你的隐蔽路径`。

### 故障 2：所有代理端口断流，节点不可达
1. **检查虚拟主出口 `tun0` 是否在线**：
   ```bash
   ip link show tun0
   ```
2. **检查底层 OpenVPN 进程集群**：
   ```bash
   ps aux | grep openvpn
   ```
3. **快速触发重连**：
   在 Web 控制台节点方块点击任意日本/美国节点旁边的「连接」，或执行：
   ```bash
   systemctl restart nxgate
   ```

### 故障 3：策略路由混乱或网络残留 (网卡删除后路由未清空)
若反复暴力 kill 导致 Linux 内核路由表残留，执行以下清理脚本恢复初始状态：
```bash
# 删除所有自定义 priority 1000 的策略路由规则
while ip rule del priority 1000 2>/dev/null; do :; done

# 清理可能残留的失效虚拟网卡
for dev in $(ip link show | grep -o 'tun[0-9]\+'); do
    ip link delete "$dev" 2>/dev/null || true
done

# 重启服务重新建立干净网卡
systemctl restart nxgate
```

### 故障 4：客户端导入订阅后节点名显示错误或乱码
- NXGate `v2.6.1.1` 起全面标准化为 **`{国家属地}-{协议}-{端口}`**（如 `JP-VLESS-443`、`KR-Hysteria2-8443`）；
- 若使用的旧客户端需要历史的出口组名，在订阅链接后增加 `?naming=group` 即可无感回退至 `[出口组名] 原节点名` 格式。

---

## 七、 组件升级与技术栈周期维护表

| 组件 / 模块 | 建议检查周期 | 升级与维护方法 | 重点注意事项 |
| :--- | :--- | :--- | :--- |
| **Go 运行与编译环境** | 每 6 个月 | 升级 VPS 本地 Go 版本至最新次高稳定版 (Minor Release) | 升级后执行 `go test ./...` 确保无任何语法/编译兼容问题。 |
| **sing-box 核心** | 每 3 个月 | 运行 `sing-box.sh` 或更新 `/usr/local/bin/sing-box` | 检查入站配置兼容性，核验 `pkg/singbox/protocols.go` 中的 API 字段。 |
| **Android 客户端** | 每季度 | 更新 Gradle Plugin、Compose 库、Android SDK target | 运行 `./gradlew assembleRelease` 确保 Lint 零 Fatal 报错。 |
| **依赖安全审计** | 每月 | 执行 `govulncheck ./...` 与 `npm --prefix web audit` | 重点排查网络代理层与加密库（filippo.io/age）的 CVE 漏洞通报。 |
