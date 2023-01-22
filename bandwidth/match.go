// Copyright 2022 someonegg. All rights reserscoreed.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bandwidth

import (
	"fmt"
	"math"
	"sort"

	"github.com/someonegg/rsdmatch"
	"github.com/someonegg/rsdmatch/distscore/china"
)

const (
	scoreSensitivity = 10.0
	bwUnit           = 50 // Mbps
)

func (m *Matcher) init() {
	if m.EnoughNodeCount == nil {
		m.ecn = DefaultEnoughNodeCount
	} else {
		m.ecn = *m.EnoughNodeCount
	}

	if m.RemoteAccessScore == nil {
		m.ras = DefaultRemoteAccessScore
	} else {
		m.ras = *m.RemoteAccessScore
	}

	if m.RejectScore == nil {
		m.rjs = DefaultRejectScore
	} else {
		m.rjs = *m.RejectScore
	}

	if m.RemoteAccessLimit == nil {
		m.ral = DefaultRemoteAccessLimit
	} else {
		m.ral = *m.RemoteAccessLimit
	}
}

func (m *Matcher) Find(supplier *rsdmatch.Supplier, buyer *rsdmatch.Buyer) rsdmatch.Affinity {
	node := supplier.Info.(*Node)
	view := buyer.Info.(*View)

	score, local := china.DistScoreOf(
		china.Location{ISP: node.ISP, Province: node.Province},
		china.Location{ISP: view.ISP, Province: view.Province},
		m.LocationProxy)
	// local only nodes
	if node.LocalOnly {
		if local {
			return rsdmatch.Affinity{
				Price: 0.0, // highest priority
				Limit: nil,
			}
		} else {
			return rsdmatch.Affinity{
				Price: score,
				Limit: nodePercentLimit(0.0),
			}
		}
	}
	// near
	if score < m.ras {
		return rsdmatch.Affinity{
			Price: score,
			Limit: nil,
		}
	}
	// remote
	if score < m.rjs {
		return rsdmatch.Affinity{
			Price: score,
			Limit: nodePercentLimit(m.ral),
		}
	}
	// reject
	return rsdmatch.Affinity{
		Price: score,
		Limit: nodePercentLimit(0.0),
	}
}

type nodePercentLimit float32

func (p nodePercentLimit) Calculate(supplierCap, buyerDemand int64) int64 {
	return int64(math.Ceil(float64(supplierCap) * float64(p)))
}

func (m *Matcher) Match(nodes []*Node, views []*View) (allocs []*Alloc, perfect bool, summary Summary) {
	m.init()

	var (
		summ       Summary
		buyerViews map[string][]string
	)

	suppliers, ispHasBW := genSuppliers(nodes)
	buyers, ispNeedsBW := genBuyers(views, summ.Scales)
	if m.AutoScale {
		summ.Scales = make(map[string]float64)
		for isp, has := range ispHasBW {
			if needs := ispNeedsBW[isp]; has > 0 && needs > 0 {
				summ.Scales[isp] = float64(has) / float64(needs)
			}
		}
		buyers, ispNeedsBW = genBuyers(views, summ.Scales)
	}
	if m.AutoMergeView {
		buyers, buyerViews = mergeBuyers(buyers, m.LocationProxy)
	}

	var (
		bwHas   int64
		bwNeeds int64
	)
	for _, has := range ispHasBW {
		bwHas += has
	}
	for _, needs := range ispNeedsBW {
		bwNeeds += needs
	}
	if m.Verbose {
		fmt.Printf("nodes: %v, views: %v, needs: %v, has: %v\n", len(suppliers), len(buyers), bwNeeds*bwUnit, bwHas*bwUnit)
		fmt.Println("")
	}
	summ.NodesCount = len(suppliers)
	summ.ViewsCount = len(buyers)
	summ.NodesBandwidth = float64(bwHas) / float64(1000/bwUnit)
	summ.ViewsBandwidth = float64(bwNeeds) / float64(1000/bwUnit)

	matches, perfect := rsdmatch.GreedyMatcher(scoreSensitivity, m.ecn,
		m.Verbose).Match(suppliers, buyers, m)
	if m.Verbose {
		fmt.Println()
	}

	{
		sort.Slice(buyers, func(i, j int) bool {
			return buyers[i].DemandRest > buyers[j].DemandRest
		})
		rests := int64(0)
		for _, buyer := range buyers {
			if rest := buyer.DemandRest; rest > 0 {
				rests += rest
				if m.Verbose {
					fmt.Println(buyer.ID, "demand:", buyer.Demand*bwUnit, "demand_rest:", rest*bwUnit)
				}
			} else {
				break
			}
		}
		if m.Verbose && rests > 0 {
			fmt.Println("total needs", rests*bwUnit)
			fmt.Println("")
		}
		summ.BandwidthNeeds = float64(rests) / float64(1000/bwUnit)
	}

	{
		sort.Slice(suppliers, func(i, j int) bool {
			return suppliers[i].CapRest > suppliers[j].CapRest
		})
		rests := int64(0)
		for _, supplier := range suppliers {
			if rest := supplier.CapRest; rest > 0 {
				rests += rest
				node := supplier.Info.(*Node)
				if m.Verbose {
					fmt.Println(node.ISP, node.Province, supplier.ID, "cap:", supplier.Cap*bwUnit, "cap_rest:", rest*bwUnit)
				}
			} else {
				break
			}
		}
		if m.Verbose && rests > 0 {
			fmt.Println("total remains", rests*bwUnit)
			fmt.Println("")
		}
		summ.BandwidthRemains = float64(rests) / float64(1000/bwUnit)
	}

	return genAllocs(matches, buyerViews), perfect, summ
}

