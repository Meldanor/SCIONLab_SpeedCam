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
	"encoding/json"
	"fmt"
	"github.com/c2h5oh/datasize"
	"github.com/scionproto/scion/go/lib/addr"
	"io/ioutil"
	"os"
	"time"
)

type InspectionResult struct {
	SpeedCamResults []map[addr.ISD_AS][]SpeedCamResult
	Start           time.Time
	Duration        time.Duration
	Graph           map[addr.ISD_AS]InspectionResultGraphNode
	Config          SpeedCamConfig
}

type InspectionResultGraphNode struct {
	Activities     []InspectionResultActivity
	Capacity       datasize.ByteSize
	CandidateScore float64
	Degree         uint

	Neighbors []addr.ISD_AS
}

type InspectionResultActivity struct {
	Start     time.Time
	Duration  time.Duration
	Bandwidth datasize.ByteSize
}

func SerializableResult(inspector *Inspector, results []map[addr.ISD_AS][]SpeedCamResult, start time.Time,
	duration time.Duration) *InspectionResult {
	result := InspectionResult{Start: start, Duration: duration, SpeedCamResults: results, Config: *inspector.config}
	result.createInspectionGraph(inspector)
	return &result
}

func (result *InspectionResult) createInspectionGraph(inspector *Inspector) {

	selector := Create(inspector.config)
	result.Graph = make(map[addr.ISD_AS]InspectionResultGraphNode)

	for k, v := range inspector.graph.nodes {

		node := InspectionResultGraphNode{}
		var neighbors []addr.ISD_AS
		for neighbor := range v.neighbors {
			neighbors = append(neighbors, neighbor)
		}
		node.Neighbors = neighbors
		node.CandidateScore = selector.calculateScore(v).score

		node.Capacity = v.info.capacity
		node.Degree = v.info.degree

		node.Activities = make([]InspectionResultActivity, 0)
		v.info.activities.Do(func(x interface{}) {
			if x == nil {
				return
			}
			act := x.(activity)
			resultActivity := InspectionResultActivity{Duration: act.duration, Start: act.start, Bandwidth: act.bandwidth}
			node.Activities = append(node.Activities, resultActivity)
		})

		result.Graph[k] = node
	}
}

func (result *InspectionResult) writeJsonResult(dir string) {
	// Check if directory is existing and if not, create it
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		MyLogger.Errorf("error creating result directory. dir: %v, err: %v", dir, err)
		return
	}

	// Format: YYYYMMDD_HHmmss
	dateFormat := "20060102_150405"
	path := fmt.Sprintf("%v/%v.json", dir, result.Start.Format(dateFormat))
	MyLogger.Debugf("Start writing result as json file %v...", path)

	data, err := json.Marshal(result)
	if err != nil {
		MyLogger.Errorf("error writing result json file. file: %v, err: %v", path, err)
		return
	}
	err = ioutil.WriteFile(path, data, 0777)
	if err != nil {
		MyLogger.Errorf("error writing result json file. file: %v, err: %v", path, err)
		return
	}
	MyLogger.Debugf("Finished writing result as json file %v", path)
}
