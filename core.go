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
package main

import (
	"fmt"
	"github.com/Meldanor/SCIONLab_SpeedCam/speed_cam"
	"github.com/c2h5oh/datasize"
	"github.com/scionproto/scion/go/lib/addr"
	"time"
)

func main() {

	// Create your own config
	config := speed_cam.Default()
	// Amount of past episodes to save
	config.Episodes = 6
	config.WeightDegree = 1.0
	config.WeightCapacity = 1.0
	config.WeightSuccess = 1.0
	config.WeightActivity = 1.0
	// Additional or fewer speedCams in  a selection
	config.SpeedCamDiff = 0

	graph := speed_cam.CreateEmpty(config)

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

	selector := speed_cam.Create(config)
	selectedSpeedCams := selector.SelectSpeedCams(graph)

	for _, v := range selectedSpeedCams {
		fmt.Printf("Selected SpeedCam: %v\n", v.IsdAs)
	}
}

func AddIsdAs(isdAsString string, graph *speed_cam.NetworkGraph) addr.ISD_AS {
	as, _ := addr.IAFromString(isdAsString)

	isdAs := *as
	graph.AddIsdAs(isdAs)

	return isdAs
}

func AddBandwidth(graph *speed_cam.NetworkGraph, as *addr.ISD_AS, bandwidths []datasize.ByteSize) {

	// Arbitrary values
	date := time.Date(2018, 02, 23, 10, 0, 0, 0, time.Local)
	duration := time.Second * 30

	for _, v := range bandwidths {
		graph.AddBandwidth(as, date, duration, v)
	}
}
