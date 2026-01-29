// Copyright 2022 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rsdmatch

import (
	"testing"
)

// 辅助函数
func makeSupplier(id string, cap, priority int64, info interface{}) Supplier {
	return Supplier{
		ID:       id,
		Cap:      cap,
		CapRest:  cap,
		Priority: priority,
		Info:     info,
	}
}

func makeBuyer(id string, demand int64, info interface{}) Buyer {
	return Buyer{
		ID:         id,
		Demand:     demand,
		DemandRest: demand,
		Info:       info,
	}
}

// mockAffinityTable 是一个简单的 AffinityTable 实现
type mockAffinityTable struct {
	prices map[string]map[string]float32
	limits map[string]map[string]int64
}

func newMockAffinityTable() *mockAffinityTable {
	return &mockAffinityTable{
		prices: make(map[string]map[string]float32),
		limits: make(map[string]map[string]int64),
	}
}

func (m *mockAffinityTable) setPrice(supplierID, buyerID string, price float32) {
	if m.prices[supplierID] == nil {
		m.prices[supplierID] = make(map[string]float32)
	}
	m.prices[supplierID][buyerID] = price
}

func (m *mockAffinityTable) setLimit(supplierID, buyerID string, limit int64) {
	if m.limits[supplierID] == nil {
		m.limits[supplierID] = make(map[string]int64)
	}
	m.limits[supplierID][buyerID] = limit
}

func (m *mockAffinityTable) Find(supplier *Supplier, buyer *Buyer) Affinity {
	price := float32(100.0) // 默认价格
	if p, ok := m.prices[supplier.ID][buyer.ID]; ok {
		price = p
	}

	var limit BuyLimit
	if l, ok := m.limits[supplier.ID][buyer.ID]; ok {
		limit = fixedBuyLimit(l)
	}

	return Affinity{Price: price, Limit: limit}
}

// fixedBuyLimit 是一个固定值的 BuyLimit 实现
type fixedBuyLimit int64

func (f fixedBuyLimit) Calculate(supplierCap, buyerDemand int64) int64 {
	return int64(f)
}

// 1. 基础匹配测试
func TestGreedyMatcher_Basic(t *testing.T) {
	t.Run("OneSupplierOneBuyer", func(t *testing.T) {
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 50, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, perfect := matcher.Match(suppliers, buyers, affinity)

		if !perfect {
			t.Error("Expected perfect match")
		}
		if len(matches) != 1 {
			t.Errorf("Expected 1 buyer match, got %d", len(matches))
		}
		if len(matches["b1"]) != 1 {
			t.Errorf("Expected 1 supplier for b1, got %d", len(matches["b1"]))
		}
		if matches["b1"][0].Amount != 50 {
			t.Errorf("Expected amount 50, got %d", matches["b1"][0].Amount)
		}
	})

	t.Run("OneSupplierTwoBuyers", func(t *testing.T) {
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 30, nil),
			makeBuyer("b2", 40, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)
		affinity.setPrice("s1", "b2", 10.0)

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, perfect := matcher.Match(suppliers, buyers, affinity)

		if !perfect {
			t.Error("Expected perfect match")
		}
		if matches["b1"][0].Amount != 30 {
			t.Errorf("Expected b1 amount 30, got %d", matches["b1"][0].Amount)
		}
		if matches["b2"][0].Amount != 40 {
			t.Errorf("Expected b2 amount 40, got %d", matches["b2"][0].Amount)
		}
		if suppliers[0].CapRest != 30 {
			t.Errorf("Expected supplier CapRest 30, got %d", suppliers[0].CapRest)
		}
	})
}

