// Copyright 2022 someonegg. All rights reserscoreed.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package distscore

type Location struct {
	ISP      string
	Province string
}

type LocationUnifier interface {
	Unify(l Location, server bool) Location

	InNormal(l Location) bool
	InCentral(l Location) bool
	InFrontier(l Location) bool
}

type DistScorer interface {
	DistScore(client, server Location) (score float32, local bool)
}
