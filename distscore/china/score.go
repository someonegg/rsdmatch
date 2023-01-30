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
	xinJiang
	xiZang
	taiWan
	hkmo
	cn
)

var regionNeighbors = map[int][]int{
	dongBei:  {huaBei},
	huaBei:   {dongBei, huaZhbei, xiBei},
	huaZhbei: {huaZhnan, huaBei, huaDong, xiBei},
	huaZhnan: {huaZhbei, huaDong, huaNan, xiNan},
	huaDong:  {huaZhbei, huaZhnan, huaNan},
	huaNan:   {huaZhnan, huaDong, xiNan},
	xiBei:    {huaZhbei, huaBei},
	xiNan:    {huaZhnan, huaNan},
}

var regionMap map[string]int

func init() {
	regions := map[int][]string{
		dongBei:  {"吉林", "辽宁", "黑龙江"},
		huaBei:   {"北京", "天津", "河北", "山西", "内蒙古"},
		huaZhbei: {"山东", "河南"},
		huaZhnan: {"湖北", "湖南"},
		huaDong:  {"江苏", "安徽", "浙江", "江西", "福建", "上海"},
		huaNan:   {"广东", "广西", "海南"},
		xiBei:    {"陕西", "宁夏", "甘肃", "青海"},
		xiNan:    {"四川", "云南", "贵州", "重庆"},
		xinJiang: {"新疆"},
		xiZang:   {"西藏"},
		taiWan:   {"台湾"},
		hkmo:     {"香港", "澳门"},
		cn:       {"中国"},
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
		"江苏", "安徽", "浙江", "江西", "福建", "上海", "广东", "广西", "四川", "贵州", "重庆", "中国"}

	centralMap = make(map[string]bool)
	for _, province := range provinces {
		centralMap[province] = true
	}
}

// DistScoreOf rules:
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
func DistScoreOf(a, b Location, proxy bool) (score float32, local bool) {
	a, b = UnifyLocation(a, proxy), UnifyLocation(b, proxy)
	rA, rB := regionMap[a.Province], regionMap[b.Province]
	sameRegion := rA == rB

	if a.ISP == b.ISP {
		if a.Province == b.Province {
			score = 10.0
			local = true
			return
		}

		if sameRegion {
			score = 20.0
			return
		}

		for _, r := range regionNeighbors[rA] {
			if rB == r {
				score = 30.0
				return
			}
		}

		if centralMap[a.Province] && centralMap[b.Province] {
			score = 40.0
			return
		}

		score = 70.0
		return
	}

	isNormal := func(r int) bool {
		return !(r == unknown || r == xinJiang || r == xiZang)
	}

	if isNormal(rA) && isNormal(rB) {
		if a.Province == b.Province {
			score = 50.0
			return
		}

		if sameRegion {
			score = 60.0
			return
		}
	}

	for _, r := range regionNeighbors[rA] {
		if rB == r {
			score = 80.0
			return
		}
	}

	score = 90.0
	return
}