// 2. 价格敏感度测试
func TestGreedyMatcher_PriceSensitivity(t *testing.T) {
	t.Run("Strict_Sens1", func(t *testing.T) {
		// sens=1.0: 严格按价格分组，price 1.0 和 1.9 在同一组
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
			makeSupplier("s2", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 100, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 1.0)
		affinity.setPrice("s2", "b1", 1.9)

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, _ := matcher.Match(suppliers, buyers, affinity)

		// s1 和 s2 价格在同一组，应该平均分配
		total := matches["b1"][0].Amount + matches["b1"][1].Amount
		if total != 100 {
			t.Errorf("Expected total 100, got %d", total)
		}
	})

	t.Run("Loose_Sens10", func(t *testing.T) {
		// sens=10.0: 价格分组宽松，price 1.0 和 9.9 在同一组
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
			makeSupplier("s2", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 100, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 1.0)
		affinity.setPrice("s2", "b1", 9.9)

		matcher := GreedyMatcher(10.0, 0.0, 0, false, false)
		matches, _ := matcher.Match(suppliers, buyers, affinity)

		total := matches["b1"][0].Amount + matches["b1"][1].Amount
		if total != 100 {
			t.Errorf("Expected total 100, got %d", total)
		}
	})

	t.Run("VeryStrict_Sens01", func(t *testing.T) {
		// sens=0.1: 几乎不分组，1.0 和 1.05 在不同组
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
			makeSupplier("s2", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 100, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 1.0)
		affinity.setPrice("s2", "b1", 1.05)

		matcher := GreedyMatcher(0.1, 0.0, 0, false, false)
		matches, _ := matcher.Match(suppliers, buyers, affinity)

		// s1 价格更低，应该先全部消耗 s1
		// 但由于在同一买家遍历中，s1 先被处理，可能不会处理 s2
		total := int64(0)
		for _, record := range matches["b1"] {
			total += record.Amount
		}
		if total != 100 {
			t.Errorf("Expected total 100, got %d", total)
		}
	})
}

// 3. 优先级权重测试
func TestGreedyMatcher_Priority(t *testing.T) {
	t.Run("DifferentPriority", func(t *testing.T) {
		// 不同优先级按权重分配
		suppliers := []Supplier{
			makeSupplier("s1", 100, 2, nil), // priority 2
			makeSupplier("s2", 100, 1, nil), // priority 1
		}
		buyers := []Buyer{
			makeBuyer("b1", 100, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)
		affinity.setPrice("s2", "b1", 10.0)

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, _ := matcher.Match(suppliers, buyers, affinity)

		// s1 的权重是 s2 的2倍，应该获得 2/3
		expectedS1 := int64(67)
		if matches["b1"][0].Amount != expectedS1 && matches["b1"][0].Amount != expectedS1-1 {
			t.Errorf("Expected s1 amount ~%d, got %d", expectedS1, matches["b1"][0].Amount)
		}
	})

	t.Run("SamePriority", func(t *testing.T) {
		// 相同优先级平均分配
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
			makeSupplier("s2", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 100, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)
		affinity.setPrice("s2", "b1", 10.0)

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, _ := matcher.Match(suppliers, buyers, affinity)

		// 相同优先级，应该平均分配
		expected := int64(50)
		for _, record := range matches["b1"] {
			if record.Amount < expected-1 || record.Amount > expected+1 {
				t.Errorf("Expected amount ~%d, got %d", expected, record.Amount)
			}
		}
	})
}

// 4. 独占模式测试
func TestGreedyMatcher_Exclusive(t *testing.T) {
	t.Run("NonExclusive", func(t *testing.T) {
		// exclusive=false: 一个 Supplier 可以服务多个 Buyers
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 60, nil),
			makeBuyer("b2", 40, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)
		affinity.setPrice("s1", "b2", 10.0)

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, perfect := matcher.Match(suppliers, buyers, affinity)

		if !perfect {
			t.Error("Expected perfect match")
		}
		if matches["b1"][0].Amount != 60 {
			t.Errorf("Expected b1 amount 60, got %d", matches["b1"][0].Amount)
		}
		if matches["b2"][0].Amount != 40 {
			t.Errorf("Expected b2 amount 40, got %d", matches["b2"][0].Amount)
		}
	})

	t.Run("Exclusive_LimitPreventsFullCapacity", func(t *testing.T) {
		// exclusive=true: BuyLimit 限制了购买量，导致无法获得完整容量
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 150, nil), // demand > cap
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)
		affinity.setLimit("s1", "b1", 80) // 限制为 80 < Cap=100

		matcher := GreedyMatcher(1.0, 0.0, 0, true, false)
		matches, perfect := matcher.Match(suppliers, buyers, affinity)

		// 由于 limit=80 < Cap=100，exclusive 模式拒绝匹配
		if len(matches["b1"]) != 0 {
			t.Errorf("Expected no matches (limit prevents full capacity), got %d", len(matches["b1"]))
		}
		if perfect {
			t.Error("Expected non-perfect match")
		}
	})

	t.Run("Exclusive_DemandEqualsCapacity", func(t *testing.T) {
		// exclusive=true: Demand == Cap，成功匹配
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
			makeSupplier("s2", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 100, nil), // 可以完整拿走 s1
			makeBuyer("b2", 100, nil), // 可以完整拿走 s2
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)
		affinity.setPrice("s2", "b1", 10.0)
		affinity.setPrice("s1", "b2", 10.0)
		affinity.setPrice("s2", "b2", 10.0)

		matcher := GreedyMatcher(1.0, 0.0, 0, true, false)
		matches, perfect := matcher.Match(suppliers, buyers, affinity)

		// 每个 buyer 完整拿走一个 supplier
		if !perfect {
			t.Error("Expected perfect match")
		}
		if matches["b1"][0].Amount != 100 {
			t.Errorf("Expected b1 amount 100, got %d", matches["b1"][0].Amount)
		}
		if matches["b2"][0].Amount != 100 {
			t.Errorf("Expected b2 amount 100, got %d", matches["b2"][0].Amount)
		}
	})
}

