// Copyright 2022 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bandwidth

import (
	"math"
	"testing"

	"github.com/someonegg/rsdmatch"
	ds "github.com/someonegg/rsdmatch/distscore"
	"github.com/someonegg/rsdmatch/distscore/china"
)

// 辅助函数
func makeNode(id, isp, province string, bw float64, priority float64) *Node {
	return &Node{
		Node:      id,
		ISP:       isp,
		Province:  province,
		Bandwidth: bw,
		Priority:  priority,
		LocalOnly: false,
	}
}

func makeView(id, isp, province string, bw float64) *View {
	return &View{
		View:      id,
		ISP:       isp,
		Province:  province,
		Bandwidth: bw,
	}
}

// 1. 测试 genSuppliers
func TestGenSuppliers(t *testing.T) {
	unifier := china.NewLocationUnifier(false)

	t.Run("NormalNodes", func(t *testing.T) {
		nodes := NodeSet{
			Elems: []*Node{
				makeNode("node1", "电信", "北京", 1.0, 1.0),
				makeNode("node2", "联通", "上海", 2.0, 0.5),
			},
		}

		suppliers, count, _ := genSuppliers(unifier, nodes)

		if count != 2 {
			t.Errorf("Expected count 2, got %d", count)
		}

		// 检查容量转换：Gbps -> Mbps (bwUnit=100)
		// node1: 1.0 Gbps = 1000 Mbps, /100 = 10 units
		if suppliers.Elems[0].Cap != 10 {
			t.Errorf("Expected node1 cap 10, got %d", suppliers.Elems[0].Cap)
		}

		// 检查优先级转换：priority*1000 + 1
		if suppliers.Elems[0].Priority != 1001 {
			t.Errorf("Expected node1 priority 1001, got %d", suppliers.Elems[0].Priority)
		}

		// 检查排序：按 Node ID 排序
		if suppliers.Elems[0].ID != "node1" || suppliers.Elems[1].ID != "node2" {
			t.Error("Expected suppliers sorted by ID")
		}
	})

	t.Run("IncompleteNode", func(t *testing.T) {
		nodes := NodeSet{
			Elems: []*Node{
				makeNode("node1", "", "北京", 1.0, 1.0), // ISP 为空
				makeNode("node2", "电信", "", 1.0, 1.0), // Province 为空
			},
		}

		suppliers, _, _ := genSuppliers(unifier, nodes)

		// 不完整的节点容量应该为 0
		if suppliers.Elems[0].Cap != 0 {
			t.Errorf("Expected incomplete node1 cap 0, got %d", suppliers.Elems[0].Cap)
		}
		if suppliers.Elems[1].Cap != 0 {
			t.Errorf("Expected incomplete node2 cap 0, got %d", suppliers.Elems[1].Cap)
		}
	})

	t.Run("PriorityConversion", func(t *testing.T) {
		nodes := NodeSet{
			Elems: []*Node{
				makeNode("node1", "电信", "北京", 1.0, 1.234), // 保留3位小数
			},
		}

		suppliers, _, _ := genSuppliers(unifier, nodes)

		// Priority = 1.234 * 1000 + 1 = 1235
		expected := int64(math.Floor(1.234*1000)) + 1
		if suppliers.Elems[0].Priority != expected {
			t.Errorf("Expected priority %d, got %d", expected, suppliers.Elems[0].Priority)
		}
	})
}

