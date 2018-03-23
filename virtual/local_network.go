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
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	sc "github.com/Meldanor/SCIONLab_SpeedCam/speed_cam"
	"github.com/scionproto/scion/go/lib/addr"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	defaultConfig = sc.Default()
	scionDir      = flag.String("scionDir", "", "Path to SCION root dir")

	episodesFlag     = flag.Int("cEpisodes", defaultConfig.Episodes, "The amount of past episodes to save")
	wDegreeFlag      = flag.Float64("cWDegree", defaultConfig.WeightDegree, "The weight for the degree")
	wCapacityFlag    = flag.Float64("cWCapacity", defaultConfig.WeightCapacity, "The weight for the capacity")
	wSuccessFlag     = flag.Float64("cWSuccess", defaultConfig.WeightSuccess, "The weight for the success")
	wActivityFlag    = flag.Float64("cWActivity", defaultConfig.WeightActivity, "The weight for the activity")
	speedCamDiffFlag = flag.Int("cSpeedCamDiff", defaultConfig.SpeedCamDiff, "Additional or fewer speed cams per episode")
	verboseFlag      = flag.Bool("verbose", defaultConfig.Verbose, "Additional output")
	resultDirFlag    = flag.String("resultDir", defaultConfig.ResultDir, "Write inspection results to that dir")

	// mock variables - the external server should handle them in a real application
	brInfos      []sc.PrometheusClientInfo
	pathRequests = make(map[string]bool)
)

func main() {

	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	flag.Parse()

	if len(*scionDir) == 0 {
		flag.Usage()
		sc.MyLogger.Criticalf("missing '-scionDir' parameter\n")
		return
	}

	config := getConfig()
	sc.MyLogger.Debugf("Config: %v\n", config)

	// parse path requests every minute and send it to mock local HTTP server
	go func() {
		logDir := *scionDir + "/logs"
		for {
			pathServerFetching(logDir, ts.URL)
			time.Sleep(1 * time.Minute)
		}
	}()

	// parse information about border router
	go func() {
		genDir := *scionDir + "/gen"
		for {
			brInfos = parseBrInformation(genDir)
			time.Sleep(1 * time.Minute)
		}
	}()

	sc.MyLogger.Debug("Wait 2 seconds to populate the data...")
	time.Sleep(2 * time.Second)
	// Initiate the speed cam algorithm
	inspector := sc.CreateEmptyGraph(config)
	requestRestFetcher := sc.PathRequestRestFetcher{FetchUrl: ts.URL + "/pathServerRequests"}
	borderRouterInfoFetcher := sc.PrometheusClientFetcher{FetcherResource: ts.URL + "/prometheusClient"}

	//Start speed cam algorithm
	go inspector.Start(requestRestFetcher, borderRouterInfoFetcher)

	sc.MyLogger.Debug("Wait 2 seconds before starting the inspection...")
	time.Sleep(2 * time.Second)

	sc.MyLogger.Debug("Starting inspection loop...")
	for {
		inspector.StartInspection()
		time.Sleep(10 * time.Second)
	}
	sc.MyLogger.Debug("Finished loop!")
}

func getConfig() *sc.SpeedCamConfig {
	return &sc.SpeedCamConfig{
		Episodes:       *episodesFlag,
		WeightDegree:   *wDegreeFlag,
		WeightCapacity: *wCapacityFlag,
		WeightSuccess:  *wSuccessFlag,
		WeightActivity: *wActivityFlag,
		SpeedCamDiff:   *speedCamDiffFlag,
		Verbose:        *verboseFlag,
		ResultDir:      *resultDirFlag}
}

// Mock a simple HTTP server to serving the data
func handler(w http.ResponseWriter, r *http.Request) {

	method := r.Method
	path := r.URL.Path
	if path == "/pathServerRequests" {
		if method == "POST" {
			handlePostPathRequests(r)
		} else if method == "GET" {
			handleGetPathRequests(w)
		}
	} else if path == "/prometheusClient" {
		if method == "GET" {
			handleGetPrometheusClient(w)
		} else {
			sc.MyLogger.Criticalf("POST /prometheusClient unsupported")
		}
	}
}

func handleGetPathRequests(w http.ResponseWriter) {

	result := make([]string, 0)

	for k := range pathRequests {
		result = append(result, k)
	}

	jsonString, err := json.Marshal(result)
	if err != nil {
		sc.MyLogger.Criticalf("error marshalling path requests, err: %v", err)
		return
	}
	w.Write(jsonString)
}

func handlePostPathRequests(r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var list []string
	decoder.Decode(&list)

	for _, v := range list {
		pathRequests[v] = true
	}
}

