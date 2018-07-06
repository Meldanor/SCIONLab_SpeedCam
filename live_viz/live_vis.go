package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Meldanor/SCIONLab_SpeedCam/speed_cam"
	"github.com/c2h5oh/datasize"
	"github.com/scionproto/scion/go/lib/addr"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

var (
	defaultConfig = speed_cam.Default()

	psRequestFetchUrlFlag    = flag.String("psUrl", "", "Url to fetch path server requests from")
	borderRouterFetchUrlFlag = flag.String("brUrl", "", "Url to fetch information about border router")

	episodesFlag     = flag.Int("cEpisodes", defaultConfig.Episodes, "The amount of past episodes to save")
	wDegreeFlag      = flag.Float64("cWDegree", defaultConfig.WeightDegree, "The weight for the degree")
	wCapacityFlag    = flag.Float64("cWCapacity", defaultConfig.WeightCapacity, "The weight for the capacity")
	wSuccessFlag     = flag.Float64("cWSuccess", defaultConfig.WeightSuccess, "The weight for the success")
	wActivityFlag    = flag.Float64("cWActivity", defaultConfig.WeightActivity, "The weight for the activity")
	speedCamDiffFlag = flag.Int("cSpeedCamDiff", defaultConfig.SpeedCamDiff, "Additional or fewer speed cams per episode")
	verboseFlag      = flag.Bool("verbose", defaultConfig.Verbose, "Additional output")
	resultDirFlag    = flag.String("resultDir", "./results/", "Write inspection results to that dir")
	maxResultsFlag   = flag.Int("maxResults", 1, "Maximum amount of files before deleting old files. Zero or negative stands for infinity.")

	scaleTypeFlag  = flag.String("scaleType", defaultConfig.ScaleType, "How many SpeedCams should be selected? Supported: const, log and linear")
	scaleParamFlag = flag.Float64("scaleParam", defaultConfig.ScaleParam, "The parameter for the scale func. Base for log, factor for linear and the const for const")

	intervalStratFlag = flag.String("intervalStrat", defaultConfig.IntervalStrategy, "Strategy for waiting. Supported: fixed, random and experience")
	intervalMinFlag   = flag.Uint("intervalMin", defaultConfig.IntervalWaitMin, "Seconds to wait at minimum till next inspection.")
	intervalMaxFlag   = flag.Uint("intervalMax", defaultConfig.IntervalWaitMax, "Seconds to wait at maximum till next inspection.")

	port = flag.Int("port", 6363, "The port to access the visualization @ http://localhost:PORT/index.html ")

	loadedVisData []byte
)

func main() {

	flag.Parse()

	if len(*psRequestFetchUrlFlag) == 0 {
		flag.Usage()
		speed_cam.MyLogger.Criticalf("missing '-psUrl' parameter\n")
		return
	}
	if len(*borderRouterFetchUrlFlag) == 0 {
		flag.Usage()
		speed_cam.MyLogger.Criticalf("missing '-brUrl' parameter\n")
		return
	}
	config := getConfig()
	speed_cam.MyLogger.Debugf("Config: %v\n", config)

	go func() {
		speed_cam.MyLogger.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), http.HandlerFunc(handler)))
	}()

	go fileWatcher(config.ResultDir)

	speed_cam.RunProgram(config, *psRequestFetchUrlFlag, *borderRouterFetchUrlFlag)

}

func getConfig() *speed_cam.SpeedCamConfig {
	return &speed_cam.SpeedCamConfig{
		Episodes:         *episodesFlag,
		WeightDegree:     *wDegreeFlag,
		WeightCapacity:   *wCapacityFlag,
		WeightSuccess:    *wSuccessFlag,
		WeightActivity:   *wActivityFlag,
		SpeedCamDiff:     *speedCamDiffFlag,
		Verbose:          *verboseFlag,
		ResultDir:        *resultDirFlag,
		MaxResults:       *maxResultsFlag,
		ScaleType:        *scaleTypeFlag,
		ScaleParam:       *scaleParamFlag,
		IntervalStrategy: *intervalStratFlag,
		IntervalWaitMin:  *intervalMinFlag,
		IntervalWaitMax:  *intervalMaxFlag,
	}
}

const IndexHtmlFile = "./index.html"

// Mock a simple HTTP server to serving the data
func handler(w http.ResponseWriter, r *http.Request) {

	method := r.Method
	if (r.URL.Path == "/index.html" || r.URL.Path == "" || r.URL.Path == "/") && method == "GET" {
		w.Header().Set("Content-Type", "text/html")
		readBytes, _ := ioutil.ReadFile(IndexHtmlFile)
		w.Write(readBytes)
		return
	} else if method == "GET" && strings.HasPrefix(r.URL.Path, "/data") {
		w.Header().Set("Content-Type", "application/json")
		w.Write(loadedVisData)
	}
}

func fileWatcher(dir string) {
	oldFilePath := ""

	for {

		files, err := ioutil.ReadDir(dir)
		if err != nil {
			speed_cam.MyLogger.Error(err)
			return
		}

		if len(files) > 1 {
			speed_cam.MyLogger.Errorf("not a single file in director '%v', but instead there were %v files", dir, len(files))
			return
		}
		if len(files) == 1 {
			filePath := path.Join(dir, files[0].Name())
			if filePath != oldFilePath {
				speed_cam.MyLogger.Debugf("New result file! old: %v, new: %v", oldFilePath, filePath)
				data, err := handleData(filePath)
				if err != nil {
					speed_cam.MyLogger.Error(err)
					return
				}
				loadedVisData, err = json.Marshal(data)
				if err != nil {
					speed_cam.MyLogger.Error(err)
					return
				}
				oldFilePath = filePath
			}
		}

		time.Sleep(20 * time.Second)
	}
}

func handleData(filePath string) (VisData, error) {
	var resultData VisData

	readBytes, err := ioutil.ReadFile(filePath)

	if err != nil {
		fmt.Printf("error with file '%v'! %v\n", filePath, err)
		return resultData, err
	}

	result := *new(speed_cam.InspectionResult)
	err = json.Unmarshal(readBytes, &result)
	if err != nil {
		fmt.Printf("error with file '%v'! %v\n", filePath, err)
		return resultData, err
	}

	resultData = transformResult(result)
	return resultData, nil
}

func transformResult(result speed_cam.InspectionResult) VisData {

	visData := VisData{Duration: result.Duration, Timestamp: result.Start.Round(time.Millisecond)}
	visData.LinkData = createLinkData(result)
	visData.NodeData = createNodeData(result)
	return visData
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
