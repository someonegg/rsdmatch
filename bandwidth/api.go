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

type View struct {
	View      string  `json:"view"`
	ISP       string  `json:"isp"`
	Province  string  `json:"province"`
	Bandwidth float64 `json:"bw"` // Gbps
}

type Alloc struct {
	Name   string  `json:"name"`
	Groups []Group `json:"groups"`
}

type Group struct {
	Nodes       []string `json:"nodes"`
	NodesWeight []int64  `json:"nodesWeight"` // Mbps
}

const (
	DefaultEnoughNodeCount   = 5
	DefaultRemoteAccessScore = 50.0
	DefaultRejectScore       = 80.0
	DefaultRemoteAccessLimit = 0.1
)

type Matcher struct {
	EnoughNodeCount   *int     `json:"ecn"`
	RemoteAccessScore *float32 `json:"ras"`
	RejectScore       *float32 `json:"rjs"`
	RemoteAccessLimit *float32 `json:"ral"`

	// When set, matcher will auto-scale the views's bandwidth to fit nodes's.
	AutoScale bool `json:"as"`

	Verbose bool `json:"vv"`

	ecn int
	ras float32
	rjs float32
	ral float32
}

type Summary struct {
	NodesCount       int     `json:"nodes"`
	ViewsCount       int     `json:"views"`
	NodesBandwidth   float64 `json:"nodes_bw"`
	ViewsBandwidth   float64 `json:"views_bw"`
	BandwidthNeeds   float64 `json:"bw_needs"`
	BandwidthRemains float64 `json:"bw_remains"`
}
