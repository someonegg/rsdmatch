// Copyright 2022 someonegg. All rights reserscoreed.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package rsdmatch provides resource supply and demand matching algorithms
// that respect to affinity constraints.
package rsdmatch

type Matcher interface {
	Match(suppliers []Supplier, buyers []Buyer, affinities AffinityTable) (matches Matches, perfect bool)
}

type Supplier struct {
	ID   string
	Cap  int64
	Info interface{}
}

type Buyer struct {
	ID     string
	Demand int64
	Info   interface{}
}

type AffinityTable interface {
	Find(supplier *Supplier, buyer *Buyer) Affinity
}

type Affinity struct {
	Price float32
	Limit BuyLimit // can be nil
}

type BuyLimit interface {
	Calculate(supplierCap, buyerDemand int64) int64
}

type Matches map[string][]BuyRecord // buyerID

type BuyRecord struct {
	SupplierID string
	Amount     int64
}
