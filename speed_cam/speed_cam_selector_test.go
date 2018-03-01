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
)

func TestNewSelector(t *testing.T) {
	selector := Create(Default())

	if selector == nil {
		t.Error("Selector is null! ")
	}
}

func TestSelection(t *testing.T) {
	selector := Create(Default())

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
	// Increase the dataSize capacity for AS17 so for this test the selected candidate is AS 1-7 (chance is much higher
	// than for other ASes)
	info := graph.nodes[*as17].info
	info.capacity = 10 * datasize.GB

	selectedCams := selector.SelectSpeedCams(graph)
	expected := 1
	if len(selectedCams) != expected {
		t.Errorf("Selected cames should be %v, but it was %v", expected, len(selectedCams))
	}

}