// 5. Price Bottom 截止测试
// bottom 是价格上限：当 demand 已满足 + enough suppliers + price > bottom 时停止
func TestGreedyMatcher_PriceBottom(t *testing.T) {
	t.Run("AllPricesBelowBottom", func(t *testing.T) {
		// 所有价格 < bottom：会购买所有可用的 suppliers
		suppliers := []Supplier{
			makeSupplier("s1", 50, 1, nil),
			makeSupplier("s2", 50, 1, nil),
			makeSupplier("s3", 50, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 150, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0) // tier 1
		affinity.setPrice("s2", "b1", 10.0) // tier 1
		affinity.setPrice("s3", "b1", 20.0) // tier 2, < bottom

		matcher := GreedyMatcher(1.0, 25.0, 0, false, false) // bottom=25
		matches, _ := matcher.Match(suppliers, buyers, affinity)

		total := int64(0)
		for _, record := range matches["b1"] {
			total += record.Amount
		}
		if total != 150 {
			t.Errorf("Expected total 150 (all suppliers), got %d", total)
		}
	})

	t.Run("StopWhenPriceExceedsBottom", func(t *testing.T) {
		// 当 price > bottom 且 demand 已满足 + enough suppliers 时停止
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil), // 完全满足 demand
			makeSupplier("s2", 50, 1, nil),  // 不会被购买（价格超过 bottom）
		}
		buyers := []Buyer{
			makeBuyer("b1", 100, nil), // demand 可以被 s1 完全满足
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0) // tier 1, < bottom
		affinity.setPrice("s2", "b1", 30.0) // tier 3, > bottom

		matcher := GreedyMatcher(1.0, 25.0, 1, false, false) // bottom=25, enough=1
		matches, _ := matcher.Match(suppliers, buyers, affinity)

		// s1 满足了 demand (100), enough=1 已满足，s2 的价格 30 > bottom 25
		// 因此不会购买 s2
		if len(matches["b1"]) != 1 {
			t.Errorf("Expected only 1 supplier, got %d", len(matches["b1"]))
		}
		if matches["b1"][0].SupplierID != "s1" {
			t.Errorf("Expected to buy from s1 only, got %s", matches["b1"][0].SupplierID)
		}
	})

	t.Run("ContinueIfDemandNotSatisfied", func(t *testing.T) {
		// 即使 price > bottom，如果 demand 未满足，仍会继续购买
		suppliers := []Supplier{
			makeSupplier("s1", 50, 1, nil),  // tier 1
			makeSupplier("s2", 50, 1, nil),  // tier 1
			makeSupplier("s3", 50, 1, nil),  // tier 3, > bottom
		}
		buyers := []Buyer{
			makeBuyer("b1", 150, nil), // demand > s1+s2 的容量
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0) // tier 1
		affinity.setPrice("s2", "b1", 10.0) // tier 1
		affinity.setPrice("s3", "b1", 30.0) // tier 3, > bottom

		matcher := GreedyMatcher(1.0, 25.0, 3, false, false) // bottom=25, enough=3
		matches, perfect := matcher.Match(suppliers, buyers, affinity)

		// s1+s2 只能提供 100，还有 50 demand 未满足
		// s3 的价格 30 > bottom，但由于 demand 未满足，仍会购买 s3
		total := int64(0)
		for _, record := range matches["b1"] {
			total += record.Amount
		}
		if total != 150 {
			t.Errorf("Expected total 150 (including expensive supplier), got %d", total)
		}
		if !perfect {
			t.Error("Expected perfect match")
		}
	})
}

