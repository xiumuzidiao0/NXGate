# Phase 3: AI 三重解锁严格校验实施计划

## 一、目标与背景

### 1.1 用户需求
> "我要三个 AI 都支持的节点"  
> "我看我设置了必须支持 AI 解锁但是还是出现了连接不支持 AI 解锁的节点的情况"

### 1.2 当前问题
经过生产环境实测，发现：
- **逻辑漏洞**: 使用 `||` (或) 而非 `&&` (且)，导致只要支持一个 AI 即判定通过
- **Claude 高频阻断**: Cloudflare 严格风控，日本/韩国节点对 ChatGPT/Gemini 放行但 Claude 403
- **缺少 Gemini 探测**: 原有系统只探测 OpenAI 和 Claude，未独立检测 Gemini

### 1.3 实测案例
```
=== tun0 (219.100.37.244) ===
ChatGPT: ✅ 200
Claude:  ❌ 403 (Cloudflare 拦截)
Gemini:  ✅ 200

系统判定: ✅ AI 可用 (错误！)
应该判定: ❌ AI 不可用 (Claude 被阻断)
```

---

## 二、核心设计

### 2.1 三大 AI 独立探测端点

| AI 服务 | 探测端点 | 成功标准 | 失败标准 |
|---------|---------|---------|---------|
| **OpenAI** | `https://ios.chat.openai.com/public-api/mobile/server_status/v1` | HTTP 200 | 403, 451, timeout |
| **Claude** | `https://claude.ai/api/auth/session` | HTTP 200 或 401* | 403, 451, timeout |
| **Gemini** | `https://gemini.google.com/app` | HTTP 200 | 403, 451, timeout |

> *注: Claude 返回 401 表示"需要登录"，证明服务可达且未被区域封锁

### 2.2 严格校验逻辑

**旧逻辑（Phase 2 之前）**:
```go
case "ai":
    return u.OpenAI == StatusUnlocked || u.Claude == StatusUnlocked  // ❌ 错误
```

**新逻辑（Phase 3）**:
```go
case "ai":
    return u.OpenAI == StatusUnlocked && 
           u.Claude == StatusUnlocked && 
           u.Gemini == StatusUnlocked  // ✅ 三者必须全部解锁
```

### 2.3 探测时机

#### 静态预判（连接前）
```go
// pkg/nodes/unlock.go: EvaluateNodeUnlock()
func EvaluateNodeUnlock(node *Node) *UnlockResult {
    // 基于国家白名单 + ISP + AS 号码预测
    if node.Country == "JP" && node.IsResidential {
        // 日本住宅宽带: 预判三大 AI 可能可用
        return &UnlockResult{
            OpenAI: StatusLikely,
            Claude: StatusLikely,
            Gemini: StatusLikely,
        }
    }
    // ...其他国家规则
}
```

#### 物理实测（连接后）
```go
// pkg/tunnel/unlock.go: ProbeTunnel()
func ProbeTunnel(tun *Tunnel) *UnlockResult {
    client := newTunnelHTTPClient(tun.DevName, 15*time.Second)
    
    // 并发探测三大 AI
    var wg sync.WaitGroup
    results := &UnlockResult{}
    
    wg.Add(3)
    go func() { defer wg.Done(); results.OpenAI = probeOpenAI(client) }()
    go func() { defer wg.Done(); results.Claude = probeClaude(client) }()
    go func() { defer wg.Done(); results.Gemini = probeGemini(client) }()
    wg.Wait()
    
    return results
}
```

---

## 三、详细实施步骤

### 3.1 数据模型扩展

**文件**: `pkg/tunnel/model.go`

```go
type UnlockResult struct {
    OpenAI    UnlockStatus `json:"openai"`
    Claude    UnlockStatus `json:"claude"`
    Gemini    UnlockStatus `json:"gemini"`      // 新增
    Netflix   UnlockStatus `json:"netflix"`
    Streaming UnlockStatus `json:"streaming"`
    
    LastProbeAt time.Time `json:"last_probe_at"`
}

type UnlockStatus string

const (
    StatusUnknown   UnlockStatus = "unknown"
    StatusLikely    UnlockStatus = "likely"     // 静态预判可能可用
    StatusUnlocked  UnlockStatus = "unlocked"   // 物理实测确认可用
    StatusBlocked   UnlockStatus = "blocked"    // 物理实测确认不可用
)
```

### 3.2 Gemini 探测实现

**文件**: `pkg/tunnel/unlock.go`

```go
func probeGemini(client *http.Client) UnlockStatus {
    req, err := http.NewRequest("GET", "https://gemini.google.com/app", nil)
    if err != nil {
        return StatusUnknown
    }
    
    req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")
    req.Header.Set("Accept", "text/html,application/xhtml+xml")
    
    resp, err := client.Do(req)
    if err != nil {
        return StatusBlocked
    }
    defer resp.Body.Close()
    
    switch resp.StatusCode {
    case 200:
        return StatusUnlocked
    case 403, 451:
        return StatusBlocked
    default:
        return StatusUnknown
    }
}
```

