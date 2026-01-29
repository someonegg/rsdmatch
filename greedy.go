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
	bottom    float32
	enough    int
	exclusive bool
	verbose   bool
}

// GreedyMatcher creates a greedy matching algorithm that matches buyers to suppliers
// based on price affinity and supplier priority.
//
// Parameters:
//   - priceSensitivity: Groups prices into tiers. Prices are divided by this value
//     and rounded down to integers for grouping. Smaller values = more granular tiers.
//     Example: sens=1.0 means prices 1.0-1.9 are in tier 1, 2.0-2.9 in tier 2, etc.
//
//   - priceBottom: Maximum acceptable price threshold. Matching stops for a buyer
//     when price exceeds this threshold (and demand is satisfied and enough suppliers
//     are matched). This acts as a price ceiling, not a floor.
//     Example: bottom=25.0 means buyer will only purchase from suppliers with price <= 25.0
//
//   - enoughSupplierCount: Target number of suppliers per buyer. If buyer's demand is
//     satisfied but matched suppliers < this count, continue trying to match more
//     suppliers (with minimum demandRest=1 to allow additional matches).
//
//   - exclusive: When true, buyer must either take the entire capacity of a supplier
//     or none at all. This ensures each supplier serves at most one buyer.
//     Suppliers with partial remaining capacity (CapRest < Cap) are rejected.
//
//   - verbose: Enable detailed logging of matching process.
//
// Matching strategy:
//   1. Sort all (supplier, buyer) pairs by: price tier → buyer → supplier priority
//   2. For each price tier, allocate supplier capacity to buyers proportionally
//      by priority (non-exclusive) or exclusively (full capacity only)
//   3. Stop matching a buyer when: demand satisfied + enough suppliers + price > bottom
func GreedyMatcher(priceSensitivity, priceBottom float32, enoughSupplierCount int, exclusive, verbose bool) Matcher {
	return greedyMatcher{priceSensitivity, priceBottom, enoughSupplierCount, exclusive, verbose}
}

type greedyAffinity struct {
	supplier *Supplier
	buyer    *Buyer

	price float32
	limit int64
}

// sensCompare compares two prices by grouping them into tiers.
// Returns negative if a < b, zero if same tier, positive if a > b.
// Price tiers are calculated by floor(price / sensitivity).
// Example: sens=1.0, price=1.0 and 1.9 both give tier=1 (same tier)
//          sens=1.0, price=1.9 and 2.0 give tier=1 and tier=2 (different tiers)
func (m greedyMatcher) sensCompare(a, b float32) int {
	iA, iB := int(a/m.sens), int(b/m.sens)
	return iA - iB
}

// ptrCompare compares two pointers by their memory address.
// Used to ensure deterministic ordering of buyers when prices are equal.
// Since pointers are stable within a single execution, this provides
// consistent ordering across multiple runs with the same input.
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

	// Sort all (supplier, buyer) pairs by priority:
	// 1. Price tier (lower is better): uses sensCompare to group prices
	// 2. Buyer (for determinism): uses pointer address for consistent ordering
	// 3. Supplier priority (higher is better): within same price tier and buyer,
	//    higher priority suppliers come first
	// This ensures we process cheapest suppliers first, and within same price,
	// prefer higher priority suppliers.
	sort.SliceStable(al, func(i, j int) bool {
		r1 := m.sensCompare(al[i].price, al[j].price)
		r2 := m.ptrCompare(unsafe.Pointer(al[i].buyer), unsafe.Pointer(al[j].buyer))
		return r1 < 0 ||
			r1 == 0 && r2 < 0 ||
			r1 == 0 && r2 == 0 &&
				al[i].supplier.Priority > al[j].supplier.Priority
	})

	matches = make(Matches, len(buyers))

	// Process affinity list in chunks grouped by (price tier, buyer)
	// Each chunk represents: all suppliers for a specific buyer at a specific price tier
	for start, end := 0, 0; start < len(al); start = end {
		buyer := al[start].buyer

		// Find the end of current group: same price tier AND same buyer
		end = start + 1
		for end < len(al) {
			if m.sensCompare(al[end].price, al[start].price) != 0 || al[end].buyer != buyer {
				break
			}
			end++
		}

		// Calculate total available capacity and weighted priority sum for this group
		available := int64(0)   // Total capacity all suppliers in this group can provide
		factorSum := int64(0)   // Sum of (capacity × priority) for proportional allocation
		for i := start; i < end; i++ {
			amount := minInt64(al[i].limit, al[i].supplier.CapRest)
			factor := amount * al[i].supplier.Priority
			available += amount
			factorSum += factor
		}

		demandRest := buyer.DemandRest

		// Special rule: if demand is already satisfied but we haven't matched enough suppliers,
		// set demandRest to 1 to allow matching additional suppliers.
		// This ensures we try to reach the target supplier count even when demand is met.
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
			// Calculate maximum amount this supplier can provide
			// Limited by both: BuyLimit (if set) and remaining capacity
			amount := minInt64(al[i].limit, al[i].supplier.CapRest)
			factor := amount * al[i].supplier.Priority

			// Non-exclusive mode: allocate proportionally by priority
			if !m.exclusive {
				may := math.Ceil(float64(factor) / float64(factorSum) * float64(demandRest))
				amount = minInt64(int64(may), amount)
			}
			factorSum -= factor

			// Skip this supplier if:
			// 1. amount <= 0: no available capacity
			// 2. exclusive mode and amount != supplier.Cap:
			//    - Supplier has been partially allocated to other buyers (CapRest < Cap)
			//    - Or BuyLimit prevents taking full capacity
			// In exclusive mode, buyer must either take entire supplier or none
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

			// Stop matching this buyer when ALL of these conditions are met:
			// 1. demandRest <= 0: remaining demand for this price tier is satisfied
			// 2. len(matches[buyer.ID]) >= m.enough: matched enough suppliers
			// 3. (exclusive OR price > bottom):
			//    - exclusive mode: always stop when demand satisfied (exclusive matches are all-or-nothing)
			//    - price > bottom: price exceeds threshold, stop to avoid expensive suppliers
			//
			// This ensures we don't continue to higher price tiers unnecessarily.
			if demandRest <= 0 && len(matches[buyer.ID]) >= m.enough &&
				(m.exclusive || m.sensCompare(al[i].price, m.bottom) > 0) {
				break
			}
			// Ensure demandRest stays at least 1 to allow continued matching
			// within this price tier (but the check above prevents moving to next tier)
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