// 2. 测试 genBuyerss
func TestGenBuyerss(t *testing.T) {
	unifier := china.NewLocationUnifier(false)

	t.Run("NormalViews", func(t *testing.T) {
		viewss := []ViewSet{
			{
				Elems: []*View{
					makeView("view1", "电信", "北京", 1.0),
					makeView("view2", "联通", "上海", 2.0),
				},
			},
		}

		buyerss, count, _ := genBuyerss(unifier, viewss, nil)

		if count != 2 {
			t.Errorf("Expected count 2, got %d", count)
		}
		if len(buyerss) != 1 {
			t.Errorf("Expected 1 buyer set, got %d", len(buyerss))
		}

		// 检查需求转换：Gbps -> Mbps (bwUnit=100)
		// view1: 1.0 Gbps = 1000 Mbps, /100 = 10 units
		// view2: 2.0 Gbps = 2000 Mbps, /100 = 20 units
		if buyerss[0].Elems[0].Demand != 20 {
			t.Errorf("Expected view2 demand 20, got %d", buyerss[0].Elems[0].Demand)
		}

		// 检查排序：按 Demand 降序
		if buyerss[0].Elems[0].Demand < buyerss[0].Elems[1].Demand {
			t.Error("Expected buyers sorted by demand descending")
		}
	})

	t.Run("WithScale", func(t *testing.T) {
		viewss := []ViewSet{
			{
				Elems: []*View{
					makeView("view1", "电信", "北京", 1.0),
				},
			},
		}

		scale := map[string]float64{"电信": 0.5}
		buyerss, _, _ := genBuyerss(unifier, viewss, scale)

		// Demand = 1.0 * 0.5 * 1000 / 100 = 5
		if buyerss[0].Elems[0].Demand != 5 {
			t.Errorf("Expected scaled demand 5, got %d", buyerss[0].Elems[0].Demand)
		}
	})

	t.Run("DefaultOption", func(t *testing.T) {
		viewss := []ViewSet{
			{
				Elems: []*View{
					makeView("view1", "电信", "北京", 1.0),
				},
				Option: nil, // 使用默认选项
			},
		}

		buyerss, _, _ := genBuyerss(unifier, viewss, nil)

		// 应该使用 DefaultViewOption
		if buyerss[0].Option == nil {
			t.Error("Expected option to be set to DefaultViewOption")
		}
		if buyerss[0].Option != DefaultViewOption {
			t.Error("Expected option to be DefaultViewOption")
		}
	})
}

// 3. 测试 mergeBuyers
func TestMergeBuyers(t *testing.T) {
	unifier := china.NewLocationUnifier(false)

	t.Run("MergeSameLocation", func(t *testing.T) {
		raws := []rsdmatch.Buyer{
			{ID: "view1", Demand: 10, Info: makeView("view1", "电信", "北京", 1.0)},
			{ID: "view2", Demand: 20, Info: makeView("view2", "电信", "北京", 2.0)}, // 相同位置
			{ID: "view3", Demand: 30, Info: makeView("view3", "联通", "上海", 3.0)},
		}

		merged, buyerViews := mergeBuyers(unifier, raws)

		// 应该合并成 2 个 buyer（北京-电信 和 上海-联通）
		if len(merged) != 2 {
			t.Errorf("Expected 2 merged buyers, got %d", len(merged))
		}

		// 北京-电信的总需求应该是 10+20=30
		beijingBuyer := merged[0]
		if beijingBuyer.Demand != 30 {
			t.Errorf("Expected Beijing demand 30, got %d", beijingBuyer.Demand)
		}

		// 检查 buyerViews
		views := buyerViews[beijingBuyer.ID]
		if len(views) != 2 {
			t.Errorf("Expected 2 views for Beijing, got %d", len(views))
		}
	})

	t.Run("SortByDemand", func(t *testing.T) {
		raws := []rsdmatch.Buyer{
			{ID: "view1", Demand: 10, Info: makeView("view1", "电信", "北京", 1.0)},
			{ID: "view2", Demand: 30, Info: makeView("view2", "联通", "上海", 3.0)},
			{ID: "view3", Demand: 20, Info: makeView("view3", "移动", "广州", 2.0)},
		}

		merged, _ := mergeBuyers(unifier, raws)

		// 应该按 Demand 降序排列
		for i := 0; i < len(merged)-1; i++ {
			if merged[i].Demand < merged[i+1].Demand {
				t.Error("Expected merged buyers sorted by demand descending")
			}
		}
	})
}

