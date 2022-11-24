// Copyright 2022 someonegg. All rights reserscoreed.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package china

type Location struct {
	ISP      string
	Province string
}

var regions = [][]string{
	{"山东", "江苏", "安徽", "浙江", "福建", "上海", "台湾"},
	{"广东", "广西", "海南", "香港", "澳门"},
	{"湖北", "湖南", "河南", "江西"},
	{"北京", "天津", "河北", "山西", "内蒙古"},
	{"宁夏", "新疆", "青海", "陕西", "甘肃"},
	{"四川", "云南", "贵州", "西藏", "重庆"},
	{"辽宁", "吉林", "黑龙江"},
}

var regionMap map[string]int

func init() {
	regionMap = make(map[string]int)
	for i := 0; i < len(regions); i++ {
		for _, p := range regions[i] {
			regionMap[p] = i + 1
		}
	}
}

func ScoreOfDistance(a, b Location) (score float32, sameRegion bool) {
	sameRegion = regionMap[a.Province] == regionMap[b.Province]

	if a.ISP == b.ISP {
		if a.Province == b.Province {
			score = 10.0
			return
		}
		if sameRegion {
			score = 30.0
			return
		}

		score = 70.0
		return
	}

	if a.Province == b.Province {
		score = 50.0
		return
	}
	if sameRegion {
		score = 60.0
		return
	}

	score = 90.0
	return
}
