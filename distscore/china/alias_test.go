// Copyright 2022 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package china

import (
	"testing"

	. "github.com/someonegg/rsdmatch/distscore"
)

func TestUnifyLocation_ISP(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		want   string
		server bool
		proxy  bool
	}{
		// 中国移动别名
		{"Mobile_CN", "移动", "移动", false, false},
		{"Mobile_Full", "中国移动", "移动", false, false},
		{"Mobile_English", "mobile", "移动", false, false},
		{"Mobile_CMCC", "cmcc", "移动", false, false},
		{"Mobile_Upper", "MOBILE", "移动", false, false}, // ASCII转小写后匹配别名

		// 中国电信别名
		{"Telecom_CN", "电信", "电信", false, false},
		{"Telecom_Full", "中国电信", "电信", false, false},
		{"Telecom_English", "telecom", "电信", false, false},
		{"Telecom_CTCC", "ctcc", "电信", false, false},
		{"Telecom_Upper", "TELECOM", "电信", false, false},

		// 中国联通别名
		{"Unicom_CN", "联通", "联通", false, false},
		{"Unicom_Full", "中国联通", "联通", false, false},
		{"Unicom_English", "unicom", "联通", false, false},
		{"Unicom_CUCC", "cucc", "联通", false, false},
		{"Unicom_Upper", "UNICOM", "联通", false, false},

		// 非别名ISP
		{"Other_ISP", "其他ISP", "其他ISP", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			loc := Location{ISP: tc.input, Province: "北京"}
			got := UnifyLocation(loc, tc.server, tc.proxy)
			if got.ISP != tc.want {
				t.Errorf("UnifyLocation(%v).ISP = %q, want %q", loc, got.ISP, tc.want)
			}
		})
	}
}

func TestUnifyLocation_Province(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		want   string
		server bool
		proxy  bool
	}{
		// 北京
		{"Beijing_CN", "北京", "北京", false, false},
		{"Beijing_Full", "北京市", "北京", false, false},
		{"Beijing_Pinyin", "beijing", "北京", false, false},
		{"Beijing_Code", "bj", "北京", false, false},

		// 上海
		{"Shanghai_CN", "上海", "上海", false, false},
		{"Shanghai_Full", "上海市", "上海", false, false},
		{"Shanghai_Pinyin", "shanghai", "上海", false, false},
		{"Shanghai_Code", "sh", "上海", false, false},

		// 广东
		{"Guangdong_CN", "广东", "广东", false, false},
		{"Guangdong_Full", "广东省", "广东", false, false},
		{"Guangdong_Pinyin", "guangdong", "广东", false, false},
		{"Guangdong_Code", "gd", "广东", false, false},

		// 西藏（有多个别名）
		{"Xizang_CN", "西藏", "西藏", false, false},
		{"Xizang_Full", "西藏自治区", "西藏", false, false},
		{"Xizang_Pinyin", "xizang", "西藏", false, false},
		{"Xizang_Code", "xz", "西藏", false, false},
		{"Xizang_English", "tibet", "西藏", false, false},

		// 新疆
		{"Xinjiang_CN", "新疆", "新疆", false, false},
		{"Xinjiang_Full", "新疆维吾尔自治区", "新疆", false, false},
		{"Xinjiang_Pinyin", "xinjiang", "新疆", false, false},
		{"Xinjiang_Code", "xj", "新疆", false, false},

		// 非别名省份
		{"Other_Province", "其他省份", "其他省份", false, false},

		// 大小写混合（省份也是ASCII转小写）
		{"Mixed_Case", "BEIJING", "北京", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			loc := Location{ISP: "电信", Province: tc.input}
			got := UnifyLocation(loc, tc.server, tc.proxy)
			if got.Province != tc.want {
				t.Errorf("UnifyLocation(%v).Province = %q, want %q", loc, got.Province, tc.want)
			}
		})
	}
}

