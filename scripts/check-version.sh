#!/usr/bin/env bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$DIR"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;36m'
PLAIN='\033[0m'
BOLD='\033[1m'

sync_version() {
    local target_ver="$1"
    # Strip any leading 'v'
    target_ver="${target_ver#v}"

    if [[ ! "$target_ver" =~ ^[0-9]+\.[0-9]+\.[0-9]+.*$ ]]; then
        echo -e "${RED}错误: 版本号格式不合法: ${target_ver} (应如 2.5.8 或小版本 2.5.7.1)${PLAIN}"
        exit 1
    fi

    echo -e "${BLUE}=== 正在统一同步版本号为 v${target_ver} ===${PLAIN}"

    # 1. VERSION
    echo "$target_ver" > VERSION
    echo -e "  [✓] 已更新 VERSION -> ${target_ver}"

    # 2. pkg/config/version.go
    sed -i -E "s/const Version = \".*\"/const Version = \"${target_ver}\"/" pkg/config/version.go
    echo -e "  [✓] 已更新 pkg/config/version.go -> const Version = \"${target_ver}\""

    # 3. install.sh
    sed -i -E "s/DEFAULT_VERSION=\".*\"/DEFAULT_VERSION=\"${target_ver}\"/" install.sh
    sed -i -E "s#download/v[0-9]+(\.[0-9]+)+#download/v${target_ver}#g" install.sh
    echo -e "  [✓] 已更新 install.sh DEFAULT_VERSION 与下载 fallback -> v${target_ver}"

    # 4. web/dist/index.html
    sed -i -E "s#id=\"sidebar-version-badge\" class=\"badge-ver\">v[^<]+<#id=\"sidebar-version-badge\" class=\"badge-ver\">v${target_ver}<#g" web/dist/index.html
    sed -i -E "s#id=\"app-version-badge\" class=\"badge badge-accent\">v[^<]+<#id=\"app-version-badge\" class=\"badge badge-accent\">v${target_ver}<#g" web/dist/index.html
    echo -e "  [✓] 已更新 web/dist/index.html 徽章 -> v${target_ver}"

    # 5. android/app/build.gradle.kts
    if [ -f "android/app/build.gradle.kts" ]; then
        sed -i -E "s/versionName = \".*\"/versionName = \"${target_ver}\"/" android/app/build.gradle.kts
        echo -e "  [✓] 已更新 android/app/build.gradle.kts -> versionName = \"${target_ver}\""
    fi

    echo -e "\n${GREEN}🎉 全部 5 处核心版本定义已统一同步至 v${target_ver}！${PLAIN}\n"
}

check_versions() {
    echo -e "${BLUE}=== 正在检查代码库全链路版本一致性 ===${PLAIN}"

    local ver_file
    ver_file=$(cat VERSION 2>/dev/null | tr -d '\r\n ' || echo "MISSING")

    local ver_go
    ver_go=$(grep -E 'const Version =' pkg/config/version.go 2>/dev/null | awk -F'"' '{print $2}' || echo "MISSING")

    local ver_install
    ver_install=$(grep -E '^DEFAULT_VERSION=' install.sh 2>/dev/null | cut -d'"' -f2 || echo "MISSING")

    local ver_web_sidebar
    ver_web_sidebar=$(grep -o -E 'id="sidebar-version-badge" class="badge-ver">v[^<]+' web/dist/index.html 2>/dev/null | sed -E 's/.*>v//' || echo "MISSING")

    local ver_web_header
    ver_web_header=$(grep -o -E 'id="app-version-badge" class="badge badge-accent">v[^<]+' web/dist/index.html 2>/dev/null | sed -E 's/.*>v//' || echo "MISSING")

    echo "  • VERSION 文件            : ${ver_file}"
    echo "  • pkg/config/version.go   : ${ver_go}"
    echo "  • install.sh 默认版本     : ${ver_install}"
    echo "  • web/dist/index.html 侧栏: ${ver_web_sidebar}"
    echo "  • web/dist/index.html 顶栏: ${ver_web_header}"

    local has_error=0
    if [ "$ver_file" != "$ver_go" ] || \
       [ "$ver_file" != "$ver_install" ] || \
       [ "$ver_file" != "$ver_web_sidebar" ] || \
       [ "$ver_file" != "$ver_web_header" ]; then
        has_error=1
    fi

    if [ "$has_error" -eq 1 ]; then
        echo -e "\n${RED}❌ 版本不一致警告：上述文件版本存在冲突！${PLAIN}"
        echo -e "${YELLOW}可以使用: ./scripts/check-version.sh --set <新版本号> 一键修复同步。${PLAIN}\n"
        exit 1
    else
        echo -e "\n${GREEN}✅ 版本核验通过：所有文件版本均完全一致 (v${ver_file})${PLAIN}\n"
    fi
}

if [ "$1" = "--set" ] || [ "$1" = "-s" ]; then
    if [ -z "$2" ]; then
        echo -e "${RED}用法: $0 --set <版本号，例如 2.5.2>${PLAIN}"
        exit 1
    fi
    sync_version "$2"
    check_versions
else
    check_versions
fi
