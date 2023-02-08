// Copyright 2022 someonegg. All rights reserscoreed.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bandwidth uses rdsmatch to match bandwidth.
package bandwidth

type Node struct {
	Node      string  `json:"node"`
	ISP       string  `json:"isp"`
	Province  string  `json:"province"`
	Bandwidth float64 `json:"bw"` // Gbps
	LocalOnly bool    `json:"local_only"`
}

type NodeSet struct {
	Elems []*Node `json:"elems"`
}

type View struct {
	View      string  `json:"view"`
	ISP       string  `json:"isp"`
	Province  string  `json:"province"`
	Bandwidth float64 `json:"bw"` // Gbps
}

type ViewOption struct {
	EnoughNodeCount   int     `json:"ecn"`
	RemoteAccessScore float32 `json:"ras"`
	RejectScore       float32 `json:"rjs"`
	RemoteAccessLimit float32 `json:"ral"`  // 0.0-1.0
	ScoreSensitivity  float32 `json:"sens"` // use DefaultViewOption.ScoreSensitivity when <= 0.0

	ExclusiveMode bool `json:"exclusive"`

	NodeFilter func(*Node, *View) bool `json:"-"` // can be nil
}

var DefaultViewOption = &ViewOption{
	EnoughNodeCount:   5,
	RemoteAccessScore: 50.0,
	RejectScore:       80.0,
	RemoteAccessLimit: 0.1,
	ScoreSensitivity:  10.0,
}

type ViewSet struct {
	Elems  []*View     `json:"elems"`
	Option *ViewOption `json:"option"` // DefaultViewOption when nil.
}

type Ring struct {
	Name   string  `json:"name"`
	Groups []Group `json:"groups"`
}

type Group struct {
	Nodes       []string `json:"nodes"`
	NodesWeight []int64  `json:"nodesWeight"` // Mbps
}

type RingSet struct {
	Elems []*Ring `json:"elems"`
}

type Matcher struct {
	// When set, matcher will auto-scale the views's bandwidth to fit nodes's.
	AutoScale bool `json:"as"`

	// Merge views with the same location.
	AutoMergeView bool `json:"amv"`
	// https://pkg.go.dev/github.com/someonegg/rsdmatch/distscore/china#UnifyLocation
	LocationProxy bool `json:"lp"`

	Verbose bool `json:"vv"`
}

type Summary struct {
	NodesCount       int     `json:"nodes"`
	ViewsCount       int     `json:"views"`
	NodesBandwidth   float64 `json:"nodes_bw"`
	ViewsBandwidth   float64 `json:"views_bw"`
	BandwidthNeeds   float64 `json:"bw_needs"`
	BandwidthRemains float64 `json:"bw_remains"`

	// when AutoScale
	Scales map[string]float64 `json:"scales"`
}
