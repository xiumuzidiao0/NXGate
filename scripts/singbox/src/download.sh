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
    tmpdir=$(mktemp -u)
    [[ ! $tmpdir ]] && {
        tmpdir=/tmp/tmp-$RANDOM
    }
    mkdir -p $tmpdir
    case $1 in
    core)
        name=$is_core_name
        tmpfile=$tmpdir/$is_core.tar.gz
        link="https://github.com/${is_core_repo}/releases/download/${latest_ver}/${is_core}-${latest_ver:1}-linux-${is_arch}.tar.gz"
        download_file
        tar zxf $tmpfile --strip-components 1 -C $is_core_dir/bin
        chmod +x $is_core_bin
        ;;
    sh)
        name="$is_core_name 脚本"
        tmpfile=$tmpdir/sh.tar.gz
        link="https://github.com/${is_sh_repo}/archive/refs/heads/main.tar.gz"
        download_file
        tar zxf $tmpfile --strip-components 1 -C $is_sh_dir
        chmod +x $is_sh_bin ${is_sh_bin/$is_core/sb}
        ;;
    caddy)
        name="Caddy"
        tmpfile=$tmpdir/caddy.tar.gz
        # https://github.com/caddyserver/caddy/releases/download/v2.6.4/caddy_2.6.4_linux_amd64.tar.gz
        link="https://github.com/${is_caddy_repo}/releases/download/${latest_ver}/caddy_${latest_ver:1}_linux_${is_arch}.tar.gz"
        download_file
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

    local dl_ok=0
    for l in "${candidate_links[@]}"; do
        if _wget -t 2 -T 20 -c "$l" -O "$tmpfile" && [[ -s "$tmpfile" ]]; then
            dl_ok=1
            break
        fi
    done

    if [[ $dl_ok -ne 1 ]]; then
        rm -rf $tmpdir
        err "\n下载 ${name} 失败 (所有镜像源均不可达).\n"
    fi
}
