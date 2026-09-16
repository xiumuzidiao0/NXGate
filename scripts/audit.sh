#!/usr/bin/env bash
# ==============================================================================
# AimiliVPN 静态代码审计与已知 CVE 依赖安全扫描工具
# 包含:
# 1. Go 官方静态分析 (go vet ./...)
# 2. Go 官方漏洞扫描 (govulncheck ./...)
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

echo -e "\n${BLUE}======================================================${PLAIN}"
echo -e "${BOLD}🛡️  [Audit] 正在执行项目代码与依赖安全审计...${PLAIN}"
echo -e "${BLUE}======================================================${PLAIN}\n"

# 1. Go Vet
echo -e "${YELLOW}[1/2] 运行 Go 官方静态代码分析 (go vet)...${PLAIN}"
go vet ./...
echo -e "${GREEN}  [✓] go vet 检查通过：无逻辑死锁、格式化串错误或隐藏隐患${PLAIN}\n"

# 2. Govulncheck
echo -e "${YELLOW}[2/2] 运行 Go 官方 CVE 依赖漏洞扫描 (govulncheck)...${PLAIN}"
if ! command -v govulncheck >/dev/null 2>&1; then
    GOPATH_BIN="$(go env GOPATH 2>/dev/null || echo "$HOME/go")/bin"
    if [ -x "${GOPATH_BIN}/govulncheck" ]; then
        export PATH="$PATH:${GOPATH_BIN}"
    else
        echo -e "  -> 本地未发现 govulncheck，正在自动拉取安装..."
        go install golang.org/x/vuln/cmd/govulncheck@latest
        export PATH="$PATH:${GOPATH_BIN}"
    fi
fi

govulncheck ./...

echo -e "\n${GREEN}======================================================${PLAIN}"
echo -e "${GREEN}🎉 [Audit] 代码审计全部通过！未发现已知安全风险。${PLAIN}"
echo -e "${GREEN}======================================================${PLAIN}\n"
