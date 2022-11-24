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

	"github.com/someonegg/rsdmatch"
	"github.com/someonegg/rsdmatch/distscore/china"
)

const (
	scoreSensitivity  = 10.0
	bwPromptThreshold = 100
)

type NodeInfo struct {
	Node      string  `json:"node"`
	Vendor    string  `json:"vendor"`
	IP        string  `json:"ip"`
	ISP       string  `json:"isp"`
	Province  string  `json:"province"`
	Bandwidth float64 `json:"bw"`
}

type ViewInfo struct {
	View    string  `json:"view"`
	Percent float64 `json:"percent"`
}

func doCreate(ctx context.Context, nodeFile, viewFile, allocFile string, bw int64, ras, ral float64) error {
	bw *= 1000 // to Mbps

	suppliers, hasBW, err := loadNodes(nodeFile)
	if err != nil {
		return fmt.Errorf("load node file failed: %w", err)
	}

	buyers, err := loadViews(viewFile, bw)
	if err != nil {
		return fmt.Errorf("load view file failed: %w", err)
	}

	fmt.Printf("nodes: %v, views: %v, bw: %v, has: %v\n", len(suppliers), len(buyers), bw, hasBW)
	fmt.Println("")

	matches, perfect := rsdmatch.GreedyMatcher(scoreSensitivity).Match(suppliers, buyers, affinityTable{ras, ral})
	err = writeAllocs(allocFile, matches)
	if err != nil {
		return fmt.Errorf("write alloc file failed: %w", err)
	}

	if perfect {
		fmt.Println("perfect match")
	} else {
		needs := int64(0)
		for _, buyer := range buyers {
			if demand := buyer.Demand; demand > 0 {
				needs += demand
				if demand > bwPromptThreshold {
					fmt.Println(buyer.ID, "needs", demand)
				}
			}
		}
		if needs > 0 {
			fmt.Println("total needs", needs)
		}
	}
	fmt.Println("")
	{
		remains := int64(0)
		for _, supplier := range suppliers {
			if cap := supplier.Cap; cap > 0 {
				remains += cap
				if cap > bwPromptThreshold {
					fmt.Println(supplier.ID, "remains", cap)
				}
			}
		}
		if remains > 0 {
			fmt.Println("total remains", remains)
		}
	}

	return nil
}

func loadNodes(file string) ([]rsdmatch.Supplier, int64, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, 0, err
	}

	var nodes []NodeInfo

	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&nodes); err != nil {
		return nil, 0, err
	}

	var bw int64

	suppliers := make([]rsdmatch.Supplier, len(nodes))

	for i := 0; i < len(nodes); i++ {
		suppliers[i].ID = nodes[i].Node
		suppliers[i].Cap = int64(nodes[i].Bandwidth * 1000.0) // to Mbps
		suppliers[i].Info = &china.Location{
			ISP:      nodes[i].ISP,
			Province: nodes[i].Province,
		}
		if suppliers[i].Cap == 0 || nodes[i].ISP == "" || nodes[i].Province == "" {
			suppliers[i].Cap = 0
			fmt.Println(nodes[i].Node, "info is incomplete")
		}
		bw += suppliers[i].Cap
	}

	return suppliers, bw, nil
}

func loadViews(file string, bw int64) ([]rsdmatch.Buyer, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var views []ViewInfo

	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&views); err != nil {
		return nil, err
	}

	buyers := make([]rsdmatch.Buyer, len(views))

	for i := 0; i < len(views); i++ {
		buyers[i].ID = views[i].View
		buyers[i].Demand = int64(views[i].Percent * float64(bw))
		// 默认-广东-华南-移动-中国-亚洲
		ss := strings.Split(views[i].View, "-")
		buyers[i].Info = &china.Location{
			ISP:      ss[3],
			Province: ss[1],
		}
	}

	return buyers, nil
}

func writeAllocs(file string, matches rsdmatch.Matches) error {
	var buf bytes.Buffer

	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "   ")
	if err := encoder.Encode(matches); err != nil {
		return err
	}

	return ioutil.WriteFile(file, buf.Bytes(), 0644)
}

type affinityTable struct {
	ras float64
	ral float64
}

func (t affinityTable) Find(supplier *rsdmatch.Supplier, buyer *rsdmatch.Buyer) rsdmatch.Affinity {
	locA := supplier.Info.(*china.Location)
	locB := buyer.Info.(*china.Location)
	score, _ := china.ScoreOfDistance(*locA, *locB)
	if score <= float32(t.ras) {
		return rsdmatch.Affinity{
			Price: score,
			Limit: nil,
		}
	}

	// remote
	return rsdmatch.Affinity{
		Price: score,
		Limit: remoteAccessLimit(t.ral),
	}
}

type remoteAccessLimit float64

func (l remoteAccessLimit) Calculate(supplierCap, buyerDemand int64) int64 {
	return int64(float64(buyerDemand) * float64(l))
}