// 4. 测试 affinityTable.Find
func TestAffinityTable_Find(t *testing.T) {
	unifier := china.NewLocationUnifier(false)
	scorer := china.NewDistScorer()

	t.Run("LocalOnly_Reject", func(t *testing.T) {
		// Case 1: LocalOnly=true + 非本地 → Limit=0
		option := &ViewOption{
			RemoteAccessScore: 50.0,
			RejectScore:       80.0,
			RemoteAccessLimit: 0.1,
		}
		table := newAffinityTable(option, unifier, scorer)

		node := makeNode("node1", "电信", "北京", 1.0, 1.0)
		node.LocalOnly = true
		view := makeView("view1", "联通", "上海", 1.0) // 不同位置

		supplier := &rsdmatch.Supplier{Info: node}
		buyer := &rsdmatch.Buyer{Info: view}

		affinity := table.Find(supplier, buyer)

		if affinity.Limit == nil {
			t.Error("Expected Limit to be set for non-local LocalOnly node")
		}
		limit := affinity.Limit.(nodePercentLimit)
		if limit != 0.0 {
			t.Errorf("Expected limit 0.0, got %f", limit)
		}
	})

	t.Run("NodeFilter_Reject", func(t *testing.T) {
		// Case 2: NodeFilter 返回 false → Limit=0
		option := &ViewOption{
			NodeFilter: func(n *Node, v *View) bool {
				return n.Node != "node1" // 拒绝 node1
			},
		}
		table := newAffinityTable(option, unifier, scorer)

		node := makeNode("node1", "电信", "北京", 1.0, 1.0)
		view := makeView("view1", "电信", "北京", 1.0)

		supplier := &rsdmatch.Supplier{Info: node}
		buyer := &rsdmatch.Buyer{Info: view}

		affinity := table.Find(supplier, buyer)

		if affinity.Limit == nil {
			t.Error("Expected Limit to be set when NodeFilter returns false")
		}
		limit := affinity.Limit.(nodePercentLimit)
		if limit != 0.0 {
			t.Errorf("Expected limit 0.0, got %f", limit)
		}
	})

	t.Run("Near_NoLimit", func(t *testing.T) {
		// Case 3: score < ras → Limit=nil (无限制)
		option := &ViewOption{
			RemoteAccessScore: 50.0, // ras
			RejectScore:       80.0,
			RemoteAccessLimit: 0.1,
		}
		table := newAffinityTable(option, unifier, scorer)

		node := makeNode("node1", "电信", "北京", 1.0, 1.0)
		view := makeView("view1", "电信", "北京", 1.0) // 相同位置，score=10 < ras

		supplier := &rsdmatch.Supplier{Info: node}
		buyer := &rsdmatch.Buyer{Info: view}

		affinity := table.Find(supplier, buyer)

		if affinity.Limit != nil {
			t.Error("Expected Limit=nil for near access (score < ras)")
		}
	})

	t.Run("Remote_WithLimit", func(t *testing.T) {
		// Case 4: score < rjs → Limit=ral
		option := &ViewOption{
			RemoteAccessScore: 20.0, // ras
			RejectScore:       80.0, // rjs
			RemoteAccessLimit: 0.5,  // ral
		}
		table := newAffinityTable(option, unifier, scorer)

		node := makeNode("node1", "电信", "新疆", 1.0, 1.0)
		view := makeView("view1", "电信", "北京", 1.0) // 远距离，score 应该在 20-80 之间

		supplier := &rsdmatch.Supplier{Info: node}
		buyer := &rsdmatch.Buyer{Info: view}

		affinity := table.Find(supplier, buyer)

		if affinity.Limit == nil {
			t.Error("Expected Limit to be set for remote access")
		}
		limit := affinity.Limit.(nodePercentLimit)
		if limit != 0.5 {
			t.Errorf("Expected limit 0.5, got %f", limit)
		}
	})

	t.Run("Reject_ZeroLimit", func(t *testing.T) {
		// Case 5: score >= rjs → Limit=0
		option := &ViewOption{
			RemoteAccessScore: 20.0,
			RejectScore:       30.0, // rjs，很低的拒绝分数
			RemoteAccessLimit: 0.1,
		}
		table := newAffinityTable(option, unifier, scorer)

		node := makeNode("node1", "联通", "新疆", 1.0, 1.0)
		view := makeView("view1", "电信", "西藏", 1.0) // 非常远，可能 score >= 30

		supplier := &rsdmatch.Supplier{Info: node}
		buyer := &rsdmatch.Buyer{Info: view}

		affinity := table.Find(supplier, buyer)

		if affinity.Limit == nil {
			t.Error("Expected Limit to be set for rejected access")
		}
		limit := affinity.Limit.(nodePercentLimit)
		if limit != 0.0 {
			t.Errorf("Expected limit 0.0 for rejected, got %f", limit)
		}
	})
}

