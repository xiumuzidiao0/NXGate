#!/bin/bash

# sing-box external control API module
# Designed for seamless integration with AimiliVPN & external management consoles
# Returns standard JSON outputs for all programmatic operations

api_err() {
    local msg="$1"
    local code="${2:-1}"
    jq -n --arg error "$msg" --argjson code "$code" '{ok: false, code: $code, error: $error}'
    exit "$code"
}

api_ok() {
    local data="${1:-{}}"
    jq -n --argjson data "$data" '{ok: true, code: 0} + $data'
}

# Convert an inbound configuration file to a clean JSON object
api_node_to_json() {
    local conf_file="$1"
    conf_file=$(basename "$conf_file")
    local full_path="$is_conf_dir/$conf_file"
    [[ ! -f "$full_path" ]] && return 1

    # Temporarily suppress human-oriented stdout
    local old_dont_show="$is_dont_show_info"
    local old_dont_exit="$is_dont_auto_exit"
    is_dont_show_info=1
    is_dont_auto_exit=1

    # Reset parsed variables
    unset is_protocol port uuid password username ss_method ss_password door_port door_addr
    unset net_type path host is_servername is_private_key is_public_key is_url net is_anytls_domain
    unset is_outbound_server is_outbound_port is_outbound_type is_outbound_user is_outbound_pass

    info "$conf_file" &>/dev/null

    is_dont_show_info="$old_dont_show"
    is_dont_auto_exit="$old_dont_exit"

    if [[ "$is_protocol" == "anytls" ]]; then
        if [[ -z "$is_anytls_domain" || "${is_anytls_domain,,}" == "auto" ]]; then
            unset is_anytls_domain
            get_ip
            is_addr="$ip"
            [[ "$ip" =~ : ]] && is_addr="[$ip]"
            is_url="anytls://$password@$is_addr:$port?insecure=1&allowInsecure=1#233boy-anytls-$is_addr"
        fi
    fi
    if [[ "$is_addr" == "auto" || -z "$is_addr" ]]; then
        get_ip
        is_addr="$ip"
        [[ "$ip" =~ : ]] && is_addr="[$ip]"
    fi
    if [[ "$is_url" =~ @auto[:#] ]]; then
        is_url="${is_url//@auto:/@$is_addr:}"
        is_url="${is_url//@auto#/@$is_addr#}"
        is_url="${is_url//-auto/-$is_addr}"
    fi

    local outbound_str="direct"
    local ob_type="${is_outbound_type:-direct}"
    local ob_server="${is_outbound_server:-}"
    local ob_port="${is_outbound_port:-0}"
    local ob_user="${is_outbound_user:-}"
    local ob_pass="${is_outbound_pass:-}"

    if [[ -n "$ob_server" && "$ob_port" -gt 0 ]]; then
        if [[ "$ob_type" == "http" ]]; then
            outbound_str="http://${ob_server}:${ob_port}"
        else
            outbound_str="socks5://${ob_server}:${ob_port}"
        fi
        if [[ -n "$ob_user" ]]; then
            local p_type="$ob_type"
            [[ "$p_type" == "socks" ]] && p_type="socks5"
            outbound_str="${p_type}://${ob_user}:${ob_pass}@${ob_server}:${ob_port}"
        fi
    fi

    local node_proto="${is_protocol:-unknown}"
    if [[ "$net" == "reality" ]]; then
        node_proto="VLESS-REALITY"
    elif [[ "$net" == "rh2" ]]; then
        node_proto="VLESS-HTTP2-REALITY"
    elif [[ "$net" == "tuic" ]]; then
        node_proto="TUIC"
    elif [[ "$net" == "hysteria2" || "$net" =~ hy* ]]; then
        node_proto="Hysteria2"
    elif [[ "$net" == "ss" ]]; then
        node_proto="Shadowsocks"
    elif [[ "$net" == "trojan" ]]; then
        node_proto="Trojan"
    elif [[ "$net" == "anytls" ]]; then
        node_proto="AnyTLS"
    elif [[ "$net" == "socks" ]]; then
        node_proto="Socks"
    elif [[ "$net" == "direct" ]]; then
        node_proto="Direct"
    elif [[ "$is_protocol" == "vmess" ]]; then
        node_proto="VMess-${net^^}"
    elif [[ "$is_protocol" == "vless" ]]; then
        node_proto="VLESS-${net^^}"
    fi

    if [[ "$conf_file" =~ -TLS- ]]; then
        local tls_prefix=$(echo "$conf_file" | sed -E 's/-[0-9]+.*//;s/-[a-zA-Z0-9.-]+\.json$//')
        [[ -n "$tls_prefix" ]] && node_proto="$tls_prefix"
    fi

    jq -n \
        --arg name "$conf_file" \
        --arg tag "$conf_file" \
        --arg protocol "$node_proto" \
        --arg raw_proto "${is_protocol:-}" \
        --arg net "${net:-tcp}" \
        --argjson port "${port:-0}" \
        --arg address "${is_addr:-}" \
        --arg uuid "${uuid:-}" \
        --arg password "${password:-}" \
        --arg username "${username:-${is_socks_user:-}}" \
        --arg ss_method "${ss_method:-}" \
        --arg sni "${is_servername:-}" \
        --arg host "${host:-}" \
        --arg path "${path:-}" \
        --arg pbk "${is_public_key:-}" \
        --arg flow "${is_flow:-}" \
        --arg network "${is_net_type:-${net:-tcp}}" \
        --arg outbound "$outbound_str" \
        --arg outbound_type "$ob_type" \
        --arg outbound_server "$ob_server" \
        --argjson outbound_port "${ob_port:-0}" \
        --arg outbound_user "$ob_user" \
        --arg url "${is_url:-}" \
        '{
            name: $name,
            tag: $tag,
            protocol: $protocol,
            raw_protocol: $raw_proto,
            network: $network,
            port: $port,
            address: $address,
            uuid: $uuid,
            password: $password,
            username: $username,
            ss_method: $ss_method,
            sni: $sni,
            host: $host,
            path: $path,
            pbk: $pbk,
            flow: $flow,
            outbound: $outbound,
            outbound_type: $outbound_type,
            outbound_server: $outbound_server,
            outbound_port: $outbound_port,
            outbound_user: $outbound_user,
            url: $url
        }'
}

# List all nodes in JSON
api_list_nodes() {
    local node_list=()
    if [[ -d "$is_conf_dir" ]]; then
        local confs
        confs=$(ls "$is_conf_dir" 2>/dev/null | grep .json$ | sed '/dynamic-port-.*-link/d')
        for conf in $confs; do
            local item
            item=$(api_node_to_json "$conf")
            if [[ -n "$item" ]]; then
                node_list+=("$item")
            fi
        done
    fi

    local joined_nodes="[]"
    if [[ ${#node_list[@]} -gt 0 ]]; then
        joined_nodes=$(printf '%s\n' "${node_list[@]}" | jq -s '.')
    fi

    jq -n --argjson nodes "$joined_nodes" \
          '{ok: true, count: ($nodes | length), nodes: $nodes}'
}

# Get detailed info for a single node in JSON
api_info_node() {
    local name="$1"
    [[ -z "$name" ]] && api_err "缺少节点名称参数 (name)"

    # Add .json suffix if missing
    [[ "$name" != *.json ]] && name="${name}.json"

    local full_path="$is_conf_dir/$name"
    if [[ ! -f "$full_path" ]]; then
        # Try finding by partial match
        local match
        match=$(ls "$is_conf_dir" 2>/dev/null | grep -i "$name" | head -n1)
        if [[ -n "$match" && -f "$is_conf_dir/$match" ]]; then
            name="$match"
        else
            api_err "找不到指定的节点配置文件: $name"
        fi
    fi

    local node_data
    node_data=$(api_node_to_json "$name")
    [[ -z "$node_data" ]] && api_err "解析节点配置失败: $name"

    jq -n --argjson node "$node_data" '{ok: true, node: $node}'
}

api_ensure_config_json() {
    if [[ -f "$is_config_json" ]]; then
        local has_final
        has_final=$(jq -r '.route.final // empty' "$is_config_json" 2>/dev/null)
        if [[ "$has_final" != "direct" ]]; then
            local updated
            updated=$(jq '.outbounds = (if (.outbounds // [] | map(select(.tag == "direct")) | length) > 0 then .outbounds else [{tag:"direct",type:"direct"}] + (.outbounds // []) end) | .route = ((.route // {}) + {final: "direct"})' "$is_config_json" 2>/dev/null)
            if [[ -n "$updated" ]]; then
                echo "$updated" > "$is_config_json"
            fi
        fi
    fi
}

api_restart_service() {
    api_ensure_config_json
    if [[ $is_systemd ]]; then
        systemctl restart "$is_core" 2>/dev/null || true
    elif [[ $is_openrc ]]; then
        rc-service "$is_core" restart 2>/dev/null || true
    else
        pkill -f "$is_core_bin" 2>/dev/null || true
        sleep 0.5
        "$is_core_bin" run -c "$is_config_json" -C "$is_conf_dir" &>/dev/null &
    fi
}

# Add a new proxy node non-interactively using original project logic
api_add_node() {
    local in_proto="$1"
    shift 1 || true

    [[ -z "$in_proto" ]] && api_err "缺少代理协议参数 (protocol)，可选: reality, rh2, hy2, tuic, ss, trojan, anytls, socks, direct, vws, wss, tws, vhu, hu, thu, vh2, h2, th2, ws, tcp, http, quic"

    local in_outbound=""
    local add_args=()

    while [[ $# -gt 0 ]]; do
        case "$1" in
        --outbound | -o | --out)
            in_outbound="$2"
            shift 2 || shift 1
            ;;
        *)
            add_args+=("$1")
            shift 1
            ;;
        esac
    done

    # If outbound was not passed with --outbound flag:
    # Check if the trailing argument is an outbound address (e.g. 127.0.0.1:7928, socks5://..., http://..., or direct)
    if [[ -z "$in_outbound" && ${#add_args[@]} -gt 0 ]]; then
        local last_idx=$((${#add_args[@]} - 1))
        local last_arg="${add_args[$last_idx]}"
        if [[ "$last_arg" =~ :[0-9]+$ || "${last_arg,,}" =~ ^(direct|none|default|null)$ || "$last_arg" =~ :// ]]; then
            local p_lower="${in_proto,,}"
            local max_proto_args=2 # default for hy2, tuic, trojan
            case "$p_lower" in
            *reality* | ss | shadowsocks | socks | anytls | direct | *-tls | *ws* | *hu* | *h2*)
                max_proto_args=3
                ;;
            esac
            if [[ ${#add_args[@]} -gt $max_proto_args ]]; then
                in_outbound="$last_arg"
                unset 'add_args[last_idx]'
            fi
        fi
    fi

    # Set non-interactive flags
    is_dont_show_info=1
    is_dont_auto_exit=1
    is_no_del_msg=1
    is_dont_test_host=1

    # Reset any inherited global state
    unset port uuid password host path ss_method is_servername is_socks_user is_socks_pass is_anytls_domain
    unset is_use_port is_use_uuid is_use_host is_use_path is_use_pass is_use_method is_use_door_addr is_use_door_port is_use_servername is_use_socks_user is_use_socks_pass is_anytls_domain

    # If outbound specified, parse and preserve it
    if [[ -n "$in_outbound" && "${in_outbound,,}" != "direct" ]]; then
        parse_outbound "$in_outbound"
        is_api_outbound=1
    else
        unset is_outbound_server is_outbound_port is_outbound_type is_outbound_user is_outbound_pass
        unset is_api_outbound
    fi

    # Snapshot existing files before addition
    local before_files
    before_files=$(ls -1 "$is_conf_dir" 2>/dev/null | grep '\.json$' | sort || true)

    # Normalize protocol specific arguments for non-interactive creation
    local p_norm="${in_proto,,}"
    case "$p_norm" in
    *-tls | wss | h2 | hu | vws | vh2 | vhu | tws | th2 | thu)
        local h_arg="${add_args[2]}"
        local c_arg="${add_args[1]}"
        if [[ -z "$h_arg" || "$h_arg" == "auto" ]]; then
            get_ip
            if [[ -n "$ip" ]]; then
                h_arg="${ip//:/-}.nip.io"
            else
                h_arg="singbox.local"
            fi
        fi
        add_args=("$h_arg" "$c_arg" "auto")
        ;;
    ss | shadowsocks)
        local p_arg="${add_args[0]:-auto}"
        local pass_arg="${add_args[1]:-auto}"
        local m_arg="${add_args[2]:-2022-blake3-aes-128-gcm}"
        [[ "$m_arg" == "auto" ]] && m_arg="2022-blake3-aes-128-gcm"
        add_args=("$p_arg" "$pass_arg" "$m_arg")
        ;;
    direct | door)
        local p_arg="${add_args[0]:-auto}"
        local d_addr="${add_args[1]:-127.0.0.1}"
        local d_port="${add_args[2]:-7928}"
        [[ "$d_addr" == "auto" ]] && d_addr="127.0.0.1"
        [[ "$d_port" == "auto" ]] && d_port="7928"
        add_args=("$p_arg" "$d_addr" "$d_port")
        ;;
    anytls*)
        local p_arg="${add_args[0]:-auto}"
        local pass_arg="${add_args[1]:-auto}"
        local dom_arg="${add_args[2]:-}"
        # If domain is empty, "auto", or does not contain a dot, clear it so AnyTLS runs in IP/self-signed mode
        if [[ -z "$dom_arg" || "${dom_arg,,}" == "auto" || ! "$dom_arg" =~ \. ]]; then
            unset is_anytls_domain
            add_args=("$p_arg" "$pass_arg")
        else
            is_anytls_domain="$dom_arg"
            add_args=("$p_arg" "$pass_arg" "$dom_arg")
        fi
        ;;
    esac

    # Execute original add logic without modifying core generation template
    local add_out
    add_out=$(add "$in_proto" "${add_args[@]}" 2>&1)
    local ret=$?
    if [[ $ret -ne 0 ]]; then
        unset is_api_outbound
        local err_msg
        err_msg=$(echo "$add_out" | grep -E "错误|err|无法|失败" | head -n 1 | sed 's/.*: //;s/^[ \t]*//')
        [[ -z "$err_msg" ]] && err_msg="创建节点失败，返回状态码: $ret"
        api_err "$err_msg"
    fi
    unset is_api_outbound

    local after_files
    after_files=$(ls -1 "$is_conf_dir" 2>/dev/null | grep '\.json$' | sort || true)

    local created_name
    created_name=$(comm -13 <(echo "$before_files") <(echo "$after_files") | head -n1 || true)
    if [[ -z "$created_name" || ! -f "$is_conf_dir/$created_name" ]]; then
        created_name="$is_config_name"
    fi
    if [[ -z "$created_name" || ! -f "$is_conf_dir/$created_name" ]]; then
        created_name=$(ls -t "$is_conf_dir" 2>/dev/null | grep '\.json$' | head -n1 || true)
    fi

    [[ -z "$created_name" ]] && api_err "节点创建可能已完成但未找到生成的配置文件"

    api_restart_service

    local node_data
    node_data=$(api_node_to_json "$created_name")

    jq -n \
        --arg msg "节点创建成功" \
        --arg target "$created_name" \
        --argjson node "$node_data" \
        '{ok: true, msg: $msg, target: $target, node: $node}'
}

# Update outbound exit of existing node(s) to local proxy (e.g. 127.0.0.1:7928) or direct
api_set_outbound() {
    local target="$1"
    local outbound="$2"

    [[ -z "$target" ]] && api_err "缺少目标节点参数 (target: 节点文件名或 'all')"
    [[ -z "$outbound" ]] && api_err "缺少出站出口参数 (outbound)，例如: 127.0.0.1:7928, socks5://127.0.0.1:7928, http://127.0.0.1:7928 或 direct"

    is_dont_show_info=1
    is_dont_auto_exit=1
    is_no_del_msg=1

    if [[ "${target,,}" == "all" ]]; then
        local count=0
        local updated=()
        if [[ -d "$is_conf_dir" ]]; then
            local confs
            confs=$(ls "$is_conf_dir" 2>/dev/null | grep .json$ | sed '/dynamic-port-.*-link/d')
            for conf in $confs; do
                change "$conf" out "$outbound" &>/dev/null
                updated+=("$conf")
                ((count++))
            done
        fi
        api_restart_service

        # sync subscription
        if [[ -f "$is_sub_json" ]]; then
            load sub.sh
            sub_sync &>/dev/null
        fi

        jq -n \
            --arg msg "已更新全部节点出站出口" \
            --arg outbound "$outbound" \
            --argjson count "$count" \
            --argjson targets "$(printf '%s\n' "${updated[@]}" | jq -R . | jq -s .)" \
            '{ok: true, msg: $msg, outbound: $outbound, updated_count: $count, targets: $targets}'
    else
        [[ "$target" != *.json ]] && target="${target}.json"
        local full_path="$is_conf_dir/$target"
        if [[ ! -f "$full_path" ]]; then
            local match
            match=$(ls "$is_conf_dir" 2>/dev/null | grep -i "$target" | head -n1)
            [[ -n "$match" ]] && target="$match" || api_err "找不到指定的节点配置文件: $target"
        fi

        change "$target" out "$outbound" &>/dev/null
        api_restart_service

        if [[ -f "$is_sub_json" ]]; then
            load sub.sh
            sub_sync &>/dev/null
        fi

        local node_data
        node_data=$(api_node_to_json "$target")

        jq -n \
            --arg msg "已更新节点出站出口" \
            --arg target "$target" \
            --arg outbound "$outbound" \
            --argjson node "$node_data" \
            '{ok: true, msg: $msg, target: $target, outbound: $outbound, node: $node}'
    fi
}

# Delete node(s)
api_del_node() {
    local target="$1"
    [[ -z "$target" ]] && api_err "缺少需要删除的目标参数 (name 或 'all')"

    is_dont_show_info=1
    is_dont_auto_exit=1
    is_no_del_msg=1

    if [[ "${target,,}" == "all" ]]; then
        local deleted=()
        if [[ -d "$is_conf_dir" ]]; then
            local confs
            confs=$(ls "$is_conf_dir" 2>/dev/null | grep .json$ | sed '/dynamic-port-.*-link/d')
            for conf in $confs; do
                del "$conf" &>/dev/null
                deleted+=("$conf")
            done
        fi
        api_restart_service
        if [[ -f "$is_sub_json" ]]; then
            load sub.sh
            sub_sync &>/dev/null
        fi
        jq -n \
            --arg msg "已清空全部代理节点" \
            --argjson deleted "$(printf '%s\n' "${deleted[@]}" | jq -R . | jq -s .)" \
            '{ok: true, msg: $msg, deleted: $deleted}'
    else
        [[ "$target" != *.json ]] && target="${target}.json"
        local full_path="$is_conf_dir/$target"
        if [[ ! -f "$full_path" ]]; then
            local match
            match=$(ls "$is_conf_dir" 2>/dev/null | grep -i "$target" | head -n1)
            [[ -n "$match" ]] && target="$match" || api_err "找不到指定的节点配置文件: $target"
        fi

        del "$target" &>/dev/null
        api_restart_service
        if [[ -f "$is_sub_json" ]]; then
            load sub.sh
            sub_sync &>/dev/null
        fi

        jq -n \
            --arg msg "节点删除成功" \
            --arg target "$target" \
            '{ok: true, msg: $msg, target: $target}'
    fi
}

# Subscription API management
api_sub_manage() {
    local action="${1:-get}"
    load sub.sh

    case "${action,,}" in
    get | info | show | status)
        if [[ ! -f "$is_sub_json" ]]; then
            jq -n '{ok: true, enabled: false, msg: "远程订阅服务尚未开启", node_count: 0, nodes: []}'
            return 0
        fi

        local port token filename sub_file host_ip sub_url
        port=$(jq -r '.port // empty' "$is_sub_json" 2>/dev/null)
        token=$(jq -r '.token // empty' "$is_sub_json" 2>/dev/null)
        filename=$(jq -r '.filename // empty' "$is_sub_json" 2>/dev/null)

        if [[ -z "$port" || -z "$token" || -z "$filename" ]]; then
            api_err "订阅配置文件 ($is_sub_json) 损坏或格式错误"
        fi

        sub_file="${is_sub_dir}/${token}/${filename}"
        [[ ! -f "$sub_file" ]] && sub_sync &>/dev/null

        get_ip
        host_ip="$ip"
        [[ $(grep ":" <<<"$ip") ]] && host_ip="[$ip]"
        sub_url="http://${host_ip}:${port}/${token}/${filename}"

        local node_urls=()
        if [[ -f "$sub_file" ]]; then
            while IFS= read -r line; do
                [[ -n "$line" ]] && node_urls+=("$line")
            done < "$sub_file"
        fi

        local is_caddy_run=false
        if is_sub_server_running 2>/dev/null || systemctl is-active sing-box-sub &>/dev/null || pgrep -f "http.server" &>/dev/null || [[ -n "$is_caddy_bin" && $(pgrep -f "$is_caddy_bin") ]]; then
            is_caddy_run=true
        fi

        local joined_urls="[]"
        if [[ ${#node_urls[@]} -gt 0 ]]; then
            joined_urls=$(printf '%s\n' "${node_urls[@]}" | jq -R . | jq -s .)
        fi

        jq -n \
            --argjson enabled true \
            --arg sub_url "$sub_url" \
            --argjson port "$port" \
            --arg token "$token" \
            --arg filename "$filename" \
            --argjson node_count "${#node_urls[@]}" \
            --argjson nodes "$joined_urls" \
            --argjson caddy_running "$is_caddy_run" \
            '{
                ok: true,
                enabled: $enabled,
                sub_url: $sub_url,
                port: $port,
                token: $token,
                filename: $filename,
                node_count: $node_count,
                nodes: $nodes,
                caddy_running: $caddy_running
            }'
        ;;

    sync | update)
        sub_sync &>/dev/null
        api_sub_manage get
        ;;

    init | new | start)
        local custom_port="$2"
        if [[ ! -f "$is_sub_json" ]]; then
            local p token hname fname
            p="$custom_port"
            [[ -z "$p" ]] && p=$(sub_port_gen)
            token=$(sub_token_gen)
            hname=$(hostname 2>/dev/null || cat /proc/sys/kernel/hostname 2>/dev/null || echo "singbox")
            hname=$(echo "$hname" | tr -cd 'a-zA-Z0-9_-')
            [[ -z "$hname" ]] && hname="singbox"
            fname="${hname}_singbox.yaml"

            mkdir -p "${is_sub_dir}/${token}"
            jq -n --argjson port "$p" \
                  --arg token "$token" \
                  --arg filename "$fname" \
                  '{port: $port, token: $token, filename: $filename, enabled: true}' >"$is_sub_json"

            sub_caddy_apply "$p" &>/dev/null
            sub_sync &>/dev/null
        fi
        api_sub_manage get
        ;;

    reset)
        sub_reset &>/dev/null
        api_sub_manage get
        ;;

    port)
        local new_port="$2"
        [[ -z "$new_port" ]] && api_err "缺少新端口参数"
        sub_port "$new_port" &>/dev/null
        api_sub_manage get
        ;;

    del | disable | stop)
        sub_del &>/dev/null
        jq -n '{ok: true, msg: "远程订阅服务已关闭并清除配置"}'
        ;;

    *)
        api_err "未知订阅操作: $action，可选: get, sync, init, reset, port, del"
        ;;
    esac
}

# Status API
api_status() {
    local core_run=false
    local core_pid=0
    local pids
    pids=$(pgrep -f "$is_core_bin" 2>/dev/null)
    if [[ -n "$pids" ]]; then
        core_run=true
        core_pid=$(echo "$pids" | head -n1)
    fi

    local caddy_run=false
    local caddy_pid=0
    if [[ -n "$is_caddy_bin" ]]; then
        local cpids
        cpids=$(pgrep -f "$is_caddy_bin" 2>/dev/null)
        if [[ -n "$cpids" ]]; then
            caddy_run=true
            caddy_pid=$(echo "$cpids" | head -n1)
        fi
    fi

    local n_count=0
    if [[ -d "$is_conf_dir" ]]; then
        n_count=$(ls "$is_conf_dir" 2>/dev/null | grep -c .json$ || true)
    fi

    local sub_on=false
    [[ -f "$is_sub_json" ]] && sub_on=true

    jq -n \
        --arg name "${is_core_name:-sing-box}" \
        --arg version "${is_core_ver:-unknown}" \
        --arg script_ver "${is_sh_ver:-unknown}" \
        --argjson core_running "$core_run" \
        --argjson core_pid "$core_pid" \
        --argjson caddy_running "$caddy_run" \
        --argjson caddy_pid "$caddy_pid" \
        --arg caddy_ver "${is_caddy_ver:-}" \
        --argjson node_count "$n_count" \
        --argjson sub_enabled "$sub_on" \
        '{
            ok: true,
            core: {
                name: $name,
                version: $version,
                script_version: $script_ver,
                running: $core_running,
                pid: $core_pid
            },
            caddy: {
                running: $caddy_running,
                pid: $caddy_pid,
                version: $caddy_ver
            },
            node_count: $node_count,
            subscription_enabled: $sub_enabled
        }'
}

# List supported protocols
api_protocols() {
    cat <<'EOF' | jq .
{
    "ok": true,
    "protocols": [
        {
            "id": "reality",
            "name": "VLESS-REALITY",
            "category": "recommended",
            "recommended": true,
            "transport": "tcp",
            "tls": "reality",
            "description": "顶级抗审查伪装，直接借用海外权威名站 TLS 指纹，无需自行配置域名",
            "args": ["port", "uuid", "sni"]
        },
        {
            "id": "rh2",
            "name": "VLESS-HTTP2-REALITY",
            "category": "recommended",
            "recommended": true,
            "transport": "http2",
            "tls": "reality",
            "description": "HTTP/2 多路复用 + REALITY TLS 双重伪装，高并发大吞吐免配域名",
            "args": ["port", "uuid", "sni"]
        },
        {
            "id": "hy2",
            "name": "Hysteria2",
            "category": "recommended",
            "recommended": true,
            "transport": "udp",
            "tls": "self-signed",
            "description": "基于定制 QUIC 协议，针对跨洋恶劣网络与高丢包极速狂飙",
            "args": ["port", "password"]
        },
        {
            "id": "tuic",
            "name": "TUIC v5",
            "category": "recommended",
            "recommended": true,
            "transport": "quic",
            "tls": "self-signed",
            "description": "基于 QUIC 拥塞控制 BBR 协议，低延迟抗抖动高可靠",
            "args": ["port", "uuid"]
        },
        {
            "id": "ss",
            "name": "Shadowsocks 2022",
            "category": "recommended",
            "recommended": true,
            "transport": "tcp/udp",
            "tls": "none",
            "description": "现代 Shadowsocks 2022 规范，性能极高客户端广泛支持",
            "args": ["port", "password", "method"]
        },
        {
            "id": "trojan",
            "name": "Trojan",
            "category": "popular",
            "recommended": false,
            "transport": "tcp",
            "tls": "tls",
            "description": "经典伪装 HTTPS 协议，流量表现为正常安全网页浏览",
            "args": ["port", "password"]
        },
        {
            "id": "anytls",
            "name": "AnyTLS",
            "category": "popular",
            "recommended": false,
            "transport": "tcp",
            "tls": "acme/tls",
            "description": "多路径自适应 TLS 传输协议，动态流量特征自愈",
            "args": ["port", "password", "domain"]
        },
        {
            "id": "socks",
            "name": "Socks5",
            "category": "popular",
            "recommended": false,
            "transport": "tcp",
            "tls": "none",
            "description": "标准 Socks5 代理入站，局域网与前置分流首选",
            "args": ["port", "username", "password"]
        },
        {
            "id": "direct",
            "name": "Direct (端口转发)",
            "category": "popular",
            "recommended": false,
            "transport": "tcp",
            "tls": "none",
            "description": "透明 TCP 端口流量中继直通",
            "args": ["port", "remote_addr", "remote_port"]
        },
        {
            "id": "vws",
            "name": "VLESS-WS-TLS",
            "category": "cdn",
            "recommended": false,
            "transport": "ws",
            "tls": "tls",
            "description": "VLESS + WebSocket + TLS，支持 Cloudflare CDN 优选反代免封",
            "args": ["host", "uuid", "path"]
        },
        {
            "id": "wss",
            "name": "VMess-WS-TLS",
            "category": "cdn",
            "recommended": false,
            "transport": "ws",
            "tls": "tls",
            "description": "VMess + WebSocket + TLS，经典 CDN 拯救被墙 IP 方案",
            "args": ["host", "uuid", "path"]
        },
        {
            "id": "tws",
            "name": "Trojan-WS-TLS",
            "category": "cdn",
            "recommended": false,
            "transport": "ws",
            "tls": "tls",
            "description": "Trojan + WebSocket + TLS，结合 CDN 隐藏源站真实地址",
            "args": ["host", "password", "path"]
        },
        {
            "id": "vhu",
            "name": "VLESS-HTTPUpgrade-TLS",
            "category": "cdn",
            "recommended": false,
            "transport": "httpupgrade",
            "tls": "tls",
            "description": "新一代 HTTPUpgrade 传输，比 WS 更省服务端 CPU 资源",
            "args": ["host", "uuid", "path"]
        },
        {
            "id": "hu",
            "name": "VMess-HTTPUpgrade-TLS",
            "category": "cdn",
            "recommended": false,
            "transport": "httpupgrade",
            "tls": "tls",
            "description": "VMess + HTTPUpgrade 现代化高性能传输",
            "args": ["host", "uuid", "path"]
        },
        {
            "id": "thu",
            "name": "Trojan-HTTPUpgrade-TLS",
            "category": "cdn",
            "recommended": false,
            "transport": "httpupgrade",
            "tls": "tls",
            "description": "Trojan + HTTPUpgrade 传输",
            "args": ["host", "password", "path"]
        },
        {
            "id": "vh2",
            "name": "VLESS-H2-TLS",
            "category": "cdn",
            "recommended": false,
            "transport": "http2",
            "tls": "tls",
            "description": "VLESS + HTTP/2 多路复用 + TLS",
            "args": ["host", "uuid", "path"]
        },
        {
            "id": "h2",
            "name": "VMess-H2-TLS",
            "category": "cdn",
            "recommended": false,
            "transport": "http2",
            "tls": "tls",
            "description": "VMess + HTTP/2 多路复用 + TLS",
            "args": ["host", "uuid", "path"]
        },
        {
            "id": "th2",
            "name": "Trojan-H2-TLS",
            "category": "cdn",
            "recommended": false,
            "transport": "http2",
            "tls": "tls",
            "description": "Trojan + HTTP/2 多路复用 + TLS",
            "args": ["host", "password", "path"]
        },
        {
            "id": "ws",
            "name": "VMess-WS",
            "category": "raw",
            "recommended": false,
            "transport": "ws",
            "tls": "none",
            "description": "纯 WebSocket 传输，便于前置 Nginx/Caddy 自建反向代理",
            "args": ["port", "uuid"]
        },
        {
            "id": "tcp",
            "name": "VMess-TCP",
            "category": "raw",
            "recommended": false,
            "transport": "tcp",
            "tls": "none",
            "description": "原生 TCP 传输，开销极低",
            "args": ["port", "uuid"]
        },
        {
            "id": "http",
            "name": "VMess-HTTP",
            "category": "raw",
            "recommended": false,
            "transport": "http",
            "tls": "none",
            "description": "伪装 HTTP/1.1 明文传输",
            "args": ["port", "uuid"]
        },
        {
            "id": "quic",
            "name": "VMess-QUIC",
            "category": "raw",
            "recommended": false,
            "transport": "quic",
            "tls": "none",
            "description": "基于原生 QUIC 协议传输",
            "args": ["port", "uuid"]
        }
    ]
}
EOF
}

# API dispatcher
api_main() {
    local action="${1:-help}"
    case "${action,,}" in
    list | nodes | get-nodes)
        api_list_nodes
        ;;
    info | node | get-node)
        api_info_node "$2"
        ;;
    add | new | create)
        api_add_node "${@:2}"
        ;;
    out | outbound | set-outbound)
        api_set_outbound "$2" "$3"
        ;;
    del | rm | delete)
        api_del_node "$2"
        ;;
    sub | subscription)
        api_sub_manage "${@:2}"
        ;;
    status | ping)
        api_status
        ;;
    restart | reload)
        api_restart_service
        api_status
        ;;
    protocols | protos)
        api_protocols
        ;;
    help | --help | -h)
        cat <<EOF
AimiliVPN & sing-box 对外控制 API 接口:
用法: $is_core api <action> [args...]

可用指令:
   api list                             以 JSON 格式输出所有活跃代理节点详情
   api info <name>                      获取指定节点详细参数与客户端连接 URL
   api add <proto> [port] [id] [sni] [outbound]
                                        无交互快速创建入站节点 (默认 auto 参数)
                                        示例: $is_core api add reality auto auto auto 127.0.0.1:7928
   api outbound <name|all> <outbound>   将现有节点链式绑定至 AimiliVPN 本地代理
                                        示例: $is_core api outbound all 127.0.0.1:7928
                                        恢复直连: $is_core api outbound all direct
   api del <name|all>                   删除指定节点或全部节点
   api sub [get|sync|init|reset|port]   远程订阅链接与节点同步管理
   api status                           获取 sing-box 进程与 Caddy 运行状态
   api protocols                        列出支持的抗封锁入站协议列表
EOF
        ;;
    *)
        api_err "未知 API 操作: $action。获取帮助请执行: $is_core api help"
        ;;
    esac
}
