// Copyright 2022 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package distscore_test

import (
	"testing"

	"github.com/someonegg/rsdmatch/distscore"
)

// mockUnifier 是一个简单的 LocationUnifier 实现，用于测试
type mockUnifier struct {
	unifyFunc func(distscore.Location, bool) distscore.Location
}

func (m mockUnifier) Unify(l distscore.Location, server bool) distscore.Location {
	if m.unifyFunc != nil {
		return m.unifyFunc(l, server)
	}
	return l
}

func (m mockUnifier) IsDeputy(l distscore.Location) bool {
	return false
}

// mockScorer 是一个简单的 DistScorer 实现，用于测试
type mockScorer struct{}

func (m mockScorer) DistScore(client, server distscore.Location) (score float32, local bool) {
	return 50.0, false
}

func TestNewComplexUnifier(t *testing.T) {
	baseUnifier := mockUnifier{}

	t.Run("NoCustomRecords", func(t *testing.T) {
		unifier := distscore.NewComplexUnifier(baseUnifier, []distscore.UnifyRecord{})
		if unifier == nil {
			t.Fatal("NewComplexUnifier returned nil")
		}

		loc := distscore.Location{ISP: "电信", Province: "北京"}
		got := unifier.Unify(loc, false)
		expected := baseUnifier.Unify(loc, false)
		if got != expected {
			t.Errorf("Unify(%+v) = %+v, want %+v (delegated to base)", loc, got, expected)
		}
	})

	t.Run("WithCustomRecords", func(t *testing.T) {
		records := []distscore.UnifyRecord{
			{
				UnifyKey: distscore.UnifyKey{
					Source: distscore.Location{ISP: "custom", Province: "custom"},
					Server: false,
				},
				UnifyVal: distscore.UnifyVal{
					Target: distscore.Location{ISP: "统一后", Province: "统一后"},
				},
			},
		}

		unifier := distscore.NewComplexUnifier(baseUnifier, records)
		if unifier == nil {
			t.Fatal("NewComplexUnifier returned nil")
		}

		loc := distscore.Location{ISP: "custom", Province: "custom"}
		got := unifier.Unify(loc, false)
		want := distscore.Location{ISP: "统一后", Province: "统一后"}
		if got != want {
			t.Errorf("Unify(%+v) = %+v, want %+v", loc, got, want)
		}
	})
}

func TestComplexUnifier_Unify_Delegation(t *testing.T) {
	baseUnifier := mockUnifier{}

	records := []distscore.UnifyRecord{
		{
			UnifyKey: distscore.UnifyKey{
				Source: distscore.Location{ISP: "special", Province: "special"},
				Server: false,
			},
			UnifyVal: distscore.UnifyVal{
				Target: distscore.Location{ISP: "special-unified", Province: "special-unified"},
			},
		},
	}

	unifier := distscore.NewComplexUnifier(baseUnifier, records)

	t.Run("CustomRecord_Match", func(t *testing.T) {
		loc := distscore.Location{ISP: "special", Province: "special"}
		got := unifier.Unify(loc, false)
		want := distscore.Location{ISP: "special-unified", Province: "special-unified"}
		if got != want {
			t.Errorf("Unify(%+v) = %+v, want %+v", loc, got, want)
		}
	})

	t.Run("DelegateToBase", func(t *testing.T) {
		loc := distscore.Location{ISP: "mobile", Province: "bj"}
		got := unifier.Unify(loc, false)
		want := loc // mockUnifier 不做转换
		if got != want {
			t.Errorf("Unify(%+v) = %+v, want %+v", loc, got, want)
		}
	})
}

func TestComplexUnifier_Unify_ServerFlag(t *testing.T) {
	baseUnifier := mockUnifier{}

	records := []distscore.UnifyRecord{
		{
			UnifyKey: distscore.UnifyKey{
				Source: distscore.Location{ISP: "test", Province: "test"},
				Server: true,
			},
			UnifyVal: distscore.UnifyVal{
				Target: distscore.Location{ISP: "server-unified", Province: "server-unified"},
			},
		},
	}

	unifier := distscore.NewComplexUnifier(baseUnifier, records)

	loc := distscore.Location{ISP: "test", Province: "test"}

	t.Run("ServerFlag_True", func(t *testing.T) {
		got := unifier.Unify(loc, true)
		want := distscore.Location{ISP: "server-unified", Province: "server-unified"}
		if got != want {
			t.Errorf("Unify(%+v, server=true) = %+v, want %+v", loc, got, want)
		}
	})

	t.Run("ServerFlag_False", func(t *testing.T) {
		got := unifier.Unify(loc, false)
		expected := baseUnifier.Unify(loc, false)
		if got != expected {
			t.Errorf("Unify(%+v, server=false) = %+v, want %+v (delegated)", loc, got, expected)
		}
	})
}

