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
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

func main() {

	logDirPtr := flag.String("logs", "", "Path to log directory")
	sendPathRequestsResource := flag.String("target", "", "the URL to send path requests to")
	timeout := flag.Duration("timeout", 24*time.Hour, "The maximum amount of time difference before ignoring the path server request")

	flag.Parse()

	if len(*logDirPtr) == 0 {
		fmt.Printf("missing 'logs' parameter\n")
		return
	}

	if len(*sendPathRequestsResource) == 0 {
		fmt.Printf("missing 'target' parameter\n")
		return
	}

	StartParseLogs(*logDirPtr, *sendPathRequestsResource, *timeout)
}

func StartParseLogs(logDir string, sendPathRequestResource string, timeout time.Duration) {

	_, err := os.Stat(logDir)

	if os.IsNotExist(err) {
		fmt.Printf("log dir '%v' does not exist\n", logDir)
		return
	}

	fmt.Printf("Log directory: %v", logDir)

	fmt.Println("Search for log files...")
	pathServerLogFiles, err := searchPathServerLogs(logDir)

	if err != nil {
		fmt.Printf("search path server logs. error: %v", err)
		return
	}

	fmt.Println("Parsing log files: ", pathServerLogFiles)
	for _, logFile := range pathServerLogFiles {
		parseLogFile(logFile, sendPathRequestResource, timeout)
	}
}

func searchPathServerLogs(root string) ([]string, error) {

	results := make([]string, 0)

	files, err := ioutil.ReadDir(root)
	if err != nil {
		return results, err
	}

	regex, err := regexp.Compile("ps.*\\.DEBUG")

	if err != nil {
		return results, err
	}

	for _, file := range files {
		filePath := root + "/" + file.Name()
		if !file.IsDir() && regex.MatchString(file.Name()) {
			results = append(results, filePath)
		}
	}

	return results, err
}

const pathRequestSubstring = "Handling PCB from"

func parseLogFile(logFilePath string, sendPathRequestsResource string, timeout time.Duration) {
	fmt.Printf("start parsing log file %v\n", logFilePath)
	inFile, err := os.Open(logFilePath)
	defer inFile.Close()

	if err != nil {
		fmt.Printf("error opening log file '%v'. err: %v\n", logFilePath, err)
		return
	}

	scanner := bufio.NewScanner(inFile)
	scanner.Split(bufio.ScanLines)

	uniquePathRequests := make(map[string]bool)
	// Subtract the duration from the time
	oldestAllowedDate := time.Now().Add((-1) * timeout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, pathRequestSubstring) && isNewPathRequest(line, oldestAllowedDate) {
			line = reducePathRequest(line)
			uniquePathRequests[line] = true
		}
	}

	fmt.Printf("parsed log file %v, found %v path requests\n", logFilePath, len(uniquePathRequests))

	pathRequestsLines := make([]string, 0)
	for k := range uniquePathRequests {
		pathRequestsLines = append(pathRequestsLines, k)
	}

	sendPathRequests(pathRequestsLines, sendPathRequestsResource)
}

func isNewPathRequest(line string, oldestAllowedDate time.Time) bool {
	timeString := line[0:19]
	timestamp, err := time.Parse("2006-01-02 15:04:05", timeString)
	if err != nil {
		fmt.Printf("time parsing error - line: '%v', err: '%v'\n", timeString, err)
		return false
	}
	return timestamp.After(oldestAllowedDate)
}

func reducePathRequest(line string) string {
	if len(line) == 0 {
		return line
	}
	i := strings.LastIndex(line, ", ")
	j := strings.LastIndex(line, " [")

	if j < i {
		j = len(line) - 1
	}
	return line[i+2 : j]
}

func sendPathRequests(rawPathRequests []string, sendPathRequestsResource string) {
	fmt.Printf("Send %v entries to SCPB\n", len(rawPathRequests))
	jsonStr, _ := json.Marshal(rawPathRequests)

	req, err := http.NewRequest("POST", sendPathRequestsResource, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		fmt.Printf("error while sending request: %v\n", err)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error while sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("error uploading path server requests. status: %v\n", resp.StatusCode)
		return
	} else {
		fmt.Printf("Entries sent! \n")
	}
}
