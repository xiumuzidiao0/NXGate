#!/bin/bash

author=xiumuzidiao0
# github=https://github.com/xiumuzidiao0/sing-box

# bash fonts colors
red='\e[31m'
yellow='\e[33m'
gray='\e[90m'
green='\e[92m'
blue='\e[94m'
magenta='\e[95m'
cyan='\e[96m'
none='\e[0m'

_red() { echo -e ${red}$@${none}; }
_blue() { echo -e ${blue}$@${none}; }
_cyan() { echo -e ${cyan}$@${none}; }
_green() { echo -e ${green}$@${none}; }
_yellow() { echo -e ${yellow}$@${none}; }
_magenta() { echo -e ${magenta}$@${none}; }
_red_bg() { echo -e "\e[41m$@${none}"; }

_rm() {
    rm -rf "$@"
}
_cp() {
    cp -rf "$@"
}
_sed() {
    sed -i "$@"
}
_mkdir() {
    mkdir -p "$@"
}

is_err=$(_red_bg 错误!)
is_warn=$(_red_bg 警告!)

err() {
    echo -e "\n$is_err $@\n"
    [[ $is_dont_auto_exit ]] && return
    exit 1
}

warn() {
    echo -e "\n$is_warn $@\n"
}

# load bash script.
load() {
    if [[ -n "$is_sh_dir" && -f "$is_sh_dir/src/$1" ]]; then
        . "$is_sh_dir/src/$1"
    elif [[ -f "$(dirname "${BASH_SOURCE[0]}")/$1" ]]; then
        . "$(dirname "${BASH_SOURCE[0]}")/$1"
    else
        . "/etc/sing-box/sh/src/$1"
    fi
}

# wget add --no-check-certificate
_wget() {
    # [[ $proxy ]] && export https_proxy=$proxy
    wget --no-check-certificate "$@"
}

# apt-get, yum, zypper or apk
cmd=$(type -P apt-get || type -P yum || type -P zypper || type -P apk)

# x64
case $(uname -m) in
amd64 | x86_64)
    is_arch="amd64"
    ;;
*aarch64* | *armv8*)
    is_arch="arm64"
    ;;
*)
    err "此脚本仅支持 64 位系统..."
    ;;
esac

is_core=sing-box
is_core_name=sing-box
is_core_dir="${SINGBOX_DIR:-/etc/$is_core}"
is_core_bin="${SINGBOX_BIN:-$is_core_dir/bin/$is_core}"
if [[ ! -x "$is_core_bin" ]]; then
    _which_bin=$(type -P sing-box 2>/dev/null || true)
    [[ -x "$_which_bin" ]] && is_core_bin="$_which_bin"
fi
is_core_repo=SagerNet/$is_core
is_conf_dir=$is_core_dir/conf
is_log_dir=/var/log/$is_core
is_sh_bin=/usr/local/bin/$is_core
is_sh_dir="${is_sh_dir:-$is_core_dir/sh}"
is_sh_repo=$author/$is_core
is_pkg="wget unzip tar qrencode bash"
is_config_json=$is_core_dir/config.json
is_caddy_bin=/usr/local/bin/caddy
is_caddy_dir=/etc/caddy
is_caddy_repo=caddyserver/caddy
is_caddyfile=$is_caddy_dir/Caddyfile
is_caddy_conf=$is_caddy_dir/$author
is_systemd=$(type -P systemctl)
is_openrc=$(type -P rc-service)
if [[ $is_systemd ]]; then
    is_caddy_service=$(systemctl list-units --full -all | grep caddy.service)
elif [[ $is_openrc ]]; then
    [[ -f /etc/init.d/caddy ]] && is_caddy_service=1
fi
is_http_port=80
is_https_port=443

# core ver
if [[ -x "$is_core_bin" ]]; then
    is_core_ver=$($is_core_bin version 2>/dev/null | head -n1 | cut -d " " -f3)
else
    is_core_ver="not-installed"
fi

# tmp tls key
is_tls_cer=$is_core_dir/bin/tls.cer
is_tls_key=$is_core_dir/bin/tls.key
if [[ -x "$is_core_bin" && (! -f "$is_tls_cer" || ! -f "$is_tls_key") ]]; then
    if mkdir -p "$is_core_dir/bin" 2>/dev/null && [[ -w "$is_core_dir/bin" ]]; then
        is_tls_tmp=${is_tls_key/key/tmp}
        if $is_core_bin generate tls-keypair tls -m 456 >$is_tls_tmp 2>/dev/null; then
            awk '/BEGIN PRIVATE KEY/,/END PRIVATE KEY/' $is_tls_tmp >$is_tls_key 2>/dev/null
            awk '/BEGIN CERTIFICATE/,/END CERTIFICATE/' $is_tls_tmp >$is_tls_cer 2>/dev/null
            rm -f $is_tls_tmp 2>/dev/null
        fi
    fi
fi

if [[ $(pgrep -f $is_core_bin) ]]; then
    is_core_status=$(_green running)
else
    is_core_status=$(_red_bg stopped)
    is_core_stop=1
fi
if [[ -f $is_caddy_bin && -d $is_caddy_dir && $is_caddy_service ]]; then
    is_caddy=1
    if [[ $is_systemd ]]; then
        [[ -f /lib/systemd/system/caddy.service && ! $(grep '\-\-adapter caddyfile' /lib/systemd/system/caddy.service) ]] && {
            load systemd.sh
            install_service caddy
            systemctl restart caddy &
        }
    fi
    is_caddy_ver=$($is_caddy_bin version | head -n1 | cut -d " " -f1)
    is_tmp_http_port=$(grep -E '^ {2,}http_port|^http_port' $is_caddyfile | grep -E -o [0-9]+)
    is_tmp_https_port=$(grep -E '^ {2,}https_port|^https_port' $is_caddyfile | grep -E -o [0-9]+)
    [[ $is_tmp_http_port ]] && is_http_port=$is_tmp_http_port
    [[ $is_tmp_https_port ]] && is_https_port=$is_tmp_https_port
    if [[ $(pgrep -f $is_caddy_bin) ]]; then
        is_caddy_status=$(_green running)
    else
        is_caddy_status=$(_red_bg stopped)
        is_caddy_stop=1
    fi
fi

load core.sh
[[ ! $args ]] && args=main
main $args
