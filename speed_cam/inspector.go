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
	"github.com/scionproto/scion/go/lib/addr"
	"regexp"
	"time"
)

type Inspector struct {
	graph   *NetworkGraph
	config  *SpeedCamConfig
	fetcher PathRequestFetcher

	active bool
}

// Creates an inspector with an empty to be explored network graph.
func CreateEmptyGraph(config *SpeedCamConfig) *Inspector {
	return CreateWithGraph(config, CreateEmpty(config))
}

// Creates an inspector with an already existing graph. This graph can also be expanded by exploration
func CreateWithGraph(config *SpeedCamConfig, graph *NetworkGraph) *Inspector {
	inspector := new(Inspector)
	inspector.config = config
	inspector.graph = graph

	return inspector
}

var isdAsRegex = regexp.MustCompile(`(\d+-\d+)`)

// Handles a path request to update the network graph.
// Input format is: ISD-AS /d>/dISD-AS
// Example: 1-1 1>1 1-5 4>3 1-6 2>1 1-7
func (inspector *Inspector) HandlePathRequest(pathRequest string) error {

	isdPairs := isdAsRegex.FindAllString(pathRequest, -1)
	if isdPairs == nil {
		return errors.New(fmt.Sprintf("Path request has invalid format or no pairs. Request:%s", pathRequest))
	}

	var isdAses []addr.ISD_AS
	for _, e := range isdPairs {
		isd_as, err := addr.IAFromString(e)
		if err != nil {
			return errors.New(fmt.Sprintf("Path request has invalid format or no pairs. Request:%s", pathRequest))
		}
		isdAses = append(isdAses, *isd_as)
	}

	// Add all ASes to the graph
	for _, e := range isdAses {
		inspector.graph.AddIsdAs(e)
	}

	// Connect ASes pair wise
	for i := 0; i < len(isdAses)-1; i++ {
		inspector.graph.ConnectIsdAses(isdAses[i], isdAses[i+1])
	}

	return nil
}

func (inspector *Inspector) Start(fetcher PathRequestFetcher) error {

	inspector.active = true
	inspector.fetcher = fetcher

	go inspector.fetchPathRequests()

	for inspector.active {
		time.Sleep(1 * time.Millisecond)
	}

	return nil
}

func (inspector *Inspector) Stop() {
	inspector.active = false
}

func (inspector *Inspector) fetchPathRequests() error {

	for inspector.active {

		pathRequests, err := inspector.fetcher.FetchPathRequests()

		for _, v := range pathRequests {
			inspector.HandlePathRequest(v)
		}
		if err != nil {
			return err
		}
		fmt.Printf("Handled %v requests\n", len(pathRequests))
		time.Sleep(1 * time.Minute)
	}

	return nil
}
