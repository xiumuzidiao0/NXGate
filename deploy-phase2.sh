#!/bin/bash
# Phase 2 部署脚本：分层超时策略
# 部署日期: 2026-09-15

set -e

SERVER="47.238.2.197"
BINARY="dist/aimilivpn_linux_amd64"

echo "=== Phase 2: 分层超时策略部署 ==="
echo "-> 上传新版本二进制..."
scp "$BINARY" "root@${SERVER}:/tmp/aimilivpn_new"

echo "-> 停止服务..."
ssh "root@${SERVER}" 'systemctl stop aimilivpn'

echo "-> 备份当前版本..."
ssh "root@${SERVER}" 'cp /usr/local/bin/aimilivpn /usr/local/bin/aimilivpn.backup-$(date +%Y%m%d-%H%M%S)'

echo "-> 替换二进制文件..."
ssh "root@${SERVER}" 'mv /tmp/aimilivpn_new /usr/local/bin/aimilivpn && chmod +x /usr/local/bin/aimilivpn'

echo "-> 启动服务..."
ssh "root@${SERVER}" 'systemctl start aimilivpn'

echo "-> 等待服务就绪..."
sleep 5

echo "-> 检查服务状态..."
ssh "root@${SERVER}" 'systemctl status aimilivpn --no-pager -l'

echo ""
echo "=== 验证分层超时策略 ==="
ssh "root@${SERVER}" << 'VERIFY_EOF'
curl -s http://localhost:8964/api/nodes/pool | jq '{
  total: .data.nodes | length,
  probeConfig: .data.config,
  sampleNode: .data.nodes[0] | {country, latency, unlock}
}'
VERIFY_EOF

echo ""
echo "=== Phase 2 部署完成 ==="
echo "新功能:"
echo "  ✓ 首次探测: TCP 12s, UDP 8s, OpenVPN 35s (宽松超时)"
echo "  ✓ 重试探测: TCP 4s, UDP 2.5s, OpenVPN 15s (严格超时)"
echo "  ✓ 最大重试次数: 2 次"
echo ""
echo "监控命令:"
echo "  journalctl -u aimilivpn -f | grep -E '(探测|超时|重试)'"
