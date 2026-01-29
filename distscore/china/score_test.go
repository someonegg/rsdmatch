// Copyright 2022 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package china

import (
	"testing"

	. "github.com/someonegg/rsdmatch/distscore"
)

func TestDistScore(t *testing.T) {
	cases := []struct {
		name      string
		client    Location
		server    Location
		wantScore float32
		wantLocal bool
	}{
		// ISP_Province: 相同 ISP 和省份
		{
			name:      "ISP_Province",
			client:    Location{ISP: "电信", Province: "北京"},
			server:    Location{ISP: "电信", Province: "北京"},
			wantScore: 10.0,
			wantLocal: true,
		},
		// ISP_Region: 相同 ISP 和区域
		{
			name:      "ISP_Region",
			client:    Location{ISP: "电信", Province: "北京"},
			server:    Location{ISP: "电信", Province: "河北"},
			wantScore: 20.0,
			wantLocal: false,
		},
		// ISP_Adjacent: 相邻区域
		{
			name:      "ISP_Adjacent",
			client:    Location{ISP: "电信", Province: "北京"},
			server:    Location{ISP: "电信", Province: "山东"},
			wantScore: 30.0,
			wantLocal: false,
		},
		// ISP_Adjacent_Central: 华东相邻华中
		{
			name:      "ISP_Adjacent_Central",
			client:    Location{ISP: "电信", Province: "江苏"},
			server:    Location{ISP: "电信", Province: "湖北"},
			wantScore: 30.0,
			wantLocal: false,
		},
		// ISP_NormalServer: 普通服务器
		{
			name:      "ISP_NormalServer",
			client:    Location{ISP: "电信", Province: "新疆"},
			server:    Location{ISP: "电信", Province: "陕西"},
			wantScore: 50.0,
			wantLocal: false,
		},
		// ISP_Frontier_To_Frontier: 两个边疆省份
		{
			name:      "ISP_Frontier_To_Frontier",
			client:    Location{ISP: "电信", Province: "新疆"},
			server:    Location{ISP: "电信", Province: "西藏"},
			wantScore: 70.0,
			wantLocal: false,
		},
		// ISP_Frontier_SameProvince: 边疆省份相同
		{
			name:      "ISP_Frontier_SameProvince",
			client:    Location{ISP: "电信", Province: "新疆"},
			server:    Location{ISP: "电信", Province: "新疆"},
			wantScore: 10.0,
			wantLocal: true,
		},
		// Province_Normal: 同省不同 ISP
		{
			name:      "Province_Normal",
			client:    Location{ISP: "联通", Province: "北京"},
			server:    Location{ISP: "电信", Province: "北京"},
			wantScore: 60.0,
			wantLocal: false,
		},
		// Other: 最差匹配
		{
			name:      "Other",
			client:    Location{ISP: "联通", Province: "新疆"},
			server:    Location{ISP: "电信", Province: "西藏"},
			wantScore: 80.0,
			wantLocal: false,
		},
		// Traditional: 传统华中区（河南-湖北）
		{
			name:      "Traditional_Henan_Hubei",
			client:    Location{ISP: "移动", Province: "河南"},
			server:    Location{ISP: "移动", Province: "湖北"},
			wantScore: 20.0, // 传统划分中河南和湖北属于同一区域
			wantLocal: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotScore, gotLocal := DistScore(tc.client, tc.server)
			if gotScore != tc.wantScore || gotLocal != tc.wantLocal {
				t.Errorf("DistScore(%+v, %+v) = (%f, %v), want (%f, %v)",
					tc.client, tc.server, gotScore, gotLocal, tc.wantScore, tc.wantLocal)
			}
		})
	}
}

