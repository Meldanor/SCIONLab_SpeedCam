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

func TestEmptyGraph(t *testing.T) {
	graph := CreateEmpty(Default())

	if len(graph.nodes) != 0 || graph.size != 0 {
		t.Error("Expected empty graph, but contains ", graph.size)
	}
}

// Test simple topology:
// 1-7  <-> 1-8 <-> 1-9
//   \     <->     /
func TestLoadGraph(t *testing.T) {

	// Create graph
	connections := make(map[addr.ISD_AS][]addr.ISD_AS)

	as17, _ := addr.IAFromString("1-7")
	as18, _ := addr.IAFromString("1-8")
	as19, _ := addr.IAFromString("1-9")

	connections[*as17] = make([]addr.ISD_AS, 2)
	connections[*as18] = make([]addr.ISD_AS, 2)
	connections[*as19] = make([]addr.ISD_AS, 2)

	connections[*as17] = append(connections[*as17], *as18, *as19)
	connections[*as18] = append(connections[*as18], *as17, *as19)
	connections[*as19] = append(connections[*as19], *as17, *as18)

	graph := Load(connections, Default())

	if graph.size != 3 {
		t.Error("Expected 3 nodes, but contains ", graph.size)
	}

	n := len(graph.nodes[*as17].neighbors)
	if n != 2 {
		t.Error("Expected 2 neighbors of 1-7, but contains ", n)
	}

	n = len(graph.nodes[*as18].neighbors)
	if n != 2 {
		t.Error("Expected 2 neighbors of 1-8, but contains ", n)
	}

	n = len(graph.nodes[*as19].neighbors)
	if n != 2 {
		t.Error("Expected 2 neighbors of 1-9, but contains ", n)
	}
}