func TestComplexUnifier_Unify_MultipleRecords(t *testing.T) {
	baseUnifier := mockUnifier{}

	records := []distscore.UnifyRecord{
		{
			UnifyKey: distscore.UnifyKey{
				Source: distscore.Location{ISP: "test", Province: "test"},
				Server: false,
			},
			UnifyVal: distscore.UnifyVal{
				Target: distscore.Location{ISP: "first", Province: "first"},
			},
		},
		{
			UnifyKey: distscore.UnifyKey{
				Source: distscore.Location{ISP: "test", Province: "test"},
				Server: false,
			},
			UnifyVal: distscore.UnifyVal{
				Target: distscore.Location{ISP: "second", Province: "second"},
			},
		},
	}

	unifier := distscore.NewComplexUnifier(baseUnifier, records)

	loc := distscore.Location{ISP: "test", Province: "test"}
	got := unifier.Unify(loc, false)
	want := distscore.Location{ISP: "second", Province: "second"}

	if got != want {
		t.Errorf("Unify(%+v) = %+v, want %+v (later record should override)", loc, got, want)
	}
}

func TestComplexUnifier_IsDeputy(t *testing.T) {
	baseUnifier := mockUnifier{}

	unifier := distscore.NewComplexUnifier(baseUnifier, []distscore.UnifyRecord{})

	t.Run("AlwaysFalse", func(t *testing.T) {
		loc := distscore.Location{ISP: "电信", Province: "北京"}
		if unifier.IsDeputy(loc) {
			t.Errorf("IsDeputy(%+v) = true, want false (mock unifier)", loc)
		}
	})
}

func TestNewComplexScorer(t *testing.T) {
	baseScorer := mockScorer{}

	t.Run("NoCustomRecords", func(t *testing.T) {
		scorer := distscore.NewComplexScorer(baseScorer, []distscore.ScoreRecord{})
		if scorer == nil {
			t.Fatal("NewComplexScorer returned nil")
		}

		client := distscore.Location{ISP: "电信", Province: "北京"}
		server := distscore.Location{ISP: "电信", Province: "北京"}
		score, local := scorer.DistScore(client, server)
		expectedScore, expectedLocal := baseScorer.DistScore(client, server)
		if score != expectedScore || local != expectedLocal {
			t.Errorf("DistScore(%+v, %+v) = (%f, %v), want (%f, %v)",
				client, server, score, local, expectedScore, expectedLocal)
		}
	})

	t.Run("WithCustomRecords", func(t *testing.T) {
		records := []distscore.ScoreRecord{
			{
				ScoreKey: distscore.ScoreKey{
					Client: distscore.Location{ISP: "special", Province: "client"},
					Server: distscore.Location{ISP: "special", Province: "server"},
				},
				ScoreVal: distscore.ScoreVal{
					Score: 99.9,
					Local: true,
				},
			},
		}

		scorer := distscore.NewComplexScorer(baseScorer, records)
		if scorer == nil {
			t.Fatal("NewComplexScorer returned nil")
		}

		client := distscore.Location{ISP: "special", Province: "client"}
		server := distscore.Location{ISP: "special", Province: "server"}
		score, local := scorer.DistScore(client, server)
		if score != 99.9 || local != true {
			t.Errorf("DistScore(%+v, %+v) = (%f, %v), want (99.9, true)",
				client, server, score, local)
		}
	})
}

func TestComplexScorer_DistScore_Delegation(t *testing.T) {
	baseScorer := mockScorer{}

	records := []distscore.ScoreRecord{
		{
			ScoreKey: distscore.ScoreKey{
				Client: distscore.Location{ISP: "custom", Province: "client"},
				Server: distscore.Location{ISP: "custom", Province: "server"},
			},
			ScoreVal: distscore.ScoreVal{
				Score: 123.45,
				Local: false,
			},
		},
	}

	scorer := distscore.NewComplexScorer(baseScorer, records)

	t.Run("CustomRecord_Match", func(t *testing.T) {
		client := distscore.Location{ISP: "custom", Province: "client"}
		server := distscore.Location{ISP: "custom", Province: "server"}
		score, local := scorer.DistScore(client, server)
		if score != 123.45 || local != false {
			t.Errorf("DistScore(%+v, %+v) = (%f, %v), want (123.45, false)",
				client, server, score, local)
		}
	})

	t.Run("DelegateToBase", func(t *testing.T) {
		client := distscore.Location{ISP: "电信", Province: "北京"}
		server := distscore.Location{ISP: "电信", Province: "河北"}
		score, local := scorer.DistScore(client, server)
		expectedScore, expectedLocal := baseScorer.DistScore(client, server)
		if score != expectedScore || local != expectedLocal {
			t.Errorf("DistScore(%+v, %+v) = (%f, %v), want (%f, %v)",
				client, server, score, local, expectedScore, expectedLocal)
		}
	})
}

