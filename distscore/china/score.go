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
)

var regionNeighbors = map[int][]int{
	dongBei:  {huaBei},
	huaBei:   {dongBei, huaZhbei, xiBei},
	huaZhbei: {huaZhnan, huaBei, huaDong, xiBei},
	huaZhnan: {huaZhbei, huaDong, huaNan, xiNan},
	huaDong:  {huaZhbei, huaZhnan, huaNan},
	huaNan:   {huaZhnan, huaDong, xiNan},
	xiBei:    {huaZhbei, huaBei, xinJiang},
	xiNan:    {huaZhnan, huaNan, xiZang},
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
		huaDong:  {"江苏", "安徽", "浙江", "江西", "福建", "上海", "台湾"},
		huaNan:   {"广东", "广西", "海南", "香港", "澳门"},
		xiBei:    {"陕西", "宁夏", "甘肃", "青海"},
		xiNan:    {"四川", "云南", "贵州", "重庆"},
		xinJiang: {"新疆"},
		xiZang:   {"西藏"},
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
		"江苏", "安徽", "浙江", "江西", "福建", "上海", "广东", "广西", "陕西", "四川", "贵州", "重庆"}

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
//	Province: 40, !(xinJiang || xiZang)
//	Region: 50, !(xinJiang || xiZang)
//	ISP_Central: 60
//	ISP: 70
//	AdjacentRegion: 80
//	Other: 90
func ScoreOfDistance(a, b Location) (score float32, sameRegion bool) {
	a, b = UnifyLocation(a), UnifyLocation(b)
	rA, rB := regionMap[a.Province], regionMap[b.Province]
	sameRegion = rA == rB

	if a.ISP == b.ISP {
		if a.Province == b.Province {
			score = 10.0
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
			score = 60
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
			score = 40.0
			return
		}

		if sameRegion {
			score = 50.0
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
