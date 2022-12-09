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
	enough  int
	verbose bool
}

func GreedyMatcher(priceSensitivity float32, enoughSupplierCount int, verbose bool) Matcher {
	return greedyMatcher{priceSensitivity, enoughSupplierCount, verbose}
}

type greedyAffinity struct {
	supplier *Supplier
	buyer    *Buyer

	price float32
	limit int64
}

func (m greedyMatcher) sensCompare(a, b float32) int {
	iA, iB := int(a/m.sens), int(b/m.sens)
	return iA - iB
}

func (m greedyMatcher) ptrCompare(a, b unsafe.Pointer) int {
	if uintptr(a) < uintptr(b) {
		return -1
	}
	if uintptr(a) > uintptr(b) {
		return 1
	}
	return 0
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
				price:    a.Price,
				limit:    math.MaxInt64,
			}
			if a.Limit != nil {
				al[n].limit = a.Limit.Calculate(suppliers[i].Cap, buyers[j].Demand)
			}
			n++
		}
	}

	sort.SliceStable(al, func(i, j int) bool {
		r1 := m.sensCompare(al[i].price, al[j].price)
		if r1 < 0 {
			return true
		}
		if r1 > 0 {
			return false
		}
		r2 := m.ptrCompare(unsafe.Pointer(al[i].buyer), unsafe.Pointer(al[j].buyer))
		if r2 < 0 {
			return true
		}
		if r2 > 0 {
			return false
		}
		return al[i].price < al[j].price
	})

	matches = make(Matches, len(buyers))

	for start, end := 0, 0; start < len(al); start = end {
		buyer := al[start].buyer

		end = start + 1
		for end < len(al) {
			if m.sensCompare(al[end].price, al[start].price) != 0 || al[end].buyer != buyer {
				break
			}
			end++
		}

		available := int64(0)
		for i := start; i < end; i++ {
			available += minInt64(al[i].limit, al[i].supplier.CapRest)
		}

		demandRest := buyer.DemandRest
		if buyer.Demand > 0 && demandRest <= 0 && len(matches[buyer.ID]) < m.enough {
			demandRest = 1
		}

		if demandRest <= 0 || available <= 0 {
			continue
		}

		if m.verbose {
			fmt.Println(buyer.ID, "demand:", buyer.Demand,
				"demand_rest:", demandRest, "available:", available)
		}

		percent := float64(demandRest) / float64(available)
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
				fmt.Println("  ", al[i].price, al[i].supplier.Info, amount)
			}
			supplier.CapRest -= amount
			allocated += amount
			matches[buyer.ID] = append(matches[buyer.ID], BuyRecord{supplier.ID, amount})
			if allocated >= demandRest && len(matches[buyer.ID]) >= m.enough {
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
