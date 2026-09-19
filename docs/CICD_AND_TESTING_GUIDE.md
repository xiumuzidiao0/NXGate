# AimiliVPN CI/CD 发布与测试流程标准化规范文档

本文档定义了 AimiliVPN（Go 高性能网关版）的代码质量准入、多端自动化测试、版本一致性同步、多架构交叉编译打包、GitHub Release 发布以及线上服务器热更新的标准操作流程（SOP）。

---

## 目录

- [一、 架构总览与发布生命周期](#一-架构总览与发布生命周期)
- [二、 自动化测试流程规范](#二-自动化测试流程规范)
  - [1. Go 后端单元与集成测试](#1-go-后端单元与集成测试)
  - [2. WebUI 跨端视口回归测试](#2-webui-跨端视口回归测试)
  - [3. 前端安全与防腐审计测试](#3-前端安全与防腐审计测试)
  - [4. 全架构交叉编译检验](#4-全架构交叉编译检验)
- [三、 版本一致性规范 (Single Source of Truth)](#三-版本一致性规范-single-source-of-truth)
  - [1. 四大版本定义文件](#1-四大版本定义文件)
  - [2. 自动化核验与一键同步工具](#2-自动化核验与一键同步工具)
- [四、 发行包构建产物标准 (Release Artifacts)](#四-发行包构建产物标准-release-artifacts)
  - [1. 9 大必备发行附件清单](#1-9-大必备发行附件清单)
  - [2. 必须包含未压缩 ELF 的关键设计原因](#2-必须包含未压缩-elf-的关键设计原因)
  - [3. SHA256 校验和标准清单](#3-sha256-校验和标准清单)
- [五、 GitHub Actions CI/CD 流水线](#五-github-actions-cicd-流水线)
  - [1. CI 持续集成工作流 (`ci.yml`)](#1-ci-持续集成工作流-ciyml)
  - [2. Release 自动化发布工作流 (`release.yml`)](#2-release-自动化发布工作流-releaseyml)
- [六、 生产环境升级与验证操作手册](#六-生产环境升级与验证操作手册)
  - [1. 终端自动化升级流程](#1-终端自动化升级流程)
  - [2. 服务端关键指标验证项](#2-服务端关键指标验证项)
- [七、 异常诊断与故障回滚指南 (Rollback Runbook)](#七-异常诊断与故障回滚指南-rollback-runbook)

---

## 一、 架构总览与发布生命周期

```
[ 本地开发 Local Dev ]
       │
       ▼
[ 1. 质量准入测试 ] ──► (Go 单元测试 + Playwright 4 端视口测试)
       │ PASS
       ▼
[ 2. 版本统一同步 ] ──► (./scripts/check-version.sh --set <新版本>)
       │
       ▼
[ 3. 推送 Git 标签 ] ──► (git tag -a vX.Y.Z -m "..." && git push origin vX.Y.Z)
       │
       ▼
[ 4. GitHub Actions ] ──► 自动触发 .github/workflows/release.yml
       │                  ├─ 严格校验 Tag 与代码版本一致性
       │                  ├─ 编译生成 4 大架构 ELF + .gz 包
       │                  ├─ 生成 SHA256SUMS.txt
       │                  └─ 创建 GitHub Release 并上传 9 个附件
       ▼
[ 5. 生产环境升级 ] ──► (在各服务器执行 `nx update` 自动拉取并重启)
```

---

## 二、 自动化测试流程规范

在发起任何代码合并（PR）或创建发布 Tag 前，必须在本地或 CI 环境中执行以下全部测试套件。

### 1. Go 后端单元与集成测试

运行项目所有子包的单元测试（强制禁用测试缓存以确保每次真实运行）：

```bash
go test -v -count=1 ./...
```

**涵盖的关键测试领域**：
- **`pkg/proxy`**：HTTP/HTTPS CONNECT 嗅探、SOCKS5 认证、RFC 1928 SOCKS5 UDP Associate 穿透协议测试；
- **`pkg/nodes`**：VPNGate 列表解析、端口预检（Port Knock TCP）、住宅 IP 智能判定白名单/黑名单；
- **`pkg/tunnel`**：隧道生命周期回收、动态组 Top-N 轮换策略、三大 AI（OpenAI/Claude/Gemini）全解锁物理探测；
- **`pkg/server`**：RESTful API 鉴权、Clash / sing-box 订阅配置文件生成。

### 2. WebUI 跨端视口回归测试

AimiliVPN WebUI 采用纯原生技术（Vanilla HTML/CSS/JS），集成了 Playwright 进行多端跨视口无头浏览器测试：

```bash
# 安装测试依赖与 Chromium 引擎（首次运行需执行）
npm --prefix web install
npx --prefix web playwright install --with-deps chromium

# 运行自动化视口回归测试
npm --prefix web run test:ui
```

**测试覆盖视口与指标**：
| 视口宽度 | 模拟目标设备 | 核心断言指标 |
| :--- | :--- | :--- |
| **`390px`** | iPhone / 常见移动端 | 底部导航栏切换正常、无任何横向溢出（`scrollWidth <= clientWidth`）、触控区域合规 |
| **`768px`** | iPad / 中型平板 | 栅格响应式自适应布局、抽屉平滑滑出 |
| **`1024px`** | 小型笔记本屏幕 | 侧边栏折叠与展开、卡片双列比例协调 |
| **`1440px`** | 宽屏桌面显示器 | 多端口分流矩阵完整展现、表格滚动条自适应 |

**附加静态断言**：
- 页面中**严禁存在任何 `on*` 内联事件处理器**（强制使用声明式 `data-action` 事件委托）；
- 全局 Geist 字体加载成功且无控制台 Uncaught Exception 报错。

### 3. 前端安全与防腐审计测试

```bash
go test -v ./web/...
```

- **`TestIndexUsesEscapedDynamicRendering`**：静态分析前端脚本，严禁在 HTML 拼接中使用未经转义的动态模板字符串，防范 XSS 漏洞；
- **`TestEmbeddedUIAssets`**：校验二进制嵌入静态资产完整性（`index.html`, `styles.css`, `app.js`, `favicon.svg`, 字体文件）。

### 4. 全架构交叉编译检验

确保 4 大主流架构均可在本地无报错完成编译：

```bash
./scripts/build.sh
```

---

## 三、 版本一致性规范 (Single Source of Truth)

为避免过去出现的“脚本提示升级成功但实际仍旧显示旧版本”问题，本项目制定严格的**版本单一信任源标准**。

### 1. 四大版本定义文件

以下 4 个文件中的版本号必须**绝对相同**（例如 `2.5.1`）：

| 序号 | 文件路径 | 版本定义形态 | 说明 |
| :--- | :--- | :--- | :--- |
| 1 | `VERSION` | `2.5.1` | 根目录纯文本标识，供安装脚本与自动化流水线直接读取 |
| 2 | `pkg/config/version.go` | `const Version = "2.5.1"` | 编译进二进制内部的版本常量，供 `--version` 输出 |
| 3 | `install.sh` | `DEFAULT_VERSION="2.5.1"` | 一键脚本内置的默认版本及 fallback 下载路径 |
| 4 | `web/dist/index.html` | 侧栏/顶栏 badge: `v2.5.1` | 前端页面直接渲染的静态徽章版本 |

### 2. 自动化核验与一键同步工具

项目内置了标准版本检查与同步脚本 `scripts/check-version.sh`：

#### 检查当前版本一致性：
```bash
./scripts/check-version.sh
```
若存在不匹配，脚本将以退出码 `1` 报错并指出具体冲突文件。

#### 一键同步所有版本：
在准备发布新版本时，**仅需执行一行命令**即可将全工程 4 处版本全部更新：
```bash
./scripts/check-version.sh --set 2.5.2
```

---

## 四、 发行包构建产物标准 (Release Artifacts)

每次在 GitHub 发布 Release 时，必须严格提供以下 **9 个标准附件**。

### 1. 9 大必备发行附件清单

```text
dist/
├── aimilivpn_linux_amd64       # 64 位 x86 Linux 原生未压缩 ELF 可执行文件
├── aimilivpn_linux_amd64.gz    # 64 位 x86 Gzip 压缩包
├── aimilivpn_linux_arm64       # 64 位 ARM (aarch64) 原生未压缩 ELF 可执行文件
├── aimilivpn_linux_arm64.gz    # 64 位 ARM Gzip 压缩包
├── aimilivpn_linux_386         # 32 位 x86 Linux 原生未压缩 ELF 可执行文件
├── aimilivpn_linux_386.gz      # 32 位 x86 Gzip 压缩包
├── aimilivpn_linux_arm         # 32 位 ARM Linux 原生未压缩 ELF 可执行文件
├── aimilivpn_linux_arm.gz      # 32 位 ARM Gzip 压缩包
└── SHA256SUMS.txt              # 包含以上全部 8 个二进制文件的哈希校验清单
```

### 2. 必须包含未压缩 ELF 的关键设计原因

- **`nx update` 极速更新机制**：生产环境中的终端管理脚本 `install.sh` 在执行自动升级时，直接拉取 `aimilivpn_linux_${GO_ARCH}` 原生 ELF。
- **避免依赖与解压失败**：某些精简或容器化生产环境未预装 `gzip` 或权限受限，直接校验 ELF 文件的 `head -c 4` 是否包含 `ELF` 标识并比对 SHA-256，保障 100% 成功替换运行。
- **双轨兼顾**：同时上传 `.gz` 格式满足手动网络受限用户的轻量下载需求。

### 3. SHA256 校验和标准清单

`SHA256SUMS.txt` 必须在 `dist/` 根目录由原生 `sha256sum aimilivpn_*` 生成，格式示例如下：
```text
<hash64>  aimilivpn_linux_386
<hash64>  aimilivpn_linux_386.gz
<hash64>  aimilivpn_linux_amd64
<hash64>  aimilivpn_linux_amd64.gz
<hash64>  aimilivpn_linux_arm
<hash64>  aimilivpn_linux_arm.gz
<hash64>  aimilivpn_linux_arm64
<hash64>  aimilivpn_linux_arm64.gz
```

---

## 五、 GitHub Actions CI/CD 流水线

项目已配置两套官方 GitHub Actions 工作流，彻底免除人工编译与上传附件可能产生的网络中断、附件状态残缺（如 `state: starter`）或哈希不对称等问题。

### 1. CI 持续集成工作流 (`.github/workflows/ci.yml`)

- **触发条件**：对 `main` 分支的 `push` 以及所有发往 `main` 的 `pull_request`。
- **执行阶段**：
  1. 运行 `./scripts/check-version.sh` 检查全项目版本是否一致；
  2. 运行 Go 后端全量单元测试与数据竞争检测；
  3. 安装 Chromium 并运行 Playwright 跨端视口 UI 回归测试；
  4. 运行 `./scripts/build.sh` 验证多架构编译输出。

### 2. Release 自动化发布工作流 (`.github/workflows/release.yml`)

- **触发条件**：推送匹配 `v*` 格式的 Git 标签（如 `v2.5.2`）。
- **自动化操作**：
  1. 校验 Git Tag 与代码内定义的版本是否 100% 吻合（不一致立即中断阻止误发）；
  2. 执行完整的自动化测试套件；
  3. 执行 `./scripts/build.sh` 一键构建 4 大架构全套包；
  4. 校验 `dist/` 目录下全部 9 个文件完整性；
  5. 自动创建 GitHub Release，将 9 个附件完整发布，并自动生成变更日志。

---

## 六、 生产环境升级与验证操作手册

### 1. 终端自动化升级流程

在任何运行 AimiliVPN 的 Linux 节点上，以 root 权限执行：

```bash
# 快捷升级命令
nx update

# 或进入主菜单选择 "9) 检查并更新核心程序"
nx
```

**更新过程日志预期**：
```text
正在检测并拉取最新发行版本 (amd64)...
  -> 尝试从源拉取预编译程序: https://github.com/xiumuzidiao0/NXGate/releases/latest/download/aimilivpn_linux_amd64 ...
  -> 二进制预编译包下载成功并通过 SHA-256 校验 (amd64)！

🎉 NXGate 已成功极速更新至最新构建 (v2.5.5) 并重启！
```

### 2. 服务端关键指标验证项

升级完成后，按顺序检查服务健康度：

```bash
# 1. 验证版本号
/opt/aimilivpn/aimilivpn --version
cat /opt/aimilivpn/VERSION

# 2. 检查 systemd 服务运行状态
systemctl status aimilivpn

# 3. 查看实时运行日志与节点探测
journalctl -u aimilivpn -n 30 --no-pager

# 4. 检查当前活跃出网隧道
curl -s http://localhost:8964/api/tunnels | jq '.data[] | {id, node: .node.ip, unlock}'
```

---

## 七、 异常诊断与故障回滚指南 (Rollback Runbook)

### 1. 常见升级异常排查

| 异常现象 | 可能原因 | 解决处置方案 |
| :--- | :--- | :--- |
| `无法获取可信 SHA256SUMS.txt` | 节点到 GitHub 网络波动，或者 Release 尚未上传完成 | 等待 Release 流水线就绪，脚本会自动轮询国内高速代理镜像源重试 |
| `下载文件 SHA-256 校验失败` | 附件被中间网络篡改或上传未完成 | 重新触发 Release 流水线发布，或使用源码就地编译方式更新 |
| `nx: command not found` | 快捷命令丢失 | 执行 `cat > /usr/bin/nx <<'EOF'\n#!/usr/bin/env bash\nexec bash /opt/aimilivpn/install.sh "$@"\nEOF\nchmod +x /usr/bin/nx` |

### 2. 服务端版本快速回滚

如果新发布的版本存在线上兼容性缺陷，可通过保留的历史备份即时回滚：

```bash
# 1. 停止运行中的服务
systemctl stop aimilivpn

# 2. 查找历史备份文件
ls -lh /usr/local/bin/aimilivpn* /opt/aimilivpn/aimilivpn*

# 3. 恢复上一版本二进制
cp -f /opt/aimilivpn/aimilivpn.backup /opt/aimilivpn/aimilivpn
chmod +x /opt/aimilivpn/aimilivpn

# 4. 重启服务
systemctl start aimilivpn
```

---

## 八、 本地 Git 钩子与分支保护策略

### 1. 本地双重守门钩子 (Git Hooks)

项目在 `.githooks/` 目录内置了开箱即用的自动化钩子，克隆项目后执行 `git config core.hooksPath .githooks` 即可全局激活：

1. **`commit-msg` (规范提交信息)**：
   - 自动拦截不符合 Conventional Commits 格式的提交。
   - 必须使用 `feat:`, `fix:`, `docs:`, `style:`, `refactor:`, `perf:`, `test:`, `ci:`, `chore:` 等语义前缀。
2. **`pre-push` (推送前质量自检)**：
   - 自动运行 `./scripts/check-version.sh`（全链路版本一致性）。
   - 自动运行 `go vet ./...`（静态代码安全分析）。
   - 自动运行 `go test -count=1 ./...`（全量子包单元测试）。
   - 自动运行 `npm --prefix web run test:ui`（4 视口回归测试）。

### 2. GitHub 分支保护策略（两套协作模式）

#### 模式 A：单人敏捷开发模式（当前默认）
- **特点**：开发者可在本地直接 `git push origin main`。
- **质量保障**：由本地 `.githooks/pre-push` 与远端 GitHub Actions CI 双重保障，任何版本冲突或测试失败均在第一时间被拦截并标红。

#### 模式 B：多人团队与 PR 审查模式
当项目进入多人协同开发阶段时，推荐开启 GitHub 官方分支保护规则：
1. 打开 GitHub 仓库 -> **Settings** -> **Branches** -> **Add branch protection rule**；
2. **Branch name pattern** 输入 `main`；
3. 勾选 **Require a pull request before merging**（要求必须通过 PR 合并代码）；
4. 勾选 **Require status checks to pass before merging**，并在列表中搜索添加 **`Test & Validate`**（绑定 CI 测试流水线）；
5. 勾选 **Do not allow bypassing the above settings**，点击保存。

---

**文档版本**：`1.1.0`  
**适用范围**：`aimili-vpngate-go v2.5.1+`