func genSuppliers(nodes []*Node) ([]rsdmatch.Supplier, map[string]int64) {
	ispBW := make(map[string]int64)

	suppliers := make([]rsdmatch.Supplier, len(nodes))

	for i, node := range nodes {
		suppliers[i].ID = node.Node
		suppliers[i].Cap = int64(math.Floor(node.Bandwidth * float64(1000/bwUnit)))
		suppliers[i].Info = node
		if node.ISP == "" || node.Province == "" {
			suppliers[i].Cap = 0
			fmt.Println("node", node.Node, "is incomplete")
		}
		ispBW[node.ISP] += suppliers[i].Cap
	}

	sort.Slice(suppliers, func(i, j int) bool {
		return suppliers[i].ID < suppliers[j].ID
	})

	return suppliers, ispBW
}

func genBuyers(views []*View, ispScale map[string]float64) ([]rsdmatch.Buyer, map[string]int64) {
	ispBW := make(map[string]int64)

	buyers := make([]rsdmatch.Buyer, len(views))

	for i, view := range views {
		buyers[i].ID = view.View
		scale := 1.0
		if s, ok := ispScale[view.ISP]; ok {
			scale = s
		}
		buyers[i].Demand = int64(math.Ceil(view.Bandwidth * scale * float64(1000/bwUnit)))
		buyers[i].Info = view
		ispBW[view.ISP] += buyers[i].Demand
	}

	sort.Slice(buyers, func(i, j int) bool {
		return buyers[i].Demand > buyers[j].Demand
	})

	return buyers, ispBW
}

func mergeBuyers(raws []rsdmatch.Buyer, locationProxy bool) (merged []rsdmatch.Buyer, buyerViews map[string][]string) {
	merged = make([]rsdmatch.Buyer, len(raws))
	buyerViews = make(map[string][]string, len(raws))

	indexes := make(map[string]int)
	next := 0
	for _, buyer := range raws {
		view := buyer.Info.(*View)
		location := china.UnifyLocation(china.Location{ISP: view.ISP, Province: view.Province}, locationProxy)
		buyerID := location.Province + "-" + location.ISP
		if idx, ok := indexes[buyerID]; ok {
			merged[idx].Demand += buyer.Demand
			buyerViews[buyerID] = append(buyerViews[buyerID], buyer.ID)
		} else {
			merged[next].ID = buyerID
			merged[next].Demand = buyer.Demand
			merged[next].Info = view
			buyerViews[buyerID] = []string{buyer.ID}
			indexes[buyerID] = next
			next++
		}

	}

	merged = merged[0:next]
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Demand > merged[j].Demand
	})
	return
}

func genAllocs(matches rsdmatch.Matches, buyerViews map[string][]string) []*Alloc {
	var allocs []*Alloc

	makeGroup := func(records []rsdmatch.BuyRecord) Group {
		group := Group{
			Nodes:       make([]string, len(records)),
			NodesWeight: make([]int64, len(records)),
		}
		for i, record := range records {
			group.Nodes[i] = record.SupplierID
			group.NodesWeight[i] = record.Amount * bwUnit
		}
		return group
	}

	for buyerID, records := range matches {
		views := buyerViews[buyerID]
		if len(views) == 0 {
			allocs = append(allocs, &Alloc{
				Name:   buyerID,
				Groups: []Group{makeGroup(records)},
			})
			continue
		}
		for _, view := range views {
			allocs = append(allocs, &Alloc{
				Name:   view,
				Groups: []Group{makeGroup(records)},
			})
		}
	}

	sort.Slice(allocs, func(i, j int) bool {
		return allocs[i].Name < allocs[j].Name
	})

	return allocs
}