// 5. 测试 AutoScale
func TestAutoScale(t *testing.T) {
	unifier := china.NewLocationUnifier(false)
	scorer := china.NewDistScorer()

	t.Run("CalculateScale", func(t *testing.T) {
		nodes := NodeSet{
			Elems: []*Node{
				makeNode("node1", "电信", "北京", 1.0, 1.0),
				makeNode("node2", "电信", "上海", 1.0, 1.0),
			},
		}

		viewss := []ViewSet{
			{
				Elems: []*View{
					makeView("view1", "电信", "北京", 2.0), // 需求大于供给
					makeView("view2", "电信", "上海", 2.0),
				},
			},
		}

		matcher := &Matcher{
			AutoScale: true,
			Unifier:   unifier,
			Scorer:    scorer,
		}

		ringss, summ := matcher.Match(nodes, viewss)

		// 检查 scale 计算：has = 20, needs = 40, scale = 0.5
		if summ.Scales == nil {
			t.Error("Expected Scales to be set")
		}
		scale, ok := summ.Scales["电信"]
		if !ok {
			t.Error("Expected scale for 电信")
		}
		if scale != 0.5 {
			t.Errorf("Expected scale 0.5, got %f", scale)
		}

		if len(ringss) != 1 {
			t.Errorf("Expected 1 RingSet, got %d", len(ringss))
		}
	})

	t.Run("ScaleWithMinMax", func(t *testing.T) {
		nodes := NodeSet{
			Elems: []*Node{
				makeNode("node1", "电信", "北京", 1.0, 1.0),
			},
		}

		viewss := []ViewSet{
			{
				Elems: []*View{
					makeView("view1", "电信", "北京", 10.0), // 需求远大于供给
				},
			},
		}

		minScale := 0.3
		maxScale := 0.7
		matcher := &Matcher{
			AutoScale:    true,
			AutoScaleMin: &minScale,
			AutoScaleMax: &maxScale,
			Unifier:      unifier,
			Scorer:       scorer,
		}

		_, summ := matcher.Match(nodes, viewss)

		// has = 10, needs = 100, scale = 0.1，但应该被限制在 min=0.3
		scale := summ.Scales["电信"]
		if scale != 0.3 {
			t.Errorf("Expected scale 0.3 (min), got %f", scale)
		}
	})
}

// 6. 测试完整匹配流程
func TestMatcher_Match(t *testing.T) {
	unifier := china.NewLocationUnifier(false)
	scorer := china.NewDistScorer()

	t.Run("SimpleMatch", func(t *testing.T) {
		nodes := NodeSet{
			Elems: []*Node{
				makeNode("node1", "电信", "北京", 1.0, 1.0),
				makeNode("node2", "电信", "上海", 1.0, 1.0),
			},
		}

		viewss := []ViewSet{
			{
				Elems: []*View{
					makeView("view1", "电信", "北京", 0.5),
					makeView("view2", "电信", "上海", 0.5),
				},
			},
		}

		matcher := &Matcher{
			Unifier: unifier,
			Scorer:  scorer,
		}

		ringss, summ := matcher.Match(nodes, viewss)

		// 验证 RingSet
		if len(ringss) != 1 {
			t.Errorf("Expected 1 RingSet, got %d", len(ringss))
		}

		// 验证 Summary
		if summ.NodesCount != 2 {
			t.Errorf("Expected NodesCount 2, got %d", summ.NodesCount)
		}
		if summ.ViewsCount != 2 {
			t.Errorf("Expected ViewsCount 2, got %d", summ.ViewsCount)
		}

		// 验证带宽单位转换：Gbps -> ? (应该是 Gbps)
		// NodesBandwidth = 2.0 Gbps
		if summ.NodesBandwidth != 2.0 {
			t.Errorf("Expected NodesBandwidth 2.0, got %f", summ.NodesBandwidth)
		}
		if summ.ViewsBandwidth != 1.0 {
			t.Errorf("Expected ViewsBandwidth 1.0, got %f", summ.ViewsBandwidth)
		}
	})

	t.Run("AutoMergeView", func(t *testing.T) {
		nodes := NodeSet{
			Elems: []*Node{
				makeNode("node1", "电信", "北京", 2.0, 1.0),
			},
		}

		viewss := []ViewSet{
			{
				Elems: []*View{
					makeView("view1", "电信", "北京", 0.5),
					makeView("view2", "电信", "北京", 0.5), // 相同位置，应该合并
				},
			},
		}

		matcher := &Matcher{
			AutoMergeView: true,
			Unifier:       unifier,
			Scorer:        scorer,
		}

		ringss, _ := matcher.Match(nodes, viewss)

		// 虽然合并了，但 genRings 会为每个原始 view 创建一个 ring
		// 所以应该有 2 个 rings
		rings := ringss[0].Elems
		if len(rings) != 2 {
			t.Errorf("Expected 2 rings (one per original view), got %d", len(rings))
		}

		// 验证两个 rings 都引用同一个 supplier group
		if len(rings[0].Groups) != 1 || len(rings[1].Groups) != 1 {
			t.Error("Expected each ring to have 1 group")
		}
	})
}