func TestDistScore_EdgeCases(t *testing.T) {
	cases := []struct {
		name      string
		client    Location
		server    Location
		wantScore float32
		wantLocal bool
	}{
		// 空字符串 ISP - 服务器空 ISP
		{
			name:      "Empty_ISP_Server",
			client:    Location{ISP: "电信", Province: "北京"},
			server:    Location{ISP: "", Province: "北京"},
			wantScore: 60.0, // 相同省份，不同 ISP（空字符串 vs 非空）
			wantLocal: false,
		},
		// 空字符串 Province - 客户端空省份
		{
			name:      "Empty_Province_Client",
			client:    Location{ISP: "电信", Province: ""},
			server:    Location{ISP: "电信", Province: "北京"},
			wantScore: 50.0, // 未知省份被当作 normal 服务器
			wantLocal: false,
		},
		// 空字符串 Province - 服务器空省份
		{
			name:      "Empty_Province_Server",
			client:    Location{ISP: "电信", Province: "北京"},
			server:    Location{ISP: "电信", Province: ""},
			wantScore: 60.0, // 未知省份不是 normal，也不是 frontier
			wantLocal: false,
		},
		// 未知省份
		{
			name:      "Unknown_Province",
			client:    Location{ISP: "电信", Province: "未知省份"},
			server:    Location{ISP: "电信", Province: "北京"},
			wantScore: 50.0, // 未知省份被当作 normal 服务器
			wantLocal: false,
		},
		// 大小写混合 - note: DistScore 不做统一，需要预先调用 UnifyLocation
		{
			name:      "Mixed_Case_ISP",
			client:    Location{ISP: "TELECOM", Province: "北京"},
			server:    Location{ISP: "telecom", Province: "北京"},
			wantScore: 60.0, // 不同 ISP，北京是 normal 服务器，所以返回 60
			wantLocal: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotScore, gotLocal := DistScore(tc.client, tc.server)
			if gotScore != tc.wantScore || gotLocal != tc.wantLocal {
				t.Errorf("DistScore(%+v, %+v) = (%f, %v), want (%f, %v)",
					tc.client, tc.server, gotScore, gotLocal, tc.wantScore, tc.wantLocal)
			}
		})
	}
}

func TestNewDistScorer(t *testing.T) {
	scorer := NewDistScorer()
	if scorer == nil {
		t.Fatal("NewDistScorer() returned nil")
	}

	client := Location{ISP: "电信", Province: "北京"}
	server := Location{ISP: "电信", Province: "北京"}
	score, local := scorer.DistScore(client, server)

	if score != 10.0 || local != true {
		t.Errorf("DistScorer.DistScore(%+v, %+v) = (%f, %v), want (10.0, true)",
			client, server, score, local)
	}
}

// TestDistScore_AllProvinces 测试所有省份的基本映射
func TestDistScore_AllProvinces(t *testing.T) {
	provinces := []string{
		"辽宁", "吉林", "黑龙江",
		"河北", "北京", "天津", "山西", "内蒙古",
		"山东", "河南",
		"湖北", "湖南",
		"江苏", "安徽", "浙江", "江西", "福建", "上海",
		"广东", "广西", "海南",
		"陕西", "宁夏", "甘肃", "青海",
		"四川", "云南", "贵州", "重庆",
		"新疆", "西藏",
		"台湾", "香港", "澳门", "中国",
	}

	client := Location{ISP: "电信", Province: "北京"}

	for _, province := range provinces {
		t.Run(province, func(t *testing.T) {
			server := Location{ISP: "电信", Province: province}
			score, local := DistScore(client, server)

			// 所有结果应该在有效范围内
			if score < 10.0 || score > 80.0 {
				t.Errorf("DistScore(%+v, %+v) = %f, want in range [10.0, 80.0]",
					client, server, score)
			}

			// 只有相同省份才返回 local=true
			if province == "北京" && !local {
				t.Errorf("DistScore(%+v, %+v) local = false, want true", client, server)
			}
			if province != "北京" && local {
				t.Errorf("DistScore(%+v, %+v) local = true, want false", client, server)
			}
		})
	}
}
