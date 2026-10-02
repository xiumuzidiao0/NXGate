#!/bin/bash

args=$@
is_sh_ver=v1.19

# Resolve script directory (following symlinks if any)
_source="${BASH_SOURCE[0]}"
while [ -h "$_source" ]; do
    _dir="$(cd -P "$(dirname "$_source")" && pwd)"
    _source="$(readlink "$_source")"
    [[ $_source != /* ]] && _source="$_dir/$_source"
done
_script_dir="$(cd -P "$(dirname "$_source")" && pwd)"

if [[ -f "$_script_dir/src/init.sh" ]]; then
    is_sh_dir="$_script_dir"
    . "$_script_dir/src/init.sh"
elif [[ -f /etc/sing-box/sh/src/init.sh ]]; then
    is_sh_dir="/etc/sing-box/sh"
    . /etc/sing-box/sh/src/init.sh
fi