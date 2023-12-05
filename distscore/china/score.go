// Copyright 2022 someonegg. All rights reserscoreed.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package china

import . "github.com/someonegg/rsdmatch/distscore"

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
		dongBei:  {"辽宁", "吉林", "黑龙江"},
		huaBei:   {"河北", "北京", "天津", "山西", "内蒙古"},
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

	regionProxy = make(map[string]string)
	for _, provinces := range regions {
		if len(provinces) <= 1 {
			continue
		}
		for i := 1; i < len(provinces); i++ {
			regionProxy[provinces[i]] = provinces[0]
		}
	}
}

var normalMap, centralMap, frontierMap map[string]bool

func init() {
	normalMap = make(map[string]bool)
	centralMap = make(map[string]bool)
	frontierMap = make(map[string]bool)

	centrals := []string{"北京", "天津", "河北", "山西", "山东", "河南", "湖北", "湖南",
		"江苏", "安徽", "浙江", "江西", "福建", "上海", "广东", "广西", "中国"}
	for _, province := range centrals {
		normalMap[province] = true
		centralMap[province] = true
	}

	normals := []string{"辽宁", "陕西", "四川", "重庆", "贵州"}
	for _, province := range normals {
		normalMap[province] = true
	}

	frontiers := []string{"新疆", "西藏"}
	for _, province := range frontiers {
		frontierMap[province] = true
	}
}

// DistScore rules:
//
//	ISP_Province: 10
//	ISP_Region: 20
//	ISP_AdjacentRegion: 30
//	ISP_Central: 40
//	ISP_ServerNormal: 50
//	ISP_!ServerFrontier: 60
//	ISP: 70
//	Province_Normal: 60
//	Other: 80
func DistScore(client, server Location) (score float32, local bool) {
	c, s := client, server
	cR, sR := regionMap[c.Province], regionMap[s.Province]

	if c.ISP == s.ISP {
		if c.Province == s.Province {
			score = 10.0
			local = true
			return
		}

		if cR == sR {
			score = 20.0
			return
		}

		if normalMap[s.Province] {
			for _, r := range regionNeighbors[cR] {
				if sR == r {
					score = 30.0
					return
				}
			}
		}

		if centralMap[c.Province] && centralMap[s.Province] {
			score = 40.0
			return
		}

		if normalMap[s.Province] {
			score = 50.0
			return
		}

		if !frontierMap[s.Province] {
			score = 60.0
			return
		}

		score = 70.0
		return
	}

	if normalMap[s.Province] {
		if c.Province == s.Province {
			score = 60.0
			return
		}
	}

	score = 80.0
	return
}

type distScorer struct{}

func (s distScorer) DistScore(client, server Location) (score float32, local bool) {
	return DistScore(client, server)
}

func NewDistScorer() DistScorer {
	return distScorer{}
}