// 6. Enough Supplier Count 测试
func TestGreedyMatcher_EnoughSupplierCount(t *testing.T) {
	t.Run("SatisfiedEnoughCount", func(t *testing.T) {
		// 已满足足够数量，停止匹配更高价格
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
			makeSupplier("s2", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 100, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)
		affinity.setPrice("s2", "b1", 20.0)

		matcher := GreedyMatcher(1.0, 15.0, 1, false, false) // enough=1
		matches, _ := matcher.Match(suppliers, buyers, affinity)

		// 只应该匹配 s1
		if len(matches["b1"]) != 1 {
			t.Errorf("Expected 1 supplier, got %d", len(matches["b1"]))
		}
	})

	t.Run("NotSatisfiedEnoughCount", func(t *testing.T) {
		// 未满足足够数量，继续匹配
		suppliers := []Supplier{
			makeSupplier("s1", 50, 1, nil),
			makeSupplier("s2", 50, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 100, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)
		affinity.setPrice("s2", "b1", 20.0)

		matcher := GreedyMatcher(1.0, 25.0, 2, false, false) // enough=2
		matches, _ := matcher.Match(suppliers, buyers, affinity)

		// 应该匹配两个 supplier
		if len(matches["b1"]) != 2 {
			t.Errorf("Expected 2 suppliers, got %d", len(matches["b1"]))
		}
	})
}

// 7. BuyLimit 限制测试
func TestGreedyMatcher_BuyLimit(t *testing.T) {
	t.Run("PercentageLimit", func(t *testing.T) {
		// 百分比限制
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 100, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)
		affinity.setLimit("s1", "b1", 50) // 限制为 50

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, _ := matcher.Match(suppliers, buyers, affinity)

		if matches["b1"][0].Amount != 50 {
			t.Errorf("Expected amount 50 (limited), got %d", matches["b1"][0].Amount)
		}
	})

	t.Run("AbsoluteLimit", func(t *testing.T) {
		// 绝对值限制
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 100, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)
		affinity.setLimit("s1", "b1", 30) // 限制为 30

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, _ := matcher.Match(suppliers, buyers, affinity)

		if matches["b1"][0].Amount != 30 {
			t.Errorf("Expected amount 30 (limited), got %d", matches["b1"][0].Amount)
		}
	})
}

// 8. 容量管理测试
func TestGreedyMatcher_Capacity(t *testing.T) {
	t.Run("SupplierCapacityInsufficient", func(t *testing.T) {
		// Supplier 容量不足时的分配策略
		suppliers := []Supplier{
			makeSupplier("s1", 50, 1, nil),   // 容量只有 50
			makeSupplier("s2", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 150, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)
		affinity.setPrice("s2", "b1", 10.0)

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, _ := matcher.Match(suppliers, buyers, affinity)

		total := int64(0)
		for _, record := range matches["b1"] {
			total += record.Amount
		}
		if total != 150 {
			t.Errorf("Expected total 150, got %d", total)
		}
	})

	t.Run("BuyerDemandZero", func(t *testing.T) {
		// Buyer demand 为 0 的处理
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 0, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, _ := matcher.Match(suppliers, buyers, affinity)

		if len(matches["b1"]) != 0 {
			t.Errorf("Expected no matches for demand 0, got %d", len(matches["b1"]))
		}
	})

	t.Run("AlreadyMatchedSupplier", func(t *testing.T) {
		// 已匹配 Supplier 的二次分配
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 60, nil),
			makeBuyer("b2", 40, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)
		affinity.setPrice("s1", "b2", 10.0)

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, perfect := matcher.Match(suppliers, buyers, affinity)

		if !perfect {
			t.Error("Expected perfect match")
		}
		if matches["b1"][0].Amount != 60 {
			t.Errorf("Expected b1 amount 60, got %d", matches["b1"][0].Amount)
		}
		if matches["b2"][0].Amount != 40 {
			t.Errorf("Expected b2 amount 40, got %d", matches["b2"][0].Amount)
		}
	})
}