// 7. 测试 genRings
func TestGenRings(t *testing.T) {
	t.Run("GenerateRings", func(t *testing.T) {
		matches := rsdmatch.Matches{
			"view1": {{SupplierID: "node1", Amount: 10}},
			"view2": {{SupplierID: "node1", Amount: 5}, {SupplierID: "node2", Amount: 5}},
		}

		buyerViews := map[string][]string{
			"view2": {"view2a", "view2b"}, // view2 被拆分
		}

		buyerDemand := map[string]int64{
			"view1": 1000,
			"view2": 500,
		}

		ringSet := genRings(matches, buyerViews, buyerDemand)

		// view1 应该生成 1 个 ring
		// view2 应该生成 2 个 rings（view2a 和 view2b）
		if len(ringSet.Elems) != 3 {
			t.Errorf("Expected 3 rings, got %d", len(ringSet.Elems))
		}

		// 检查 NodesWeight 转换：Amount * bwUnit
		ring := ringSet.Elems[0]
		if ring.Groups[0].NodesWeight[0] != 10*100 { // 10 * 100
			t.Errorf("Expected NodesWeight 1000, got %d", ring.Groups[0].NodesWeight[0])
		}
	})

	t.Run("SortByName", func(t *testing.T) {
		matches := rsdmatch.Matches{
			"view2": {{SupplierID: "node2", Amount: 5}},
			"view1": {{SupplierID: "node1", Amount: 10}},
		}

		ringSet := genRings(matches, nil, nil)

		// 应该按 Name 排序
		if ringSet.Elems[0].Name > ringSet.Elems[1].Name {
			t.Error("Expected rings sorted by Name ascending")
		}
	})
}

// 8. 测试边界条件
func TestEdgeCases(t *testing.T) {
	t.Run("EmptyNodeSet", func(t *testing.T) {
		nodes := NodeSet{Elems: []*Node{}}
		viewss := []ViewSet{}

		matcher := &Matcher{
			Unifier: china.NewLocationUnifier(false),
			Scorer:  china.NewDistScorer(),
		}

		ringss, summ := matcher.Match(nodes, viewss)

		if len(ringss) != 0 {
			t.Errorf("Expected 0 RingSets, got %d", len(ringss))
		}
		if summ.NodesCount != 0 {
			t.Errorf("Expected NodesCount 0, got %d", summ.NodesCount)
		}
	})

	t.Run("EmptyViewSet", func(t *testing.T) {
		nodes := NodeSet{
			Elems: []*Node{
				makeNode("node1", "电信", "北京", 1.0, 1.0),
			},
		}
		viewss := []ViewSet{}

		matcher := &Matcher{
			Unifier: china.NewLocationUnifier(false),
			Scorer:  china.NewDistScorer(),
		}

		ringss, _ := matcher.Match(nodes, viewss)

		if len(ringss) != 0 {
			t.Errorf("Expected 0 RingSets, got %d", len(ringss))
		}
	})

	t.Run("ZeroBandwidth", func(t *testing.T) {
		nodes := NodeSet{
			Elems: []*Node{
				makeNode("node1", "电信", "北京", 0.0, 1.0), // 0 带宽
			},
		}

		viewss := []ViewSet{
			{
				Elems: []*View{
					makeView("view1", "电信", "北京", 0.0), // 0 带宽
				},
			},
		}

		matcher := &Matcher{
			Unifier: china.NewLocationUnifier(false),
			Scorer:  china.NewDistScorer(),
		}

		_, summ := matcher.Match(nodes, viewss)

		if summ.NodesBandwidth != 0 {
			t.Errorf("Expected NodesBandwidth 0, got %f", summ.NodesBandwidth)
		}
		if summ.ViewsBandwidth != 0 {
			t.Errorf("Expected ViewsBandwidth 0, got %f", summ.ViewsBandwidth)
		}
	})
}

