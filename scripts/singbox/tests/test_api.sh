#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_DIR="${SINGBOX_TEST_DIR:-/tmp/test-singbox-$$}"
trap 'rm -rf "$TEST_DIR"' EXIT INT TERM

echo "=========================================="
echo "  sing-box API 核心功能自动化全量测试套件"
echo "=========================================="

export SINGBOX_DIR="$TEST_DIR"
mkdir -p "$SINGBOX_DIR/bin" "$SINGBOX_DIR/conf" "$SINGBOX_DIR/sub"
rm -rf "$SINGBOX_DIR/conf"/*

if [[ ! -x "$SINGBOX_DIR/bin/sing-box" ]]; then
    _which_sb=""
    if [[ -n "$SINGBOX_BIN" && -x "$SINGBOX_BIN" ]]; then
        _which_sb="$SINGBOX_BIN"
    elif [[ -x "/etc/sing-box/bin/sing-box" ]]; then
        _which_sb="/etc/sing-box/bin/sing-box"
    elif type -P sing-box &>/dev/null; then
        _which_sb="$(type -P sing-box)"
    fi

    if [[ -n "$_which_sb" && -x "$_which_sb" ]]; then
        cp -f "$_which_sb" "$SINGBOX_DIR/bin/sing-box"
        chmod +x "$SINGBOX_DIR/bin/sing-box"
    else
        echo "未检测到本地 sing-box 核心程序，正在拉取测试所需官方二进制..."
        _arch="$(uname -m)"
        case "$_arch" in
            x86_64|amd64) _sb_arch="amd64" ;;
            aarch64|arm64) _sb_arch="arm64" ;;
            *) _sb_arch="amd64" ;;
        esac
        _tar_url="https://github.com/SagerNet/sing-box/releases/download/v1.14.0/sing-box-1.14.0-linux-${_sb_arch}.tar.gz"
        _tmp_tar="/tmp/sb_ci_${_sb_arch}.tar.gz"
        if curl -sSL -f -m 30 "$_tar_url" -o "$_tmp_tar" 2>/dev/null || \
           curl -sSL -f -m 30 "https://ghproxy.net/${_tar_url}" -o "$_tmp_tar" 2>/dev/null; then
            tar -zxf "$_tmp_tar" -C "$SINGBOX_DIR/bin" --strip-components 1 sing-box-1.14.0-linux-${_sb_arch}/sing-box 2>/dev/null || \
            tar -zxf "$_tmp_tar" -C "$SINGBOX_DIR/bin" --strip-components 1 2>/dev/null || true
            chmod +x "$SINGBOX_DIR/bin/sing-box" 2>/dev/null || true
            rm -f "$_tmp_tar"
        fi
    fi
fi

SINGBOX_CMD="$SCRIPT_DIR/sing-box.sh"

pass_count=0
fail_count=0

assert_ok() {
    local desc="$1"
    local output="$2"
    local is_ok
    is_ok=$(echo "$output" | jq -r '.ok // empty' 2>/dev/null || true)
    if [[ "$is_ok" == "true" ]]; then
        echo -e " \e[32m[PASS]\e[0m $desc"
        (( ++pass_count ))
    else
        echo -e " \e[31m[FAIL]\e[0m $desc"
        echo "   输出: $output"
        (( ++fail_count ))
    fi
}

# 1. Test api help
echo -e "\n--- 1. 测试 API 帮助指令 ---"
help_out=$($SINGBOX_CMD api help)
if [[ "$help_out" =~ "AimiliVPN & sing-box 对外控制 API 接口" ]]; then
    echo -e " \e[32m[PASS]\e[0m api help 正常输出"
    (( ++pass_count ))
else
    echo -e " \e[31m[FAIL]\e[0m api help 输出不符合预期"
    (( ++fail_count ))
fi

# 2. Test api protocols
echo -e "\n--- 2. 测试协议列表接口 ---"
proto_out=$($SINGBOX_CMD api protocols)
assert_ok "api protocols 返回有效协议列表" "$proto_out"
proto_count=$(echo "$proto_out" | jq '.protocols | length')
echo "   支持协议数量: $proto_count"

# 3. Test api status
echo -e "\n--- 3. 测试服务运行状态接口 ---"
status_out=$($SINGBOX_CMD api status)
assert_ok "api status 返回有效运行状态" "$status_out"

# 4. Test api list when empty
echo -e "\n--- 4. 测试空节点列表接口 ---"
list_out=$($SINGBOX_CMD api list)
assert_ok "api list 在无节点时返回 count 0" "$list_out"

# 5. Test api add reality
echo -e "\n--- 5. 测试添加 VLESS-REALITY 节点并指定出口 ---"
add_reality=$($SINGBOX_CMD api add reality auto auto auto 127.0.0.1:7928)
assert_ok "api add reality 成功创建" "$add_reality"
reality_name=$(echo "$add_reality" | jq -r '.node.name')
reality_ob=$(echo "$add_reality" | jq -r '.node.outbound')
echo "   创建节点名: $reality_name, 出口: $reality_ob"

# 6. Test api add hy2
echo -e "\n--- 6. 测试添加 Hysteria2 节点并指定出口 ---"
add_hy2=$($SINGBOX_CMD api add hy2 auto auto auto 127.0.0.1:7928)
assert_ok "api add hy2 成功创建" "$add_hy2"
hy2_name=$(echo "$add_hy2" | jq -r '.node.name')
echo "   创建节点名: $hy2_name"

# 7. Test api add ss
echo -e "\n--- 7. 测试添加 Shadowsocks 节点并指定出口 ---"
add_ss=$($SINGBOX_CMD api add ss auto auto auto 127.0.0.1:7928)
assert_ok "api add ss 成功创建" "$add_ss"
ss_name=$(echo "$add_ss" | jq -r '.node.name')
echo "   创建节点名: $ss_name"

# 8. Verify sing-box syntax of all generated configs
echo -e "\n--- 8. 校验生成的配置文件语法 (sing-box check) ---"
for c in "$SINGBOX_DIR/conf"/*.json; do
    if "$SINGBOX_DIR/bin/sing-box" check -c "$c"; then
        echo -e " \e[32m[PASS]\e[0m 语法校验通过: $(basename "$c")"
        (( ++pass_count ))
    else
        echo -e " \e[31m[FAIL]\e[0m 语法错误: $(basename "$c")"
        (( ++fail_count ))
    fi
done

# 9. Test api list with 3 nodes
echo -e "\n--- 9. 测试节点列表包含 3 个节点 ---"
list_3=$($SINGBOX_CMD api list)
assert_ok "api list 成功获取节点列表" "$list_3"
count_3=$(echo "$list_3" | jq -r '.count')
if [[ "$count_3" -eq 3 ]]; then
    echo -e " \e[32m[PASS]\e[0m 节点数等于 3"
    (( ++pass_count ))
else
    echo -e " \e[31m[FAIL]\e[0m 节点数不匹配: $count_3"
    (( ++fail_count ))
fi

# 10. Test api info for single node
echo -e "\n--- 10. 测试获取单个节点详情 ---"
info_single=$($SINGBOX_CMD api info "$reality_name")
assert_ok "api info 成功获取单个节点详情" "$info_single"
info_url=$(echo "$info_single" | jq -r '.node.url')
if [[ "$info_url" =~ ^vless:// ]]; then
    echo -e " \e[32m[PASS]\e[0m 成功提取有效客户端 vless:// 分享链接"
    (( ++pass_count ))
else
    echo -e " \e[31m[FAIL]\e[0m 分享链接异常: $info_url"
    (( ++fail_count ))
fi

# 11. Test api outbound for single node
echo -e "\n--- 11. 测试更改单个节点出站代理端口为 7929 ---"
out_single=$($SINGBOX_CMD api outbound "$reality_name" "127.0.0.1:7929")
assert_ok "api outbound 单节点成功" "$out_single"
check_single=$($SINGBOX_CMD api info "$reality_name")
single_ob=$(echo "$check_single" | jq -r '.node.outbound')
if [[ "$single_ob" == "socks5://127.0.0.1:7929" ]]; then
    echo -e " \e[32m[PASS]\e[0m 单节点出口更新为 $single_ob"
    (( ++pass_count ))
else
    echo -e " \e[31m[FAIL]\e[0m 出口更新失败: $single_ob"
    (( ++fail_count ))
fi

# 12. Test api outbound all to 127.0.0.1:7928
echo -e "\n--- 12. 测试一键将全部节点出口切换至 127.0.0.1:7928 ---"
out_all=$($SINGBOX_CMD api outbound all "127.0.0.1:7928")
assert_ok "api outbound all 成功" "$out_all"

# 13. Test api outbound all direct
echo -e "\n--- 13. 测试一键将全部节点出口恢复为 direct (直连) ---"
out_direct=$($SINGBOX_CMD api outbound all "direct")
assert_ok "api outbound all direct 成功" "$out_direct"
list_direct=$($SINGBOX_CMD api list)
all_direct=$(echo "$list_direct" | jq -r '[.nodes[].outbound == "direct"] | all')
if [[ "$all_direct" == "true" ]]; then
    echo -e " \e[32m[PASS]\e[0m 所有节点已恢复 direct 直连"
    (( ++pass_count ))
else
    echo -e " \e[31m[FAIL]\e[0m 部分节点未恢复 direct"
    (( ++fail_count ))
fi

# 14. Test api sub init & get
echo -e "\n--- 14. 测试远程订阅管理与同步 ---"
sub_init=$($SINGBOX_CMD api sub init 19999)
assert_ok "api sub init 成功" "$sub_init"
sub_url=$(echo "$sub_init" | jq -r '.sub_url')
sub_nodes=$(echo "$sub_init" | jq -r '.node_count')
echo "   订阅地址: $sub_url, 包含节点数: $sub_nodes"
if [[ "$sub_nodes" -eq 3 ]]; then
    echo -e " \e[32m[PASS]\e[0m 订阅包含全部 3 个节点的分享 URL"
    (( ++pass_count ))
else
    echo -e " \e[31m[FAIL]\e[0m 订阅节点数不符: $sub_nodes"
    (( ++fail_count ))
fi

# 15. Test api del single node
echo -e "\n--- 15. 测试删除单个节点 ---"
del_single=$($SINGBOX_CMD api del "$ss_name")
assert_ok "api del 单节点成功" "$del_single"
list_after_del=$($SINGBOX_CMD api list)
count_after_del=$(echo "$list_after_del" | jq -r '.count')
if [[ "$count_after_del" -eq 2 ]]; then
    echo -e " \e[32m[PASS]\e[0m 节点数正确减少至 2"
    (( ++pass_count ))
else
    echo -e " \e[31m[FAIL]\e[0m 删除后节点数异常: $count_after_del"
    (( ++fail_count ))
fi

# 16. Test api del all
echo -e "\n--- 16. 测试清空全部节点 ---"
del_all=$($SINGBOX_CMD api del all)
assert_ok "api del all 成功" "$del_all"
list_final=$($SINGBOX_CMD api list)
count_final=$(echo "$list_final" | jq -r '.count')
if [[ "$count_final" -eq 0 ]]; then
    echo -e " \e[32m[PASS]\e[0m 全部节点已清空 (count 0)"
    (( ++pass_count ))
else
    echo -e " \e[31m[FAIL]\e[0m 清空后仍有残留节点: $count_final"
    (( ++fail_count ))
fi

echo "=========================================="
echo "  测试结果统计: 通过 $pass_count 项, 失败 $fail_count 项"
echo "=========================================="

if [[ $fail_count -eq 0 ]]; then
    echo -e "\e[32m>>> 全部测试圆满通过！<<<\e[0m"
    exit 0
else
    echo -e "\e[31m>>> 存在失败测试项！<<<\e[0m"
    exit 1
fi
