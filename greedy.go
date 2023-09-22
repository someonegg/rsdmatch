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
	sens      float32
	enough    int
	exclusive bool
	verbose   bool
}

func GreedyMatcher(priceSensitivity float32, enoughSupplierCount int, exclusive, verbose bool) Matcher {
	return greedyMatcher{priceSensitivity, enoughSupplierCount, exclusive, verbose}
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
		r2 := m.ptrCompare(unsafe.Pointer(al[i].buyer), unsafe.Pointer(al[j].buyer))
		return r1 < 0 ||
			r1 == 0 && r2 < 0 ||
			r1 == 0 && r2 == 0 &&
				al[i].supplier.Priority > al[j].supplier.Priority
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
		factorSum := int64(0)
		for i := start; i < end; i++ {
			amount := minInt64(al[i].limit, al[i].supplier.CapRest)
			factor := amount * al[i].supplier.Priority
			available += amount
			factorSum += factor
		}

		demandRest := buyer.DemandRest
		if buyer.Demand > 0 && demandRest <= 0 && len(matches[buyer.ID]) < m.enough {
			demandRest = 1
		}

		if demandRest <= 0 || available <= 0 || factorSum <= 0 {
			continue
		}

		if m.verbose {
			fmt.Println(buyer.ID,
				"demand:", buyer.Demand, "demand_rest:", demandRest,
				"available:", available, "factor_sum:", factorSum)
		}

		for i := start; i < end; i++ {
			if factorSum <= 0 {
				break
			}

			supplier := al[i].supplier
			amount := minInt64(al[i].limit, al[i].supplier.CapRest)
			factor := amount * al[i].supplier.Priority
			if !m.exclusive {
				may := math.Ceil(float64(factor) / float64(factorSum) * float64(demandRest))
				amount = minInt64(int64(may), amount)
			}
			factorSum -= factor
			if amount <= 0 || (m.exclusive && amount != supplier.Cap) {
				continue
			}

			records, recorded := matches[buyer.ID], false
			for i := 0; i < len(records); i++ {
				if records[i].SupplierID == supplier.ID {
					recorded = true
					records[i].Amount += amount
					break
				}
			}
			if !recorded {
				records = append(records, BuyRecord{supplier.ID, amount})
			}
			matches[buyer.ID] = records

			if m.verbose {
				fmt.Println("  ", al[i].price, al[i].supplier.Info, amount, factor)
			}

			supplier.CapRest -= amount
			buyer.DemandRest -= amount
			demandRest -= amount
			if demandRest <= 0 && len(matches[buyer.ID]) >= m.enough {
				break
			}
			demandRest = maxInt64(1, demandRest)
		}

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

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