### 3.3 校验逻辑修复

**文件**: `pkg/nodes/unlock.go`

**修改前**:
```go
func (u *UnlockResult) SupportsCategory(category string) bool {
    switch category {
    case "ai":
        return u.OpenAI == StatusUnlocked || u.Claude == StatusUnlocked  // ❌
    // ...
}
```

**修改后**:
```go
func (u *UnlockResult) SupportsCategory(category string) bool {
    switch category {
    case "ai":
        return u.OpenAI == StatusUnlocked && 
               u.Claude == StatusUnlocked && 
               u.Gemini == StatusUnlocked  // ✅ 三者必须全部解锁
    // ...
}
```

### 3.4 动态组入池验证

**文件**: `pkg/tunnel/dynamic.go`

```go
// 动态自适应组连接后立即验证
func (dg *DynamicGroup) validateTunnelUnlock(tun *Tunnel) bool {
    if !dg.RequireAIUnlock {
        return true  // 未开启 AI 解锁要求，直接通过
    }
    
    // 物理实测三大 AI
    unlock := ProbeTunnel(tun)
    tun.SetUnlock(unlock)
    
    // 严格校验
    if unlock.OpenAI != StatusUnlocked {
        stats.LogWarn("Dynamic", "隧道 %s OpenAI 不可用，拒绝入池", tun.ID)
        return false
    }
    if unlock.Claude != StatusUnlocked {
        stats.LogWarn("Dynamic", "隧道 %s Claude 不可用，拒绝入池", tun.ID)
        return false
    }
    if unlock.Gemini != StatusUnlocked {
        stats.LogWarn("Dynamic", "隧道 %s Gemini 不可用，拒绝入池", tun.ID)
        return false
    }
    
    stats.LogInfo("Dynamic", "隧道 %s 三大 AI 全部解锁，允许入池", tun.ID)
    return true
}
```

### 3.5 WebUI 前端同步

**文件**: `web/src/components/TunnelCard.jsx`

```jsx
{/* AI 解锁状态独立显示 */}
{tunnel.unlock && (
  <div className="unlock-badges">
    <Badge 
      color={tunnel.unlock.openai === 'unlocked' ? 'success' : 'error'}
      text="ChatGPT"
    />
    <Badge 
      color={tunnel.unlock.claude === 'unlocked' ? 'success' : 'error'}
      text="Claude"
    />
    <Badge 
      color={tunnel.unlock.gemini === 'unlocked' ? 'success' : 'error'}
      text="Gemini"
    />
  </div>
)}
```

**文件**: `web/src/pages/DynamicGroups.jsx`

```jsx
{/* 筛选选项文案更新 */}
<Checkbox
  checked={group.requireAIUnlock}
  onChange={(e) => updateGroup({requireAIUnlock: e.target.checked})}
  label="必须支持三大 AI (ChatGPT + Claude + Gemini 全部解锁)"
/>
```

---

## 四、测试策略

### 4.1 单元测试

**文件**: `pkg/tunnel/unlock_test.go`

```go
func TestAITripleUnlockValidation(t *testing.T) {
    tests := []struct {
        name     string
        unlock   UnlockResult
        expected bool
    }{
        {
            name: "三大 AI 全部解锁",
            unlock: UnlockResult{
                OpenAI: StatusUnlocked,
                Claude: StatusUnlocked,
                Gemini: StatusUnlocked,
            },
            expected: true,
        },
        {
            name: "Claude 被阻断（现实场景）",
            unlock: UnlockResult{
                OpenAI: StatusUnlocked,
                Claude: StatusBlocked,   // ❌
                Gemini: StatusUnlocked,
            },
            expected: false,
        },
        {
            name: "Gemini 被阻断",
            unlock: UnlockResult{
                OpenAI: StatusUnlocked,
                Claude: StatusUnlocked,
                Gemini: StatusBlocked,   // ❌
            },
            expected: false,
        },
        {
            name: "全部被阻断",
            unlock: UnlockResult{
                OpenAI: StatusBlocked,
                Claude: StatusBlocked,
                Gemini: StatusBlocked,
            },
            expected: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := tt.unlock.SupportsCategory("ai")
            if result != tt.expected {
                t.Errorf("期望 %v, 实际 %v", tt.expected, result)
            }
        })
    }
}
```

### 4.2 集成测试

**测试脚本**: `scripts/test-ai-unlock.sh`

