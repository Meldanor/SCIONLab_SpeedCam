// Copyright 2018 ETH Zurich, OvGU Magdeburg
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package for a bandwidth regulation algorithm named SpeedCam. Further information here: URL_TO_THESIS
package speed_cam

import (
	"github.com/c2h5oh/datasize"
	"github.com/scionproto/scion/go/lib/addr"
	"testing"
	"time"
)

// Playground to test the speedcam algorithm without setting up a SCION network
func TestSpeedCamPlayground(t *testing.T) {

	// Create your own config
	config := Default()
	// Amount of past episodes to save
	config.episodes = 6
	config.weightDegree = 1.0
	config.weightCapacity = 1.0
	config.weightSuccess = 1.0
	config.weightActivity = 1.0
	// Additional or fewer speedCams in  a selection
	config.speedCamDiff = 0

	graph := CreateEmpty(config)

	// Create ASes and add them to the graph
	as17 := AddIsdAs("1-7", graph)
	as18 := AddIsdAs("1-8", graph)
	as19 := AddIsdAs("1-9", graph)

	// Connect ASes
	graph.ConnectIsdAses(as17, as18)
	graph.ConnectIsdAses(as17, as19)

	// Add artificial bandwidth (Which was recorded in the path)
	AddBandwidth(graph, &as17, []datasize.ByteSize{2 * datasize.GB, 3 * datasize.GB})
	AddBandwidth(graph, &as18, []datasize.ByteSize{4 * datasize.GB, 10 * datasize.GB})

	selector := Create(config)
	selectedSpeedCams := selector.SelectSpeedCams(graph)

	for _, v := range selectedSpeedCams {
		t.Logf("Selected SpeedCam: %v", v.isdAs)
	}
}

func AddIsdAs(isdAsString string, graph *NetworkGraph) addr.ISD_AS {
	as, _ := addr.IAFromString(isdAsString)

	isdAs := *as
	graph.AddIsdAs(isdAs)

	return isdAs
}

func AddBandwidth(graph *NetworkGraph, as *addr.ISD_AS, bandwidths []datasize.ByteSize) {
	node := graph.nodes[*as]

	// Arbitrary values
	date := time.Date(2018, 02, 23, 10, 0, 0, 0, time.Local)
	duration := time.Second * 30

	for _, v := range bandwidths {
		node.info.AddActivity(date, duration, v)
	}
}