func TestUnifyLocation_ProxyMunici(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		want   string
		proxy  bool
		server bool
	}{
		// 直辖市代理模式
		{"Beijing_Proxy", "北京", "河北", true, false},
		{"Tianjin_Proxy", "天津", "河北", true, false},
		{"Shanghai_Proxy", "上海", "江苏", true, false},
		{"Chongqing_Proxy", "重庆", "四川", true, false},
		{"Ningxia_Proxy", "宁夏", "甘肃", true, false},

		// 代理模式关闭
		{"Beijing_NoProxy", "北京", "北京", false, false},
		{"Tianjin_NoProxy", "天津", "天津", false, false},
		{"Shanghai_NoProxy", "上海", "上海", false, false},
		{"Chongqing_NoProxy", "重庆", "重庆", false, false},
		{"Ningxia_NoProxy", "宁夏", "宁夏", false, false},

		// 非直辖市不受影响
		{"Normal_Province", "广东", "广东", true, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			loc := Location{ISP: "电信", Province: tc.input}
			got := UnifyLocation(loc, tc.server, tc.proxy)
			if got.Province != tc.want {
				t.Errorf("UnifyLocation(proxy=%v, %+v).Province = %q, want %q",
					tc.proxy, loc, got.Province, tc.want)
			}
		})
	}
}

func TestInNormal(t *testing.T) {
	cases := []struct {
		name     string
		location Location
		want     bool
	}{
		// Normal 省份
		{"Normal_Beijing", Location{ISP: "电信", Province: "北京"}, true},
		{"Normal_Shanghai", Location{ISP: "电信", Province: "上海"}, true},
		{"Normal_Guangdong", Location{ISP: "电信", Province: "广东"}, true},
		{"Normal_Liaoning", Location{ISP: "电信", Province: "辽宁"}, true},
		{"Normal_Shaanxi", Location{ISP: "电信", Province: "陕西"}, true},
		{"Normal_Sichuan", Location{ISP: "电信", Province: "四川"}, true},
		{"Normal_Chongqing", Location{ISP: "电信", Province: "重庆"}, true},
		{"Normal_Guizhou", Location{ISP: "电信", Province: "贵州"}, true},

		// 非Normal（边疆）
		{"Frontier_Xinjiang", Location{ISP: "电信", Province: "新疆"}, false},
		{"Frontier_Xizang", Location{ISP: "电信", Province: "西藏"}, false},

		// 别名应该也能识别
		{"Normal_Alias_bj", Location{ISP: "电信", Province: "bj"}, true},
		{"Normal_Alias_beijing", Location{ISP: "电信", Province: "beijing"}, true},

		// 未知省份
		{"Unknown_Province", Location{ISP: "电信", Province: "未知"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := InNormal(tc.location)
			if got != tc.want {
				t.Errorf("InNormal(%+v) = %v, want %v", tc.location, got, tc.want)
			}
		})
	}
}

func TestInCentral(t *testing.T) {
	cases := []struct {
		name     string
		location Location
		want     bool
	}{
		// Central 省份（中部发达地区）
		{"Central_Beijing", Location{ISP: "电信", Province: "北京"}, true},
		{"Central_Shanghai", Location{ISP: "电信", Province: "上海"}, true},
		{"Central_Guangdong", Location{ISP: "电信", Province: "广东"}, true},
		{"Central_Jiangsu", Location{ISP: "电信", Province: "江苏"}, true},
		{"Central_Hubei", Location{ISP: "电信", Province: "湖北"}, true},
		{"Central_Henan", Location{ISP: "电信", Province: "河南"}, true},

		// 非Central（虽然normal但不是central）
		{"Normal_NotCentral_Liaoning", Location{ISP: "电信", Province: "辽宁"}, false},
		{"Normal_NotCentral_Shaanxi", Location{ISP: "电信", Province: "陕西"}, false},
		{"Normal_NotCentral_Sichuan", Location{ISP: "电信", Province: "四川"}, false},

		// 边疆
		{"Frontier_Xinjiang", Location{ISP: "电信", Province: "新疆"}, false},
		{"Frontier_Xizang", Location{ISP: "电信", Province: "西藏"}, false},

		// 别名
		{"Central_Alias_bj", Location{ISP: "电信", Province: "bj"}, true},
		{"Central_Alias_gd", Location{ISP: "电信", Province: "gd"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := InCentral(tc.location)
			if got != tc.want {
				t.Errorf("InCentral(%+v) = %v, want %v", tc.location, got, tc.want)
			}
		})
	}
}

