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
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Meldanor/SCIONLab_SpeedCam/speed_cam"
	"github.com/c2h5oh/datasize"
	"github.com/scionproto/scion/go/lib/addr"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

var (
	resultDir     = flag.String("resultDir", "", "Directory containing the result files")
	indexHtmlFile = flag.String("indexHtml", "", "index.html pointer")
	port          = flag.Int("port", 6363, "The port to access the visualization @ http://localhost:PORT/index.html ")

	loadedVisData [][]byte
)

func main() {

	flag.Parse()

	if len(*resultDir) == 0 {
		fmt.Printf("-resultDir flag missing!\n")
		flag.Usage()
		return
	}

	if len(*indexHtmlFile) == 0 {
		fmt.Printf("-indexHtml flag missing!\n")
		flag.Usage()
		return

	}
	fmt.Printf("Result dir: '%v', port: '%v'\n", *resultDir, *port)

	results, err := loadData(*resultDir)

	if err != nil {
		fmt.Printf("error loading result files. err: %v\n", err)
	}

	fmt.Println("Calculcate visualization data...")

	visData := transformResult(results)
	fmt.Printf("Data size: %v\n", len(visData))

	// Pre marshall the data to JSON to save time
	for _, v := range visData {
		data, err := json.Marshal(v)
		if err != nil {
			fmt.Printf("marshalling data. err: %v\n", err)
			return
		}
		loadedVisData = append(loadedVisData, data)
	}

	fmt.Println("Finished calculating visualization data!")

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), http.HandlerFunc(handler)))

	for {
		time.Sleep(1 * time.Second)
	}
}

// Mock a simple HTTP server to serving the data
func handler(w http.ResponseWriter, r *http.Request) {

	method := r.Method
	if r.URL.Path == "/index.html" && method == "GET" {
		w.Header().Set("Content-Type", "text/html")
		readBytes, _ := ioutil.ReadFile(*indexHtmlFile)
		w.Write(readBytes)
		return
	} else if r.URL.Path == "/dataSize" && method == "GET" {
		w.Write([]byte(strconv.Itoa(len(loadedVisData))))
		return
	} else if method == "GET" && strings.HasPrefix(r.URL.Path, "/data") {
		values := r.URL.Query()
		indexS := values.Get("index")
		index, _ := strconv.ParseInt(indexS, 10, 32)
		w.Header().Set("Content-Type", "application/json")
		w.Write(loadedVisData[index])
	}

}

func loadData(dir string) ([]speed_cam.InspectionResult, error) {

	var results []speed_cam.InspectionResult

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return results, err
	}

	for _, file := range files {
		fileName := path.Join(dir, file.Name())
		readBytes, err := ioutil.ReadFile(fileName)

		if err != nil {
			return results, err
		}

		result := *new(speed_cam.InspectionResult)
		err = json.Unmarshal(readBytes, &result)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}

	return results, nil
}

func transformResult(results []speed_cam.InspectionResult) []VisData {

	var visDataSlice []VisData

	for _, r := range results {

		visData := VisData{Duration: r.Duration, Timestamp: r.Start}
		visData.LinkData = createLinkData(r)
		visData.NodeData = createNodeData(r)
		visDataSlice = append(visDataSlice, visData)
	}

	return visDataSlice
}

func createNodeData(result speed_cam.InspectionResult) []NodeData {

	var nodeDataSlice []NodeData

	for k, v := range result.Graph {
		nodeData := NodeData{Id: k.String(), Degree: v.Degree, CandidateScore: uint(v.CandidateScore)}
		nodeIsdAs, _ := addr.IAFromString(nodeData.Id)
		n := 0
		bandwidth := datasize.B
		for _, result := range result.SpeedCamResults {
			for _, v := range result {
				for _, r := range v {
					if r.Source == *nodeIsdAs {
						nodeData.WasSpeedCam = true
					}
					if r.Source == *nodeIsdAs || r.Neighbor == *nodeIsdAs {
						bandwidth += r.BandwidthIn + r.BandwidthOut
						n++
					}
				}
			}
		}
		if n == 0 {
			nodeData.AvgBytes = 0
		} else {
			nodeData.AvgBytes = float64(bandwidth) / float64(n)
		}

		nodeDataSlice = append(nodeDataSlice, nodeData)
	}

	return nodeDataSlice
}

func createLinkData(result speed_cam.InspectionResult) []LinkData {
	graph := result.Graph

	reducedGraph := make(map[string][]string)
	for k, v := range graph {
		isdAs := k.String()

		for _, n := range v.Neighbors {
			nIsdAs := n.String()

			// Was the neighbor already added?
			if _, exists := reducedGraph[nIsdAs]; !exists {
				reducedGraph[isdAs] = append(reducedGraph[isdAs], nIsdAs)
			}

		}
	}

	var linksSlice []LinkData
	for k, v := range reducedGraph {
		for _, n := range v {
			linkData := LinkData{Source: k, Target: n}
			linkData.AvgBytes = averageLinkBandwidth(k, n, result) + averageLinkBandwidth(n, k, result)
			linksSlice = append(linksSlice, linkData)
		}
	}

	return linksSlice
}

func averageLinkBandwidth(source string, target string, result speed_cam.InspectionResult) float64 {
	n := 0
	bandwidthTotal := datasize.B

	sourceIsdAs, _ := addr.IAFromString(source)
	targetIsdAs, _ := addr.IAFromString(target)
	for _, v := range result.SpeedCamResults {

		// check source->target way
		results, exists := v[*sourceIsdAs]
		// Are the results from target to source -> count them
		if exists && results[0].Neighbor == *targetIsdAs {
			for _, r := range results {
				bandwidthTotal += r.BandwidthIn + r.BandwidthOut
				n++
			}
		}

		// Check target->source way
		results, exists = v[*targetIsdAs]
		// Are the results from target to source -> count them
		if exists && results[0].Source == *sourceIsdAs {
			for _, r := range results {
				bandwidthTotal += r.BandwidthIn + r.BandwidthOut
				n++
			}
		}
	}

	if n == 0 {
		return 0
	}
	return float64(bandwidthTotal) / float64(n)
}

type VisData struct {
	NodeData  []NodeData
	LinkData  []LinkData
	Timestamp time.Time
	Duration  time.Duration
}

type NodeData struct {
	Id             string
	Degree         uint
	CandidateScore uint
	AvgBytes       float64
	WasSpeedCam    bool
}

type LinkData struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	AvgBytes float64
}
