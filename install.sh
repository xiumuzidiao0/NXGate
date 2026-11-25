#!/usr/bin/env bash
set -e

# ==============================================================================
# NXGate (Go 高性能多出口网关) 一键安装与终端管理脚本
# 支持使用 systemd 的 Linux: Ubuntu / Debian / CentOS / RHEL / AlmaLinux / Rocky / Fedora
# OpenRC Alpine / 无 systemd 容器请使用独立进程或 Docker Compose。
# ==============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;36m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
PLAIN='\033[0m'
BOLD='\033[1m'

NXGATE_LANG="${NXGATE_LANG:-}"

normalize_lang() {
    case "${1,,}" in
        en|en-*|en_*|english|2) printf 'en' ;;
        zh|zh-*|zh_*|chinese|cn|1) printf 'zh' ;;
        *) return 1 ;;
    esac
}

is_en() {
    [ "${NXGATE_LANG:-zh}" = "en" ]
}

msg() {
    if is_en; then
        printf '%s' "$2"
    else
        printf '%s' "$1"
    fi
}

load_saved_lang() {
    local normalized="" raw=""
    if [ -n "${NXGATE_LANG:-}" ]; then
        if normalized=$(normalize_lang "${NXGATE_LANG}" 2>/dev/null); then
            NXGATE_LANG="$normalized"
            export NXGATE_LANG
            return 0
        fi
        NXGATE_LANG=""
    fi
    if [ -f "${CONFIG_FILE}" ]; then
        raw=$(grep '^NXGATE_LANG=' "${CONFIG_FILE}" 2>/dev/null | head -n1 | cut -d'=' -f2- | tr -d '\r\n ' || true)
        if [ -n "$raw" ] && normalized=$(normalize_lang "$raw" 2>/dev/null); then
            NXGATE_LANG="$normalized"
            export NXGATE_LANG
            return 0
        fi
    fi
    return 1
}

select_language() {
    local lang_choice="" normalized=""
    while true; do
        echo -e "\n${BLUE}==================================================================${PLAIN}"
        echo -e "${BLUE}           NXGate Installer / NXGate 安装管理脚本                 ${PLAIN}"
        echo -e "${BLUE}==================================================================${PLAIN}"
        echo -e "  ${GREEN}[1]${PLAIN} 简体中文 (默认 / Default)"
        echo -e "  ${GREEN}[2]${PLAIN} English"
        echo -e "${BLUE}==================================================================${PLAIN}"
        prompt_input "Select language / 请选择语言 [1-2] (Default/默认: 1): " "1" lang_choice
        if normalized=$(normalize_lang "$lang_choice" 2>/dev/null); then
            NXGATE_LANG="$normalized"
            export NXGATE_LANG
            if is_en; then
                echo -e "${GREEN}Language set to English.${PLAIN}"
            else
                echo -e "${GREEN}已选择简体中文。${PLAIN}"
            fi
            return 0
        fi
        echo -e "${RED}Invalid choice / 输入无效，请输入 1 或 2。${PLAIN}"
    done
}

safe_clear() {
    [ -t 1 ] && [ -n "$TERM" ] && clear 2>/dev/null || true
}

is_elf_binary() {
    local f="$1"
    [ -f "$f" ] && [ -s "$f" ] && head -c 4 "$f" 2>/dev/null | grep -q 'ELF'
}

INSTALL_DIR="/opt/nxgate"
if [ -d "/opt/aimilivpn" ] && [ ! -d "/opt/nxgate" ]; then
    INSTALL_DIR="/opt/aimilivpn" # 只识别旧目录；读取状态不迁移文件
fi
BIN_PATH="${INSTALL_DIR}/nxgate"
CONFIG_FILE="${INSTALL_DIR}/config.env"
SERVICE_FILE="/etc/systemd/system/nxgate.service"
GITHUB_REPO="https://github.com/xiumuzidiao0/NXGate.git"
DEFAULT_VERSION="2.6.5.42"
REQUIRED_GO_VERSION="1.26.11"

get_app_version() {
    if [ -f "${INSTALL_DIR}/VERSION" ]; then
        cat "${INSTALL_DIR}/VERSION" | tr -d '\r\n '
    elif [ -f "${BIN_PATH}" ]; then
        "${BIN_PATH}" --version 2>/dev/null | awk '{print $NF}' | tr -d 'v' || echo "$DEFAULT_VERSION"
    else
        echo "$DEFAULT_VERSION"
    fi
}

# 1. 检查 Root 权限
check_root() {
    if [ "$(id -u)" != "0" ]; then
        if [ "${NXGATE_LANG:-}" = "en" ]; then
            echo -e "${RED}Error: This script must be run as root. Please run: sudo bash $0${PLAIN}"
        elif [ "${NXGATE_LANG:-}" = "zh" ]; then
            echo -e "${RED}错误: 必须以 root 权限运行此脚本。请使用: sudo bash $0${PLAIN}"
        else
            echo -e "${RED}错误: 必须以 root 权限运行此脚本 (sudo bash $0) / Error: Must run as root (sudo bash $0)${PLAIN}"
        fi
        exit 1
    fi
}

# 2. 检查操作系统包管理器
detect_os() {
    OS_TYPE=""
    PKG_MGR=""
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS_TYPE=$ID
    fi

    case "$OS_TYPE" in
        ubuntu|debian)
            PKG_MGR="apt-get"
            export DEBIAN_FRONTEND=noninteractive
            ;;
        centos|rhel|rocky|almalinux|fedora|ol|amzn)
            if command -v dnf >/dev/null 2>&1; then
                PKG_MGR="dnf"
            else
                PKG_MGR="yum"
            fi
            ;;
        *)
            echo -e "${RED}$(msg "错误: 不支持的操作系统 ($OS_TYPE)！" "Error: Unsupported OS ($OS_TYPE)!")${PLAIN}"
            exit 1
            ;;
    esac
}

# 3. 检测 CPU 架构
detect_arch() {
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64|amd64)
            GO_ARCH="amd64"
            ;;
        aarch64|arm64)
            GO_ARCH="arm64"
            ;;
        i386|i686)
            GO_ARCH="386"
            ;;
        armv7*|armv6*|arm*)
            GO_ARCH="arm"
            ;;
        *)
            echo "$(msg "不支持的 CPU 架构: $ARCH" "Unsupported CPU architecture: $ARCH")" >&2
            return 1
            ;;
    esac
    export GO_ARCH
}

# 多源加速下载官方预编译二进制程序 (支持国内外网络自适应)
download_release_binary() {
    local target_file="$1"
    detect_arch

    local urls=()
    readarray -t urls < <(nxgate_github_urls \
        "https://github.com/xiumuzidiao0/NXGate/releases/latest/download/nxgate_linux_${GO_ARCH}" \
        "https://github.com/xiumuzidiao0/NXGate/releases/download/v${DEFAULT_VERSION}/nxgate_linux_${GO_ARCH}")

    for u in "${urls[@]}"; do
        echo -e "  -> $(msg "尝试从源拉取预编译程序" "Fetching prebuilt binary from"): ${u} ..."
        local target_bin_name
        target_bin_name=$(basename "$u")

        local checksum_url
        if [[ "$u" == *"/latest/"* ]]; then
            checksum_url="https://github.com/xiumuzidiao0/NXGate/releases/latest/download/SHA256SUMS.txt"
        else
            checksum_url="https://github.com/xiumuzidiao0/NXGate/releases/download/v${DEFAULT_VERSION}/SHA256SUMS.txt"
        fi

        local expected_hash
        expected_hash=$(curl -sSL -f -m 20 "$checksum_url" 2>/dev/null | awk -v file="$target_bin_name" '{
            gsub(/\r/, "");
            f1=$1; f2=$2; sub(/^\*/, "", f2); sub(/^\.\//, "", f2);
            if (f2 == file && length(f1) == 64) { print f1; exit }
            if (f1 == file && length(f2) == 64) { print f2; exit }
        }' | tr '[:upper:]' '[:lower:]')
        if [[ ! "$expected_hash" =~ ^[0-9a-fA-F]{64}$ ]]; then
            echo -e "${YELLOW}  -> $(msg "无法获取可信 SHA256SUMS.txt，跳过该下载源" "Unable to fetch valid SHA256SUMS.txt; skipping source")${PLAIN}"
            continue
        fi

        if curl -fL --progress-bar --connect-timeout 15 --max-time 300 --speed-limit 1024 --speed-time 60 "$u" -o "${target_file}.tmp" && [ -s "${target_file}.tmp" ]; then
            if head -c 4 "${target_file}.tmp" | grep -q 'ELF'; then
                local actual_hash
                actual_hash=$(sha256sum "${target_file}.tmp" | awk '{print $1}' | tr '[:upper:]' '[:lower:]')
                if [ "$actual_hash" != "$expected_hash" ]; then
                    echo -e "${RED}  -> $(msg "下载文件 SHA-256 校验失败，拒绝安装" "SHA-256 verification failed; rejecting binary")${PLAIN}"
                    rm -f "${target_file}.tmp"
                    continue
                fi
                mv -f "${target_file}.tmp" "${target_file}"
                chmod +x "${target_file}"
                echo -e "${GREEN}  -> $(msg "二进制预编译包下载成功并通过 SHA-256 校验 (${GO_ARCH})！" "Prebuilt binary downloaded and SHA-256 verified (${GO_ARCH})!")${PLAIN}"
                return 0
            else
                rm -f "${target_file}.tmp"
            fi
        else
            rm -f "${target_file}.tmp" 2>/dev/null || true
        fi
    done
    return 1
}

# 4. 安装基础依赖
install_dependencies() {
    echo -e "\n${YELLOW}[1/4] $(msg "正在安装系统基础依赖..." "Installing base system dependencies...")${PLAIN}"
    if [ "$PKG_MGR" = "apt-get" ]; then
        apt-get update -q || true
        apt-get install -y jq openvpn curl git ca-certificates iptables iproute2 psmisc
    elif [ "$PKG_MGR" = "dnf" ] || [ "$PKG_MGR" = "yum" ]; then
        if [ "$OS_TYPE" != "fedora" ] && [ "$OS_TYPE" != "amzn" ]; then
            $PKG_MGR install -y epel-release || true
        fi
        $PKG_MGR install -y jq openvpn curl git ca-certificates iptables iproute psmisc || \
        $PKG_MGR install -y jq openvpn curl git ca-certificates iptables iproute2 psmisc
    fi
}

# 5. 安装或确保 Go 编译环境
version_ge() {
    local current="${1#go}"
    local required="${2#go}"
    local c_major c_minor c_patch r_major r_minor r_patch
    IFS=. read -r c_major c_minor c_patch <<< "$current"
    IFS=. read -r r_major r_minor r_patch <<< "$required"
    c_major=${c_major:-0}; c_minor=${c_minor:-0}; c_patch=${c_patch:-0}
    r_major=${r_major:-0}; r_minor=${r_minor:-0}; r_patch=${r_patch:-0}
    c_patch=${c_patch%%[^0-9]*}; r_patch=${r_patch%%[^0-9]*}
    [ "$c_major" -gt "$r_major" ] || {
        [ "$c_major" -eq "$r_major" ] &&
        { [ "$c_minor" -gt "$r_minor" ] ||
          { [ "$c_minor" -eq "$r_minor" ] && [ "$c_patch" -ge "$r_patch" ]; }; }
    }
}

go_version_supported() {
    local version
    version=$(go env GOVERSION 2>/dev/null || true)
    [ -n "$version" ] && version_ge "$version" "$REQUIRED_GO_VERSION"
}

ensure_go() {
    detect_arch
    if command -v go >/dev/null 2>&1; then
        if go_version_supported; then
            echo -e "${GREEN}$(msg "检测到受支持的 Go" "Supported Go detected"): $(go version)${PLAIN}"
            return 0
        fi
        echo -e "${YELLOW}$(msg "检测到 Go 版本过低: $(go version)，需要 ${REQUIRED_GO_VERSION}+" "Go version too low: $(go version), requires ${REQUIRED_GO_VERSION}+")${PLAIN}"
    fi

    if [ -x "/usr/local/go/bin/go" ]; then
        export PATH="/usr/local/go/bin:$PATH"
        if go_version_supported; then
            return 0
        fi
    fi

    if [ -x "$HOME/.local/go/bin/go" ]; then
        export PATH="$HOME/.local/go/bin:$PATH"
        if go_version_supported; then
            return 0
        fi
    fi

    local bundled_go="${INSTALL_DIR}/toolchains/go${REQUIRED_GO_VERSION}/bin"
    if [ -x "$bundled_go/go" ]; then
        export PATH="$bundled_go:$PATH"
        if go_version_supported; then
            return 0
        fi
    fi
    install_verified_go
}

# 仅在官方元数据与下载包 SHA256 匹配后，安装到 NXGate 专用版本目录。
install_verified_go() {
    local go_arch="$GO_ARCH"
    [ "$go_arch" = arm ] && go_arch=armv6l
    local go_tar="go${REQUIRED_GO_VERSION}.linux-${go_arch}.tar.gz"
    local work expected actual tool_root target stage
    work=$(mktemp -d) || return 1
    if ! curl --proto '=https' --proto-redir '=https' -fsSL --connect-timeout 15 --max-time 60 \
        'https://go.dev/dl/?mode=json&include=all' -o "$work/releases.json"; then
        rm -rf "$work"
        echo "$(msg '无法取得 Go 官方校验元数据，未安装 Go。' 'Failed to retrieve official Go verification metadata; Go was not installed.')" >&2
        return 1
    fi
    expected=$(jq -er --arg version "go${REQUIRED_GO_VERSION}" --arg name "$go_tar" --arg arch "$go_arch" \
        '[.[] | select(.version == $version) | .files[] |
          select(.filename == $name and .os == "linux" and .arch == $arch and .kind == "archive") |
          .sha256] | if length == 1 then .[0] else error("missing or ambiguous archive") end' "$work/releases.json") || {
        rm -rf "$work"; return 1;
    }
    if [[ ! "$expected" =~ ^[0-9a-f]{64}$ ]]; then
        rm -rf "$work"; return 1
    fi
    echo "$(msg "正在下载 Go ${REQUIRED_GO_VERSION} (${go_arch})，低带宽请耐心等待..." "Downloading Go ${REQUIRED_GO_VERSION} (${go_arch})...")"
    if ! curl --proto '=https' --proto-redir '=https' -fL --progress-bar --connect-timeout 15 \
        --max-time 900 --speed-limit 1024 --speed-time 60 "https://go.dev/dl/${go_tar}" -o "$work/go.tar.gz"; then
        rm -rf "$work"; return 1
    fi
    actual=$(sha256sum "$work/go.tar.gz")
    if [ "${actual%% *}" != "$expected" ]; then
        echo "$(msg 'Go SHA256 校验失败，拒绝解压；原有 Go 未改动。' 'Go SHA256 verification failed; refusing extraction, existing Go untouched.')" >&2
        rm -rf "$work"; return 1
    fi
    tool_root="${INSTALL_DIR}/toolchains"
    target="$tool_root/go${REQUIRED_GO_VERSION}"
    if [ -e "$target" ]; then
        # 不覆盖已有工具目录；损坏的目录需管理员核查。
        if [ ! -x "$target/bin/go" ] || ! "$target/bin/go" version | grep -Fq "go${REQUIRED_GO_VERSION} "; then
            echo "$(msg "已有 $target 无效，请检查后再安装；未覆盖。" "Existing $target is invalid; check it before installing (not overwritten).")" >&2
            rm -rf "$work"; return 1
        fi
    else
        mkdir -p "$tool_root" || { rm -rf "$work"; return 1; }
        stage=$(mktemp -d "$tool_root/.go-stage.XXXXXX") || { rm -rf "$work"; return 1; }
        if ! tar -xzf "$work/go.tar.gz" -C "$stage" ||
            [ ! -x "$stage/go/bin/go" ] ||
            ! "$stage/go/bin/go" version | grep -Fq "go${REQUIRED_GO_VERSION} "; then
            rm -rf "$stage" "$work"; return 1
        fi
        if ! mv -T "$stage/go" "$target"; then
            rm -rf "$stage" "$work"; return 1
        fi
        rmdir "$stage"
    fi
    rm -rf "$work"
    export PATH="$target/bin:$PATH"
    go_version_supported
}