func TestInFrontier(t *testing.T) {
	cases := []struct {
		name     string
		location Location
		want     bool
	}{
		// 边疆省份
		{"Frontier_Xinjiang", Location{ISP: "电信", Province: "新疆"}, true},
		{"Frontier_Xizang", Location{ISP: "电信", Province: "西藏"}, true},

		// 非边疆
		{"NotFrontier_Beijing", Location{ISP: "电信", Province: "北京"}, false},
		{"NotFrontier_Shanghai", Location{ISP: "电信", Province: "上海"}, false},
		{"NotFrontier_Guangdong", Location{ISP: "电信", Province: "广东"}, false},

		// Normal但非边疆
		{"Normal_NotFrontier_Liaoning", Location{ISP: "电信", Province: "辽宁"}, false},
		{"Normal_NotFrontier_Shaanxi", Location{ISP: "电信", Province: "陕西"}, false},

		// 别名
		{"Frontier_Alias_xj", Location{ISP: "电信", Province: "xj"}, true},
		{"Frontier_Alias_xinjiang", Location{ISP: "电信", Province: "xinjiang"}, true},
		{"Frontier_Alias_xz", Location{ISP: "电信", Province: "xz"}, true},
		{"Frontier_Alias_tibet", Location{ISP: "电信", Province: "tibet"}, true},

		// 未知省份
		{"Unknown_Province", Location{ISP: "电信", Province: "未知"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := InFrontier(tc.location)
			if got != tc.want {
				t.Errorf("InFrontier(%+v) = %v, want %v", tc.location, got, tc.want)
			}
		})
	}
}

func TestNewLocationUnifier(t *testing.T) {
	// 测试代理模式开启
	unifier := NewLocationUnifier(true)
	if unifier == nil {
		t.Fatal("NewLocationUnifier(true) returned nil")
	}

	loc := Location{ISP: "电信", Province: "北京"}
	got := unifier.Unify(loc, false)
	if got.Province != "河北" {
		t.Errorf("LocationUnifier.Unify(%+v, false).Province = %q, want '河北'", loc, got.Province)
	}

	// 测试 IsDeputy
 deputyLoc := Location{ISP: "电信", Province: "北京"}
	if !unifier.IsDeputy(deputyLoc) {
		t.Errorf("LocationUnifier.IsDeputy(%+v) = false, want true", deputyLoc)
	}

	notDeputyLoc := Location{ISP: "电信", Province: "新疆"}
	if unifier.IsDeputy(notDeputyLoc) {
		t.Errorf("LocationUnifier.IsDeputy(%+v) = true, want false", notDeputyLoc)
	}
}

func TestLocationUnifier_Unify(t *testing.T) {
	cases := []struct {
		name   string
		input  Location
		server bool
		proxy  bool
		want   Location
	}{
		// ISP统一 + 省份统一 + 代理
		{
			name:   "Full_Unify",
			input:  Location{ISP: "mobile", Province: "bj"},
			server: false,
			proxy:  true,
			want:   Location{ISP: "移动", Province: "河北"},
		},
		// 仅ISP统一
		{
			name:   "ISP_Only",
			input:  Location{ISP: "telecom", Province: "未知省份"},
			server: false,
			proxy:  false,
			want:   Location{ISP: "电信", Province: "未知省份"},
		},
		// 仅省份统一
		{
			name:   "Province_Only",
			input:  Location{ISP: "未知ISP", Province: "gd"},
			server: false,
			proxy:  false,
			want:   Location{ISP: "未知ISP", Province: "广东"},
		},
		// 不需要统一
		{
			name:   "No_Unify",
			input:  Location{ISP: "电信", Province: "北京"},
			server: false,
			proxy:  false,
			want:   Location{ISP: "电信", Province: "北京"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			unifier := NewLocationUnifier(tc.proxy)
			got := unifier.Unify(tc.input, tc.server)
			if got != tc.want {
				t.Errorf("LocationUnifier.Unify(%+v, %v) = %+v, want %+v",
					tc.input, tc.server, got, tc.want)
			}
		})
	}
}
