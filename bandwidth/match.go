// Copyright 2022 someonegg. All rights reserscoreed.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bandwidth

import (
	"fmt"
	"math"
	"sort"

	"github.com/someonegg/rsdmatch"
	ds "github.com/someonegg/rsdmatch/distscore"
	"github.com/someonegg/rsdmatch/distscore/china"
)

const bwUnit = 100 // Mbps

func (o *ViewOption) Fix() {
	if ral := o.RemoteAccessLimit; !(ral >= 0.0 && ral <= 1.0) {
		o.RemoteAccessLimit = DefaultViewOption.RemoteAccessLimit
		fmt.Println("RemoteAccessLimit fixed")
	}
	if o.ScoreSensitivity <= 0.0 {
		o.ScoreSensitivity = DefaultViewOption.ScoreSensitivity
		fmt.Println("ScoreSensitivity fixed")
	}
}

type affinityTable struct {
	ras    float32
	rjs    float32
	ral    float32
	filter func(*Node, *View) bool

	unifier ds.LocationUnifier
	scorer  ds.DistScorer
}

func newAffinityTable(o *ViewOption, unifier ds.LocationUnifier, scorer ds.DistScorer) rsdmatch.AffinityTable {
	return &affinityTable{
		ras:     o.RemoteAccessScore,
		rjs:     o.RejectScore,
		ral:     o.RemoteAccessLimit,
		filter:  o.NodeFilter,
		unifier: unifier,
		scorer:  scorer,
	}
}