# 辅助生成随机字符串与密码 (连续流式读取，消除熵池截断)
rand_str() {
    local len="${1:-8}"
    LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom 2>/dev/null | head -c "$len" || echo "nxgate"
}

rand_pass() {
    local len="${1:-12}"
    LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom 2>/dev/null | head -c "$len" || echo "nxgate123"
}

# 始终从控制终端读取；不消费 curl | bash 的脚本流。
require_terminal() {
    local tty_fd
    if ! { exec {tty_fd}<>/dev/tty; } 2>/dev/null; then
        if [ "${NXGATE_LANG:-}" = "en" ]; then
            echo 'Interactive terminal required; download the script in SSH and run it with bash. Installation aborted.' >&2
        elif [ "${NXGATE_LANG:-}" = "zh" ]; then
            echo '需要交互终端；请在 SSH 终端下载脚本后用 bash 执行。未继续安装。' >&2
        else
            echo '需要交互终端；请在 SSH 终端下载脚本后用 bash 执行。/ Interactive terminal required; download the script in SSH and run it with bash.' >&2
        fi
        exit 1
    fi
    if [ ! -t "$tty_fd" ]; then
        exec {tty_fd}>&-
        if [ "${NXGATE_LANG:-}" = "en" ]; then
            echo 'No interactive terminal available; installation aborted.' >&2
        elif [ "${NXGATE_LANG:-}" = "zh" ]; then
            echo '没有可用的交互终端，未继续安装。' >&2
        else
            echo '没有可用的交互终端，未继续安装。/ No interactive terminal available; installation aborted.' >&2
        fi
        exit 1
    fi
    exec {tty_fd}>&-
}

require_systemd() {
    if ! command -v systemctl >/dev/null 2>&1 || [ ! -d /run/systemd/system ] ||
        ! systemctl show --property=Version --value >/dev/null 2>&1; then
        echo "$(msg '此安装器需要正在运行的 systemd；OpenRC Alpine/普通容器请使用独立进程或 Docker Compose。未安装依赖或修改系统。' 'This installer requires a running systemd instance; for OpenRC Alpine or plain containers, use the standalone binary or Docker Compose. No changes were made.')" >&2
        exit 1
    fi
}

# 提示出现前关闭回显，避免快速粘贴的密码在 read -s 生效前泄露。
# 子进程的 EXIT trap 在 EOF/中断时也恢复原终端状态。
read_terminal_secret() (
    local fd="$1" message="$2" saved value
    saved=$(stty -g <&"$fd") || exit 1
    trap 'stty "$saved" <&"$fd"' EXIT
    trap 'exit 130' INT
    trap 'exit 143' TERM
    stty -echo <&"$fd" || exit 1
    printf '%s' "$message" >&"$fd"
    if ! IFS= read -r -u "$fd" value; then
        printf '\n%s\n' "$(msg '输入已中断，停止操作。' 'Input aborted; stopping operation.')" >&"$fd"
        exit 1
    fi
    printf '\n' >&"$fd"
    printf '%s' "$value"
)

prompt_input() {
    local prompt_msg="$1" default_val="$2" result_var="$3"
    local input_val="" tty_fd
    require_terminal
    exec {tty_fd}<>/dev/tty
    if [ "${4:-}" = secret ]; then
        if ! input_val=$(read_terminal_secret "$tty_fd" "$prompt_msg"); then
            exec {tty_fd}>&-; exit 1
        fi
    else
        printf '%s' "$prompt_msg" >&"$tty_fd"
        if ! IFS= read -r -u "$tty_fd" input_val; then
            printf '\n%s\n' "$(msg '输入已中断，停止操作。' 'Input aborted; stopping operation.')" >&"$tty_fd"
            exec {tty_fd}>&-; exit 1
        fi
    fi
    exec {tty_fd}>&-
    printf -v "$result_var" '%s' "${input_val:-$default_val}"
}

# 6. 交互式自定义配置与保存
configure_install_params() {
    if [ -f "$(settings_json_path)" ] && ! command -v jq >/dev/null 2>&1; then
        echo "$(msg '已有 settings.json，需要 jq 读取保存设置；请先安装 jq，避免用初始化值覆盖。' 'Existing settings.json requires jq to read saved settings; install jq first to avoid overwriting them.')" >&2
        return 1
    fi
    local def_web_host="${UI_HOST:-0.0.0.0}"
    local def_web_port="8787"
    local def_path=$(rand_str 8)
    local def_user="admin"
    local def_pass=$(rand_pass 12)
    local def_proxy_port="7928"

    # 如果已有配置文件，优先以已有值作为默认
    if [ -f "${CONFIG_FILE}" ]; then
        local exist_web_host exist_web exist_path exist_user exist_pass exist_proxy
        exist_web_host=$(get_config_val "UI_HOST")
        [ -n "$exist_web_host" ] && def_web_host="$exist_web_host"
        exist_web=$(get_config_val "UI_PORT") || return 1
        exist_path=$(get_config_val "UI_PATH") || return 1
        exist_user=$(get_config_val "UI_USERNAME") || return 1
        exist_pass=$(get_config_val "UI_PASSWORD") || return 1
        exist_proxy=$(get_config_val "LOCAL_PROXY_PORT") || return 1

        [ -n "$exist_web" ] && def_web_port="$exist_web"
        [ -n "$exist_path" ] && def_path="$exist_path"
        [ -n "$exist_user" ] && def_user="$exist_user"
        [ -n "$exist_pass" ] && def_pass="$exist_pass"
        [ -n "$exist_proxy" ] && def_proxy_port="$exist_proxy"
    fi

    # 只允许单行地址，避免调用环境中的 UI_HOST 注入 EnvironmentFile 的其他配置。
    [[ "$def_web_host" =~ ^[a-zA-Z0-9_.:-]+$ || "$def_web_host" =~ ^\[[a-fA-F0-9:]+\]$ ]] || {
        echo "$(msg 'UI_HOST 必须是单个 IP 或主机名，不能带协议、空格或控制字符。' 'UI_HOST must be a single IP or hostname without schemes, spaces, or control characters.')" >&2
        return 1
    }

    echo -e "\n${BLUE}==================================================================${PLAIN}"
    if is_en; then
        echo -e "${BLUE}              NXGate Initial Deployment Configuration              ${PLAIN}"
        echo -e "${BLUE}     (Press Enter to accept recommended or random defaults)       ${PLAIN}"
    else
        echo -e "${BLUE}              NXGate 初始部署参数自定义配置                       ${PLAIN}"
        echo -e "${BLUE}  (直接按回车可全部采用括号内的推荐默认值或安全随机值)             ${PLAIN}"
    fi
    echo -e "${BLUE}==================================================================${PLAIN}"

    # 1. Web 端口
    while true; do
        prompt_input "$(msg "1. 请输入 Web 管理控制台端口 [1-65535] (默认: ${def_web_port}): " "1. Enter Web console port [1-65535] (default: ${def_web_port}): ")" "${def_web_port}" custom_web_port
        if [[ "$custom_web_port" =~ ^[0-9]+$ ]] && [ "$custom_web_port" -ge 1 ] && [ "$custom_web_port" -le 65535 ]; then
            break
        else
            echo -e "${RED}$(msg '错误: 端口必须是 1 至 65535 之间的纯数字！' 'Error: Port must be a number between 1 and 65535!')${PLAIN}"
        fi
    done

    # 2. 安全访问路径
    prompt_input "$(msg "2. 请输入 Web 后台安全访问路径 (默认: ${def_path}): " "2. Enter Web admin secret path prefix (default: ${def_path}): ")" "${def_path}" custom_path
    custom_path=$(echo "${custom_path}" | tr -d '/' | tr -d ' ')
    [ -z "$custom_path" ] && custom_path="${def_path}"

    # 3. 管理员账号
    prompt_input "$(msg "3. 请输入 Web 管理员账号 (默认: ${def_user}): " "3. Enter Web admin username (default: ${def_user}): ")" "${def_user}" custom_user
    [ -z "$custom_user" ] && custom_user="${def_user}"

    # 4. 管理员密码
    prompt_input "$(msg "4. 请输入 Web 管理员密码 (回车保留原密码或使用随机密码，输入不回显): " "4. Enter Web admin password (Enter to keep/generate default, hidden input): ")" "${def_pass}" custom_pass secret
    [ -z "$custom_pass" ] && custom_pass="${def_pass}"

    # 5. 代理出站端口
    while true; do
        prompt_input "$(msg "5. 请输入 本地统一代理监听端口 [1-65535] (默认: ${def_proxy_port}): " "5. Enter local unified proxy port [1-65535] (default: ${def_proxy_port}): ")" "${def_proxy_port}" custom_proxy_port
        if [[ "$custom_proxy_port" =~ ^[0-9]+$ ]] && [ "$custom_proxy_port" -ge 1 ] && [ "$custom_proxy_port" -le 65535 ]; then
            if [ "$custom_proxy_port" = "$custom_web_port" ]; then
                echo -e "${RED}$(msg "错误: 代理端口不能与 Web 管理端口 ($custom_web_port) 相同！" "Error: Proxy port cannot match Web console port ($custom_web_port)!")${PLAIN}"
            else
                break
            fi
        else
            echo -e "${RED}$(msg '错误: 端口必须是 1 至 65535 之间的纯数字！' 'Error: Port must be a number between 1 and 65535!')${PLAIN}"
        fi
    done

    if is_en; then
        echo -e "\n${GREEN}------------------- Deployment Summary -------------------${PLAIN}"
        echo -e " Web Bind Host   : ${CYAN}${def_web_host}${PLAIN} (public access depends on firewall/NAT; HTTPS reverse proxy recommended)"
        echo -e " Web Admin Port  : ${CYAN}${custom_web_port}${PLAIN}"
        echo -e " Secret URL Path : ${CYAN}/${custom_path}${PLAIN}"
        echo -e " Admin Username  : ${YELLOW}${custom_user}${PLAIN}"
        echo -e " Admin Password  : Configured (save securely after installation)"
        echo -e " Local Proxy Port: ${CYAN}${custom_proxy_port}${PLAIN}"
        echo -e "${GREEN}----------------------------------------------------------${PLAIN}"
    else
        echo -e "\n${GREEN}------------------- 您配置的参数确认 -------------------${PLAIN}"
        echo -e " Web 监听地址   : ${CYAN}${def_web_host}${PLAIN}（公网访问仍受防火墙/NAT 限制；建议 HTTPS 反代）"
        echo -e " Web 管理端口   : ${CYAN}${custom_web_port}${PLAIN}"
        echo -e " 安全访问路径   : ${CYAN}/${custom_path}${PLAIN}"
        echo -e " 管理员账号     : ${YELLOW}${custom_user}${PLAIN}"
        echo -e " 管理员密码     : 已设置（安装完成后请妥善保存）"
        echo -e " 本地代理端口   : ${CYAN}${custom_proxy_port}${PLAIN}"
        echo -e "${GREEN}-------------------------------------------------------${PLAIN}"
    fi

    local confirmed
    prompt_input "$(msg '确认保存以上配置并安装 NXGate？[y/N]: ' 'Save this configuration and install NXGate? [y/N]: ')" "n" confirmed
    case "${confirmed,,}" in y|yes) ;; *) echo "$(msg '已取消，未保存配置或安装。' 'Cancelled; no configuration saved or installed.')"; exit 1 ;; esac
    mkdir -p "${INSTALL_DIR}/data"
    # 创建时即限制权限，不等写入明文凭据后再收紧。
    (umask 077; : > "${CONFIG_FILE}")
    chmod 600 "${CONFIG_FILE}"
    cat > "${CONFIG_FILE}" <<EOF
