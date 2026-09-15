package nodes

import (
	"testing"
)

func TestResidentialDetector_CDNBlacklist(t *testing.T) {
	d := NewResidentialDetector()

	testCases := []struct {
		name     string
		ip       string
		asn      int
		isp      string
		rdns     string
		wantType NodeType
		wantConf int
	}{
		{
			name:     "Cloudflare CDN",
			ip:       "104.16.0.1",
			asn:      13335,
			isp:      "Cloudflare Inc",
			rdns:     "",
			wantType: TypeDatacenter,
			wantConf: 100,
		},
		{
			name:     "台湾中华电信（家宽白名单）",
			ip:       "1.34.0.1",
			asn:      3462,
			isp:      "Chunghwa Telecom",
			rdns:     "1-34-0-1.hinet-ip.hinet.net",
			wantType: TypeResidential,
			wantConf: 90,
		},
		{
			name:     "AWS EC2（云厂商黑名单）",
			ip:       "18.220.0.1",
			asn:      16509,
			isp:      "Amazon.com Inc",
			rdns:     "ec2-18-220-0-1.us-east-2.compute.amazonaws.com",
			wantType: TypeDatacenter,
			wantConf: 90,
		},
		{
			name:     "日本 NTT OCN（家宽白名单）",
			ip:       "220.152.0.1",
			asn:      4713,
			isp:      "NTT Communications",
			rdns:     "220-152-0-1.osaka.home.ne.jp",
			wantType: TypeResidential,
			wantConf: 90,
		},
		{
			name:     "DigitalOcean VPS（黑名单）",
			ip:       "167.99.0.1",
			asn:      14061,
			isp:      "DigitalOcean LLC",
			rdns:     "",
			wantType: TypeDatacenter,
			wantConf: 90,
		},
		{
			name:     "美国 Comcast 家宽（白名单）",
			ip:       "73.0.0.1",
			asn:      7922,
			isp:      "Comcast Cable Communications",
			rdns:     "c-73-0-0-1.hsd1.ca.comcast.net",
			wantType: TypeResidential,
			wantConf: 90,
		},
		{
			name:     "ISP 名称包含 broadband（家宽关键词）",
			ip:       "100.0.0.1",
			asn:      0,
			isp:      "Generic Broadband Services",
			rdns:     "",
			wantType: TypeResidential,
			wantConf: 70,
		},
		{
			name:     "ISP 名称包含 hosting（机房关键词）",
			ip:       "200.0.0.1",
			asn:      0,
			isp:      "Example Hosting Solutions",
			rdns:     "",
			wantType: TypeDatacenter,
			wantConf: 75,
		},
		{
			name:     "rDNS 包含 dynamic（家宽特征）",
			ip:       "50.0.0.1",
			asn:      0,
			isp:      "Unknown ISP",
			rdns:     "dynamic-50-0-0-1.example.net",
			wantType: TypeResidential,
			wantConf: 65,
		},
		{
			name:     "rDNS 包含 server（机房特征）",
			ip:       "60.0.0.1",
			asn:      0,
			isp:      "Unknown ISP",
			rdns:     "server-60-0-0-1.example.com",
			wantType: TypeDatacenter,
			wantConf: 60,
		},
		{
			name:     "信息不足（未知）",
			ip:       "1.2.3.4",
			asn:      0,
			isp:      "",
			rdns:     "",
			wantType: TypeUnknown,
			wantConf: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := d.Classify(tc.ip, tc.asn, tc.isp, tc.rdns)

			if result.Type != tc.wantType {
				t.Errorf("Type mismatch: got %v, want %v", result.Type, tc.wantType)
			}

			if result.Confidence != tc.wantConf {
				t.Errorf("Confidence mismatch: got %d, want %d", result.Confidence, tc.wantConf)
			}

			t.Logf("✓ [%s] Type=%s Confidence=%d Reason=%s",
				tc.name, result.Type, result.Confidence, result.Reason)
		})
	}
}

func TestResidentialDetector_GlobalISPs(t *testing.T) {
	d := NewResidentialDetector()

	// 测试全球主流运营商 ASN 白名单
	testCases := []struct {
		name string
		asn  int
		want NodeType
	}{
		{"台湾中华电信", 3462, TypeResidential},
		{"日本 KDDI", 2516, TypeResidential},
		{"韩国 KT", 4766, TypeResidential},
		{"美国 Comcast", 7922, TypeResidential},
		{"英国 BT", 2856, TypeResidential},
		{"德国 Deutsche Telekom", 3320, TypeResidential},
		{"澳大利亚 Telstra", 1221, TypeResidential},
		{"Cloudflare (黑名单)", 13335, TypeDatacenter},
		{"AWS (黑名单)", 16509, TypeDatacenter},
		{"Google Cloud (黑名单)", 15169, TypeDatacenter},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := d.Classify("1.2.3.4", tc.asn, "", "")
			if result.Type != tc.want {
				t.Errorf("ASN %d: got %v, want %v", tc.asn, result.Type, tc.want)
			}
		})
	}
}

func TestResidentialDetector_KeywordPriority(t *testing.T) {
	d := NewResidentialDetector()

	// 测试机房关键词优先级高于家宽关键词（避免误判）
	result := d.Classify("1.2.3.4", 0, "Broadband Hosting Services", "")
	if result.Type != TypeDatacenter {
		t.Errorf("Expected datacenter for mixed keywords, got %v", result.Type)
	}

	t.Logf("✓ 机房关键词优先级验证通过: Type=%s Confidence=%d", result.Type, result.Confidence)
}