func (t *affinityTable) Find(supplier *rsdmatch.Supplier, buyer *rsdmatch.Buyer) rsdmatch.Affinity {
	node := supplier.Info.(*Node)
	view := buyer.Info.(*View)

	score, local := t.scorer.DistScore(
		t.unifier.Unify(ds.Location{ISP: view.ISP, Province: view.Province}, false),
		t.unifier.Unify(ds.Location{ISP: node.ISP, Province: node.Province}, true))
	// filter
	if t.filter != nil && !t.filter(node, view) {
		return rsdmatch.Affinity{
			Price: score,
			Limit: nodePercentLimit(0.0),
		}
	}
	// local only
	if node.LocalOnly && !local {
		return rsdmatch.Affinity{
			Price: score,
			Limit: nodePercentLimit(0.0),
		}
	}
	// near
	if score < t.ras {
		return rsdmatch.Affinity{
			Price: score,
			Limit: nil,
		}
	}
	// remote
	if score < t.rjs {
		return rsdmatch.Affinity{
			Price: score,
			Limit: nodePercentLimit(t.ral),
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

func (m *Matcher) Match(nodes NodeSet, viewss []ViewSet) (ringss []RingSet, summ Summary) {
	if m.Unifier == nil {
		m.Unifier = china.NewLocationUnifier(m.ProxyMunici)
	}
	if m.Scorer == nil {
		m.Scorer = china.NewDistScorer()
	}

	suppliers, supplierCount, ispHasBW := genSuppliers(m.Unifier, nodes)
	buyerss, buyerCount, ispNeedsBW := genBuyerss(m.Unifier, viewss, summ.Scales)
	if m.AutoScale {
		summ.Scales = make(map[string]float64)
		for isp, has := range ispHasBW {
			if needs := ispNeedsBW[isp]; has > 0 && needs > 0 {
				scale := float64(has) / float64(needs)
				switch {
				case m.AutoScaleMin != nil && scale < *m.AutoScaleMin:
					scale = *m.AutoScaleMin
				case m.AutoScaleMax != nil && scale > *m.AutoScaleMax:
					scale = *m.AutoScaleMax
				}
				summ.Scales[isp] = scale
			}
		}
		buyerss, buyerCount, ispNeedsBW = genBuyerss(m.Unifier, viewss, summ.Scales)
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
	summ.NodesCount = supplierCount
	summ.ViewsCount = buyerCount
	summ.NodesBandwidth = float64(bwHas) / float64(1000/bwUnit)
	summ.ViewsBandwidth = float64(bwNeeds) / float64(1000/bwUnit)
	if m.Verbose {
		fmt.Printf("nodes: %v, views: %v, needs: %v, has: %v\n", supplierCount, buyerCount, bwNeeds*bwUnit, bwHas*bwUnit)
		fmt.Println("")
	}

	for _, buyers := range buyerss {
		var buyerViews map[string][]string
		if m.AutoMergeView {
			buyers.Elems, buyerViews = mergeBuyers(m.Unifier, buyers.Elems)
			if m.Verbose {
				fmt.Println("merged views:")
				for _, views := range buyerViews {
					if len(views) > 1 {
						fmt.Println("  ", views)
					}
				}
				fmt.Println("")
			}
		}

		matches, _ := rsdmatch.GreedyMatcher(buyers.Option.ScoreSensitivity, buyers.Option.ScoreSensitivity,
			buyers.Option.EnoughNodeCount, buyers.Option.ExclusiveMode, m.Verbose).Match(
			suppliers.Elems, buyers.Elems, newAffinityTable(buyers.Option, m.Unifier, m.Scorer))
		if m.Verbose {
			fmt.Println()
		}

		buyerDemand := make(map[string]int64)
		{
			elems := buyers.Elems
			sort.Slice(elems, func(i, j int) bool {
				return elems[i].DemandRest > elems[j].DemandRest
			})
			rests := int64(0)
			for _, elem := range elems {
				buyerDemand[elem.ID] += elem.Demand * bwUnit
				if rest := elem.DemandRest; rest > 0 {
					rests += rest
					if m.Verbose {
						fmt.Println(elem.ID, "demand:", elem.Demand*bwUnit, "demand_rest:", rest*bwUnit)
					}
				}
			}
			if m.Verbose && rests > 0 {
				fmt.Println("total needs", rests*bwUnit)
				fmt.Println("")
			}
			summ.BandwidthNeeds += float64(rests) / float64(1000/bwUnit)
		}

		ringss = append(ringss, genRings(matches, buyerViews, buyerDemand))
	}

	{
		elems := suppliers.Elems
		sort.Slice(elems, func(i, j int) bool {
			return elems[i].CapRest > elems[j].CapRest
		})
		rests := int64(0)
		for _, elem := range elems {
			if rest := elem.CapRest; rest > 0 {
				rests += rest
				node := elem.Info.(*Node)
				if m.Verbose {
					fmt.Println(node.ISP, node.Province, elem.ID, "cap:", elem.Cap*bwUnit, "cap_rest:", rest*bwUnit)
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

	return
}

type supplierSet struct {
	Elems []rsdmatch.Supplier
}

func genSuppliers(unifier ds.LocationUnifier, nodes NodeSet) (supplierSet, int, map[string]int64) {
	ispBW := make(map[string]int64)

	suppliers := make([]rsdmatch.Supplier, len(nodes.Elems))

	for i, node := range nodes.Elems {
		location := unifier.Unify(ds.Location{ISP: node.ISP, Province: node.Province}, true)
		suppliers[i].ID = node.Node
		suppliers[i].Cap = int64(math.Floor(node.Bandwidth * float64(1000/bwUnit)))
		if node.ISP == "" || node.Province == "" {
			suppliers[i].Cap = 0
			fmt.Println("node", node.Node, "is incomplete")
		}
		suppliers[i].CapRest = suppliers[i].Cap
		suppliers[i].Priority = int64(node.Priority*1000) + 1
		suppliers[i].Info = node
		if unifier.IsDeputy(ds.Location{ISP: node.ISP, Province: node.Province}) {
			ispBW[location.ISP] += suppliers[i].Cap
		}
	}

	sort.Slice(suppliers, func(i, j int) bool {
		return suppliers[i].ID < suppliers[j].ID
	})

	return supplierSet{suppliers}, len(suppliers), ispBW
}

type buyerSet struct {
	Elems  []rsdmatch.Buyer
	Option *ViewOption
}

func genBuyerss(unifier ds.LocationUnifier, viewss []ViewSet, ispScale map[string]float64) ([]buyerSet, int, map[string]int64) {
	count := 0
	ispBW := make(map[string]int64)

	var buyerss []buyerSet

	for _, views := range viewss {
		buyers := make([]rsdmatch.Buyer, len(views.Elems))

		for i, view := range views.Elems {
			location := unifier.Unify(ds.Location{ISP: view.ISP, Province: view.Province}, false)
			buyers[i].ID = view.View
			scale := 1.0
			if s, ok := ispScale[location.ISP]; ok {
				scale = s
			}
			buyers[i].Demand = int64(math.Ceil(view.Bandwidth * scale * float64(1000/bwUnit)))
			buyers[i].DemandRest = buyers[i].Demand
			buyers[i].Info = view
			if unifier.IsDeputy(ds.Location{ISP: view.ISP, Province: view.Province}) {
				ispBW[location.ISP] += buyers[i].Demand
			}
		}

		sort.Slice(buyers, func(i, j int) bool {
			return buyers[i].Demand > buyers[j].Demand
		})

		option := views.Option
		if option == nil {
			option = DefaultViewOption
		}
		option.Fix()

		buyerss = append(buyerss, buyerSet{buyers, option})
		count += len(buyers)
	}

	return buyerss, count, ispBW
}

func mergeBuyers(unifier ds.LocationUnifier, raws []rsdmatch.Buyer) (merged []rsdmatch.Buyer, buyerViews map[string][]string) {
	merged = make([]rsdmatch.Buyer, len(raws))
	buyerViews = make(map[string][]string, len(raws))

	indexes := make(map[string]int)
	next := 0
	for _, buyer := range raws {
		view := buyer.Info.(*View)
		location := unifier.Unify(ds.Location{ISP: view.ISP, Province: view.Province}, false)
		buyerID := location.Province + "-" + location.ISP
		if idx, ok := indexes[buyerID]; ok {
			merged[idx].Demand += buyer.Demand
			merged[idx].DemandRest = merged[idx].Demand
			buyerViews[buyerID] = append(buyerViews[buyerID], buyer.ID)
		} else {
			idx = next
			next++
			merged[idx].ID = buyerID
			merged[idx].Demand = buyer.Demand
			merged[idx].DemandRest = merged[idx].Demand
			merged[idx].Info = view
			buyerViews[buyerID] = []string{buyer.ID}
			indexes[buyerID] = idx
		}

	}

	merged = merged[0:next]
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Demand > merged[j].Demand
	})
	return
}

func genRings(matches rsdmatch.Matches, buyerViews map[string][]string, buyerDemand map[string]int64) RingSet {
	var rings []*Ring

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
		demand := buyerDemand[buyerID]
		if len(views) == 0 {
			rings = append(rings, &Ring{
				Name:   buyerID,
				Groups: []Group{makeGroup(records)},
				Demand: demand,
			})
			continue
		}
		for _, view := range views {
			rings = append(rings, &Ring{
				Name:   view,
				Groups: []Group{makeGroup(records)},
				Demand: demand,
			})
		}
	}

	sort.Slice(rings, func(i, j int) bool {
		return rings[i].Name < rings[j].Name
	})

	return RingSet{rings}
}
