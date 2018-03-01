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
	"github.com/scionproto/scion/go/lib/addr"
	"testing"
)

func TestCreateEmpty(t *testing.T) {
	defaultConfig := Default()
	inspector := CreateEmptyGraph(defaultConfig)

	if inspector.graph.size != 0 {
		t.Error("Expected empty graph, but contains ", inspector.graph.size)
	}
}

func TestAddSingleIsdAs(t *testing.T) {
	defaultConfig := Default()
	inspector := CreateEmptyGraph(defaultConfig)
	inspector.HandlePathRequest("1-7")

	if inspector.graph.size != 1 {
		t.Error("Expected graph with one AS, but contains ", inspector.graph.size)
	}
}

func TestAddAndConnect(t *testing.T) {
	defaultConfig := Default()
	inspector := CreateEmptyGraph(defaultConfig)
	inspector.HandlePathRequest("1-1 1>1 1-5")

	if inspector.graph.size != 2 {
		t.Error("Expected graph with two ASes, but contains ", inspector.graph.size)
	}

	isdAs1, _ := addr.IAFromString("1-1")
	isdAs2, _ := addr.IAFromString("1-5")

	_, connected := inspector.graph.nodes[*isdAs1].neighbors[*isdAs2]

	if !connected {
		t.Error("Expected ASes connecte, but they werent")
	}
}
