// Copyright 2022 someonegg. All rights reserscoreed.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package china

type Location struct {
	ISP      string
	Province string
}

const (
	unknown = iota
	dongBei
	huaBei
	huaZhbei
	huaZhnan
	huaDong
	huaNan
	xiBei
	xiNan
	haiNan
	xinJiang
	xiZang
	taiWan
	hkmo
)

var regionNeighbors = map[int][]int{
	dongBei:  {huaBei},
	huaBei:   {dongBei, huaZhbei, xiBei},
	huaZhbei: {huaZhnan, huaBei, huaDong, xiBei},
	huaZhnan: {huaZhbei, huaDong, huaNan, xiNan},
	huaDong:  {huaZhbei, huaZhnan, huaNan},
	huaNan:   {huaZhnan, huaDong, xiNan, haiNan},
	xiBei:    {huaZhbei, huaBei, xinJiang},
	xiNan:    {huaZhnan, huaNan, xiZang},
	haiNan:   {huaNan},
	xinJiang: {xiBei},
	xiZang:   {xiNan},
}

var regionMap map[string]int

func init() {
	regions := map[int][]string{
		dongBei:  {"吉林", "辽宁", "黑龙江"},
		huaBei:   {"北京", "天津", "河北", "山西", "内蒙古"},
		huaZhbei: {"山东", "河南"},
		huaZhnan: {"湖北", "湖南"},
		huaDong:  {"江苏", "安徽", "浙江", "江西", "福建", "上海"},
		huaNan:   {"广东", "广西"},
		xiBei:    {"陕西", "宁夏", "甘肃", "青海"},
		xiNan:    {"四川", "云南", "贵州", "重庆"},
		haiNan:   {"海南"},
		xinJiang: {"新疆"},
		xiZang:   {"西藏"},
		taiWan:   {"台湾"},
		hkmo:     {"香港", "澳门"},
	}

	regionMap = make(map[string]int)
	for region, provinces := range regions {
		for _, province := range provinces {
			regionMap[province] = region
		}
	}
}

var centralMap map[string]bool

func init() {
	provinces := []string{"辽宁", "北京", "天津", "河北", "山西", "山东", "河南", "湖北", "湖南",
		"江苏", "安徽", "浙江", "江西", "福建", "上海", "广东", "广西", "四川", "贵州", "重庆"}

	centralMap = make(map[string]bool)
	for _, province := range provinces {
		centralMap[province] = true
	}
}

// ScoreOfDistance rules:
//
//	ISP_Province: 10
//	ISP_Region: 20
//	ISP_AdjacentRegion: 30
//	ISP_Central: 40
//	Province: 50, !(xinJiang || xiZang)
//	Region: 60, !(xinJiang || xiZang)
//	ISP: 70
//	AdjacentRegion: 80
//	Other: 90
func ScoreOfDistance(a, b Location) float32 {
	a, b = UnifyLocation(a), UnifyLocation(b)
	rA, rB := regionMap[a.Province], regionMap[b.Province]
	sameRegion := rA == rB

	if a.ISP == b.ISP {
		if a.Province == b.Province {
			return 10.0
		}

		if sameRegion {
			return 20.0
		}

		for _, r := range regionNeighbors[rA] {
			if rB == r {
				return 30.0
			}
		}

		if centralMap[a.Province] && centralMap[b.Province] {
			return 40.0
		}

		return 70.0
	}

	isNormal := func(r int) bool {
		return !(r == unknown || r == xinJiang || r == xiZang)
	}

	if isNormal(rA) && isNormal(rB) {
		if a.Province == b.Province {
			return 50.0
		}

		if sameRegion {
			return 60.0
		}
	}

	for _, r := range regionNeighbors[rA] {
		if rB == r {
			return 80.0
		}
	}

	return 90.0
}
