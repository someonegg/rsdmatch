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
	bwUnit           = 100 // Mbps
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

	var summ Summary

	suppliers, bwHas := genSuppliers(nodes)
	buyers, bwNeeds := genBuyers(views, 1.0)
	if m.AutoScale {
		buyers, bwNeeds = genBuyers(views, float64(bwHas)/float64(bwNeeds))
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

	return genAllocs(matches), perfect, summ
}

func genSuppliers(nodes []*Node) ([]rsdmatch.Supplier, int64) {
	var bwHas int64

	suppliers := make([]rsdmatch.Supplier, len(nodes))

	for i, node := range nodes {
		suppliers[i].ID = node.Node
		suppliers[i].Cap = int64(math.Floor(node.Bandwidth * float64(1000/bwUnit)))
		suppliers[i].Info = node
		if node.ISP == "" || node.Province == "" {
			suppliers[i].Cap = 0
			fmt.Println("node", node.Node, "is incomplete")
		}
		bwHas += suppliers[i].Cap
	}

	sort.Slice(suppliers, func(i, j int) bool {
		return suppliers[i].ID < suppliers[j].ID
	})

	return suppliers, bwHas
}

func genBuyers(views []*View, scale float64) ([]rsdmatch.Buyer, int64) {
	var bwNeeds int64

	buyers := make([]rsdmatch.Buyer, len(views))

	for i, view := range views {
		buyers[i].ID = view.View
		buyers[i].Demand = int64(math.Ceil(view.Bandwidth * scale * float64(1000/bwUnit)))
		buyers[i].Info = view
		bwNeeds += buyers[i].Demand
	}

	sort.Slice(buyers, func(i, j int) bool {
		return buyers[i].Demand > buyers[j].Demand
	})

	return buyers, bwNeeds
}

func genAllocs(matches rsdmatch.Matches) []*Alloc {
	var allocs []*Alloc

	for buyerID, records := range matches {
		group := Group{
			Nodes:       make([]string, len(records)),
			NodesWeight: make([]int64, len(records)),
		}
		for i, record := range records {
			group.Nodes[i] = record.SupplierID
			group.NodesWeight[i] = record.Amount * bwUnit
		}
		allocs = append(allocs, &Alloc{
			Name:   buyerID,
			Groups: []Group{group},
		})
	}

	sort.Slice(allocs, func(i, j int) bool {
		return allocs[i].Name < allocs[j].Name
	})

	return allocs
}
