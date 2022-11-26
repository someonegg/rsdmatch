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
	"math"
	"sort"
	"strings"

	"github.com/someonegg/rsdmatch"
	"github.com/someonegg/rsdmatch/distscore/china"
)

const (
	scoreSensitivity = 10.0
	bwUnit           = 100 // Mbps
)

var limitOfMode = make(map[string]float64)

type NodeInfo struct {
	Node      string  `json:"node"`
	Vendor    string  `json:"vendor"`
	IP        string  `json:"ip"`
	ISP       string  `json:"isp"`
	Province  string  `json:"province"`
	Mode      string  `json:"mode"`
	Bandwidth float64 `json:"bw"`
}

type ViewInfo struct {
	View    string  `json:"view"`
	Percent float64 `json:"percent"`
}

type Allocs struct {
	Views []AllocView `json:"views"`
}

type AllocView struct {
	Name   string       `json:"name"`
	Groups []AllocGroup `json:"groups"`
}

type AllocGroup struct {
	Nodes       []string `json:"nodes"`
	NodesWeight []int64  `json:"nodesWeight"`
}

func doCreate(ctx context.Context, nodeFile, viewFile, allocFile string,
	bw int64, ras, ral, rjs float64, verbose bool) error {
	bw *= (1000 / bwUnit)

	suppliers, bwHas, err := loadNodes(nodeFile)
	if err != nil {
		return fmt.Errorf("load node file failed: %w", err)
	}

	buyers, bwNeeds, err := loadViews(viewFile, bw)
	if err != nil {
		return fmt.Errorf("load view file failed: %w", err)
	}

	fmt.Printf("nodes: %v, views: %v, needs: %v, has: %v\n", len(suppliers), len(buyers), bwNeeds*bwUnit, bwHas*bwUnit)
	fmt.Println("")

	matches, perfect := rsdmatch.GreedyMatcher(scoreSensitivity, verbose).Match(suppliers, buyers,
		affinityTable{
			ras: ras,
			ral: ral,
			rjs: rjs,
		})
	fmt.Println("")

	err = writeAllocs(allocFile, matches)
	if err != nil {
		return fmt.Errorf("write alloc file failed: %w", err)
	}

	if perfect {
		fmt.Println("perfect match")
	} else {
		sort.Slice(buyers, func(i, j int) bool {
			return buyers[i].DemandRest > buyers[j].DemandRest
		})
		needs := int64(0)
		for _, buyer := range buyers {
			if demandRest := buyer.DemandRest; demandRest > 0 {
				needs += demandRest
				fmt.Println(buyer.ID, "demand:", buyer.Demand*bwUnit, "demand_rest:", demandRest*bwUnit)
			}
		}
		if needs > 0 {
			fmt.Println("total needs", needs*bwUnit)
		}
	}
	fmt.Println("")
	{
		sort.Slice(suppliers, func(i, j int) bool {
			return suppliers[i].CapRest > suppliers[j].CapRest
		})
		remains := int64(0)
		for _, supplier := range suppliers {
			if capRest := supplier.CapRest; capRest > 0 {
				remains += capRest
				loc := supplier.Info.(*china.Location)
				fmt.Println(loc.ISP, loc.Province, supplier.ID, "cap:", supplier.Cap*bwUnit, "cap_rest:", capRest*bwUnit)
			}
		}
		if remains > 0 {
			fmt.Println("total remains", remains*bwUnit)
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

	var bwHas int64

	suppliers := make([]rsdmatch.Supplier, len(nodes))

	for i := 0; i < len(nodes); i++ {
		suppliers[i].ID = nodes[i].Node
		suppliers[i].Cap = int64(math.Ceil(nodes[i].Bandwidth * float64(1000/bwUnit)))
		suppliers[i].Info = &china.Location{
			ISP:      nodes[i].ISP,
			Province: nodes[i].Province,
		}
		if suppliers[i].Cap == 0 || nodes[i].ISP == "" || nodes[i].Province == "" {
			suppliers[i].Cap = 0
			fmt.Println(nodes[i].Node, "info is incomplete")
		}
		if modl, ok := limitOfMode[nodes[i].Mode]; ok {
			suppliers[i].Cap = int64(math.Ceil(float64(suppliers[i].Cap) * modl))
		}
		bwHas += suppliers[i].Cap
	}

	return suppliers, bwHas, nil
}

func loadViews(file string, bw int64) ([]rsdmatch.Buyer, int64, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, 0, err
	}

	var views []ViewInfo

	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&views); err != nil {
		return nil, 0, err
	}

	var bwNeeds int64

	buyers := make([]rsdmatch.Buyer, len(views))

	for i := 0; i < len(views); i++ {
		buyers[i].ID = views[i].View
		buyers[i].Demand = int64(math.Ceil(views[i].Percent * float64(bw)))
		// 默认-广东-华南-移动-中国-亚洲
		ss := strings.Split(views[i].View, "-")
		buyers[i].Info = &china.Location{
			ISP:      ss[3],
			Province: ss[1],
		}
		bwNeeds += buyers[i].Demand
	}

	return buyers, bwNeeds, nil
}

func writeAllocs(file string, matches rsdmatch.Matches) error {
	var views []AllocView

	for buyerID, records := range matches {
		group := AllocGroup{
			Nodes:       make([]string, len(records)),
			NodesWeight: make([]int64, len(records)),
		}
		for i, record := range records {
			group.Nodes[i] = record.SupplierID
			group.NodesWeight[i] = record.Amount * bwUnit
		}
		ss := strings.Split(buyerID, "-")
		views = append(views, AllocView{
			Name:   ss[1] + "-" + ss[3],
			Groups: []AllocGroup{group},
		})
	}

	sort.Slice(views, func(i, j int) bool {
		return views[i].Name < views[j].Name
	})

	var buf bytes.Buffer

	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "   ")
	if err := encoder.Encode(Allocs{views}); err != nil {
		return err
	}

	return ioutil.WriteFile(file, buf.Bytes(), 0644)
}

type affinityTable struct {
	ras float64
	ral float64
	rjs float64
}

func (t affinityTable) Find(supplier *rsdmatch.Supplier, buyer *rsdmatch.Buyer) rsdmatch.Affinity {
	locA := supplier.Info.(*china.Location)
	locB := buyer.Info.(*china.Location)
	score, _ := china.ScoreOfDistance(*locA, *locB)
	if float64(score) < t.ras {
		return rsdmatch.Affinity{
			Price: score,
			Limit: nil,
		}
	}
	if float64(score) < t.rjs {
		// remote
		return rsdmatch.Affinity{
			Price: score,
			Limit: remoteAccessLimit(t.ral),
		}
	}
	// reject
	return rsdmatch.Affinity{
		Price: score,
		Limit: remoteAccessLimit(0.0),
	}
}

type remoteAccessLimit float64

func (l remoteAccessLimit) Calculate(supplierCap, buyerDemand int64) int64 {
	return int64(math.Ceil(float64(supplierCap) * float64(l)))
}
