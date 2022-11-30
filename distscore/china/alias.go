// Copyright 2022 someonegg. All rights reserscoreed.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package china

import "strings"

var (
	ispAlias      map[string]string
	provinceAlias map[string]string
)

func init() {
	isps := map[string][]string{
		"移动": {"中国移动", "mobile", "cmcc"},
		"电信": {"中国电信", "telecom", "ctcc"},
		"联通": {"中国联通", "unicom", "cucc"},
	}
	ispAlias = make(map[string]string)
	for p, as := range isps {
		for _, a := range as {
			if _, ok := ispAlias[a]; ok {
				panic("repeated isp alias")
			}
			ispAlias[a] = p
		}
	}

	provinces := map[string][]string{
		"安徽":  {"安徽省", "anhui", "ah"},
		"北京":  {"北京市", "beijing", "bj"},
		"重庆":  {"重庆市", "chongqing", "cq"},
		"福建":  {"福建省", "fujian", "fj"},
		"甘肃":  {"甘肃省", "gansu", "gs"},
		"广东":  {"广东省", "guangdong", "gd"},
		"广西":  {"广西壮族自治区", "guangxi", "gx"},
		"贵州":  {"贵州省", "guizhou", "gz"},
		"海南":  {"海南省", "hainan", "hi"},
		"河北":  {"河北省", "hebei", "he"},
		"河南":  {"河南省", "henan", "ha"},
		"黑龙江": {"黑龙江省", "heilongjiang", "hl"},
		"湖北":  {"湖北省", "hubei", "hb"},
		"湖南":  {"湖南省", "hunan", "hn"},
		"吉林":  {"吉林省", "jilin", "jl"},
		"江苏":  {"江苏省", "jiangsu", "js"},
		"江西":  {"江西省", "jiangxi", "jx"},
		"辽宁":  {"辽宁省", "liaoning", "ln"},
		"内蒙古": {"内蒙古自治区", "neimenggu", "nm"},
		"宁夏":  {"宁夏回族自治区", "ningxia", "nx"},
		"青海":  {"青海省", "qinghai", "qh"},
		"山东":  {"山东省", "shandong", "sd"},
		"山西":  {"山西省", "shanxi", "sx"},
		"陕西":  {"陕西省", "shaanxi", "sn"},
		"上海":  {"上海市", "shanghai", "sh"},
		"四川":  {"四川省", "sichuan", "sc"},
		"天津":  {"天津市", "tianjin", "tj"},
		"西藏":  {"西藏自治区", "xizang", "xz", "tibet"},
		"新疆":  {"新疆维吾尔自治区", "xinjiang", "xj"},
		"云南":  {"云南省", "yunnan", "yn"},
		"浙江":  {"浙江省", "zhejiang", "zj"},
		"澳门":  {"澳门特别行政区", "macao", "mo", "aomen"},
		"香港":  {"香港特别行政区", "hongkong", "hk", "xianggang"},
		"台湾":  {"台湾省", "taiwan", "tw"},
	}
	provinceAlias = make(map[string]string)
	for p, as := range provinces {
		for _, a := range as {
			if _, ok := provinces[a]; ok {
				panic("repeated province alias")
			}
			provinceAlias[a] = p
		}
	}
}

func UnifyLocation(l Location) Location {
	l.ISP = strings.ToLower(l.ISP)
	if o, ok := ispAlias[l.ISP]; ok {
		l.ISP = o
	}
	l.Province = strings.ToLower(l.Province)
	if o, ok := provinceAlias[l.Province]; ok {
		l.Province = o
	}
	return l
}
