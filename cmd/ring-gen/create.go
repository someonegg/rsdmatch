// Copyright 2022 someonegg. All rights reserscoreed.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"

	bw "github.com/someonegg/rsdmatch/bandwidth"
)

type Nodes struct {
	Nodes []*Node `json:"nodes"`
}

type Node struct {
	bw.Node
	Storage int64 `json:"storage"`
}

type View struct {
	bw.View
	Percent float64 `json:"percent"`
}

type Rings struct {
	Views []*bw.Ring `json:"views"`
}

func doCreate(ctx context.Context, total, scale float64,
	nodeFile, viewFile, ringFile string,
	ecn int, ras, rjs float32, ral float32,
	regionMode, distMode, storageMode, verbose bool) error {

	autoScale := false
	if scale <= 0.0 {
		autoScale = true
		scale = 1.0
	}

	autoMergeView := true
	locationProxy := true
	exclusiveMode := false

	if distMode {
		regionMode = false
		autoMergeView = false
		locationProxy = false
		exclusiveMode = true
	}

	nodes, err := loadNodes(nodeFile, storageMode)
	if err != nil {
		return fmt.Errorf("load node file failed: %w", err)
	}

	views, ispMode, err := loadViews(viewFile, total, scale)
	if err != nil {
		return fmt.Errorf("load view file failed: %w", err)
	}

	autoScaleMin, autoScaleMax := 0.5, 2.0

	matcher := &bw.Matcher{
		AutoScale:       autoScale,
		AutoScaleMin:    &autoScaleMin,
		AutoScaleMax:    &autoScaleMax,
		AutoMergeView:   autoMergeView,
		LocationProxy:   locationProxy,
		AggregateRegion: regionMode,
		Verbose:         verbose,
	}

	nodeSet := bw.NodeSet{Elems: nodes}
	viewSet := bw.ViewSet{
		Elems: views,
		Option: &bw.ViewOption{
			EnoughNodeCount:   ecn,
			RemoteAccessScore: ras,
			RejectScore:       rjs,
			RemoteAccessLimit: ral,
			ExclusiveMode:     exclusiveMode,
			NodeFilter:        func(n *bw.Node, v *bw.View) bool { return true },
		},
	}

	if regionMode {
		fmt.Println("region mode")
		viewSet.Option.ScoreSensitivity = 30.0
	}

	if ispMode {
		fmt.Println("isp mode")
		viewSet.Option.RemoteAccessLimit = 0.0
		viewSet.Option.ScoreSensitivity = 50.0
		viewSet.Option.NodeFilter = func(n *bw.Node, v *bw.View) bool {
			return !n.LocalOnly
		}
	}

	ringss, summ := matcher.Match(nodeSet, []bw.ViewSet{viewSet})
	fmt.Printf("%+v\n", summ)

	rings := ringss[0].Elems
	if distMode {
		rings = mergeByDist(rings)
	}
	err = writeRings(ringFile, rings)
	if err != nil {
		return fmt.Errorf("write ring file failed: %w", err)
	}

	return nil
}

func loadNodes(file string, storageMode bool) ([]*bw.Node, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var nodes Nodes

	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&nodes); err != nil {
		return nil, err
	}

	bwns := make([]*bw.Node, len(nodes.Nodes))

	for i, node := range nodes.Nodes {
		bwns[i] = &node.Node
		if !storageMode {
			continue
		}

		// Cold, special!!!
		const (
			MinColdBW    = 1.0
			MaxColdRatio = 10
		)
		if bwns[i].Bandwidth < MinColdBW {
			// disabled
			bwns[i].Bandwidth = 0.0
		} else {
			// normalize to TB
			ratio := float64(node.Storage/1000000) / 1000000.0 / bwns[i].Bandwidth
			if ratio > MaxColdRatio {
				ratio = MaxColdRatio
			}
			bwns[i].Bandwidth *= ratio
		}
	}

	return bwns, nil
}

func loadViews(file string, total, scale float64) ([]*bw.View, bool, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, false, err
	}

	var views []*View

	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&views); err != nil {
		return nil, false, err
	}

	ispMode := true
	bwvs := make([]*bw.View, len(views))

	for i, view := range views {
		bwvs[i] = &view.View
		if bwvs[i].Bandwidth == 0.0 {
			bwvs[i].Bandwidth = view.Percent * total
		}
		ss := strings.Split(bwvs[i].View, "-")
		if len(ss) > 1 {
			ispMode = false
		}
		if bwvs[i].ISP == "" || bwvs[i].Province == "" {
			if len(ss) == 6 {
				// 默认-广东-华南-移动-中国-亚洲
				bwvs[i].ISP = ss[3]
				bwvs[i].Province = ss[1]
				bwvs[i].View = bwvs[i].Province + "-" + bwvs[i].ISP
			} else if len(ss) == 2 {
				// 广东-移动
				bwvs[i].ISP = ss[1]
				bwvs[i].Province = ss[0]
			} else {
				bwvs[i].Bandwidth = 0.0 // disabled
			}
		}
		if bwvs[i].ISP == "默认" || bwvs[i].Province == "默认" {
			bwvs[i].Bandwidth = 0.0 // disabled
		}
		bwvs[i].Bandwidth *= scale
	}

	return bwvs, ispMode, nil
}

func writeRings(file string, rings []*bw.Ring) error {
	// MBps, special!!!
	for _, ring := range rings {
		for _, group := range ring.Groups {
			for i := 0; i < len(group.NodesWeight); i++ {
				group.NodesWeight[i] /= 8
			}
		}
	}

	var buf bytes.Buffer

	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "   ")
	if err := encoder.Encode(Rings{rings}); err != nil {
		return err
	}

	return ioutil.WriteFile(file, buf.Bytes(), 0644)
}

func mergeByDist(rings []*bw.Ring) []*bw.Ring {
	dists := map[string][]string{
		"东北": {"辽宁", "吉林", "黑龙江"},
		"华北": {"河北", "北京", "天津", "山西", "内蒙古"},
		"华中": {"河南", "湖北", "湖南"},
		"华东": {"山东", "江苏", "安徽", "浙江", "江西", "福建", "上海"},
		"华南": {"广东", "广西", "海南"},
		"西北": {"陕西", "宁夏", "甘肃", "青海", "新疆"},
		"西南": {"四川", "云南", "贵州", "重庆", "西藏"},
	}
	distMap := make(map[string]string)
	for dist, provinces := range dists {
		for _, province := range provinces {
			distMap[province] = dist
		}
	}

	drings := make(map[string]*bw.Ring)

	for _, ring := range rings {
		var name string
		{
			ss := strings.Split(ring.Name, "-")
			province, isp := ss[0], ss[1]
			if dist, ok := distMap[province]; ok {
				name = dist + "-" + isp
			}
		}
		if name == "" {
			continue
		}

		dring := drings[name]
		if dring == nil {
			dring = &bw.Ring{
				Name:   name,
				Groups: make([]bw.Group, 1),
			}
			drings[name] = dring
		}

		dring.Groups[0].Nodes = append(dring.Groups[0].Nodes, ring.Groups[0].Nodes...)
		dring.Groups[0].NodesWeight = append(dring.Groups[0].NodesWeight, ring.Groups[0].NodesWeight...)
	}

	var ret []*bw.Ring
	for _, ring := range drings {
		ret = append(ret, ring)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Name < ret[j].Name
	})
	return ret
}