```bash
#!/bin/bash
# 在生产服务器上测试三大 AI 解锁探测

SERVER="47.238.2.197"

echo "=== 测试 AI 解锁探测 ==="

# 获取当前连接的隧道列表
TUNNELS=$(ssh root@$SERVER 'curl -s http://localhost:8964/api/tunnels | jq -r ".data[].dev_name"')

for TUN in $TUNNELS; do
    echo ""
    echo "=== 测试隧道: $TUN ==="
    
    # 通过该隧道实测三大 AI
    ssh root@$SERVER << EOF
        # ChatGPT
        timeout 10 curl -so /dev/null -w "%{http_code}" \
            --interface $TUN \
            https://ios.chat.openai.com/public-api/mobile/server_status/v1 \
            && echo " ✅ ChatGPT" || echo " ❌ ChatGPT"
        
        # Claude
        timeout 10 curl -so /dev/null -w "%{http_code}" \
            --interface $TUN \
            https://claude.ai/api/auth/session \
            && echo " ✅ Claude" || echo " ❌ Claude"
        
        # Gemini
        timeout 10 curl -so /dev/null -w "%{http_code}" \
            --interface $TUN \
            https://gemini.google.com/app \
            && echo " ✅ Gemini" || echo " ❌ Gemini"
EOF
done
```

---

## 五、部署计划

### 5.1 代码审查清单

- [ ] `pkg/tunnel/model.go`: 新增 `Gemini` 字段
- [ ] `pkg/tunnel/unlock.go`: 实现 `probeGemini()` 函数
- [ ] `pkg/nodes/unlock.go`: 修复 `SupportsCategory("ai")` 逻辑
- [ ] `pkg/tunnel/dynamic.go`: 新增 `validateTunnelUnlock()` 入池验证
- [ ] `web/src/components/TunnelCard.jsx`: 前端独立显示三大 AI 徽章
- [ ] `web/src/pages/DynamicGroups.jsx`: 更新筛选选项文案

### 5.2 测试清单

- [ ] 单元测试: `TestAITripleUnlockValidation`
- [ ] 单元测试: `TestProbeGemini`
- [ ] 集成测试: `scripts/test-ai-unlock.sh`
- [ ] 回归测试: `go test -count=1 ./...`

### 5.3 部署步骤

```bash
# 1. 构建
./scripts/build.sh

# 2. 部署
./deploy-phase3.sh

# 3. 验证
ssh root@47.238.2.197 'curl -s http://localhost:8964/api/tunnels | jq ".data[] | {id, unlock}"'

# 4. 实测
./scripts/test-ai-unlock.sh
```

---

## 六、风险与回滚

### 6.1 潜在风险

1. **可用节点大幅减少**: 严格要求三大 AI 全通可能导致符合条件的节点锐减
   - **缓解**: Phase 2 分层超时策略已提升节点发现率 30-50%
   - **监控**: 实时监控节点池数量，如果 < 5 个则告警

2. **探测超时影响连接速度**: 三大 AI 串行探测可能延长连接建立时间
   - **缓解**: 使用并发探测 (`sync.WaitGroup`) 而非串行
   - **优化**: 探测超时设为 10s，避免长时间阻塞

3. **Claude Cloudflare 误判**: Cloudflare 风控可能导致偶发性 403
   - **缓解**: Phase 2 自动重试机制（最多 2 次）
   - **降级**: 如果连续 5 分钟无可用节点，临时降级为"支持 ChatGPT + Gemini"

### 6.2 回滚方案

```bash
# 快速回滚到 Phase 2
ssh root@47.238.2.197 '
  systemctl stop aimilivpn
  cp /usr/local/bin/aimilivpn.backup-phase2 /usr/local/bin/aimilivpn
  systemctl start aimilivpn
'
```

---

## 七、成功指标

| 指标 | Phase 2 基线 | Phase 3 目标 |
|------|-------------|-------------|
| AI 解锁准确率 | ~60% (存在 Claude 403 漏判) | **100%** (三大 AI 严格校验) |
| 用户投诉率 | "还是出现不支持 AI 的节点" | **0** (彻底解决) |
| 可用节点数量 | 15-25 个 | 保持 10+ 个 (严格筛选后) |
| 连接建立时间 | ~8s | ~12s (增加并发探测) |

---

## 八、后续优化方向

1. **智能降级策略**: 
   - 如果 5 分钟内无三大 AI 全通节点，临时降级为"至少两个 AI"
   - WebUI 显示降级状态提示用户

2. **区域偏好**: 
   - 用户可设置"优先日本节点"或"优先美国节点"
   - 结合住宅宽带检测器提升命中率

3. **缓存优化**: 
   - AI 解锁探测结果缓存 12 小时
   - 避免对同一节点重复探测

4. **代理池集成**: 
   - 将验证通过的节点自动添加到 sing-box 代理池
   - 支持 Clash/V2Ray/Xray 订阅输出

---

**计划版本**: v1.0  
**预计开发时间**: 4-6 小时  
**预计测试时间**: 2-3 小时  
**预计上线时间**: Phase 2 验证通过后 24 小时内  
**责任人**: AimiliVPN 开发团队