func handleGetPrometheusClient(w http.ResponseWriter) {
	jsonString, err := json.Marshal(brInfos)
	if err != nil {
		sc.MyLogger.Criticalf("error marshalling border router information, err: %v", err)
		return
	}
	w.Write(jsonString)
}

func pathServerFetching(logDir string, url string) {
	cmd := exec.Command("go", "run", "ps_request_parser/ps_request_parser.go", "-logs="+logDir, "-target="+url+"/pathServerRequests")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func parseBrInformation(genDir string) []sc.PrometheusClientInfo {

	var locBrInfos []sc.PrometheusClientInfo
	regex, err := regexp.Compile("br\\d+-\\d+-\\d+$")
	if err != nil {
		sc.MyLogger.Criticalf("error regex %v", err)
		return locBrInfos
	}
	brFiles := make([]brFile, 0)
	// look for br configuration files and temporary save them for parsing
	err = filepath.Walk(genDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		if info.IsDir() && regex.MatchString(path) {
			brInfoFile := brFile{configFile: path + "/supervisord.conf", topologyFile: path + "/topology.json"}
			brFiles = append(brFiles, brInfoFile)
		}

		return nil
	})

	if err != nil {
		sc.MyLogger.Criticalf("error walking the path %q: %v\n", genDir, err)
		return locBrInfos
	}

	// Parse information from config files
	locBrInfos = make([]sc.PrometheusClientInfo, 0)
	for _, v := range brFiles {
		locBrInfos = append(locBrInfos, parseBrFiles(v))
	}
	return locBrInfos
}

type brFile struct {
	configFile   string
	topologyFile string
}

func parseBrFiles(brInfoFile brFile) sc.PrometheusClientInfo {

	info := sc.PrometheusClientInfo{}

	parseBrConfigFile(brInfoFile.configFile, &info)
	parseBrTopologyFile(brInfoFile.topologyFile, &info)

	return info
}

func parseBrConfigFile(brConfigFilePath string, info *sc.PrometheusClientInfo) {

	readBytes, err := ioutil.ReadFile(brConfigFilePath)
	if err != nil {
		sc.MyLogger.Criticalf("error reading config file '%v', err: %v\n", brConfigFilePath, err)
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(string(readBytes)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "command") {
			brId := extractCommandInfo(line, "id", brConfigFilePath)
			ipPort := extractCommandInfo(line, "prom", brConfigFilePath)
			ipPort = strings.Replace(ipPort, "[", "", -1)
			ipPort = strings.Replace(ipPort, "]", "", -1)

			info.BrId = brId
			split := strings.Split(ipPort, ":")
			info.Ip = "http://" + split[0]
			info.Port, err = strconv.Atoi(split[1])
			if err != nil {
				sc.MyLogger.Criticalf("error parsing port in string '%v' ; err: %v", ipPort, err)
			}

			break
		}
	}
}

func extractCommandInfo(line string, command string, config string) string {
	cmdStr := "-" + command + "="
	indexStart := strings.Index(line, cmdStr)
	if indexStart == -1 {
		sc.MyLogger.Criticalf("no '%v' parameter in config '%v'\n", cmdStr, config)
		return ""
	}
	indexEnd := strings.Index(line[indexStart:], "\" ")
	if indexEnd == -1 {
		sc.MyLogger.Criticalf("no '%v' parameter in config '%v'\n", cmdStr, config)
		return ""
	}

	indexEnd += indexStart
	indexStart += len(cmdStr)

	return line[indexStart:indexEnd]
}

func parseBrTopologyFile(topologyFile string, info *sc.PrometheusClientInfo) {

	readBytes, err := ioutil.ReadFile(topologyFile)

	if err != nil {
		sc.MyLogger.Criticalf("error reading topology file '%v', err: %v\n", topologyFile, err)
		return
	}

	root := make(map[string]interface{})
	err = json.Unmarshal(readBytes, &root)

	if err != nil {
		sc.MyLogger.Criticalf("error reading topology file '%v', err: %v\n", topologyFile, err)
		return
	}
	sourceIsdAs, _ := addr.IAFromString(root["ISD_AS"].(string))
	info.SourceIsdAs = *sourceIsdAs
	// Go down the hierarchy
	borderRouters := root["BorderRouters"].(map[string]interface{})
	borderRouterInfoObject := borderRouters[info.BrId].(map[string]interface{})
	interfacesObject := borderRouterInfoObject["Interfaces"].(map[string]interface{})
	if len(interfacesObject) > 1 {
		sc.MyLogger.Criticalf("error parsing topology file '%v', too many interfaces!", topologyFile)
		return
	}
	for _, v := range interfacesObject {
		value := v.(map[string]interface{})
		targetIsdAs, _ := addr.IAFromString(value["ISD_AS"].(string))
		info.TargetIsdAs = *targetIsdAs
		break
	}
}
