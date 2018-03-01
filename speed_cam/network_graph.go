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
	"errors"
	"fmt"
	"github.com/c2h5oh/datasize"
	"github.com/scionproto/scion/go/lib/addr"
	"time"
)

// A symmetric graph representing the topology of a SCION network
type NetworkGraph struct {
	nodes  map[addr.ISD_AS]networkNode
	size   uint32
	config *SpeedCamConfig
}

// Creates an empty graph without any ASes inside
func CreateEmpty(config *SpeedCamConfig) *NetworkGraph {
	graph := new(NetworkGraph)
	graph.nodes = make(map[addr.ISD_AS]networkNode)
	graph.size = 0
	graph.config = config
	return graph
}

// Creates graph from a connection list. The information values are default ones
func Load(connections map[addr.ISD_AS][]addr.ISD_AS, config *SpeedCamConfig) *NetworkGraph {
	graph := CreateEmpty(config)

	for k, neighbors := range connections {
		graph.AddIsdAs(k)

		for _, neighbor := range neighbors {
			graph.ConnectIsdAses(k, neighbor)
		}
	}

	return graph
}

// Adds an AS to the graph without connections or information about it.
// Duplicate ISD-ASes are permitted and will result in an error.
func (graph *NetworkGraph) AddIsdAs(isdAs addr.ISD_AS) error {
	_, exists := graph.nodes[isdAs]
	// Do not add an existing AS twice
	if exists {
		return errors.New(fmt.Sprintf("Duplicate ISD-AS %v added to graph", isdAs))
	}
	node := new(networkNode)
	node.IsdAs = isdAs
	node.info = NewInfo(isdAs, graph.config)
	node.neighbors = make(map[addr.ISD_AS]networkNode)
	graph.nodes[isdAs] = *node
	graph.size++
	return nil
}

// Connects two ASes with each other and increases their degrees by one.
// The both ASes must be added to the graph or the call will result in an error, so will already connected ASes.
func (graph *NetworkGraph) ConnectIsdAses(source addr.ISD_AS, target addr.ISD_AS) error {

	sourceNode, exists := graph.nodes[source]
	if !exists {
		return errors.New(fmt.Sprintf("Source %v not existing in graph", source))
	}

	targetNode, exists := graph.nodes[target]
	if !exists {
		return errors.New(fmt.Sprintf("Target %v not existing in graph", target))
	}

	_, exists = sourceNode.neighbors[target]
	_, exists2 := targetNode.neighbors[source]

	if exists || exists2 {
		return errors.New(fmt.Sprintf("Source %v and target %v are already connected", source, target))
	}

	// TODO: Use a reduced graph instead of a mirrored, redundant
	sourceNode.neighbors[target] = targetNode
	targetNode.neighbors[source] = sourceNode
	sourceNode.info.degree += 1
	targetNode.info.degree += 1

	return nil
}

func (graph *NetworkGraph) AddBandwidth(isdAs *addr.ISD_AS, start time.Time, duration time.Duration, bandwidth datasize.ByteSize) error {

	node, exists := graph.nodes[*isdAs]
	if !exists {
		return errors.New(fmt.Sprintf("AS %v not added to graph!", isdAs))
	}

	node.info.AddActivity(start, duration, bandwidth)

	return nil
}

type networkNode struct {
	IsdAs     addr.ISD_AS
	info      *speedCamInfo
	neighbors map[addr.ISD_AS]networkNode
}
