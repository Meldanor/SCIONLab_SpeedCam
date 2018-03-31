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
	"github.com/op/go-logging"
	"github.com/scionproto/scion/go/lib/addr"
	"regexp"
	"time"
)

type Inspector struct {
	graph         *NetworkGraph
	config        *SpeedCamConfig
	fetcher       PathRequestFetcher
	brInfoFetcher PrometheusClientFetcher

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

	// Disable debug logging
	if !config.Verbose {
		logging.SetLevel(logging.INFO, "speedcam")
	}

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

func (inspector *Inspector) Start(fetcher PathRequestFetcher, clientFetcher PrometheusClientFetcher) error {

	inspector.active = true
	inspector.fetcher = fetcher
	inspector.brInfoFetcher = clientFetcher

	go inspector.fetchPathRequests()
	go inspector.fetchBrInfo()

	for inspector.active {
		time.Sleep(1 * time.Millisecond)
	}

	return nil
}

func (inspector *Inspector) Stop() {
	inspector.active = false
}

func (inspector *Inspector) StartInspection() {

	startTime := time.Now()
	MyLogger.Info("Start inspection!")
	if inspector.graph.size == 0 {
		MyLogger.Warning("Network graph is empty (as far as I know). Inspection aborted.")
		return
	}

	selector := Create(inspector.config)
	selectSpeedCams := selector.SelectSpeedCams(inspector.graph)
	clientInfos := inspector.brInfoFetcher.Info
	clientInfoGrouped := groupBySource(clientInfos)

	size := len(selectSpeedCams)
	resultChannel := make(chan map[addr.ISD_AS][]SpeedCamResult, size)
	defer close(resultChannel)

	inspectionDuration := 30 * time.Second
	for _, selectedSpeedCam := range selectSpeedCams {
		MyLogger.Debugf("Initiate speed cam on '%v'\n", selectedSpeedCam.IsdAs)
		info := clientInfoGrouped[selectedSpeedCam.IsdAs]
		speedCam := CreateSpeedCam(selectedSpeedCam.IsdAs, inspectionDuration)
		MyLogger.Debugf("Start speed cam on '%v' for 30 seconds\n", selectedSpeedCam.IsdAs)
		go func(cam *SpeedCam, c chan map[addr.ISD_AS][]SpeedCamResult) {

			c <- cam.Measure(info, 5*time.Second)
		}(speedCam, resultChannel)
	}

	var inspectionResults []map[addr.ISD_AS][]SpeedCamResult
	for i := 0; i < size; i++ {
		inspectionResults = append(inspectionResults, <-resultChannel)
	}
	inspector.aggregateResults(inspectionResults, startTime, inspectionDuration)
	presentResults(inspectionResults)
	// If a result dir was specified -> write results to it
	if len(inspector.config.ResultDir) != 0 {
		serializeResult := SerializableResult(inspector, inspectionResults, startTime, inspectionDuration)
		serializeResult.writeJsonResult(inspector.config.ResultDir)
	}
	MyLogger.Info("Inspection finished!")
}

func presentResults(results []map[addr.ISD_AS][]SpeedCamResult) {

	for i := 0; i < len(results); i++ {
		measureResults := results[i]
		MyLogger.Debugf("Results of %v: \n", i+1)
		for k, v := range measureResults {
			MyLogger.Debugf("\tResults for %v:\n", k)
			for _, result := range v {
				MyLogger.Debugf("\t\tLink: %v<->%v Timestamp: %v, In: %v/s, Out: %v/s\n",
					result.Neighbor, result.Source, result.Timestamp, result.BandwidthIn.HR(), result.BandwidthOut.HR())
			}
		}
	}
}

func (inspector *Inspector) aggregateResults(results []map[addr.ISD_AS][]SpeedCamResult, start time.Time,
	inspectionDuration time.Duration) {

	bandwidthPerNode := make(map[addr.ISD_AS]datasize.ByteSize)

	for _, m := range results {
		for _, v := range m {

			for _, result := range v {
				sourceBandwidth, exists := bandwidthPerNode[result.Source]
				if !exists {
					sourceBandwidth = 0
				}
				sourceBandwidth += datasize.ByteSize(uint64(result.BandwidthOut) / uint64(len(v)))
				bandwidthPerNode[result.Source] = sourceBandwidth

				targetBandwidth, exists := bandwidthPerNode[result.Neighbor]
				if !exists {
					targetBandwidth = 0
				}
				targetBandwidth += datasize.ByteSize(uint64(result.BandwidthIn) / uint64(len(v)))
				bandwidthPerNode[result.Neighbor] = targetBandwidth
			}
		}
	}

	for key, v := range bandwidthPerNode {
		info := inspector.graph.nodes[key].info
		MyLogger.Debugf("Add activity to node '%v', start time: %v, duration: %v, average bytes/s: %v",
			key, start, inspectionDuration, v.HR())
		info.AddActivity(start, inspectionDuration, v)
	}

}

func groupBySource(clientInfos []PrometheusClientInfo) map[addr.ISD_AS][]PrometheusClientInfo {
	result := make(map[addr.ISD_AS][]PrometheusClientInfo)

	for _, clientInfo := range clientInfos {
		k := clientInfo.SourceIsdAs
		result[k] = append(result[k], clientInfo)
	}

	return result
}

func (inspector *Inspector) fetchPathRequests() error {

	for inspector.active {

		pathRequests, err := inspector.fetcher.FetchPathRequests()

		for _, v := range pathRequests {
			inspector.HandlePathRequest(v)
		}
		if err != nil {
			MyLogger.Criticalf("error polling path requests, err: %v\n", err)
			return err
		}
		MyLogger.Debugf("Handled %v path requests\n", len(pathRequests))
		time.Sleep(5 * time.Minute)
	}

	return nil
}

func (inspector *Inspector) fetchBrInfo() error {

	for inspector.active {

		err := inspector.brInfoFetcher.PollData()

		if err != nil {
			MyLogger.Criticalf("error polling border router information, err: %v\n", err)
			return err
		}
		MyLogger.Debugf("Polled %v border router information\n", len(inspector.brInfoFetcher.Info))
		time.Sleep(5 * time.Minute)
	}
	return nil
}
