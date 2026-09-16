# AimiliVPN 开发者贡献与工程规范指南 (Contributing Guide)

感谢您对 AimiliVPN 项目的关注与贡献！为了保证代码的高可用性、网络穿透核心稳定性以及跨架构编译的一致性，请在开发与提交代码时遵循本指南。

---

## 目录

- [一、 本地开发环境准备](#一-本地开发环境准备)
- [二、 Git 钩子配置 (必做)](#二-git-钩子配置-必做)
- [三、 分支工作流规范](#三-分支工作流规范)
- [四、 提交信息规范 (Conventional Commits)](#四-提交信息规范-conventional-commits)
- [五、 质量准入与测试要求](#五-质量准入与测试要求)
- [六、 敏感信息隔离要求](#六-敏感信息隔离要求)
- [七、 发布新版本流程](#七-发布新版本流程)

---

## 一、 本地开发环境准备

项目推荐的开发工具版本：
- **Go**: `1.23` 或更高版本（支持最新标准库与模块指令）
- **Node.js**: `20.x` 或更高版本（用于 WebUI 自动化视口回归测试）
- **Linux 环境**: 推荐 Ubuntu / Debian / Fedora / Arch，支持网络命名空间与虚拟网络设备（tun）

```bash
# 1. 克隆代码库
git clone https://github.com/xiumuzidiao0/aimili-vpngate-go.git
cd aimili-vpngate-go

# 2. 安装前端测试依赖
npm --prefix web install
npx --prefix web playwright install --with-deps chromium

# 3. 复制环境变量配置示例
cp config.env.example config.env
```

---

## 二、 Git 钩子配置 (必做)

项目在 `.githooks/` 目录中内置了本地自动化守门钩子：
- **`commit-msg`**：校验提交信息前缀是否符合规范；
- **`pre-push`**：在推送代码前自动运行版本核验、静态分析、Go 测试和前端多端测试。

首次克隆项目后，请在仓库根目录执行一次激活指令：

```bash
git config core.hooksPath .githooks
```

---

## 三、 分支工作流规范

- **`main` 分支**：主分支，代码必须时刻保持 100% 测试通过且随时可发布。
- **特性开发分支**：建议从 `main` 切出新分支进行开发：
  - 新功能：`feat/feature-name`（如 `feat/socks5-udp`）
  - 修复补丁：`fix/bug-description`（如 `fix/ci-singbox-500`）
  - 性能优化：`perf/optimization-name`（如 `perf/port-knock`）
- 完成开发并确保本地测试全部通过后，向 `main` 发起 Pull Request。

---

## 四、 提交信息规范 (Conventional Commits)

本项目强制执行 Conventional Commits 格式：
```text
<type>(<可选模块>): <简明描述>
```

### 允许的 `<type>` 前缀：

| 前缀 | 说明 | 示例 |
| :--- | :--- | :--- |
| **`feat`** | 新功能 | `feat(proxy): 支持 SOCKS5 UDP Associate 穿透出海` |
| **`fix`** | 缺陷修复 | `fix(server): 解决 CI 环境下缺失 singbox 报错 500` |
| **`docs`** | 文档更新 | `docs(ci): 补充 CI/CD 流水线与发布规范指南` |
| **`style`** | UI/样式调整 | `style(web): 优化控制台深空色彩梯度与卡片阴影` |
| **`refactor`** | 代码重构 | `refactor(web): 移除系统设置中冗余的自动轮换 Tab` |
| **`perf`** | 性能优化 | `perf(nodes): 引入 TCP 端口预检减少 90% 拨号耗时` |
| **`test`** | 测试用例 | `test(nodes): 补充端口敲门与延迟分层测试用例` |
| **`ci`** | 流水线配置 | `ci(actions): 集成 govulncheck 依赖安全审计` |
| **`chore`** | 杂项/版本维护 | `chore(release): bump version to v2.5.2` |

---

## 五、 质量准入与测试要求

在发起 PR 或推送代码前，请确保以下三个测试命令全部返回 PASS：

```bash
# 1. 静态代码分析与已知 CVE 依赖扫描
./scripts/audit.sh

# 2. Go 后端全量子包单元与集成测试
go test -count=1 ./...

# 3. WebUI 4 大视口 (390/768/1024/1440px) 回归测试
npm --prefix web run test:ui
```

---

## 六、 敏感信息隔离要求

- **严禁提交敏感信息**：代码库 `.gitignore` 已默认忽略 `config.env`、`*.pem`、`*.key`、`*.crt` 等文件。
- 绝不在代码注释、文档或 Git 历史中硬编码服务器登录密码、API Token 或个人私有配置。
- 生产配置请通过环境变量或安全的外部配置挂载传入。

---

## 七、 发布新版本流程

项目已完全实现发布自动化。当准备发布新版本时：

```bash
# 运行一键发布辅助脚本（自动完成版本同步、测试、提交、打Tag与推送）
./scripts/release.sh 2.5.2
```

推送 Tag 后，GitHub Actions 会自动接管编译并在 GitHub Releases 中发布包含 4 大架构共 9 个完整产物的发行版。服务器只需执行 `ml update` 即可完成极速升级。