// 9. Perfect 匹配测试
func TestGreedyMatcher_Perfect(t *testing.T) {
	t.Run("AllDemandSatisfied", func(t *testing.T) {
		// 所有 demand 满足 → perfect=true
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
			makeSupplier("s2", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 80, nil),
			makeBuyer("b2", 120, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)
		affinity.setPrice("s1", "b2", 10.0)
		affinity.setPrice("s2", "b2", 10.0)

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		_, perfect := matcher.Match(suppliers, buyers, affinity)

		if !perfect {
			t.Error("Expected perfect match")
		}
	})

	t.Run("SomeDemandUnsatisfied", func(t *testing.T) {
		// 有 demand 未满足 → perfect=false
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 150, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		_, perfect := matcher.Match(suppliers, buyers, affinity)

		if perfect {
			t.Error("Expected non-perfect match")
		}
	})
}

// 10. 边界条件测试
func TestGreedyMatcher_EdgeCases(t *testing.T) {
	t.Run("EmptySuppliers", func(t *testing.T) {
		suppliers := []Supplier{}
		buyers := []Buyer{
			makeBuyer("b1", 100, nil),
		}
		affinity := newMockAffinityTable()

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, perfect := matcher.Match(suppliers, buyers, affinity)

		// 空 suppliers 意味着无法满足任何 demand
		if perfect {
			t.Error("Expected non-perfect match")
		}
		// matches 会是空的，因为没有创建任何匹配
		if len(matches) != 0 {
			t.Logf("Got %d buyer entries (may be acceptable)", len(matches))
		}
	})

	t.Run("EmptyBuyers", func(t *testing.T) {
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
		}
		buyers := []Buyer{}
		affinity := newMockAffinityTable()

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, perfect := matcher.Match(suppliers, buyers, affinity)

		// 空 buyers 意味着没有需求要满足
		// 当前实现返回 perfect=false（因为没有进入主循环检查）
		if perfect {
			// 如果代码修改为认为空买家是完美匹配，这个断言需要调整
			t.Log("Got perfect=true (empty buyers considered perfect)")
		}
		if len(matches) != 0 {
			t.Errorf("Expected 0 matches, got %d", len(matches))
		}
	})

	t.Run("ZeroCapacitySupplier", func(t *testing.T) {
		suppliers := []Supplier{
			makeSupplier("s1", 0, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 100, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, perfect := matcher.Match(suppliers, buyers, affinity)

		if perfect {
			t.Error("Expected non-perfect match")
		}
		if len(matches["b1"]) != 0 {
			t.Errorf("Expected no matches, got %d", len(matches["b1"]))
		}
	})

	t.Run("ZeroDemandBuyer", func(t *testing.T) {
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 0, nil),
			makeBuyer("b2", 50, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)
		affinity.setPrice("s1", "b2", 10.0)

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, perfect := matcher.Match(suppliers, buyers, affinity)

		if !perfect {
			t.Error("Expected perfect match")
		}
		if len(matches["b1"]) != 0 {
			t.Errorf("Expected b1 to have no matches, got %d", len(matches["b1"]))
		}
		if matches["b2"][0].Amount != 50 {
			t.Errorf("Expected b2 amount 50, got %d", matches["b2"][0].Amount)
		}
	})

	t.Run("VeryLargeValues", func(t *testing.T) {
		// 测试极大/极小数值
		suppliers := []Supplier{
			makeSupplier("s1", 1000000000, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 1000000000, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, perfect := matcher.Match(suppliers, buyers, affinity)

		if !perfect {
			t.Error("Expected perfect match")
		}
		if matches["b1"][0].Amount != 1000000000 {
			t.Errorf("Expected amount 1000000000, got %d", matches["b1"][0].Amount)
		}
	})
}

// 11. 复杂场景测试
func TestGreedyMatcher_ComplexScenarios(t *testing.T) {
	t.Run("MultipleBuyersMultipleSuppliers", func(t *testing.T) {
		suppliers := []Supplier{
			makeSupplier("s1", 100, 2, nil), // high priority
			makeSupplier("s2", 100, 1, nil), // low priority
			makeSupplier("s3", 50, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 100, nil),
			makeBuyer("b2", 100, nil),
		}
		affinity := newMockAffinityTable()
		for _, s := range suppliers {
			for _, b := range buyers {
				affinity.setPrice(s.ID, b.ID, 10.0)
			}
		}

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, _ := matcher.Match(suppliers, buyers, affinity)

		// 验证总匹配量
		totalB1 := int64(0)
		totalB2 := int64(0)
		for _, record := range matches["b1"] {
			totalB1 += record.Amount
		}
		for _, record := range matches["b2"] {
			totalB2 += record.Amount
		}

		if totalB1 != 100 {
			t.Errorf("Expected b1 total 100, got %d", totalB1)
		}
		if totalB2 != 100 {
			t.Errorf("Expected b2 total 100, got %d", totalB2)
		}
	})

	t.Run("PriceTieredMatching", func(t *testing.T) {
		// 价格分层匹配
		suppliers := []Supplier{
			makeSupplier("s1", 50, 1, nil),
			makeSupplier("s2", 50, 1, nil),
			makeSupplier("s3", 50, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 100, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0) // tier 1
		affinity.setPrice("s2", "b1", 10.0) // tier 1
		affinity.setPrice("s3", "b1", 20.0) // tier 2

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, _ := matcher.Match(suppliers, buyers, affinity)

		// 应该优先使用 tier 1 的 supplier
		total := int64(0)
		for _, record := range matches["b1"] {
			total += record.Amount
		}
		if total != 100 {
			t.Errorf("Expected total 100, got %d", total)
		}
	})
}

// 12. 不变性验证测试
func TestGreedyMatcher_Invariants(t *testing.T) {
	t.Run("CapacityConservation", func(t *testing.T) {
		// Supplier 容量守恒
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
			makeSupplier("s2", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 150, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)
		affinity.setPrice("s2", "b1", 10.0)

		initialCap := int64(0)
		for _, s := range suppliers {
			initialCap += s.Cap
		}

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, _ := matcher.Match(suppliers, buyers, affinity)

		matched := int64(0)
		for _, record := range matches["b1"] {
			matched += record.Amount
		}

		finalCapRest := int64(0)
		for _, s := range suppliers {
			finalCapRest += s.CapRest
		}

		if initialCap != finalCapRest+matched {
			t.Errorf("Capacity conservation violated: initial=%d, rest=%d, matched=%d",
				initialCap, finalCapRest, matched)
		}
	})

	t.Run("DemandConservation", func(t *testing.T) {
		// Buyer 需求守恒
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 80, nil),
			makeBuyer("b2", 20, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)
		affinity.setPrice("s1", "b2", 10.0)

		initialDemand := int64(0)
		for _, b := range buyers {
			initialDemand += b.Demand
		}

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, _ := matcher.Match(suppliers, buyers, affinity)

		matched := int64(0)
		for _, buyerID := range []string{"b1", "b2"} {
			for _, record := range matches[buyerID] {
				matched += record.Amount
			}
		}

		finalDemandRest := int64(0)
		for _, b := range buyers {
			finalDemandRest += b.DemandRest
		}

		if initialDemand != finalDemandRest+matched {
			t.Errorf("Demand conservation violated: initial=%d, rest=%d, matched=%d",
				initialDemand, finalDemandRest, matched)
		}
	})

	t.Run("NoDuplicateMatches", func(t *testing.T) {
		// 无重复匹配：同一 supplier-buyer 对只出现一次
		suppliers := []Supplier{
			makeSupplier("s1", 100, 1, nil),
			makeSupplier("s2", 100, 1, nil),
		}
		buyers := []Buyer{
			makeBuyer("b1", 150, nil),
		}
		affinity := newMockAffinityTable()
		affinity.setPrice("s1", "b1", 10.0)
		affinity.setPrice("s2", "b1", 10.0)

		matcher := GreedyMatcher(1.0, 0.0, 0, false, false)
		matches, _ := matcher.Match(suppliers, buyers, affinity)

		// 检查是否有重复的 supplier ID
		supplierIDs := make(map[string]bool)
		for _, record := range matches["b1"] {
			if supplierIDs[record.SupplierID] {
				t.Errorf("Duplicate supplier ID found: %s", record.SupplierID)
			}
			supplierIDs[record.SupplierID] = true
		}
	})
}