// 9. 测试 ViewOption.Fix
func TestViewOption_Fix(t *testing.T) {
	t.Run("FixInvalidRAL", func(t *testing.T) {
		option := &ViewOption{
			RemoteAccessLimit: 1.5, // 无效值，应该被修正
		}

		option.Fix()

		if option.RemoteAccessLimit != DefaultViewOption.RemoteAccessLimit {
			t.Errorf("Expected RemoteAccessLimit to be fixed to default, got %f", option.RemoteAccessLimit)
		}
	})

	t.Run("FixInvalidSens", func(t *testing.T) {
		option := &ViewOption{
			ScoreSensitivity: -1.0, // 无效值，应该被修正
		}

		option.Fix()

		if option.ScoreSensitivity != DefaultViewOption.ScoreSensitivity {
			t.Errorf("Expected ScoreSensitivity to be fixed to default, got %f", option.ScoreSensitivity)
		}
	})

	t.Run("ValidOption", func(t *testing.T) {
		option := &ViewOption{
			RemoteAccessLimit: 0.5,
			ScoreSensitivity:  25.0,
		}

		originalRAL := option.RemoteAccessLimit
		originalSens := option.ScoreSensitivity

		option.Fix()

		if option.RemoteAccessLimit != originalRAL {
			t.Error("Valid RemoteAccessLimit should not be changed")
		}
		if option.ScoreSensitivity != originalSens {
			t.Error("Valid ScoreSensitivity should not be changed")
		}
	})
}

// 10. 测试 nodePercentLimit
func TestNodePercentLimit(t *testing.T) {
	t.Run("CalculateLimit", func(t *testing.T) {
		limit := nodePercentLimit(0.5) // 50%

		// supplierCap = 100, buyerDemand = 200
		// limit = 100 * 0.5 = 50
		result := limit.Calculate(100, 200)

		if result != 50 {
			t.Errorf("Expected limit 50, got %d", result)
		}
	})

	t.Run("CeilingRounding", func(t *testing.T) {
		limit := nodePercentLimit(0.333) // 33.3%

		// supplierCap = 100
		// limit = Ceil(100 * 0.333) = Ceil(33.3) = 34
		result := limit.Calculate(100, 200)

		expected := int64(math.Ceil(100 * 0.333))
		if result != expected {
			t.Errorf("Expected limit %d, got %d", expected, result)
		}
	})
}

// 11. 测试默认值
func TestDefaults(t *testing.T) {
	t.Run("DefaultViewOptionValues", func(t *testing.T) {
		if DefaultViewOption.EnoughNodeCount != 5 {
			t.Errorf("Expected default EnoughNodeCount 5, got %d", DefaultViewOption.EnoughNodeCount)
		}
		if DefaultViewOption.RemoteAccessScore != 50.0 {
			t.Errorf("Expected default RemoteAccessScore 50.0, got %f", DefaultViewOption.RemoteAccessScore)
		}
		if DefaultViewOption.RejectScore != 80.0 {
			t.Errorf("Expected default RejectScore 80.0, got %f", DefaultViewOption.RejectScore)
		}
		if DefaultViewOption.RemoteAccessLimit != 0.1 {
			t.Errorf("Expected default RemoteAccessLimit 0.1, got %f", DefaultViewOption.RemoteAccessLimit)
		}
		if DefaultViewOption.ScoreSensitivity != 10.0 {
			t.Errorf("Expected default ScoreSensitivity 10.0, got %f", DefaultViewOption.ScoreSensitivity)
		}
	})
}

// 12. 测试 Deputy 判断
func TestDeputy(t *testing.T) {
	unifier := china.NewLocationUnifier(false)

	t.Run("CentralProvince", func(t *testing.T) {
		// 北京是 Central 地区，应该是 Deputy
		location := ds.Location{ISP: "电信", Province: "北京"}

		if !unifier.IsDeputy(location) {
			t.Error("Expected Beijing to be Deputy")
		}
	})

	t.Run("FrontierProvince", func(t *testing.T) {
		// 新疆是边疆地区，不应该是 Deputy
		location := ds.Location{ISP: "电信", Province: "新疆"}

		if unifier.IsDeputy(location) {
			t.Error("Expected Xinjiang to not be Deputy")
		}
	})
}
