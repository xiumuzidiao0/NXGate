# 家宽/住宅 IP 智能甄选引擎

## 功能概述

基于 `freesub` 项目的五层瀑布式判定机制，新增智能家宽甄选引擎，用于精准识别 VPNGate 志愿者节点的 IP 类型（家庭宽带/机房/移动网络），帮助用户筛选优质家宽节点。

## 核心特性

### 1. 五层瀑布式判定机制（按优先级排序）

| 层级 | 判定依据 | 置信度 | 说明 |
|------|---------|--------|------|
| 1 | **CDN 网段黑名单** | 100% | Cloudflare、Fastly、Akamai 核心 CIDR 段 |
| 2 | **ASN 白名单** | 90% | 182 个全球主流家宽运营商（中华电信、NTT、Comcast 等） |
| 3 | **ASN 黑名单** | 90% | 78 个云厂商/机房（AWS、Azure、Hetzner、阿里云等） |
| 4 | **ISP 名称关键词** | 70-75% | broadband/telecom（家宽）、hosting/cloud（机房） |
| 5 | **rDNS 兜底** | 60-65% | dynamic/dhcp（家宽）、server/vps（机房） |

### 2. 覆盖全球主流运营商

#### 亚洲
- **台湾**：中华电信、台湾大哥大、远传电信、亚太电信、凯擘大宽频
- **日本**：NTT OCN、KDDI、SoftBank、IIJ、So-net、Rakuten
- **韩国**：KT、SK Broadband、LG Uplus
- **香港/新加坡**：PCCW、HKBN、HGC、StarHub、Singtel

#### 欧美
- **美国**：Comcast、Charter (Spectrum)、Cox、AT&T、Verizon
- **英国**：BT、Virgin Media、Sky Broadband、TalkTalk
- **德国**：Deutsche Telekom、Vodafone、O2
- **法国**：Orange、Bouygues、SFR、Free.fr
- **加拿大**：Rogers、Shaw、Bell Canada
- **澳洲**：Telstra、TPG、Optus

### 3. 云厂商黑名单（78 家主流 IDC）

- 美国云：AWS、GCP、Azure、Cloudflare、DigitalOcean、Vultr、Linode
- 欧洲云：Hetzner、OVH、Aruba IT
- 亚洲云：阿里云、腾讯云、中国电信云

## 技术实现

### 文件结构

```
pkg/nodes/
├── residential.go       # 家宽甄选引擎核心逻辑
├── residential_test.go  # 完整单元测试（11 个测试用例）
├── enrich.go           # IP 信息丰富化（集成家宽判定）
└── model.go            # Node 数据模型扩展
```

### 核心 API

```go
// 初始化甄选引擎
detector := NewResidentialDetector()

// 执行五层瀑布式判定
result := detector.Classify(
    ip,    // IP 地址
    asn,   // 自治系统编号
    isp,   // ISP 名称
    rdns,  // 反向 DNS 记录
)

// 返回结果
type Classification struct {
    Type       NodeType  // residential / datacenter / mobile / unknown
    Confidence int       // 置信度 0-100
    Reason     string    // 判定依据（便于调试）
}
```

### Node 数据模型扩展

```go
type Node struct {
    // ... 原有字段 ...
    
    // 新增家宽甄选字段
    IPType        string `json:"ip_type"`        // residential/hosting/mobile/unknown
    ASN           int    `json:"asn"`            // 自治系统编号
    RDNS          string `json:"rdns"`           // 反向DNS记录
    IsResidential bool   `json:"is_residential"` // 家宽标识
    ResConfidence int    `json:"res_confidence"` // 置信度 0-100
    ResReason     string `json:"res_reason"`     // 判定依据
}
```

## 使用场景

### 1. 节点列表接口自动标注

```bash
GET /api/nodes?type=residential
```

返回结果自动包含家宽判定信息：

```json
{
  "id": "vpn123456789",
  "hostname": "vpn123456789.example.net",
  "ip": "220.152.10.50",
  "country_short": "JP",
  "ip_type": "residential",
  "isp": "NTT Communications",
  "asn": 4713,
  "is_residential": true,
  "res_confidence": 90,
  "res_reason": "Residential ISP ASN whitelist"
}
```

### 2. 前端筛选与展示

- 节点列表显示 🏠 图标标识家宽节点
- 支持按 IP 类型筛选：`residential` / `hosting` / `mobile`
- 展示置信度百分比与判定依据

### 3. 自动化爬虫优先级

爬虫后台自动为高置信度家宽节点分配更高的**信誉评分**与**探测频率**。

## 测试验证

### 运行单元测试

```bash
cd /home/xmzd/aimili-vpngate-go
go test -v ./pkg/nodes -run TestResidential
```

### 测试覆盖

- ✅ CDN 网段黑名单（Cloudflare 104.16.0.1）
- ✅ 家宽运营商白名单（台湾中华电信 ASN 3462）
- ✅ 云厂商黑名单（AWS ASN 16509）
- ✅ ISP 名称关键词（broadband、hosting）
- ✅ rDNS 特征（dynamic、server）
- ✅ 关键词优先级（机房关键词 > 家宽关键词）
- ✅ 全球 10 大运营商 ASN 验证

**测试结果：11 个测试用例全部 PASS**

## 判定示例

### 案例 1：台湾中华电信（家宽）

```
IP: 1.34.123.45
ASN: 3462 (Chunghwa Telecom)
ISP: Chunghwa Telecom
rDNS: 1-34-123-45.hinet-ip.hinet.net

判定结果：residential (置信度 90%)
依据：Residential ISP ASN whitelist
```

### 案例 2：AWS EC2（机房）

```
IP: 18.220.15.30
ASN: 16509 (Amazon.com Inc)
ISP: Amazon.com Inc
rDNS: ec2-18-220-15-30.us-east-2.compute.amazonaws.com

判定结果：datacenter (置信度 90%)
依据：Cloud provider ASN blacklist
```

### 案例 3：日本家庭宽带（家宽）

```
IP: 220.152.10.50
ASN: 4713 (NTT OCN)
ISP: NTT Communications
rDNS: 220-152-10-50.osaka.home.ne.jp

判定结果：residential (置信度 90%)
依据：Residential ISP ASN whitelist
```

### 案例 4：信息不足（未知）

```
IP: 1.2.3.4
ASN: 0
ISP: ""
rDNS: ""

判定结果：unknown (置信度 0%)
依据：Insufficient data for classification
```

## 性能优化

1. **内存缓存**：ASN 黑白名单预加载到 map，O(1) 查询
2. **CIDR 索引**：CDN 网段使用 `net.IPNet` 快速匹配
3. **关键词优先级**：机房特征优先判定，避免误报
4. **批量查询**：ip-api.com 支持每次 50 个 IP 批量查询
5. **7 天缓存**：本地 JSON 文件缓存 IP 探测结果

## 后续扩展

- [ ] 添加更多地区运营商（印度、巴西、中东）
- [ ] 集成 MaxMind GeoIP2 ISP 数据库
- [ ] 支持 IPv6 家宽判定
- [ ] 添加家宽节点历史稳定性追踪
- [ ] 前端可视化置信度雷达图

## 参考资料

- [freesub 五层瀑布式判定](https://github.com/username/freesub)
- [ip-api.com 批量查询 API](http://ip-api.com/docs/api:batch)
- [IANA ASN 注册表](https://www.iana.org/assignments/as-numbers/)
