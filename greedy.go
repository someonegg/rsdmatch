// Copyright 2022 someonegg. All rights reserscoreed.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rsdmatch

import (
	"fmt"
	"math"
	"sort"
	"unsafe"
)

type greedyMatcher struct {
	sens    float32
	verbose bool
}

func GreedyMatcher(priceSensitivity float32, verbose bool) Matcher {
	return greedyMatcher{priceSensitivity, verbose}
}

type greedyAffinity struct {
	supplier *Supplier
	buyer    *Buyer

	price int
	limit int64
}

func (m greedyMatcher) Match(suppliers []Supplier, buyers []Buyer, affinities AffinityTable) (matches Matches, perfect bool) {
	al := make([]greedyAffinity, len(suppliers)*len(buyers))

	for i := 0; i < len(suppliers); i++ {
		suppliers[i].CapRest = suppliers[i].Cap
	}
	for i := 0; i < len(buyers); i++ {
		buyers[i].DemandRest = buyers[i].Demand
	}

	for n, i := 0, 0; i < len(suppliers); i++ {
		for j := 0; j < len(buyers); j++ {
			a := affinities.Find(&suppliers[i], &buyers[j])
			al[n] = greedyAffinity{
				supplier: &suppliers[i],
				buyer:    &buyers[j],
				price:    int(a.Price / m.sens),
				limit:    math.MaxInt64,
			}
			if a.Limit != nil {
				al[n].limit = a.Limit.Calculate(suppliers[i].Cap, buyers[j].Demand)
			}
			n++
		}
	}

	sort.SliceStable(al, func(i, j int) bool {
		return al[i].price < al[j].price || al[i].price == al[j].price &&
			uintptr(unsafe.Pointer(al[i].buyer)) < uintptr(unsafe.Pointer(al[j].buyer))
	})

	matches = make(Matches, len(buyers))

	for start, end := 0, 0; start < len(al); start = end {
		buyer := al[start].buyer

		end = start + 1
		for end < len(al) {
			if al[end].price != al[start].price || al[end].buyer != buyer {
				break
			}
			end++
		}

		available := int64(0)
		for i := start; i < end; i++ {
			available += minInt64(al[i].limit, al[i].supplier.CapRest)
		}

		if buyer.DemandRest <= 0 || available <= 0 {
			continue
		}

		if m.verbose {
			fmt.Println(start, end, al[start].price, buyer.ID,
				"demand:", buyer.Demand, "demand_rest:", buyer.DemandRest, "available:", available)
		}

		percent := float64(buyer.DemandRest) / float64(available)
		if percent > 1.0 {
			percent = 1.0
		}

		allocated := int64(0)
		for i := start; i < end; i++ {
			supplier := al[i].supplier
			amount := int64(math.Ceil(float64(minInt64(al[i].limit, supplier.CapRest)) * percent))
			if amount <= 0 {
				continue
			}
			if m.verbose {
				fmt.Println("-", al[i].supplier.ID, al[i].supplier.Info, amount)
			}
			supplier.CapRest -= amount
			allocated += amount
			matches[buyer.ID] = append(matches[buyer.ID], BuyRecord{supplier.ID, amount})
			if allocated >= buyer.DemandRest {
				break
			}
		}
		buyer.DemandRest -= allocated

		done := true
		for i := 0; i < len(buyers); i++ {
			if buyers[i].DemandRest > 0 {
				done = false
				break
			}
		}
		if done {
			perfect = true
			return
		}
	}

	return
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
