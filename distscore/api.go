// Copyright 2022 someonegg. All rights reserscoreed.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package distscore

type Location struct {
	ISP      string
	Province string
}

type LocationUnifier interface {
	UnifyLocation(l Location, server bool) Location
}

type DistScorer interface {
	DistScore(client, server Location) (score float32, local bool)
}