# NXGate 运行环境变量配置
DATA_DIR=${INSTALL_DIR}/data
NXGATE_LANG=${NXGATE_LANG:-zh}
UI_HOST=${def_web_host}
UI_PORT=${custom_web_port}
UI_PATH=${custom_path}
UI_USERNAME=${custom_user}
UI_PASSWORD=${custom_pass}
LOCAL_PROXY_HOST=127.0.0.1
LOCAL_PROXY_PORT=${custom_proxy_port}
LOCAL_PROXY_MAX_CONNECTIONS=512
CHECK_INTERVAL_SECONDS=20
FETCH_INTERVAL_SECONDS=900
EOF
    chmod 600 "${CONFIG_FILE}"
}

# 7. 编译并部署二进制文件
build_and_deploy() {
    echo -e "\n${YELLOW}[2/4] $(msg '正在准备 NXGate 可执行程序...' 'Preparing NXGate binary...')${PLAIN}"
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local dl_ok=0

    # 1. 优先尝试极速下载已发布的官方跨平台静态二进制文件 (约 6MB, 支持多镜像源)
    if download_release_binary "${BIN_PATH}"; then
        dl_ok=1
    fi

    # 2. 若无法下载发行包，自动降级至就地编译
    if [ "$dl_ok" = "0" ]; then
        if [ -f "${SCRIPT_DIR}/bin/nxgate" ]; then
            echo -e "  -> $(msg '发现本地已预编译二进制文件，直接安装部署...' 'Found local prebuilt binary; deploying...')"
            cp -f "${SCRIPT_DIR}/bin/nxgate" "${BIN_PATH}"
        elif [ -f "${SCRIPT_DIR}/cmd/nxgate/main.go" ]; then
            echo -e "  -> $(msg '正在从本地源码就地编译目标程序 (约需 10-30 秒)...' 'Building from local source (takes ~10-30s)...')"
            ensure_go
            cd "${SCRIPT_DIR}"
            CGO_ENABLED=0 go build -ldflags="-s -w" -o "${BIN_PATH}" ./cmd/nxgate
        else
            echo -e "  -> $(msg '正在从 GitHub 官方仓库拉取最新源码并构建...' 'Cloning latest source from GitHub and building...')"
            ensure_go
            TMP_DIR=$(mktemp -d)
            git clone --depth 1 "${GITHUB_REPO}" "${TMP_DIR}"
            cd "${TMP_DIR}"
            echo -e "  -> $(msg '正在就地编译二进制程序 (低配 VPS 首次编译约需 20~40 秒，请耐心稍候)...' 'Compiling binary (first build on low-spec VPS takes ~20-40s)...')"
            CGO_ENABLED=0 go build -ldflags="-s -w" -o "${BIN_PATH}" ./cmd/nxgate
            rm -rf "${TMP_DIR}"
        fi
    fi

    if ! is_elf_binary "${BIN_PATH}"; then
        echo -e "${RED}$(msg "错误: 二进制可执行文件 ${BIN_PATH} 准备失败或不是合法的 ELF 程序！" "Error: Binary ${BIN_PATH} is missing or not a valid ELF executable!")${PLAIN}"
        exit 1
    fi

    chmod +x "${BIN_PATH}"
    echo "${DEFAULT_VERSION}" > "${INSTALL_DIR}/VERSION"

    # 部署本地兜底镜像
    mkdir -p "${INSTALL_DIR}/mirror"
    if [ ! -f "${INSTALL_DIR}/mirror/vpngate.csv" ] || [ ! -s "${INSTALL_DIR}/mirror/vpngate.csv" ]; then
        echo -e "  -> $(msg '正在下载初始节点快照镜像...' 'Downloading initial node snapshot mirror...')"
        local mirror_csv_urls=(
            "https://raw.githubusercontent.com/xiumuzidiao0/NXGate/data/mirror/vpngate.csv"
            "https://fastly.jsdelivr.net/gh/xiumuzidiao0/NXGate@data/mirror/vpngate.csv"
            "https://cdn.jsdelivr.net/gh/xiumuzidiao0/NXGate@data/mirror/vpngate.csv"
            "https://ghproxy.net/https://raw.githubusercontent.com/xiumuzidiao0/NXGate/data/mirror/vpngate.csv"
        )
        for m_url in "${mirror_csv_urls[@]}"; do
            if curl -sSL -f -m 10 "$m_url" -o "${INSTALL_DIR}/mirror/vpngate.csv" 2>/dev/null && [ -s "${INSTALL_DIR}/mirror/vpngate.csv" ]; then
                break
            fi
        done
    fi

    # 确保完整的管理脚本安装到 ${INSTALL_DIR}/install.sh
    if [ -f "${BASH_SOURCE[0]}" ] && [ -s "${BASH_SOURCE[0]}" ]; then
        cp -f "${BASH_SOURCE[0]}" "${INSTALL_DIR}/install.sh"
    else
        echo -e "  -> $(msg '正在下载本地管理脚本至' 'Downloading management script to') ${INSTALL_DIR}/install.sh ..."
        curl -sSL "https://raw.githubusercontent.com/xiumuzidiao0/NXGate/main/install.sh" -o "${INSTALL_DIR}/install.sh"
    fi
    chmod +x "${INSTALL_DIR}/install.sh"

    register_shortcuts
}

# 7.4 注册全局快捷命令 (nx, nxgate, aimili)
register_shortcuts() {
    rm -f /usr/bin/nxgate /usr/local/bin/nxgate /usr/bin/aimilivpn /usr/local/bin/aimilivpn /usr/bin/ml /usr/local/bin/ml /usr/bin/nx /usr/local/bin/nx

    cat > /usr/bin/nx <<EOF
#!/usr/bin/env bash
if [ -n "\$1" ]; then
    exec bash ${INSTALL_DIR}/install.sh "\$@"
else
    exec bash ${INSTALL_DIR}/install.sh menu
fi
EOF
    chmod +x /usr/bin/nx
    cp -f /usr/bin/nx /usr/local/bin/nx 2>/dev/null || true
    cp -f /usr/bin/nx /usr/bin/nxgate 2>/dev/null || true
    cp -f /usr/bin/nx /usr/local/bin/nxgate 2>/dev/null || true
    cp -f /usr/bin/nx /usr/bin/aimili 2>/dev/null || true
    cp -f /usr/bin/nx /usr/local/bin/aimili 2>/dev/null || true
}

# 不接管主机 SSH、防火墙或策略路由。回环监听不需要公网放行。
# 对外服务由管理员按实际监听地址在现用防火墙/云安全组中显式配置。
configure_firewall() {
    if is_en; then
        echo "Firewall and routing left unchanged. Port ${1}/tcp defaults to local use; for remote admin use an SSH tunnel or HTTPS reverse proxy."
    else
        echo "未修改防火墙或路由。端口 ${1}/tcp 默认用于本机；远程管理建议 SSH 隧道或 HTTPS 反向代理。"
    fi
}

# 8. 安装 systemd 服务
install_service() {
    echo -e "\n${YELLOW}[3/4] $(msg '正在配置系统守护进程服务...' 'Configuring systemd service...')${PLAIN}"
    cat > "${SERVICE_FILE}" <<EOF
[Unit]
Description=NXGate Go Gateway Service
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=${INSTALL_DIR}
Environment="GOGC=50"
Environment="GOMEMLIMIT=96MiB"
EnvironmentFile=${CONFIG_FILE}
ExecStart=${BIN_PATH}
Restart=always
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

    # 停止并清理旧版 aimilivpn 服务 (若存在)
    systemctl stop aimilivpn.service 2>/dev/null || true
    systemctl disable aimilivpn.service 2>/dev/null || true
    rm -f /etc/systemd/system/aimilivpn.service 2>/dev/null || true

    systemctl daemon-reload
    systemctl enable nxgate.service
    systemctl restart nxgate.service

    local web_port=$(get_config_val "UI_PORT")
    configure_firewall "${web_port}"
}

