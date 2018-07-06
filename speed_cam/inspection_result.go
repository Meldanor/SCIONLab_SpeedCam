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
	"path"
	"regexp"
	"sort"
	"time"
)

type InspectionResult struct {
	SpeedCamResults []map[addr.IA][]SpeedCamResult
	Start           time.Time
	Duration        time.Duration
	Graph           map[addr.IA]InspectionResultGraphNode
	Config          SpeedCamConfig
}

type InspectionResultGraphNode struct {
	Activities     []InspectionResultActivity
	Capacity       datasize.ByteSize
	CandidateScore float64
	Degree         uint

	Neighbors []addr.IA
}

type InspectionResultActivity struct {
	Start     time.Time
	Duration  time.Duration
	Bandwidth datasize.ByteSize
}

func SerializableResult(inspector *Inspector, results []map[addr.IA][]SpeedCamResult, start time.Time,
	duration time.Duration) *InspectionResult {
	result := InspectionResult{Start: start, Duration: duration, SpeedCamResults: results, Config: *inspector.config}
	result.createInspectionGraph(inspector)
	return &result
}

func (result *InspectionResult) createInspectionGraph(inspector *Inspector) {

	selector := Create(inspector.config)
	result.Graph = make(map[addr.IA]InspectionResultGraphNode)

	for k, v := range inspector.graph.nodes {

		node := InspectionResultGraphNode{}
		var neighbors []addr.IA
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

	if !result.Config.StoreInfiniteFiles() {
		err = result.deleteOldFiles(dir)
		if err != nil {
			MyLogger.Errorf("error deleting old files: %v, err: %v", dir, err)
			return
		}
	}

	// Format: YYYYMMDD_HHmmss
	dateFormat := "20060102_150405"
	filePath := fmt.Sprintf("%v/%v.json", dir, result.Start.Format(dateFormat))
	MyLogger.Debugf("Start writing result as json file %v...", filePath)

	data, err := json.Marshal(result)
	if err != nil {
		MyLogger.Errorf("error writing result json file. file: %v, err: %v", filePath, err)
		return
	}
	err = ioutil.WriteFile(filePath, data, 0777)
	if err != nil {
		MyLogger.Errorf("error writing result json file. file: %v, err: %v", filePath, err)
		return
	}
	MyLogger.Debugf("Finished writing result as json file %v", filePath)
}

func (result *InspectionResult) deleteOldFiles(dir string) error {

	allFiles, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	// Filter only inspection result files
	regex, err := regexp.Compile("\\d{8}_\\d{6}\\.json")
	if err != nil {
		return err
	}

	files := make([]os.FileInfo, 0)
	for _, v := range allFiles {
		name := v.Name()
		if regex.MatchString(name) {
			files = append(files, v)
		}

	}

	// File amount is in range, do not delete anything
	// +1 because after the clean up there will be a new file created, so delete one additional
	diff := (len(files) + 1) - result.Config.MaxResults
	if diff < 0 {
		return nil
	}

	MyLogger.Infof("Delete %v old log files", diff)
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime().Before(files[j].ModTime())
	})

	deletedFiles := make([]string, 0)
	for i := 0; i < diff; i++ {
		filePath := path.Join(dir, files[i].Name())
		err = os.Remove(filePath)
		if err != nil {
			MyLogger.Errorf("error removing file %v, err: %v", filePath, err)
		} else {
			deletedFiles = append(deletedFiles, filePath)
		}
	}

	MyLogger.Infof("Removed %v files. File names: %v", len(deletedFiles), deletedFiles)

	return nil
}
