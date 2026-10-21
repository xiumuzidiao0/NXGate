# shellcheck shell=bash
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

get_latest_version() {
    case $1 in
    core)
        name=$is_core_name
        url="https://api.github.com/repos/${is_core_repo}/releases/latest?v=$RANDOM"
        ;;
    sh)
        latest_ver="main"
        return 0
        ;;
    caddy)
        name="Caddy"
        url="https://api.github.com/repos/$is_caddy_repo/releases/latest?v=$RANDOM"
        ;;
    esac
    latest_ver=$(_wget -qO- $url 2>/dev/null | grep tag_name | grep -E -o 'v([0-9.]+)')
    if [[ ! $latest_ver ]]; then
        case $1 in
        core)
            latest_ver=$(curl -sIL -m 10 "https://github.com/${is_core_repo}/releases/latest" 2>/dev/null | grep -i '^location:' | grep -E -o 'v[0-9.]+' | head -n1)
            [[ ! $latest_ver ]] && latest_ver="v1.14.0"
            ;;
        caddy)
            latest_ver=$(curl -sIL -m 10 "https://github.com/${is_caddy_repo}/releases/latest" 2>/dev/null | grep -i '^location:' | grep -E -o 'v[0-9.]+' | head -n1)
            [[ ! $latest_ver ]] && latest_ver="v2.8.4"
            ;;
        esac
    fi
    [[ ! $latest_ver ]] && {
        err "获取 ${name} 最新版本失败."
    }
    unset name url
}
download() {
    latest_ver=$2
    [[ ! $latest_ver ]] && get_latest_version $1
    # tmp dir
    tmpdir=$(mktemp -d) || return 1
    local expected_sha=""
    case $1 in
    core)
        name=$is_core_name
        tmpfile=$tmpdir/$is_core.tar.gz
        link="https://github.com/${is_core_repo}/releases/download/${latest_ver}/${is_core}-${latest_ver:1}-linux-${is_arch}.tar.gz"
        expected_sha=$(singbox_core_sha256 "$latest_ver" "$is_arch") || {
            # Other versions need the official API digest, never mirror metadata.
            command -v jq >/dev/null 2>&1 || { rm -rf "$tmpdir"; err "需要 jq 读取官方 SHA256"; return 1; }
            nxgate_download_file "$tmpdir/release.json" "" "https://api.github.com/repos/${is_core_repo}/releases/tags/${latest_ver}" || { rm -rf "$tmpdir"; return 1; }
            expected_sha=$(jq -r --arg file "${is_core}-${latest_ver:1}-linux-${is_arch}.tar.gz" '.assets[] | select(.name == $file) | .digest // empty' "$tmpdir/release.json")
            expected_sha=${expected_sha#sha256:}
        }
        if [[ ! "$expected_sha" =~ ^[a-f0-9]{64}$ ]]; then
            rm -rf "$tmpdir"; err "缺少官方 SHA256，拒绝安装核心"; return 1
        fi
        download_file || return 1
        tar -xOzf "$tmpfile" "${is_core}-${latest_ver:1}-linux-${is_arch}/${is_core}" > "$tmpdir/core" || { rm -rf "$tmpdir"; return 1; }
        [[ -s "$tmpdir/core" ]] || { rm -rf "$tmpdir"; return 1; }
        # Preserve the official package's optional runtime library/license while
        # never extracting arbitrary archive paths into the installation directory.
        local member
        for member in libcronet.so LICENSE; do
            if tar -tzf "$tmpfile" | grep -Fx "${is_core}-${latest_ver:1}-linux-${is_arch}/$member" >/dev/null; then
                tar -xOzf "$tmpfile" "${is_core}-${latest_ver:1}-linux-${is_arch}/$member" > "$tmpdir/$member" || { rm -rf "$tmpdir"; return 1; }
                mv -f "$tmpdir/$member" "$(dirname "$is_core_bin")/$member" || { rm -rf "$tmpdir"; return 1; }
            fi
        done
        chmod +x "$tmpdir/core" && mv -f "$tmpdir/core" "$is_core_bin" || { rm -rf "$tmpdir"; return 1; }
        ;;
    sh)
        name="$is_core_name 脚本"
        tmpfile=$tmpdir/sh.tar.gz
        link="https://github.com/${is_sh_repo}/archive/refs/heads/main.tar.gz"
        download_file || return 1
        tar zxf $tmpfile --strip-components 1 -C $is_sh_dir
        chmod +x $is_sh_bin ${is_sh_bin/$is_core/sb}
        ;;
    caddy)
        name="Caddy"
        tmpfile=$tmpdir/caddy.tar.gz
        # https://github.com/caddyserver/caddy/releases/download/v2.6.4/caddy_2.6.4_linux_amd64.tar.gz
        link="https://github.com/${is_caddy_repo}/releases/download/${latest_ver}/caddy_${latest_ver:1}_linux_${is_arch}.tar.gz"
        download_file || return 1
        tar zxf $tmpfile -C $tmpdir
        cp -f $tmpdir/caddy $is_caddy_bin
        chmod +x $is_caddy_bin
        ;;
    esac
    rm -rf $tmpdir
    unset latest_ver
}
download_file() {
    local candidate_links=("$link")
    if [[ "$link" =~ ^https://github\.com/ ]]; then
        candidate_links+=(
            "https://ghproxy.net/${link}"
            "https://mirror.ghproxy.com/${link}"
            "https://gh-proxy.com/${link}"
        )
    fi

    if ! nxgate_download_file "$tmpfile" "${expected_sha:-}" "${candidate_links[@]}"; then
        rm -rf "$tmpdir"
        err "下载 ${name} 失败；见上方 HTTP/curl/校验错误。"
        return 1
    fi
}