# 9. JSON 是网页设置的权威源；EnvironmentFile 只负责初始化及其他变量。
settings_json_key() {
    case "$1" in
        UI_PORT) echo ui_port ;;
        UI_PATH) echo ui_path ;;
        UI_USERNAME) echo ui_username ;;
        UI_PASSWORD) echo ui_password ;;
        LOCAL_PROXY_PORT) echo proxy_port ;;
        *) return 0 ;;
    esac
}
settings_json_path() {
    local data_dir
    data_dir=$(grep '^DATA_DIR=' "${CONFIG_FILE}" 2>/dev/null | head -1 | cut -d= -f2- | tr -d '\r')
    data_dir=${data_dir:-${INSTALL_DIR}/data}
    case "$data_dir" in /*) ;; *) data_dir="${INSTALL_DIR}/${data_dir}" ;; esac
    printf '%s/settings.json' "$data_dir"
}
get_config_val() {
    local key="$1" json_key settings_file
    json_key=$(settings_json_key "$key")
    settings_file=$(settings_json_path)
    if [ -n "$json_key" ] && [ -f "$settings_file" ]; then
        if ! command -v jq >/dev/null; then
            [ "${NXGATE_LANG:-zh}" = "en" ] && echo "jq is required to read persisted settings; install jq first" >&2 || echo "读取持久化设置需要 jq；请先安装 jq" >&2
            return 1
        fi
        jq -er --arg key "$json_key" '.[$key]' "$settings_file"
        return
    fi
    if [ -f "${CONFIG_FILE}" ]; then
        grep "^${key}=" "${CONFIG_FILE}" | cut -d'=' -f2- | tr -d '\r\n'
    fi
}

# 10. 已有持久化设置时写入同一权威源，避免 CLI 改密后重启恢复旧密码。
set_config_val() {
    local key="$1" val="$2" json_key settings_file tmp
    if [[ "$val" == *$'\n'* || "$val" == *$'\r'* ]]; then
        [ "${NXGATE_LANG:-zh}" = "en" ] && echo "Config value cannot contain newlines" >&2 || echo "配置值不能含换行符" >&2
        return 1
    fi
    json_key=$(settings_json_key "$key")
    settings_file=$(settings_json_path)
    if [ -n "$json_key" ] && [ -f "$settings_file" ]; then
        if ! command -v jq >/dev/null; then
            [ "${NXGATE_LANG:-zh}" = "en" ] && echo "jq is required to update persisted settings; install jq first" >&2 || echo "更新持久化设置需要 jq；请先安装 jq" >&2
            return 1
        fi
        tmp=$(mktemp "${settings_file}.XXXXXX") || return 1
        if ! jq --arg key "$json_key" --arg value "$val" \
            '.[$key] = (if $key == "ui_port" or $key == "proxy_port" then ($value|tonumber) else $value end)' \
            "$settings_file" > "$tmp"; then
            rm -f "$tmp"; return 1
        fi
        chmod 600 "$tmp" && mv -f "$tmp" "$settings_file"
        return
    fi
    if [ -f "${CONFIG_FILE}" ]; then
        tmp=$(mktemp "${CONFIG_FILE}.XXXXXX") || return 1
        # Do not interpolate passwords into sed expressions (/, &, and | are valid password characters).
        grep -v "^${key}=" "$CONFIG_FILE" > "$tmp" || true
        printf '%s=%s\n' "$key" "$val" >> "$tmp"
        chmod 600 "$tmp" && mv -f "$tmp" "$CONFIG_FILE"
    fi
}

# 11. 获取服务器公网 IP
is_ipv4_address() {
    local value="$1" octet
    [[ "$value" =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]] || return 1
    local -a octets
    IFS=. read -r -a octets <<< "$value"
    for octet in "${octets[@]}"; do
        [ "$((10#$octet))" -le 255 ] || return 1
    done
}
get_public_ip() {
    local ip source
    for source in https://api.ipify.org https://ip.sb https://checkip.amazonaws.com; do
        ip=$(curl -fsS4 --connect-timeout 2 --max-time 3 "$source" 2>/dev/null) || continue
        ip=${ip//$'\r'/}; ip=${ip//$'\n'/}
        if is_ipv4_address "$ip"; then printf '%s\n' "$ip"; return; fi
    done
    printf '%s\n' '<SERVER-IP>'
}

# 监听地址不是访问地址；回环/指定地址不探测公网 IP，IPv6 URL 需加方括号。
get_web_access_url() {
    local host port path
    host=$(get_config_val "UI_HOST"); host=${host:-0.0.0.0}
    host=${host#[}; host=${host%]}
    port=$(get_config_val "UI_PORT")
    path=$(get_config_val "UI_PATH"); path=${path#/}; path=${path%/}
    case "$host" in
        0.0.0.0|::) host=$(get_public_ip) ;;
        *:*) host="[$host]" ;;
    esac
    printf 'http://%s:%s/%s/\n' "$host" "$port" "$path"
}

get_web_bind_address() {
    local host port
    host=$(get_config_val "UI_HOST"); host=${host:-0.0.0.0}
    host=${host#[}; host=${host%]}
    case "$host" in *:*) host="[$host]" ;; esac
    port=$(get_config_val "UI_PORT")
    printf '%s:%s\n' "$host" "$port"
}

# 12. 打印安装成功面板
print_install_success() {
    local web_url=$(get_web_access_url)
    local web_bind=$(get_web_bind_address)
    local path=$(get_config_val "UI_PATH")
    local user=$(get_config_val "UI_USERNAME")
    local pass=$(get_config_val "UI_PASSWORD")
    local proxy_port=$(get_config_val "LOCAL_PROXY_PORT")
    local ver=$(get_app_version)

    local sb_status=$(show_singbox_status)

    echo -e "\n${GREEN}==================================================================${PLAIN}"
    if is_en; then
        echo -e "${GREEN}        🎉 NXGate (Multi-Egress Gateway v${ver}) Installed!           ${PLAIN}"
        echo -e "${GREEN}==================================================================${PLAIN}"
        echo -e " ${BOLD}Web Bind Address${PLAIN} : ${CYAN}${web_bind}${PLAIN}"
        echo -e " ${BOLD}Web Console URL${PLAIN}  : ${CYAN}${web_url}${PLAIN}"
        echo -e " ${BOLD}Admin Username${PLAIN}   : ${YELLOW}${user}${PLAIN}"
        echo -e " ${BOLD}Admin Password${PLAIN}   : ${YELLOW}${pass}${PLAIN}"
        echo -e " ${BOLD}Local Proxy${PLAIN}      : ${GREEN}127.0.0.1:${proxy_port}${PLAIN} (HTTP/HTTPS/SOCKS5 unified port)"
        echo -e " ${BOLD}sing-box Gateway${PLAIN} : ${sb_status} (VLESS-REALITY / Hysteria2)"
        echo -e " ${BOLD}Current Version${PLAIN}  : ${YELLOW}v${ver}${PLAIN}"
        echo -e " ${BOLD}CLI Control${PLAIN}      : Run ${CYAN}nx${PLAIN} anytime to open the terminal manager"
        echo -e " ${YELLOW}⚠️ Notes${PLAIN}          : 1. Include the path prefix when visiting: ${CYAN}/${path}/${PLAIN} (other paths return 404)"
        echo -e "                   2. Replace <SERVER-IP> with a reachable host/IP; 0.0.0.0 means all IPv4 interfaces"
        echo -e "                   3. Public access depends on firewall/NAT rules; HTTPS reverse proxy is recommended"
    else
        echo -e "${GREEN}        🎉 NXGate (自适应多出口路由网关 v${ver}) 安装部署完成！        ${PLAIN}"
        echo -e "${GREEN}==================================================================${PLAIN}"
        echo -e " ${BOLD}Web 配置监听${PLAIN}   : ${CYAN}${web_bind}${PLAIN}"
        echo -e " ${BOLD}Web 管理控制台${PLAIN} : ${CYAN}${web_url}${PLAIN}"
        echo -e " ${BOLD}管理账号${PLAIN}       : ${YELLOW}${user}${PLAIN}"
        echo -e " ${BOLD}管理密码${PLAIN}       : ${YELLOW}${pass}${PLAIN}"
        echo -e " ${BOLD}本地自适应代理${PLAIN} : ${GREEN}127.0.0.1:${proxy_port}${PLAIN} (HTTP/HTTPS/SOCKS5 单端口)"
        echo -e " ${BOLD}边缘抗封锁网关${PLAIN} : ${sb_status} (VLESS-REALITY / Hysteria2)"
        echo -e " ${BOLD}当前程序版本${PLAIN}   : ${YELLOW}v${ver}${PLAIN}"
        echo -e " ${BOLD}终端管理命令${PLAIN}   : 在终端随时输入 ${CYAN}nx${PLAIN} 唤出管理控制中心"
        echo -e " ${YELLOW}⚠️ 访问提示${PLAIN}       : 1. 访问时请务必带上后缀路径: ${CYAN}/${path}/${PLAIN} (未带路径将隐藏返回 404)"
        echo -e "                   2. <SERVER-IP> 请替换为可达的服务器 IP/域名；0.0.0.0 仅表示监听所有 IPv4 接口"
        echo -e "                   3. 公网访问仍需防火墙/NAT 允许；HTTP 明文传输凭据，建议 HTTPS 反代并限制来源"
    fi
    echo -e "${GREEN}==================================================================${PLAIN}\n"
}

# ==============================================================================
# 终端交互式管理功能 (CLI Menu)
# ==============================================================================

show_service_status() {
    if systemctl is-active --quiet nxgate 2>/dev/null || systemctl is-active --quiet aimilivpn 2>/dev/null; then
        echo -e "${GREEN}$(msg '● 运行中 (Active)' '● Active')${PLAIN}"
    else
        echo -e "${RED}$(msg '● 已停止 (Inactive)' '● Inactive')${PLAIN}"
    fi
}

menu_status() {
    local web_url=$(get_web_access_url)
    local web_bind=$(get_web_bind_address)

    local user=$(get_config_val "UI_USERNAME")
    local pass=$(get_config_val "UI_PASSWORD")
    local proxy_port=$(get_config_val "LOCAL_PROXY_PORT")
    local ver=$(get_app_version)
    local sb_status=$(show_singbox_status)

    echo -e "\n${BLUE}==================================================================${PLAIN}"
    if is_en; then
        echo -e "${BLUE}        ⚡ NXGate (Multi-Egress Gateway) Runtime Status           ${PLAIN}"
        echo -e "${BLUE}==================================================================${PLAIN}"
        echo -e " ${BOLD}Service Status${PLAIN}   : $(show_service_status)"
        echo -e " ${BOLD}sing-box Status${PLAIN}  : ${sb_status}"
        echo -e " ${BOLD}Web Bind Address${PLAIN} : ${CYAN}${web_bind}${PLAIN}"
        echo -e " ${BOLD}Web Console URL${PLAIN}  : ${CYAN}${web_url}${PLAIN}"
        echo -e " ${BOLD}Admin Login${PLAIN}      : ${YELLOW}${user}${PLAIN} / ${YELLOW}${pass}${PLAIN}"
        echo -e " ${BOLD}Local Proxy Port${PLAIN} : ${GREEN}127.0.0.1:${proxy_port}${PLAIN} (HTTP/SOCKS5 unified port)"
        echo -e " ${BOLD}Current Version${PLAIN}  : ${YELLOW}v${ver}${PLAIN}"
        echo -e " Note: Replace <SERVER-IP> with a reachable address; firewall/NAT applies, HTTPS reverse proxy recommended."
    else
        echo -e "${BLUE}        ⚡ NXGate (自适应多出口路由网关) 运行状态信息            ${PLAIN}"
        echo -e "${BLUE}==================================================================${PLAIN}"
        echo -e " ${BOLD}服务运行状态${PLAIN}   : $(show_service_status)"
        echo -e " ${BOLD}sing-box 入站状态${PLAIN}: ${sb_status}"
        echo -e " ${BOLD}Web 配置监听${PLAIN}   : ${CYAN}${web_bind}${PLAIN}"
        echo -e " ${BOLD}Web 控制台入口${PLAIN}  : ${CYAN}${web_url}${PLAIN}"
        echo -e " ${BOLD}管理账号/密码${PLAIN}   : ${YELLOW}${user}${PLAIN} / ${YELLOW}${pass}${PLAIN}"
        echo -e " ${BOLD}本地代理端口${PLAIN}     : ${GREEN}127.0.0.1:${proxy_port}${PLAIN} (HTTP/SOCKS5 单端口自适应)"
        echo -e " ${BOLD}当前程序版本${PLAIN}   : ${YELLOW}v${ver}${PLAIN}"
        echo -e " 访问提示：<SERVER-IP> 请替换为可达服务器地址；公网仍受防火墙/NAT 限制，建议 HTTPS 反代。"
    fi
    echo -e "${BLUE}==================================================================${PLAIN}\n"
}

menu_start() {
    systemctl start nxgate 2>/dev/null || systemctl start aimilivpn 2>/dev/null || true
    echo -e "${GREEN}$(msg '已发送启动指令。' 'Start command sent.')${PLAIN}"
    sleep 1
}

menu_stop() {
    systemctl stop nxgate 2>/dev/null || systemctl stop aimilivpn 2>/dev/null || true
    echo -e "${YELLOW}$(msg '已发送停止指令。' 'Stop command sent.')${PLAIN}"
    sleep 1
}

menu_restart() {
    systemctl restart nxgate 2>/dev/null || systemctl restart aimilivpn 2>/dev/null || true
    echo -e "${GREEN}$(msg '已发送重启指令。' 'Restart command sent.')${PLAIN}"
    sleep 1
}

menu_logs() {
    echo -e "${CYAN}$(msg '正在查看实时运行日志 (按 Ctrl+C 退出)...' 'Streaming live logs (press Ctrl+C to exit)...')${PLAIN}"
    journalctl -u nxgate -u aimilivpn -f -n 50
}

menu_modify_credentials() {
    while true; do
        safe_clear
        local curr_user=$(get_config_val "UI_USERNAME")
        local curr_pass=$(get_config_val "UI_PASSWORD")
        echo -e "${BLUE}=======================================================${PLAIN}"
        if is_en; then
            echo -e "${BLUE}              Admin Credentials Management             ${PLAIN}"
            echo -e "${BLUE}=======================================================${PLAIN}"
            echo -e "  Current Username: ${YELLOW}${curr_user}${PLAIN}"
            echo -e "  Current Password: ${YELLOW}${curr_pass}${PLAIN}"
            echo -e "-------------------------------------------------------"
            echo -e "  [1] Set custom username & password"
            echo -e "  [2] Generate a random 12-char strong password"
            echo -e "  [0] Back to main menu"
        else
            echo -e "${BLUE}                 管理账号与密码管理                    ${PLAIN}"
            echo -e "${BLUE}=======================================================${PLAIN}"
            echo -e "  当前管理账号: ${YELLOW}${curr_user}${PLAIN}"
            echo -e "  当前管理密码: ${YELLOW}${curr_pass}${PLAIN}"
            echo -e "-------------------------------------------------------"
            echo -e "  [1] 自定义修改账号与密码"
            echo -e "  [2] 一键随机重置 12 位安全强密码"
            echo -e "  [0] 返回主菜单"
        fi
        echo -e "${BLUE}=======================================================${PLAIN}"
        prompt_input "$(msg '请选择操作 [0-2]: ' 'Select an option [0-2]: ')" "" cred_choice

        case "$cred_choice" in
            1)
                echo ""
                prompt_input "$(msg '请输入新管理账号 (回车保持不变): ' 'Enter new username (Enter to keep current): ')" "" new_u
                prompt_input "$(msg '请输入新管理密码 (不能为空, 回车保持不变): ' 'Enter new password (Enter to keep current): ')" "" new_p secret
                [ -n "$new_u" ] && set_config_val "UI_USERNAME" "$new_u"
                [ -n "$new_p" ] && set_config_val "UI_PASSWORD" "$new_p"
                systemctl restart nxgate 2>/dev/null || systemctl restart aimilivpn 2>/dev/null || true
                echo -e "${GREEN}$(msg '账号密码已更新并重启服务！' 'Credentials updated and service restarted!')${PLAIN}"
                sleep 1.5
                ;;
            2)
                local rand_p=$(rand_pass 12)
                set_config_val "UI_PASSWORD" "$rand_p"
                systemctl restart nxgate 2>/dev/null || systemctl restart aimilivpn 2>/dev/null || true
                echo -e "${GREEN}$(msg '密码已成功重置为' 'Password reset to'): ${YELLOW}${rand_p}${PLAIN}"
                prompt_input "$(msg '按回车键继续...' 'Press Enter to continue...')" "" ignored_input
                ;;
            0)
                break
                ;;
            *)
                echo -e "${RED}$(msg '输入无效' 'Invalid selection')${PLAIN}"; sleep 1
                ;;
        esac
    done
}

menu_modify_ports_and_path() {
    while true; do
        safe_clear
        local curr_web=$(get_config_val "UI_PORT")
        local curr_proxy=$(get_config_val "LOCAL_PROXY_PORT")
        local curr_path=$(get_config_val "UI_PATH")
        local ip=$(get_public_ip)

        echo -e "${BLUE}=======================================================${PLAIN}"
        if is_en; then
            echo -e "${BLUE}            Ports & Admin Secret Path Settings         ${PLAIN}"
            echo -e "${BLUE}=======================================================${PLAIN}"
            echo -e "  Current Web Port   : ${CYAN}${curr_web}${PLAIN}"
            echo -e "  Current Proxy Port : ${CYAN}${curr_proxy}${PLAIN}"
            echo -e "  Current Secret Path: ${CYAN}/${curr_path}${PLAIN}"
            echo -e "  Current Console URL: ${YELLOW}http://${ip}:${curr_web}/${curr_path}${PLAIN}"
            echo -e "-------------------------------------------------------"
            echo -e "  [1] Change Web console port"
            echo -e "  [2] Change local adaptive proxy port"
            echo -e "  [3] Change Web admin secret path"
            echo -e "  [4] Generate a random secret path"
            echo -e "  [0] Back to main menu"
        else
            echo -e "${BLUE}               端口与后台安全路径管理                  ${PLAIN}"
            echo -e "${BLUE}=======================================================${PLAIN}"
            echo -e "  当前 Web 管理端口 : ${CYAN}${curr_web}${PLAIN}"
            echo -e "  当前 本地代理端口 : ${CYAN}${curr_proxy}${PLAIN}"
            echo -e "  当前 安全访问路径 : ${CYAN}/${curr_path}${PLAIN}"
            echo -e "  当前完整访问入口  : ${YELLOW}http://${ip}:${curr_web}/${curr_path}${PLAIN}"
            echo -e "-------------------------------------------------------"
            echo -e "  [1] 修改 Web 管理后台端口"
            echo -e "  [2] 修改 本地自适应代理端口"
            echo -e "  [3] 修改 后台安全访问路径"
            echo -e "  [4] 一键随机重新生成安全路径 (防扫描防爆破)"
            echo -e "  [0] 返回主菜单"
        fi
        echo -e "${BLUE}=======================================================${PLAIN}"
        prompt_input "$(msg '请选择操作 [0-4]: ' 'Select an option [0-4]: ')" "" p_choice

        case "$p_choice" in
            1)
                prompt_input "$(msg '请输入新的 Web 管理端口 [1-65535]: ' 'Enter new Web console port [1-65535]: ')" "" n_web
                if [[ "$n_web" =~ ^[0-9]+$ ]] && [ "$n_web" -ge 1 ] && [ "$n_web" -le 65535 ]; then
                    if [ "$n_web" = "$curr_proxy" ]; then
                        echo -e "${RED}$(msg '错误: Web 端口不能与代理端口相同！' 'Error: Web port cannot match proxy port!')${PLAIN}"; sleep 1.5
                    else
                        set_config_val "UI_PORT" "$n_web"
                        configure_firewall "$n_web"
                        systemctl restart nxgate 2>/dev/null || systemctl restart aimilivpn 2>/dev/null || true
                        echo -e "${GREEN}$(msg "Web 管理端口已更新为 $n_web 并重启生效！" "Web port updated to $n_web and service restarted!")${PLAIN}"; sleep 1.5
                    fi
                else
                    echo -e "${RED}$(msg '端口必须为 1-65535 的数字！' 'Port must be a number between 1 and 65535!')${PLAIN}"; sleep 1.5
                fi
                ;;
            2)
                prompt_input "$(msg '请输入新的 本地代理端口 [1-65535]: ' 'Enter new local proxy port [1-65535]: ')" "" n_proxy
                if [[ "$n_proxy" =~ ^[0-9]+$ ]] && [ "$n_proxy" -ge 1 ] && [ "$n_proxy" -le 65535 ]; then
                    if [ "$n_proxy" = "$curr_web" ]; then
                        echo -e "${RED}$(msg '错误: 代理端口不能与 Web 端口相同！' 'Error: Proxy port cannot match Web port!')${PLAIN}"; sleep 1.5
                    else
                        set_config_val "LOCAL_PROXY_PORT" "$n_proxy"
                        systemctl restart nxgate 2>/dev/null || systemctl restart aimilivpn 2>/dev/null || true
                        echo -e "${GREEN}$(msg "本地代理端口已更新为 $n_proxy 并重启生效！" "Local proxy port updated to $n_proxy and service restarted!")${PLAIN}"; sleep 1.5
                    fi
                else
                    echo -e "${RED}$(msg '端口必须为 1-65535 的数字！' 'Port must be a number between 1 and 65535!')${PLAIN}"; sleep 1.5
                fi
                ;;
            3)
                prompt_input "$(msg '请输入新的安全访问路径 (无需斜杠，例如 admin 或 myvpn): ' 'Enter new secret path (no slashes, e.g. admin or myvpn): ')" "" n_path
                n_path=$(echo "${n_path}" | tr -d '/' | tr -d ' ')
                if [ -n "$n_path" ]; then
                    set_config_val "UI_PATH" "$n_path"
                    systemctl restart nxgate 2>/dev/null || systemctl restart aimilivpn 2>/dev/null || true
                    echo -e "${GREEN}$(msg "安全路径已更新为 /${n_path} 并重启生效！" "Secret path updated to /${n_path} and service restarted!")${PLAIN}"; sleep 1.5
                fi
                ;;
            4)
                local rand_p=$(rand_str 8)
                set_config_val "UI_PATH" "$rand_p"
                systemctl restart nxgate 2>/dev/null || systemctl restart aimilivpn 2>/dev/null || true
                echo -e "${GREEN}$(msg '安全访问路径已随机更新为' 'Secret path randomized to'): ${YELLOW}/${rand_p}${PLAIN}"
                prompt_input "$(msg '按回车键继续...' 'Press Enter to continue...')" "" ignored_input
                ;;
            0)
                break
                ;;
            *)
                echo -e "${RED}$(msg '输入无效' 'Invalid selection')${PLAIN}"; sleep 1
                ;;
        esac
    done
}

menu_switch_node() {
    local port=$(get_config_val "UI_PORT")
    local user=$(get_config_val "UI_USERNAME")
    local pass=$(get_config_val "UI_PASSWORD")

    echo -e "${CYAN}$(msg '正在请求快速切换至当前延迟最低的最优节点...' 'Requesting fast switch to lowest-latency node...')${PLAIN}"
    curl -s -X POST -u "${user}:${pass}" "http://127.0.0.1:${port}/api/connect" >/dev/null && \
        echo -e "${GREEN}$(msg '已向守护程序发送节点切换指令！' 'Node switch command sent!')${PLAIN}" || \
        echo -e "${RED}$(msg '连接失败，请确认服务已启动。' 'Connection failed; verify that the service is running.')${PLAIN}"
    sleep 1.5
}

menu_list_nodes() {
    local port=$(get_config_val "UI_PORT")
    local user=$(get_config_val "UI_USERNAME")
    local pass=$(get_config_val "UI_PASSWORD")

    echo -e "\n${CYAN}$(msg '正在获取当前优质候选节点列表...' 'Fetching candidate node list...')${PLAIN}"
    curl -s -u "${user}:${pass}" "http://127.0.0.1:${port}/api/nodes" | grep -o '{"id":[^}]*}' | head -n 10 | while read -r line; do
        echo "$line"
    done
    echo ""
    prompt_input "$(msg '按回车键返回主菜单...' 'Press Enter to return to main menu...')" "" ignored_input
}

menu_update() {
    detect_arch
    echo -e "\n${YELLOW}$(msg "正在检测并拉取最新发行版本 (${GO_ARCH})..." "Checking and fetching latest release (${GO_ARCH})...")${PLAIN}"

    if download_release_binary "${BIN_PATH}.tmp"; then
        if ! is_elf_binary "${BIN_PATH}.tmp"; then
            echo -e "${RED}$(msg '错误: 下载的更新文件不是有效的 ELF 二进制程序，拒绝应用！' 'Error: Downloaded update is not a valid ELF binary; aborting!')${PLAIN}"
            rm -f "${BIN_PATH}.tmp"
            prompt_input "$(msg '按回车键返回主菜单...' 'Press Enter to return to main menu...')" "" ignored_input
            return 1
        fi
        mv -f "${BIN_PATH}.tmp" "${BIN_PATH}"
        chmod +x "${BIN_PATH}"
        curl -sSL -f "https://raw.githubusercontent.com/xiumuzidiao0/NXGate/main/VERSION" -o "${INSTALL_DIR}/VERSION" 2>/dev/null || true
        curl -sSL -f "https://raw.githubusercontent.com/xiumuzidiao0/NXGate/main/install.sh" -o "${INSTALL_DIR}/install.sh" 2>/dev/null || true
        chmod +x "${INSTALL_DIR}/install.sh" 2>/dev/null || true
        register_shortcuts
        systemctl restart nxgate 2>/dev/null || systemctl restart aimilivpn 2>/dev/null || true
        local new_ver=$(get_app_version)
        echo -e "\n${GREEN}🎉 $(msg "NXGate 已成功极速更新至最新构建 (v${new_ver}) 并重启！" "NXGate updated to v${new_ver} and restarted!")${PLAIN}"
        echo -e " ${BOLD}$(msg '快捷指令已注册' 'Shortcut ready')${PLAIN}: $(msg "在终端随时输入 ${CYAN}nx${PLAIN} 唤出管理控制中心" "Run ${CYAN}nx${PLAIN} anytime to open the terminal manager")"
        prompt_input "$(msg '按回车键返回主菜单...' 'Press Enter to return to main menu...')" "" ignored_input
        return
    fi

    echo -e "\n${YELLOW}$(msg '预编译发行包下载失败，正在从 GitHub 源码编译更新...' 'Prebuilt download failed; building update from GitHub source...')${PLAIN}"
    TMP_DIR=$(mktemp -d)
    if git clone --depth 1 "${GITHUB_REPO}" "${TMP_DIR}"; then
        cd "${TMP_DIR}"
        ensure_go
        if command -v go >/dev/null 2>&1; then
            echo -e "  -> $(msg '正在编译二进制程序 (通常需要 20-30 秒)...' 'Building binary (typically takes 20-30s)...')"
            if CGO_ENABLED=0 go build -ldflags="-s -w" -o "${BIN_PATH}" ./cmd/nxgate; then
                chmod +x "${BIN_PATH}"
                cp -f "${TMP_DIR}/install.sh" "${INSTALL_DIR}/install.sh" 2>/dev/null || true
                chmod +x "${INSTALL_DIR}/install.sh" 2>/dev/null || true
                cp -f "${TMP_DIR}/VERSION" "${INSTALL_DIR}/VERSION" 2>/dev/null || true
                register_shortcuts
                systemctl restart nxgate 2>/dev/null || systemctl restart aimilivpn 2>/dev/null || true
                echo -e "\n${GREEN}🎉 $(msg '源码就地编译更新完成并已重启服务！' 'Built update from source and restarted service!') (v$(get_app_version))${PLAIN}"
                echo -e " ${BOLD}$(msg '快捷指令已注册' 'Shortcut ready')${PLAIN}: $(msg "在终端随时输入 ${CYAN}nx${PLAIN} 唤出管理控制中心" "Run ${CYAN}nx${PLAIN} anytime to open the terminal manager")"
            else
                echo -e "${RED}$(msg '编译失败！' 'Build failed!')${PLAIN}"
            fi
        else
            echo -e "${RED}$(msg '未找到 Go 编译环境，更新失败。' 'Go toolchain not found; update failed.')${PLAIN}"
        fi
        rm -rf "${TMP_DIR}"
    else
        echo -e "${RED}$(msg '拉取源码失败，请检查网络连接。' 'Failed to clone repository; check network connectivity.')${PLAIN}"
    fi
    prompt_input "$(msg '按回车键返回主菜单...' 'Press Enter to return to main menu...')" "" ignored_input
}

menu_uninstall() {
    echo -e "\n${RED}$(msg '警告: 即将完全卸载 NXGate 服务及其配置文件！' 'Warning: This will completely uninstall NXGate and its configuration files!')${PLAIN}"
    prompt_input "$(msg '确认卸载吗？(y/N): ' 'Confirm uninstall? (y/N): ')" "" confirm
    if [ "$confirm" = "y" ] || [ "$confirm" = "Y" ]; then
        systemctl stop nxgate 2>/dev/null || true
        systemctl disable nxgate 2>/dev/null || true
        systemctl stop aimilivpn 2>/dev/null || true
        systemctl disable aimilivpn 2>/dev/null || true
        rm -f "${SERVICE_FILE}" /etc/systemd/system/aimilivpn.service
        systemctl daemon-reload
        rm -f /usr/local/bin/nxgate /usr/bin/nxgate /usr/local/bin/nx /usr/bin/nx /usr/local/bin/aimilivpn /usr/bin/aimilivpn /usr/local/bin/ml /usr/bin/ml /usr/local/bin/aimili /usr/bin/aimili
        rm -rf "${INSTALL_DIR}" /opt/aimilivpn 2>/dev/null || true

        if type -P sing-box &>/dev/null || [ -d "/etc/sing-box" ]; then
            prompt_input "$(msg '是否同时卸载边缘抗封锁网关 sing-box 及其配置？[y/N]: ' 'Also uninstall sing-box and its configs? [y/N]: ')" "" un_sb
            if [ "${un_sb,,}" = "y" ]; then
                systemctl stop sing-box 2>/dev/null || true
                systemctl disable sing-box 2>/dev/null || true
                rm -f /etc/systemd/system/sing-box.service /lib/systemd/system/sing-box.service
                rm -rf /etc/sing-box /var/log/sing-box
                rm -f /usr/local/bin/sing-box /usr/local/bin/sb
                echo -e "${GREEN}✓ $(msg 'sing-box 已完全卸载。' 'sing-box uninstalled.')${PLAIN}"
            fi
        fi

        echo -e "${GREEN}$(msg 'NXGate 已完全卸载干净。' 'NXGate has been completely uninstalled.')${PLAIN}"
        exit 0
    else
        echo -e "$(msg '已取消卸载。' 'Uninstall cancelled.')"
        sleep 1
    fi
}

show_singbox_status() {
    if type -P sing-box &>/dev/null || [ -x "/usr/local/bin/sing-box" ] || [ -f "/etc/sing-box/sh/sing-box.sh" ]; then
        if systemctl is-active --quiet sing-box 2>/dev/null || pgrep -f "/etc/sing-box/bin/sing-box" &>/dev/null || pgrep -f "sing-box" &>/dev/null; then
            echo -e "${GREEN}$(msg '● 运行中' '● Running')${PLAIN}"
        else
            echo -e "${YELLOW}$(msg '○ 已停止' '○ Stopped')${PLAIN}"
        fi
    else
        echo -e "${RED}$(msg '未安装' 'Not Installed')${PLAIN} ($(msg '按 11 一键安装' 'Press 11 to install'))"
    fi
}

# Kept identical in install.sh and embedded download.sh: the installer also runs via stdin.
nxgate_download_file() {
    local target="$1" expected="$2"
    shift 2
    local budget="${NXGATE_DOWNLOAD_TIMEOUT:-600}"
    local stall="${NXGATE_DOWNLOAD_STALL:-60}"
    if [[ ! "$budget" =~ ^[0-9]{1,4}$ ]] || ((10#$budget < 60 || 10#$budget > 1800)); then
        echo 'NXGATE_DOWNLOAD_TIMEOUT 必须为 60–1800 秒。' >&2
        return 1
    fi
    if [[ ! "$stall" =~ ^[0-9]{1,3}$ ]] || ((10#$stall < 5 || 10#$stall > 300)); then
        echo 'NXGATE_DOWNLOAD_STALL 必须为 5–300 秒。' >&2
        return 1
    fi
    budget=$((10#$budget)); stall=$((10#$stall))
    local stage url attempt rc status bytes actual
    stage=$(mktemp -d "${TMPDIR:-/tmp}/nxgate-download.XXXXXX") || return 1
    for url in "$@"; do
        for attempt in 1 2; do
            rm -f "$stage/payload"
            echo "  -> 下载源: $url (尝试 $attempt；上限 ${budget}s，连接 15s，低于 1 KiB/s 持续 ${stall}s 切源)" >&2
            rc=0
            curl --location --fail --show-error --progress-bar \
                --connect-timeout 15 --max-time "$budget" \
                --speed-limit 1024 --speed-time "$stall" \
                --output "$stage/payload" --write-out '%{http_code} %{size_download}\n' \
                "$url" > "$stage/status" || rc=$?
            read -r status bytes < "$stage/status" || true
            status=${status:-000}; bytes=${bytes:-0}
            if [ "$rc" = 0 ] && [ -s "$stage/payload" ]; then
                if [ -n "$expected" ]; then
                    actual=$(sha256sum "$stage/payload")
                    actual=${actual%% *}
                    if [ "$actual" != "$expected" ]; then
                        echo '  SHA256 校验失败，丢弃该源文件并切换；不会安装。' >&2
                        break
                    fi
                fi
                if mv -f "$stage/payload" "$target"; then
                    rm -rf "$stage"
                    return 0
                fi
                rm -rf "$stage"
                return 1
            fi
            echo "  下载失败: curl=$rc HTTP=$status 已接收=${bytes}字节；部分文件已丢弃。" >&2
            # Only transient server errors get one same-source retry. Never restart
            # a slow/stalled/timeout transfer on that source, nor retry a 404.
            case "$status" in
                429|500|502|503|504)
                    if [ "$attempt" = 1 ]; then sleep 2; continue; fi ;;
            esac
            break
        done
    done
    rm -rf "$stage"
    echo '所有下载源失败；请按上方 HTTP/curl 错误判断，不一定是带宽问题。' >&2
    return 1
}

# Digests from SagerNet's official GitHub release asset metadata (v1.14.0).
# Pinning them here also authenticates archives fetched through third-party mirrors.
singbox_core_sha256() {
    case "$1:$2" in
        v1.14.0:amd64) echo 2375de6999f4f56ab46b4fc5ddf26a6aba1d3e61a0f4e7ddec2f4690457d5f63 ;;
        v1.14.0:arm64) echo 04d9b40bc98dc55b6f509ce3292145c65478f65866bea64826ebb2f382385088 ;;
        v1.14.0:386) echo 543efac8f7aa3821a57da3fa394e7c47f383602a9592e674b5eaabb0cedc0d2c ;;
        v1.14.0:armv7) echo 1a8a205e9429c6317f30c5ec112d13345966a91348ae942c9c1645d4b6140063 ;;
        *) return 1 ;;
    esac
}

nxgate_github_urls() {
    local url
    local raw_mirrors="${GITHUB_MIRRORS-${NXGATE_GITHUB_MIRRORS-https://ghproxy.net,https://mirror.ghproxy.com,https://gh-proxy.com}}"
    local mirrors=()
    local entry prefix
    if [[ -n "$raw_mirrors" && "${raw_mirrors,,}" != "direct" && "${raw_mirrors,,}" != "none" ]]; then
        IFS=',' read -r -a mirrors <<< "$raw_mirrors"
    fi
    for url in "$@"; do
        printf '%s\n' "$url"
        if [[ "$url" =~ ^https://(github\.com|raw\.githubusercontent\.com)/ ]]; then
            for entry in "${mirrors[@]}"; do
                prefix="${entry#"${entry%%[![:space:]]*}"}"
                prefix="${prefix%"${prefix##*[![:space:]]}"}"
                prefix="${prefix%/}"
                [[ "$prefix" =~ ^https:// ]] && printf '%s/%s\n' "$prefix" "$url"
            done
        fi
    done
}

do_install_singbox() {
    echo -e "\n${YELLOW}$(msg '正在准备安装部署 sing-box 边缘抗封锁网关...' 'Preparing sing-box edge gateway installation...')${PLAIN}"
    detect_arch
    detect_os

    # 1. 确保基础依赖工具已安装 (jq, tar)
    local need_pkgs=()
    type -P jq &>/dev/null || need_pkgs+=("jq")
    type -P tar &>/dev/null || need_pkgs+=("tar")
    if [ ${#need_pkgs[@]} -gt 0 ]; then
        echo -e "  -> $(msg '正在安装基础依赖' 'Installing base dependencies'): ${need_pkgs[*]} ..."
        if [ -n "$PKG_MGR" ]; then
            $PKG_MGR install -y "${need_pkgs[@]}" &>/dev/null || true
        fi
    fi

    # 2. 部署 sing-box 管理脚本到 /etc/sing-box/sh (零外部 GitHub 脚本请求)
    mkdir -p /etc/sing-box/sh /etc/sing-box/bin /etc/sing-box/conf /var/log/sing-box

    local scripts_installed=0
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

    # 策略 1: 本地源码目录 scripts/singbox
    if [ -d "${script_dir}/scripts/singbox/src" ]; then
        echo -e "  -> $(msg "发现本地工程管理脚本 (${script_dir}/scripts/singbox)，正在同步..." "Syncing local management scripts (${script_dir}/scripts/singbox)...")"
        cp -rf "${script_dir}/scripts/singbox/"* /etc/sing-box/sh/
        scripts_installed=1
    elif [ -d "${INSTALL_DIR}/scripts/singbox/src" ]; then
        echo -e "  -> $(msg "发现已部署工程管理脚本 (${INSTALL_DIR}/scripts/singbox)，正在同步..." "Syncing deployed management scripts (${INSTALL_DIR}/scripts/singbox)...")"
        cp -rf "${INSTALL_DIR}/scripts/singbox/"* /etc/sing-box/sh/
        scripts_installed=1
    fi

    # 策略 2: 利用 nxgate 可执行文件内置内嵌脚本自解压 (--extract-singbox)
    if [ "$scripts_installed" = "0" ]; then
        local candidate_bins=("${BIN_PATH}" "${script_dir}/nxgate" "$(type -P nxgate || true)")
        for b in "${candidate_bins[@]}"; do
            if [ -n "$b" ] && [ -x "$b" ]; then
                echo -e "  -> $(msg "正在通过 NXGate 二进制程序 (${b}) 自释放管理脚本..." "Extracting bundled management scripts from (${b})...")"
                if "$b" --extract-singbox /etc/sing-box/sh 2>/dev/null; then
                    scripts_installed=1
                    break
                fi
            fi
        done
    fi

    # 策略 3: 用户 HOME 目录历史残留本地源码 (兼容降级)
    if [ "$scripts_installed" = "0" ] && [ -n "${HOME}" ] && [ -d "${HOME}/sing-box/src" ]; then
        echo -e "  -> $(msg "发现用户主目录历史源码 (${HOME}/sing-box)，正在复制..." "Copying scripts from ${HOME}/sing-box...")"
        cp -rf "${HOME}/sing-box/"* /etc/sing-box/sh/
        scripts_installed=1
    fi

    # 策略 4: 从 NXGate 官方仓库源拉取内嵌管理脚本 (多 CDN 镜像容灾，针对旧二进制或极简部署环境)
    if [ "$scripts_installed" = "0" ]; then
        echo -e "  -> $(msg '尝试从 NXGate 官方代码库拉取工程管理脚本...' 'Fetching management scripts from NXGate repository...')"
        local repo_tar="/tmp/nxgate_repo_$$.tar.gz"
        local repo_urls=()
        readarray -t repo_urls < <(nxgate_github_urls "https://github.com/xiumuzidiao0/NXGate/archive/refs/heads/main.tar.gz")
        for ru in "${repo_urls[@]}"; do
            echo -e "    • $(msg '尝试拉取源' 'Trying source'): ${ru} ..."
            if curl -sSL -f -m 25 "$ru" -o "$repo_tar" 2>/dev/null && [ -s "$repo_tar" ]; then
                local tmp_repo_dir="/tmp/nxgate_repo_dir_$$"
                mkdir -p "$tmp_repo_dir"
                if tar -zxf "$repo_tar" -C "$tmp_repo_dir" 2>/dev/null; then
                    local found_src=$(find "$tmp_repo_dir" -type d -path "*/scripts/singbox/src" 2>/dev/null | head -n1)
                    if [ -n "$found_src" ]; then
                        local found_sb_dir="$(dirname "$found_src")"
                        cp -rf "$found_sb_dir/"* /etc/sing-box/sh/
                        scripts_installed=1
                    fi
                fi
                rm -rf "$tmp_repo_dir" "$repo_tar"
                [ "$scripts_installed" = "1" ] && break
            fi
        done
        rm -f "$repo_tar"
    fi

    if [ "$scripts_installed" = "0" ] || [ ! -f "/etc/sing-box/sh/sing-box.sh" ]; then
        echo -e "${RED}✗ $(msg '无法定位或提取 sing-box 管理脚本，安装中断。' 'Unable to locate or extract sing-box scripts; installation aborted.')${PLAIN}"
        return 1
    fi

    # 设置权限与软链接
    chmod +x /etc/sing-box/sh/sing-box.sh /etc/sing-box/sh/src/*.sh 2>/dev/null || true
    ln -sf /etc/sing-box/sh/sing-box.sh /usr/local/bin/sing-box
    ln -sf /etc/sing-box/sh/sing-box.sh /usr/local/bin/sb
    echo -e "  ${GREEN}✓ $(msg 'sing-box 管理脚本与控制接口部署就绪' 'sing-box scripts and CLI ready')${PLAIN}"

    # 3. 部署 sing-box 官方核心二进制 (/etc/sing-box/bin/sing-box)
    local sb_bin="/etc/sing-box/bin/sing-box"
    local need_download_core=1

    if [ -x "$sb_bin" ]; then
        local cur_ver=$("$sb_bin" version 2>/dev/null | head -n1 | awk '{print $3}' || true)
        if [ -n "$cur_ver" ]; then
            echo -e "  -> $(msg "检测到已存在 sing-box 核心二进制 (${cur_ver})，跳过重复下载。" "Existing sing-box binary detected (${cur_ver}); skipping download.")"
            need_download_core=0
        fi
    fi

    if [ "$need_download_core" = "1" ]; then
        local sb_ver="v1.14.0"
        local sb_arch="$GO_ARCH"
        case "$sb_arch" in
            amd64) sb_arch="amd64" ;;
            arm64) sb_arch="arm64" ;;
            386)   sb_arch="386" ;;
            arm)   sb_arch="armv7" ;;
            *)     sb_arch="amd64" ;;
        esac

        local core_urls=()
        readarray -t core_urls < <(nxgate_github_urls "https://github.com/SagerNet/sing-box/releases/download/${sb_ver}/sing-box-${sb_ver#v}-linux-${sb_arch}.tar.gz")

        local core_stage core_tar expected core_member
        expected=$(singbox_core_sha256 "$sb_ver" "$sb_arch") || {
            echo "$(msg '缺少该版本/架构的可信 SHA256，拒绝安装。' 'Missing trusted SHA256 for this version/architecture; refusing install.')" >&2
            return 1
        }
        core_stage=$(mktemp -d "${TMPDIR:-/tmp}/nxgate-singbox.XXXXXX") || return 1
        core_tar="$core_stage/core.tar.gz"
        if ! nxgate_download_file "$core_tar" "$expected" "${core_urls[@]}"; then
            rm -rf "$core_stage"
            return 1
        fi
        # Extract only the expected member, not arbitrary archive paths.
        core_member="sing-box-${sb_ver#v}-linux-${sb_arch}/sing-box"
        if ! tar -xOzf "$core_tar" "$core_member" > "$core_stage/sing-box" || [ ! -s "$core_stage/sing-box" ]; then
            echo "$(msg 'sing-box 压缩包缺少核心文件或已损坏；未安装。' 'sing-box archive is corrupt or missing core binary; not installed.')" >&2
            rm -rf "$core_stage"
            return 1
        fi
        if ! chmod +x "$core_stage/sing-box" || ! mv -f "$core_stage/sing-box" "$sb_bin"; then
            rm -rf "$core_stage"
            return 1
        fi
        rm -rf "$core_stage"
        echo -e "  ${GREEN}✓ $(msg 'sing-box 官方核心二进制部署成功（SHA256 已验证）' 'sing-box core binary installed (SHA256 verified)')${PLAIN}"

    fi

    # 4. 确保基础配置 config.json 存在
    if [ ! -f "/etc/sing-box/config.json" ]; then
        cat >/etc/sing-box/config.json <<'EOF'
{
  "log": {
    "output": "/var/log/sing-box/access.log",
    "level": "info",
    "timestamp": true
  },
  "dns": {},
  "outbounds": [
    {
      "tag": "direct",
      "type": "direct"
    }
  ],
  "route": {
    "final": "direct"
  }
}
EOF
    fi

    # 5. 注册并启动 systemd 服务
    if command -v systemctl >/dev/null 2>&1; then
        cat >/etc/systemd/system/sing-box.service <<'EOF'
[Unit]
Description=sing-box Service
Documentation=https://sing-box.sagernet.org/
After=network.target nss-lookup.target

[Service]
User=root
NoNewPrivileges=true
ExecStart=/etc/sing-box/bin/sing-box run -c /etc/sing-box/config.json -C /etc/sing-box/conf
Restart=on-failure
RestartPreventExitStatus=23
LimitNPROC=10000
LimitNOFILE=1048576
PrivateTmp=true
ProtectSystem=full

[Install]
WantedBy=multi-user.target
EOF
        systemctl daemon-reload
        systemctl enable sing-box || return 1
        systemctl restart sing-box || return 1
        echo -e "  ${GREEN}✓ $(msg 'sing-box.service 守护进程已注册并启动' 'sing-box.service registered and started')${PLAIN}"
    fi

    # 组件安装不创建入站，也不修改现有入站。

    echo -e "${GREEN}✓ $(msg 'sing-box 边缘抗封锁网关安装部署成功！' 'sing-box edge gateway installed!')${PLAIN}"
    return 0
}

# 一个显式确认只对应一次创建；失败不能宣称已创建。
create_initial_singbox_node() {
    local proxy_port="$1" create_node sb_bin
    prompt_input "$(msg '创建 VLESS-REALITY 入站？将监听所有地址 (::)，可能对公网开放；请先确认访问控制。[y/N]: ' 'Create a VLESS-REALITY inbound? It binds all addresses (::) and may be publicly reachable; verify access controls first. [y/N]: ')" "n" create_node
    case "${create_node,,}" in y|yes) ;; *) return 0 ;; esac
    sb_bin=$(type -P sing-box) || return 1
    if "$sb_bin" api add reality auto auto auto "127.0.0.1:${proxy_port}"; then
        echo "$(msg "VLESS-REALITY 入站已创建，出口配置为本地代理 ${proxy_port}；请核对代理认证和连通性。" "VLESS-REALITY inbound created, outbound set to local proxy ${proxy_port}; verify proxy auth and connectivity.")"
    else
        echo "$(msg '创建入站失败，请检查 sing-box 日志和已有配置；未自动重试。' 'Failed to create inbound; check sing-box logs and existing config (not retried).')" >&2
        return 1
    fi
}

singbox_available() {
    type -P sing-box >/dev/null 2>&1 || [ -x /usr/local/bin/sing-box ]
}

setup_singbox_integration() {
    echo -e "\n${YELLOW}[4/4] $(msg '可选 sing-box 集成' 'Optional sing-box integration')${PLAIN}"
    local proxy_port install_choice
    proxy_port=$(get_config_val "LOCAL_PROXY_PORT")
    proxy_port=${proxy_port:-7928}
    if singbox_available; then
        echo "$(msg '检测到已有 sing-box；保留现有配置，不自动创建或修改入站。' 'Existing sing-box detected; preserving current config without modifying inbounds.')"
        return 0
    fi
    prompt_input "$(msg '是否安装可选 sing-box 组件（不会自动创建入站）？[y/N]: ' 'Install optional sing-box component (no inbound created automatically)? [y/N]: ')" "n" install_choice
    case "${install_choice,,}" in y|yes) ;; *) return 0 ;; esac
    # 不在 if 函数条件中执行安装，以免 Bash 关闭函数内部的 errexit。
    do_install_singbox
    create_initial_singbox_node "$proxy_port"
}

menu_singbox() {
    while true; do
        safe_clear
        local sb_cmd=""
        if type -P sing-box &>/dev/null; then
            sb_cmd="sing-box"
        elif [ -x "/usr/local/bin/sing-box" ]; then
            sb_cmd="/usr/local/bin/sing-box"
        elif [ -f "/etc/sing-box/sh/sing-box.sh" ]; then
            sb_cmd="bash /etc/sing-box/sh/sing-box.sh"
        fi

        echo -e "${BLUE}==================================================================${PLAIN}"
        if is_en; then
            echo -e "${BLUE}           🚀 sing-box Edge Gateway Control Center                ${PLAIN}"
        else
            echo -e "${BLUE}      🚀 sing-box 边缘抗封锁入站网关控制中心                      ${PLAIN}"
        fi
        echo -e "${BLUE}==================================================================${PLAIN}"

        if [ -z "$sb_cmd" ]; then
            if is_en; then
                echo -e "  ${BOLD}Status${PLAIN}: ${RED}sing-box server is not installed${PLAIN}"
                echo -e ""
                echo -e "  ${BOLD}Overview${PLAIN}:"
                echo -e "  sing-box provides VLESS-REALITY, Hysteria2, and other edge inbound"
                echo -e "  protocols and chains client traffic into NXGate's local egress pool."
                echo -e "${BLUE}------------------------------------------------------------------${PLAIN}"
                echo -e "  ${GREEN}[1]${PLAIN} Install sing-box edge gateway now"
                echo -e "  ${YELLOW}[0]${PLAIN} Back to main menu"
            else
                echo -e "  ${BOLD}当前状态${PLAIN}: ${RED}尚未安装 sing-box 服务端${PLAIN}"
                echo -e ""
                echo -e "  ${BOLD}功能说明${PLAIN}:"
                echo -e "  sing-box 能够提供 VLESS-REALITY（借用海外权威名站 TLS 指纹）、"
                echo -e "  Hysteria2（UDP 极速狂飙）等顶级抗封锁协议，让客户端安全直连 VPS，"
                echo -e "  并将流量链式分流给 NXGate 维护的全球住宅家宽 IP 出口矩阵。"
                echo -e "${BLUE}------------------------------------------------------------------${PLAIN}"
                echo -e "  ${GREEN}[1]${PLAIN} 立即一键安装部署 sing-box 边缘网关"
                echo -e "  ${YELLOW}[0]${PLAIN} 返回上级主菜单"
            fi
            echo -e "${BLUE}==================================================================${PLAIN}"
            prompt_input "$(msg '请输入选项 [0-1]: ' 'Select an option [0-1]: ')" "" sb_choice
            case "$sb_choice" in
                1)
                    do_install_singbox
                    prompt_input "$(msg '按回车键继续...' 'Press Enter to continue...')" "" ignored_input
                    ;;
                0) return ;;
                *) echo -e "${RED}$(msg '输入无效' 'Invalid selection')${PLAIN}"; sleep 1 ;;
            esac
            continue
        fi

        local proxy_port=$(get_config_val "LOCAL_PROXY_PORT")
        [ -z "$proxy_port" ] && proxy_port="7928"

        local status_str=$(show_singbox_status)
        local n_count=0
        if [ -d "/etc/sing-box/conf" ]; then
            n_count=$(ls /etc/sing-box/conf 2>/dev/null | grep -c '\.json$' || true)
        fi

        if is_en; then
            echo -e "  ${BOLD}Service Status${PLAIN}   : ${status_str}"
            echo -e "  ${BOLD}Active Inbounds${PLAIN}  : ${CYAN}${n_count}${PLAIN} configs"
            echo -e "  ${BOLD}NXGate Egress${PLAIN}    : ${GREEN}127.0.0.1:${proxy_port}${PLAIN}"
            echo -e "${BLUE}------------------------------------------------------------------${PLAIN}"
            echo -e "  ${GREEN}[1]${PLAIN} Route all sing-box inbounds via NXGate (${proxy_port})"
            echo -e "  ${GREEN}[2]${PLAIN} Reset all sing-box inbounds to direct"
            echo -e "  ${GREEN}[3]${PLAIN} List active inbounds & client share URIs"
            echo -e "  ${GREEN}[4]${PLAIN} Add new inbound (REALITY / Hysteria2 / TUIC / SS)"
            echo -e "  ${GREEN}[5]${PLAIN} Start / Stop / Restart sing-box service"
            echo -e "  ${GREEN}[6]${PLAIN} Print remote subscription URLs (Clash / Base64)"
            echo -e "  ${GREEN}[7]${PLAIN} Open native sing-box CLI menu"
            echo -e "  ${YELLOW}[8]${PLAIN} Reinstall / Update sing-box"
            echo -e "  ${RED}[9]${PLAIN} Uninstall sing-box"
            echo -e "  ${YELLOW}[0]${PLAIN} Back to main menu"
        else
            echo -e "  ${BOLD}服务运行状态${PLAIN} : ${status_str}"
            echo -e "  ${BOLD}活跃入站节点${PLAIN} : ${CYAN}${n_count}${PLAIN} 个配置"
            echo -e "  ${BOLD}NXGate出口${PLAIN}  : ${GREEN}127.0.0.1:${proxy_port}${PLAIN}"
            echo -e "${BLUE}------------------------------------------------------------------${PLAIN}"
            echo -e "  ${GREEN}[1]${PLAIN} 一键将所有 sing-box 入站接入 NXGate 住宅出口 (${proxy_port})"
            echo -e "  ${GREEN}[2]${PLAIN} 一键将所有 sing-box 入站恢复直连 (direct)"
            echo -e "  ${GREEN}[3]${PLAIN} 查看入站节点列表与客户端分享链接 (vless://, hy2://)"
            echo -e "  ${GREEN}[4]${PLAIN} 快速新建抗封锁入站节点 (REALITY / Hysteria2)"
            echo -e "  ${GREEN}[5]${PLAIN} 启动 / 停止 / 重启 sing-box 服务"
            echo -e "  ${GREEN}[6]${PLAIN} 获取并复制远程订阅链接 (Clash / 小火箭)"
            echo -e "  ${GREEN}[7]${PLAIN} 打开 sing-box 原生终端交互菜单"
            echo -e "  ${YELLOW}[8]${PLAIN} 重新安装 / 更新 sing-box"
            echo -e "  ${RED}[9]${PLAIN} 卸载 sing-box"
            echo -e "  ${YELLOW}[0]${PLAIN} 返回上级主菜单"
        fi
        echo -e "${BLUE}==================================================================${PLAIN}"
        prompt_input "$(msg '请输入选项 [0-9]: ' 'Select an option [0-9]: ')" "" sb_choice

        case "$sb_choice" in
            1)
                echo -e "\n${YELLOW}$(msg "正在将全部 sing-box 入站出口切换至 127.0.0.1:${proxy_port} ..." "Routing all sing-box inbounds via 127.0.0.1:${proxy_port} ...")${PLAIN}"
                $sb_cmd api outbound all "127.0.0.1:${proxy_port}"
                echo -e "${GREEN}✓ $(msg '全部入站节点流量已成功链式绑定至 NXGate 住宅代理出口！' 'All inbounds chained to NXGate proxy egress!')${PLAIN}"
                prompt_input "$(msg '按回车键继续...' 'Press Enter to continue...')" "" ignored_input
                ;;
            2)
                echo -e "\n${YELLOW}$(msg '正在将全部 sing-box 入站出口恢复为 direct (直连) ...' 'Resetting all sing-box inbounds to direct...')${PLAIN}"
                $sb_cmd api outbound all "direct"
                echo -e "${GREEN}✓ $(msg '全部入站节点已恢复 VPS 本机直连出网！' 'All inbounds reset to direct VPS egress!')${PLAIN}"
                prompt_input "$(msg '按回车键继续...' 'Press Enter to continue...')" "" ignored_input
                ;;
            3)
                echo -e "\n${YELLOW}$(msg '---------------- 当前活跃入站节点清单 ----------------' '---------------- Active Inbound Nodes ----------------')${PLAIN}"
                $sb_cmd api list | jq -r '.nodes[] | "----------------------------------------\nTag: \(.name)\nPort: :\(.port)\nProto: \(.protocol) (\(.network))\nOutbound: \(.outbound)\nURI: \(.url)\n----------------------------------------"' 2>/dev/null || $sb_cmd api list
                prompt_input "$(msg '按回车键继续...' 'Press Enter to continue...')" "" ignored_input
                ;;
            4)
                echo "$(msg '新建入站默认监听所有地址 (::)，可能对公网开放。请先配置访问控制。' 'New inbounds bind all addresses (::) and may be publicly reachable. Verify access controls first.')"
                local expose_confirm
                prompt_input "$(msg '确认创建公开监听的入站？[y/N]: ' 'Confirm creating a publicly listening inbound? [y/N]: ')" "n" expose_confirm
                case "${expose_confirm,,}" in y|yes) ;; *) continue ;; esac
                echo -e "\n${YELLOW}$(msg '请选择要新建的协议类型:' 'Select protocol type:')${PLAIN}"
                if is_en; then
                    echo -e " 1) VLESS-REALITY (Recommended: domain-free TLS fingerprint)"
                    echo -e " 2) Hysteria2     (Recommended: high-throughput UDP)"
                    echo -e " 3) TUIC          (QUIC BBR low-latency)"
                    echo -e " 4) Shadowsocks   (Classic 2022 lightweight)"
                else
                    echo -e " 1) VLESS-REALITY (推荐：免域名免证书，借用亚马逊/苹果 TLS 指纹)"
                    echo -e " 2) Hysteria2     (推荐：UDP 极速抗丢包狂飙)"
                    echo -e " 3) TUIC          (QUIC BBR 拥塞控制，低延迟)"
                    echo -e " 4) Shadowsocks   (经典 2022 轻量协议)"
                fi
                prompt_input "$(msg '请选择 [1-4, 默认 1]: ' 'Select [1-4, default 1]: ')" "" p_choice
                local p_proto="reality"
                [ "$p_choice" = "2" ] && p_proto="hy2"
                [ "$p_choice" = "3" ] && p_proto="tuic"
                [ "$p_choice" = "4" ] && p_proto="ss"

                prompt_input "$(msg '请输入监听端口 [直接回车 auto 自动分配]: ' 'Enter listen port [Enter for auto]: ')" "" p_port
                [ -z "$p_port" ] && p_port="auto"

                prompt_input "$(msg "是否直接链式接入 NXGate 本地代理出口 (127.0.0.1:${proxy_port})？[Y/n]: " "Chain outbound to NXGate local proxy (127.0.0.1:${proxy_port})? [Y/n]: ")" "" p_chain
                local p_outbound="127.0.0.1:${proxy_port}"
                if [ "${p_chain,,}" = "n" ] || [ "${p_chain,,}" = "no" ]; then
                    p_outbound="direct"
                fi

                echo -e "\n${YELLOW}$(msg '正在创建并部署入站节点...' 'Creating and deploying inbound node...')${PLAIN}"
                $sb_cmd api add "$p_proto" "$p_port" auto auto "$p_outbound"
                echo -e "${GREEN}✓ $(msg '节点创建完成！' 'Inbound node created!')${PLAIN}"
                prompt_input "$(msg '按回车键继续...' 'Press Enter to continue...')" "" ignored_input
                ;;
            5)
                echo -e "\n 1) $(msg '启动' 'Start')  2) $(msg '停止' 'Stop')  3) $(msg '重启' 'Restart')"
                prompt_input "$(msg '请选择 [1-3]: ' 'Select [1-3]: ')" "" s_op
                case "$s_op" in
                    1) systemctl start sing-box && echo -e "${GREEN}$(msg '服务已启动' 'Service started')${PLAIN}" ;;
                    2) systemctl stop sing-box && echo -e "${YELLOW}$(msg '服务已停止' 'Service stopped')${PLAIN}" ;;
                    3) systemctl restart sing-box && echo -e "${GREEN}$(msg '服务已重启' 'Service restarted')${PLAIN}" ;;
                esac
                sleep 1
                ;;
            6)
                echo -e "\n${YELLOW}$(msg '正在获取 NXGate 原生订阅链接...' 'Fetching NXGate subscription URLs...')${PLAIN}"
                local ui_port sub_tok pub_ip
                ui_port=$(get_config_val "UI_PORT")
                [ -z "$ui_port" ] && ui_port="8787"
                if [ -s "${INSTALL_DIR}/data/sub_token.txt" ]; then
                    sub_tok=$(tr -d '\r\n ' < "${INSTALL_DIR}/data/sub_token.txt")
                else
                    sub_tok=$(get_config_val "UI_PATH")
                    sub_tok="${sub_tok#/}"
                    sub_tok="${sub_tok%/}"
                fi
                pub_ip=$(curl -sS -m 4 https://api.ipify.org 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}')
                [ -z "$pub_ip" ] && pub_ip="127.0.0.1"
                if [ -n "$sub_tok" ]; then
                    echo -e "  ${BOLD}$(msg '通用订阅 (Base64)' 'Universal (Base64)')${PLAIN}: ${GREEN}http://${pub_ip}:${ui_port}/${sub_tok}/api/singbox/subscription${PLAIN}"
                    echo -e "  ${BOLD}Clash / Mihomo${PLAIN}    : ${GREEN}http://${pub_ip}:${ui_port}/${sub_tok}/api/singbox/subscription/clash${PLAIN}"
                else
                    echo -e "  ${BOLD}$(msg '通用订阅 (Base64)' 'Universal (Base64)')${PLAIN}: ${GREEN}http://${pub_ip}:${ui_port}/api/singbox/subscription${PLAIN}"
                    echo -e "  ${BOLD}Clash / Mihomo${PLAIN}    : ${GREEN}http://${pub_ip}:${ui_port}/api/singbox/subscription/clash${PLAIN}"
                fi
                prompt_input "$(msg '按回车键继续...' 'Press Enter to continue...')" "" ignored_input
                ;;
            7)
                $sb_cmd
                ;;
            8)
                do_install_singbox
                prompt_input "$(msg '按回车键继续...' 'Press Enter to continue...')" "" ignored_input
                ;;
            9)
                prompt_input "$(msg '确认彻底卸载 sing-box 及其所有配置文件吗？[y/N]: ' 'Confirm full uninstall of sing-box and its configs? [y/N]: ')" "" un_confirm
                if [ "${un_confirm,,}" = "y" ] || [ "${un_confirm,,}" = "yes" ]; then
                    systemctl stop sing-box 2>/dev/null || true
                    systemctl disable sing-box 2>/dev/null || true
                    rm -f /etc/systemd/system/sing-box.service /lib/systemd/system/sing-box.service
                    rm -rf /etc/sing-box /var/log/sing-box
                    rm -f /usr/local/bin/sing-box /usr/local/bin/sb
                    systemctl daemon-reload 2>/dev/null || true
                    echo -e "${GREEN}✓ $(msg 'sing-box 已彻底卸载干净！' 'sing-box completely uninstalled!')${PLAIN}"
                fi
                prompt_input "$(msg '按回车键继续...' 'Press Enter to continue...')" "" ignored_input
                ;;
            0) return ;;
            *) echo -e "${RED}$(msg '输入无效' 'Invalid selection')${PLAIN}"; sleep 1 ;;
        esac
    done
}

main_menu() {
    while true; do
        safe_clear
        local web_url=$(get_web_access_url)
        local web_bind=$(get_web_bind_address)

        local user=$(get_config_val "UI_USERNAME")
        local pass=$(get_config_val "UI_PASSWORD")
        local proxy_port=$(get_config_val "LOCAL_PROXY_PORT")
        local ver=$(get_app_version)
        local sb_status=$(show_singbox_status)

        echo -e "${BLUE}==================================================================${PLAIN}"
        if is_en; then
            echo -e "${BLUE}        ⚡ NXGate (Multi-Egress Gateway) Control Panel  v${ver}     ${PLAIN}"
            echo -e "${BLUE}==================================================================${PLAIN}"
            echo -e "  ${BOLD}Service Status${PLAIN}   : $(show_service_status)"
            echo -e "  ${BOLD}sing-box Status${PLAIN}  : ${sb_status}"
            echo -e "  ${BOLD}Web Bind Address${PLAIN} : ${CYAN}${web_bind}${PLAIN}"
            echo -e "  ${BOLD}Web Console URL${PLAIN}  : ${CYAN}${web_url}${PLAIN}"
            echo -e "  ${BOLD}Admin Login${PLAIN}      : ${YELLOW}${user}${PLAIN} / ${YELLOW}${pass}${PLAIN}"
            echo -e "  ${BOLD}Local Proxy Port${PLAIN} : ${GREEN}127.0.0.1:${proxy_port}${PLAIN} (HTTP & SOCKS5)"
            echo -e "${BLUE}------------------------------------------------------------------${PLAIN}"
            echo -e "  ${GREEN}[1]${PLAIN} Start service             ${GREEN}[2]${PLAIN} Stop service"
            echo -e "  ${GREEN}[3]${PLAIN} Restart service           ${GREEN}[4]${PLAIN} View live logs"
            echo -e "  ${GREEN}[5]${PLAIN} Switch best egress node   ${GREEN}[6]${PLAIN} List candidate nodes"
            echo -e "  ${GREEN}[7]${PLAIN} Modify admin credentials  ${GREEN}[8]${PLAIN} Modify ports & secret path"
            echo -e "  ${GREEN}[9]${PLAIN} Check & update binary     ${RED}[10]${PLAIN} Uninstall NXGate"
            echo -e "  ${CYAN}[11]${PLAIN} 🚀 Manage / Install sing-box edge gateway"
            echo -e "  ${CYAN}[12]${PLAIN} 🌐 Language: English (Switch to 简体中文)"
            echo -e "  ${YELLOW}[0]${PLAIN} Exit"
        else
            echo -e "${BLUE}      ⚡ NXGate (自适应多出口路由网关) 终端控制中心  v${ver}      ${PLAIN}"
            echo -e "${BLUE}==================================================================${PLAIN}"
            echo -e "  ${BOLD}服务运行状态${PLAIN}   : $(show_service_status)"
            echo -e "  ${BOLD}sing-box 入站状态${PLAIN}: ${sb_status}"
            echo -e "  ${BOLD}Web 配置监听${PLAIN}   : ${CYAN}${web_bind}${PLAIN}"
            echo -e "  ${BOLD}Web 控制台入口${PLAIN}  : ${CYAN}${web_url}${PLAIN}"
            echo -e "  ${BOLD}管理账号/密码${PLAIN}   : ${YELLOW}${user}${PLAIN} / ${YELLOW}${pass}${PLAIN}"
            echo -e "  ${BOLD}本地代理端口${PLAIN}     : ${GREEN}127.0.0.1:${proxy_port}${PLAIN} (HTTP & SOCKS5 单端口自适应)"
            echo -e "${BLUE}------------------------------------------------------------------${PLAIN}"
            echo -e "  ${GREEN}[1]${PLAIN} 启动服务               ${GREEN}[2]${PLAIN} 停止服务"
            echo -e "  ${GREEN}[3]${PLAIN} 重启服务               ${GREEN}[4]${PLAIN} 查看实时运行日志"
            echo -e "  ${GREEN}[5]${PLAIN} 智能切换最优节点       ${GREEN}[6]${PLAIN} 查看当前候选节点列表"
            echo -e "  ${GREEN}[7]${PLAIN} 修改管理账号/密码      ${GREEN}[8]${PLAIN} 修改 Web/代理端口与安全路径"
            echo -e "  ${GREEN}[9]${PLAIN} 检查并在线更新版本     ${RED}[10]${PLAIN} 卸载 NXGate"
            echo -e "  ${CYAN}[11]${PLAIN} 🚀 管理 / 安装 sing-box 边缘抗封锁入站网关"
            echo -e "  ${CYAN}[12]${PLAIN} 🌐 界面语言: 简体中文 (切换为 English)"
            echo -e "  ${YELLOW}[0]${PLAIN} 退出终端管理"
        fi
        echo -e "${BLUE}==================================================================${PLAIN}"
        prompt_input "$(msg '请输入选项 [0-12]: ' 'Select an option [0-12]: ')" "" choice

        case "$choice" in
            1) menu_start ;;
            2) menu_stop ;;
            3) menu_restart ;;
            4) menu_logs ;;
            5) menu_switch_node ;;
            6) menu_list_nodes ;;
            7) menu_modify_credentials ;;
            8) menu_modify_ports_and_path ;;
            9) menu_update ;;
            10) menu_uninstall ;;
            11) menu_singbox ;;
            12)
                if is_en; then NXGATE_LANG="zh"; else NXGATE_LANG="en"; fi
                export NXGATE_LANG
                set_config_val "NXGATE_LANG" "$NXGATE_LANG" 2>/dev/null || true
                ;;
            0) exit 0 ;;
            *) echo -e "${RED}$(msg '输入无效，请重新选择' 'Invalid selection; try again')${PLAIN}"; sleep 1 ;;
        esac
    done
}

# ==============================================================================
# 脚本主执行入口
# ==============================================================================
main() {
    load_saved_lang || true
    check_root
    # status 保持只读；不进行自动修复、补命令、目录迁移或网络策略修改。
    if [ "${1:-}" = status ]; then
        menu_status
        return
    fi
    require_systemd
    case "${1:-}" in
        start|stop|restart)
            if [ ! -f "${SERVICE_FILE}" ] && [ -f /etc/systemd/system/aimilivpn.service ]; then
                systemctl "$1" aimilivpn
            else
                systemctl "$1" nxgate
            fi
            return
            ;;
        ""|menu|update|upgrade|9|install) ;;
        *) echo "$(msg "未知命令: $1" "Unknown command: $1")" >&2; return 1 ;;
    esac
    require_terminal
    if [ -z "${NXGATE_LANG:-}" ]; then
        select_language
    fi
    detect_os
    detect_arch
    case "${1:-}" in
        update|upgrade|9) menu_update; return ;;
        menu) main_menu; return ;;
    esac
    if [ -f "${BIN_PATH}" ] && [ -f "${SERVICE_FILE}" ] && [ -z "${1:-}" ]; then
        main_menu
        return
    fi
    # 先收集并确认配置；预检/EOF/取消时不安装依赖、不改网络策略。
    configure_install_params
    install_dependencies
    build_and_deploy
    install_service
    setup_singbox_integration
    print_install_success
}

# 可 source 函数用于隔离回归；curl | bash 时 BASH_SOURCE 为空。
if [ -z "${BASH_SOURCE[0]:-}" ] || [ "${BASH_SOURCE[0]}" = "$0" ]; then
    main "$@"
fi
