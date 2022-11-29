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
	"strings"

	bw "github.com/someonegg/rsdmatch/bandwidth"
)

var limitOfMode = make(map[string]float64)

type Node struct {
	bw.Node
	Mode string `json:"mode"`
}

type View struct {
	bw.View
	Percent float64 `json:"percent"`
}

type Allocs struct {
	Views []*bw.Alloc `json:"views"`
}

func doCreate(ctx context.Context, total float64, nodeFile, viewFile, allocFile string,
	ecn int, ras, rjs float32, ral float32, verbose bool) error {

	nodes, err := loadNodes(nodeFile)
	if err != nil {
		return fmt.Errorf("load node file failed: %w", err)
	}

	views, err := loadViews(viewFile, total)
	if err != nil {
		return fmt.Errorf("load view file failed: %w", err)
	}

	matcher := &bw.Matcher{
		EnoughNodeCount:   &ecn,
		RemoteAccessScore: &ras,
		RejectScore:       &rjs,
		RemoteAccessLimit: &ral,
		Verbose:           verbose,
	}

	allocs, perfect := matcher.Match(nodes, views)
	if perfect {
		fmt.Println("perfect match")
	}

	err = writeAllocs(allocFile, allocs)
	if err != nil {
		return fmt.Errorf("write alloc file failed: %w", err)
	}

	return nil
}

func loadNodes(file string) ([]*bw.Node, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var nodes []*Node

	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&nodes); err != nil {
		return nil, err
	}

	bwns := make([]*bw.Node, len(nodes))

	for i, node := range nodes {
		bwns[i] = &node.Node
		if modl, ok := limitOfMode[node.Mode]; ok {
			bwns[i].Bandwidth *= modl
		}
	}

	return bwns, nil
}

func loadViews(file string, total float64) ([]*bw.View, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var views []*View

	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&views); err != nil {
		return nil, err
	}

	bwvs := make([]*bw.View, len(views))

	for i, view := range views {
		bwvs[i] = &view.View
		if bwvs[i].Bandwidth == 0.0 {
			bwvs[i].Bandwidth = view.Percent * total
		}
		if bwvs[i].ISP == "" || bwvs[i].Province == "" {
			ss := strings.Split(bwvs[i].View, "-")
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
	}

	return bwvs, nil
}

func writeAllocs(file string, allocs []*bw.Alloc) error {
	// MBps, special!!!
	for _, alloc := range allocs {
		for _, group := range alloc.Groups {
			for i := 0; i < len(group.NodesWeight); i++ {
				group.NodesWeight[i] /= 8
			}
		}
	}

	var buf bytes.Buffer

	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "   ")
	if err := encoder.Encode(Allocs{allocs}); err != nil {
		return err
	}

	return ioutil.WriteFile(file, buf.Bytes(), 0644)
}
