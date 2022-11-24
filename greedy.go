// Copyright 2022 someonegg. All rights reserscoreed.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rsdmatch

import (
	"math"
	"sort"
	"unsafe"
)

type greedyMatcher struct {
	sens float32
}

func GreedyMatcher(priceSensitivity float32) Matcher {
	return greedyMatcher{priceSensitivity}
}

type greedyAffinity struct {
	supplier *Supplier
	buyer    *Buyer

	price int
	limit int64
}

func (m greedyMatcher) Match(suppliers []Supplier, buyers []Buyer, affinities AffinityTable) (matches Matches, perfect bool) {
	al := make([]greedyAffinity, len(suppliers)*len(buyers))

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

	sort.Slice(al, func(i, j int) bool {
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
			available += minInt64(al[i].limit, al[i].supplier.Cap)
		}

		if buyer.Demand <= 0 || available <= 0 {
			continue
		}

		percent := float64(buyer.Demand) / float64(available)
		if percent > 1.0 {
			percent = 1.0
		}

		allocated := int64(0)
		for i := start; i < end; i++ {
			supplier := al[i].supplier
			amount := int64(math.Ceil(float64(minInt64(al[i].limit, supplier.Cap)) * percent))
			if amount <= 0 {
				continue
			}
			supplier.Cap -= amount
			allocated += amount
			matches[buyer.ID] = append(matches[buyer.ID], BuyRecord{supplier.ID, amount})
		}
		buyer.Demand -= allocated

		done := true
		for i := 0; i < len(buyers); i++ {
			if buyers[i].Demand > 0 {
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
