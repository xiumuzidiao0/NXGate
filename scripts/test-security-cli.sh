#!/usr/bin/env bash
# Local-only regression for installer config helpers. Does not execute installer/systemctl/network calls.
set -euo pipefail
ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
command -v jq >/dev/null
source <(sed -n '/^settings_json_key()/,/^# 11/{ /^# 11/d; p; }' "$ROOT/install.sh")
INSTALL_DIR=$(mktemp -d)
trap 'rm -rf "$INSTALL_DIR"' EXIT
CONFIG_FILE="$INSTALL_DIR/config.env"
mkdir "$INSTALL_DIR/data"
printf 'DATA_DIR=%s/data\nUI_PASSWORD=old-env\nUI_PORT=8787\n' "$INSTALL_DIR" > "$CONFIG_FILE"
set_config_val UI_PASSWORD 'bootstrap|with&characters'
test "$(get_config_val UI_PASSWORD)" = 'bootstrap|with&characters'
printf '%s' '{"ui_username":"admin","ui_password":"new-saved","ui_path":"secret","ui_port":8787,"proxy_port":7928,"age_encrypt_enabled":true}' > "$INSTALL_DIR/data/settings.json"
test "$(get_config_val UI_PASSWORD)" = new-saved
set_config_val UI_PASSWORD 'new-password|with&characters"and\slash'
test "$(get_config_val UI_PASSWORD)" = 'new-password|with&characters"and\slash'
set_config_val UI_PORT 8788
test "$(jq -r '.ui_port|type' "$INSTALL_DIR/data/settings.json")" = number
test "$(get_config_val UI_PORT)" = 8788
test "$(jq -r '.age_encrypt_enabled' "$INSTALL_DIR/data/settings.json")" = true
test "$(stat -c %a "$INSTALL_DIR/data/settings.json")" = 600
printf 'broken' > "$INSTALL_DIR/data/settings.json"
if set_config_val UI_PASSWORD 'must-not-succeed' 2>/dev/null; then echo 'corrupt settings accepted' >&2; exit 1; fi
test "$(cat "$INSTALL_DIR/data/settings.json")" = broken
echo 'PASS: CLI initialization, JSON precedence, password round-trip, numeric port, unrelated values, permissions, corrupt-file preservation'