func TestComplexScorer_DistScore_MultipleRecords(t *testing.T) {
	baseScorer := mockScorer{}

	records := []distscore.ScoreRecord{
		{
			ScoreKey: distscore.ScoreKey{
				Client: distscore.Location{ISP: "test", Province: "client"},
				Server: distscore.Location{ISP: "test", Province: "server"},
			},
			ScoreVal: distscore.ScoreVal{
				Score: 10.0,
				Local: false,
			},
		},
		{
			ScoreKey: distscore.ScoreKey{
				Client: distscore.Location{ISP: "test", Province: "client"},
				Server: distscore.Location{ISP: "test", Province: "server"},
			},
			ScoreVal: distscore.ScoreVal{
				Score: 20.0,
				Local: true,
			},
		},
	}

	scorer := distscore.NewComplexScorer(baseScorer, records)

	client := distscore.Location{ISP: "test", Province: "client"}
	server := distscore.Location{ISP: "test", Province: "server"}
	score, local := scorer.DistScore(client, server)

	if score != 20.0 || local != true {
		t.Errorf("DistScore(%+v, %+v) = (%f, %v), want (20.0, true) (later record overrides)",
			client, server, score, local)
	}
}

func TestComplexScorer_DistScore_KeyOrder(t *testing.T) {
	baseScorer := mockScorer{}

	records := []distscore.ScoreRecord{
		{
			ScoreKey: distscore.ScoreKey{
				Client: distscore.Location{ISP: "A", Province: "X"},
				Server: distscore.Location{ISP: "B", Province: "Y"},
			},
			ScoreVal: distscore.ScoreVal{
				Score: 100.0,
				Local: false,
			},
		},
	}

	scorer := distscore.NewComplexScorer(baseScorer, records)

	// 正向匹配
	client1 := distscore.Location{ISP: "A", Province: "X"}
	server1 := distscore.Location{ISP: "B", Province: "Y"}
	score1, _ := scorer.DistScore(client1, server1)
	if score1 != 100.0 {
		t.Errorf("DistScore(%+v, %+v) = %f, want 100.0", client1, server1, score1)
	}

	// 反向不匹配（委托给基础 scorer）
	client2 := distscore.Location{ISP: "B", Province: "Y"}
	server2 := distscore.Location{ISP: "A", Province: "X"}
	score2, _ := scorer.DistScore(client2, server2)
	expectedScore, _ := baseScorer.DistScore(client2, server2)
	if score2 == 100.0 {
		t.Errorf("DistScore(%+v, %+v) = %f, should not match custom record (order sensitive)",
			client2, server2, score2)
	}
	if score2 != expectedScore {
		t.Errorf("DistScore(%+v, %+v) = %f, want %f (delegated)",
			client2, server2, score2, expectedScore)
	}
}

func TestComplexUnifier_KeyOrder(t *testing.T) {
	baseUnifier := mockUnifier{}

	records := []distscore.UnifyRecord{
		{
			UnifyKey: distscore.UnifyKey{
				Source: distscore.Location{ISP: "test", Province: "test"},
				Server: true,
			},
			UnifyVal: distscore.UnifyVal{
				Target: distscore.Location{ISP: "server-target", Province: "server-target"},
			},
		},
		{
			UnifyKey: distscore.UnifyKey{
				Source: distscore.Location{ISP: "test", Province: "test"},
				Server: false,
			},
			UnifyVal: distscore.UnifyVal{
				Target: distscore.Location{ISP: "client-target", Province: "client-target"},
			},
		},
	}

	unifier := distscore.NewComplexUnifier(baseUnifier, records)

	loc := distscore.Location{ISP: "test", Province: "test"}

	// Server=true 匹配第一条记录
	got1 := unifier.Unify(loc, true)
	want1 := distscore.Location{ISP: "server-target", Province: "server-target"}
	if got1 != want1 {
		t.Errorf("Unify(%+v, server=true) = %+v, want %+v", loc, got1, want1)
	}

	// Server=false 匹配第二条记录
	got2 := unifier.Unify(loc, false)
	want2 := distscore.Location{ISP: "client-target", Province: "client-target"}
	if got2 != want2 {
		t.Errorf("Unify(%+v, server=false) = %+v, want %+v", loc, got2, want2)
	}
}
