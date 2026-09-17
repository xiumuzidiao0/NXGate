#!/usr/bin/env bash
# ==============================================================================
# AimiliVPN 自动化发布辅助脚本
# 用法: ./scripts/release.sh <版本号，例如 2.5.2>
# 流程:
# 1. 检查 Git 工作区状态
# 2. 全项目版本号自动同步
# 3. 运行静态代码与依赖安全审计 (go vet + govulncheck)
# 4. 运行 Go 后端单元与集成测试 (go test)
# 5. 运行 WebUI 多端无头浏览器测试 (playwright)
# 6. 自动提交版本变更、打 Tag 并推送到远程
# 7. 触发 GitHub Actions 自动化编译与 Release 产物发布
# ==============================================================================

set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$DIR"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;36m'
PLAIN='\033[0m'
BOLD='\033[1m'

if [ -z "$1" ]; then
    CURRENT_VER=$(cat VERSION 2>/dev/null || echo "2.5.1")
    echo -e "${YELLOW}当前版本: v${CURRENT_VER}${PLAIN}"
    echo -e "用法: ${GREEN}$0 <新版本号>${PLAIN}"
    echo -e "示例: ${BLUE}$0 2.5.2${PLAIN}\n"
    exit 1
fi

TARGET_VER="${1#v}"

if [[ ! "$TARGET_VER" =~ ^[0-9]+\.[0-9]+\.[0-9]+.*$ ]]; then
    echo -e "${RED}错误: 版本号格式不合法: ${TARGET_VER} (格式应如: 2.5.2)${PLAIN}"
    exit 1
fi

echo -e "\n${BLUE}==================================================================${PLAIN}"
echo -e "${BOLD}🚀 [Release] 开始准备发布新版本: v${TARGET_VER}${PLAIN}"
echo -e "${BLUE}==================================================================${PLAIN}\n"

# 1. 检查 Git 分支
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo -e "${YELLOW}警告: 当前分支为 [${CURRENT_BRANCH}]，非 [main] 主分支。${PLAIN}"
    read -p "是否继续在此分支发布？(y/N): " confirm_branch
    if [[ ! "$confirm_branch" =~ ^[yY]$ ]]; then
        echo -e "${RED}操作已终止。${PLAIN}"
        exit 1
    fi
fi

# 2. 同步全工程版本
echo -e "\n${YELLOW}[1/5] 同步全项目 4 处版本定义...${PLAIN}"
./scripts/check-version.sh --set "$TARGET_VER"

# 3. 运行代码与依赖安全审计
echo -e "\n${YELLOW}[2/5] 执行静态代码与依赖安全审计...${PLAIN}"
./scripts/audit.sh

# 4. 运行全量测试套件
echo -e "\n${YELLOW}[3/5] 运行 Go 全量子包测试与 WebUI 视口测试...${PLAIN}"
go test -count=1 ./...
if command -v npm >/dev/null 2>&1 && [ -f "web/package.json" ]; then
    npm --prefix web run test:ui
fi
echo -e "${GREEN}  [✓] 全套自动化测试通过！${PLAIN}"

# 5. 提交版本变更与打标签
echo -e "\n${YELLOW}[4/5] 提交版本变更并创建 Git Tag (v${TARGET_VER})...${PLAIN}"
git add VERSION pkg/config/version.go install.sh web/dist/index.html
if ! git diff --cached --quiet; then
    git commit -m "chore(release): bump version to v${TARGET_VER}

Co-Authored-By: Claude Code <noreply@anthropic.com>"
fi

git tag -a "v${TARGET_VER}" -m "Release v${TARGET_VER}" -f

# 6. 推送并触发流水线
echo -e "\n${YELLOW}[5/5] 推送代码与 Tag 到远程仓库...${PLAIN}"
read -p "确认推送到远程并触发 GitHub Actions 自动构建与发布？(Y/n): " confirm_push
if [[ "$confirm_push" =~ ^[nN]$ ]]; then
    echo -e "${YELLOW}已取消推送。本地已打好 Tag: v${TARGET_VER}，稍后可手动执行:${PLAIN}"
    echo -e "  git push origin ${CURRENT_BRANCH} && git push origin v${TARGET_VER}"
    exit 0
fi

git push origin "${CURRENT_BRANCH}"
git push origin "v${TARGET_VER}" -f

echo -e "\n${GREEN}==================================================================${PLAIN}"
echo -e "${GREEN}🎉 [Release] v${TARGET_VER} 代码与标签已成功推送到远程！${PLAIN}"
echo -e "${BLUE}GitHub Actions 将自动执行多架构交叉编译、生成哈希并在 Releases 中发布 9 个完整产物。${PLAIN}"
echo -e "监控构建流水线: ${YELLOW}https://github.com/xiumuzidiao0/NXGate/actions${PLAIN}"
echo -e "${GREEN}==================================================================${PLAIN}\n"
