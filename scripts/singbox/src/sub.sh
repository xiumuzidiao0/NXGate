#!/bin/bash

# sing-box remote subscription module
# managed by 233boy script

is_sub_dir=$is_core_dir/sub
is_sub_json=$is_core_dir/sub.json
is_sub_caddy_conf=$is_caddy_conf/sub.conf

sub_token_gen() {
    local token
    token=$(tr -dc 'a-z0-9' </dev/urandom 2>/dev/null | head -c 16)
    if [[ ! $token || ${#token} -lt 16 ]]; then
        token=$(cat /proc/sys/kernel/random/uuid 2>/dev/null | sed 's/-//g' | head -c 16)
    fi
    echo "$token"
}

sub_port_gen() {
    local is_count=0
    local tmp_port
    while :; do
        ((is_count++))
        if [[ $is_count -ge 233 ]]; then
            err "自动获取订阅可用端口失败."
        fi
        tmp_port=$(shuf -i 10000-65535 -n 1)
        [[ ! $(is_test port_used $tmp_port) ]] && break
    done
    echo "$tmp_port"
}

sub_server_apply() {
    local port=$1
    mkdir -p "$is_sub_dir"

    # Stop and clean legacy caddy sub conf if exists
    if [[ -f "$is_sub_caddy_conf" ]]; then
        rm -f "$is_sub_caddy_conf"
        if [[ -n "$is_caddy_bin" && $(pgrep -f "$is_caddy_bin") ]]; then
            manage restart caddy &>/dev/null || true
        fi
    fi

    # Detect lightweight HTTP server (python3 / python / busybox)
    local py_bin
    py_bin=$(type -P python3 || type -P python || echo "")

    if [[ -d /run/systemd/system || -d /lib/systemd/system || -d /etc/systemd/system ]]; then
        if [[ -n "$py_bin" ]]; then
            cat >/lib/systemd/system/sing-box-sub.service <<-EOF
[Unit]
Description=sing-box Remote Subscription Service
After=network.target

[Service]
Type=simple
WorkingDirectory=${is_sub_dir}
ExecStart=${py_bin} -m http.server ${port} --directory ${is_sub_dir}
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF
            systemctl daemon-reload &>/dev/null || true
            systemctl enable sing-box-sub &>/dev/null || true
            systemctl restart sing-box-sub &>/dev/null || true
        elif [[ $(type -P busybox) ]]; then
            cat >/lib/systemd/system/sing-box-sub.service <<-EOF
[Unit]
Description=sing-box Remote Subscription Service
After=network.target

[Service]
Type=simple
ExecStart=$(type -P busybox) httpd -f -p ${port} -h ${is_sub_dir}
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF
            systemctl daemon-reload &>/dev/null || true
            systemctl enable sing-box-sub &>/dev/null || true
            systemctl restart sing-box-sub &>/dev/null || true
        fi
    else
        # Fallback for non-systemd environments (background daemon)
        pkill -f "http.server $port" 2>/dev/null || true
        if [[ -n "$py_bin" ]]; then
            nohup "$py_bin" -m http.server "$port" --directory "$is_sub_dir" >/dev/null 2>&1 &
        elif [[ $(type -P busybox) ]]; then
            busybox httpd -p "$port" -h "$is_sub_dir" 2>/dev/null || true
        fi
    fi
}

sub_caddy_apply() {
    sub_server_apply "$@"
}

is_sub_server_running() {
    if systemctl is-active sing-box-sub &>/dev/null; then
        return 0
    fi
    if pgrep -f "http.server.*$is_sub_dir" &>/dev/null || pgrep -f "httpd -f -p" &>/dev/null; then
        return 0
    fi
    local port
    port=$(jq -r '.port // empty' "$is_sub_json" 2>/dev/null)
    if [[ -n "$port" ]] && pgrep -f "http.server $port" &>/dev/null; then
        return 0
    fi
    return 1
}

sub_sync() {
    [[ ! -f $is_sub_json ]] && return 0
    local sub_token sub_filename sub_file
    sub_token=$(jq -r '.token // empty' "$is_sub_json" 2>/dev/null)
    sub_filename=$(jq -r '.filename // empty' "$is_sub_json" 2>/dev/null)
    [[ ! $sub_token || ! $sub_filename ]] && return 0

    local target_dir="${is_sub_dir}/${sub_token}"
    mkdir -p "$target_dir"
    sub_file="${target_dir}/${sub_filename}"

    local tmp_file="/tmp/sub-$RANDOM-$$.txt"
    rm -f "$tmp_file"

    if [[ -d $is_conf_dir ]]; then
        local old_dont_show=$is_dont_show_info
        for conf in $(ls "$is_conf_dir" 2>/dev/null | grep .json$ | sed '/dynamic-port-.*-link/d'); do
            is_dont_show_info=1
            unset is_protocol is_url net host is_anytls_domain is_addr
            info "$conf" &>/dev/null
            if [[ $is_url ]]; then
                if [[ $is_url =~ @auto[:#] ]]; then
                    get_ip
                    is_url="${is_url//@auto:/@$ip:}"
                    is_url="${is_url//@auto#/@$ip#}"
                    is_url="${is_url//-auto/-$ip}"
                fi
                echo "$is_url" >>"$tmp_file"
            fi
        done
        is_dont_show_info=$old_dont_show
    fi

    if [[ -f $tmp_file ]]; then
        mv -f "$tmp_file" "$sub_file"
    else
        > "$sub_file"
    fi
    chmod 644 "$sub_file" 2>/dev/null
}

sub_init() {
    local port token hostname_val filename
    port=$(sub_port_gen)
    token=$(sub_token_gen)
    hostname_val=$(hostname 2>/dev/null || cat /proc/sys/kernel/hostname 2>/dev/null || echo "singbox")
    hostname_val=$(echo "$hostname_val" | tr -cd 'a-zA-Z0-9_-')
    [[ ! $hostname_val ]] && hostname_val="singbox"
    filename="${hostname_val}_singbox.yaml"

    mkdir -p "${is_sub_dir}/${token}"
    jq -n --argjson port "$port" \
          --arg token "$token" \
          --arg filename "$filename" \
          '{port: $port, token: $token, filename: $filename, enabled: true}' >"$is_sub_json"

    sub_caddy_apply "$port"
    sub_sync
    sleep 1
    _green "\n已成功初始化并开启远程订阅服务!"
}

sub_info() {
    if [[ ! -f $is_sub_json ]]; then
        warn "当前尚未开启远程订阅功能."
        echo -ne "是否立即开启远程订阅服务? [y/n] (默认 y): "
        read -r is_sub_choice
        [[ ! $is_sub_choice ]] && is_sub_choice="y"
        if [[ ${is_sub_choice,,} == 'y' || ${is_sub_choice,,} == 'yes' ]]; then
            sub_init
        else
            return 0
        fi
    fi

    local port token filename sub_file sub_url node_count host_ip
    port=$(jq -r '.port // empty' "$is_sub_json" 2>/dev/null)
    token=$(jq -r '.token // empty' "$is_sub_json" 2>/dev/null)
    filename=$(jq -r '.filename // empty' "$is_sub_json" 2>/dev/null)

    if [[ ! $port || ! $token || ! $filename ]]; then
        warn "订阅配置文件格式异常，正在重新生成..."
        sub_init
        port=$(jq -r '.port' "$is_sub_json")
        token=$(jq -r '.token' "$is_sub_json")
        filename=$(jq -r '.filename' "$is_sub_json")
    fi

    sub_file="${is_sub_dir}/${token}/${filename}"
    [[ ! -f $sub_file ]] && sub_sync

    get_ip
    host_ip="$ip"
    [[ $(grep ":" <<<"$ip") ]] && host_ip="[$ip]"
    sub_url="http://${host_ip}:${port}/${token}/${filename}"
    node_count=$(grep -c '://' "$sub_file" 2>/dev/null || true)
    [[ ! $node_count ]] && node_count=0

    local is_color=44
    msg "\n-------------- 远程订阅 (Subscription) -------------"
    msg "订阅地址 (URL) \t=  \e[4;${is_color}m${sub_url}\e[0m"
    msg "监听端口 (port) \t=  \e[${is_color}m${port}\e[0m"
    msg "文件路径 (path) \t=  \e[${is_color}m/${token}/${filename}\e[0m"
    msg "包含节点 (nodes) \t=  \e[${is_color}m${node_count} 个\e[0m"
    if is_sub_server_running; then
        msg "服务状态 (status) \t=  $(_green 运行中) (HTTP Server)"
    else
        msg "服务状态 (status) \t=  $(_red_bg 已停止) (HTTP Server)"
    fi

    warn "提示: 如无法访问订阅链接，请检查防火墙或云服务器安全组是否已放行端口 ($port)."

    msg "\n------------- 二维码 (QR Code) -------------"
    msg
    if [[ $(type -P qrencode) ]]; then
        qrencode -t ANSI "${sub_url}"
    else
        msg "请安装 qrencode: $(_green "$cmd update -y; $cmd install qrencode -y")"
    fi
    msg
    msg "若无法直接扫描，可通过以下在线链接生成二维码:"
    msg "\e[4;${is_color}mhttps://233boy.github.io/tools/qr.html#${sub_url}\e[0m\n"
}

sub_reset() {
    [[ ! -f $is_sub_json ]] && {
        sub_init
        sub_info
        return 0
    }
    local old_token
    old_token=$(jq -r '.token // empty' "$is_sub_json" 2>/dev/null)
    [[ $old_token && -d "${is_sub_dir}/${old_token}" ]] && rm -rf "${is_sub_dir}/${old_token}"

    local port token hostname_val filename
    port=$(sub_port_gen)
    token=$(sub_token_gen)
    hostname_val=$(hostname 2>/dev/null || cat /proc/sys/kernel/hostname 2>/dev/null || echo "singbox")
    hostname_val=$(echo "$hostname_val" | tr -cd 'a-zA-Z0-9_-')
    [[ ! $hostname_val ]] && hostname_val="singbox"
    filename="${hostname_val}_singbox.yaml"

    mkdir -p "${is_sub_dir}/${token}"
    jq -n --argjson port "$port" \
          --arg token "$token" \
          --arg filename "$filename" \
          '{port: $port, token: $token, filename: $filename, enabled: true}' >"$is_sub_json"

    sub_caddy_apply "$port"
    sub_sync
    sleep 1
    _green "\n已重置订阅服务（新端口与新路径）:"
    sub_info
}

sub_port_change() {
    [[ ! -f $is_sub_json ]] && {
        err "当前未开启远程订阅，请先使用: $is_core sub 开启."
    }
    local new_port=$1
    if [[ ! $new_port ]]; then
        ask string new_port "请输入新的订阅监听端口 (10000-65535):"
    fi
    if [[ ! $(is_test port "$new_port") ]]; then
        err "请输入正确的端口 (1-65535)."
    fi
    if [[ $(is_test port_used "$new_port") ]]; then
        err "无法使用 ($new_port) 端口，该端口已被占用."
    fi

    cat <<<"$(jq --argjson port "$new_port" '.port = $port' "$is_sub_json")" >"$is_sub_json"
    sub_caddy_apply "$new_port"
    _green "\n已更新订阅端口为: $new_port\n"
    sub_info
}

sub_del() {
    [[ ! -f $is_sub_json && ! -f $is_sub_caddy_conf ]] && {
        warn "当前未配置远程订阅服务."
        return 0
    }
    if [[ ! $is_dont_auto_exit ]]; then
        echo -ne "是否确认停用并删除远程订阅服务? [y/n]: "
        read -r is_del_choice
        if [[ ${is_del_choice,,} != 'y' && ${is_del_choice,,} != 'yes' ]]; then
            msg "已取消操作."
            return 0
        fi
    fi
    systemctl stop sing-box-sub &>/dev/null || true
    systemctl disable sing-box-sub &>/dev/null || true
    rm -f /lib/systemd/system/sing-box-sub.service
    systemctl daemon-reload &>/dev/null || true
    pkill -f "http.server" 2>/dev/null || true
    rm -rf "$is_sub_dir" "$is_sub_json" "$is_sub_caddy_conf"
    _green "\n已停用并清理远程订阅服务.\n"
}

sub_main() {
    case ${1,,} in
    "" | info)
        sub_info
        ;;
    new | reset)
        sub_reset
        ;;
    update | sync)
        sub_sync
        _green "\n已更新订阅节点内容.\n"
        sub_info
        ;;
    port)
        sub_port_change "$2"
        ;;
    del | rm | stop | off)
        sub_del
        ;;
    *)
        msg "\n使用方法: $(_green $is_core sub) [info | new | update | port | del]\n"
        msg "   $is_core sub               查看订阅地址与二维码"
        msg "   $is_core sub new           重置并重新生成随机端口与路径"
        msg "   $is_core sub update        手动刷新订阅中的节点列表"
        msg "   $is_core sub port [port]   修改订阅端口"
        msg "   $is_core sub del           停用并删除订阅服务\n"
        ;;
    esac
}
