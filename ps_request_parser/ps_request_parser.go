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
)

func main() {

	logDirPtr := flag.String("logs", "", "Path to log directory")
	sendPathRequestsResource := flag.String("target", "", "the URL to send path requests to")

	flag.Parse()

	if len(*logDirPtr) == 0 {
		fmt.Printf("missing 'logs' parameter\n")
		return
	}

	if len(*sendPathRequestsResource) == 0 {
		fmt.Printf("missing 'target' parameter\n")
		return
	}

	StartParseLogs(*logDirPtr, *sendPathRequestsResource)
}

func StartParseLogs(logDir string, sendPathRequestResource string) {

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
		parseLogFile(logFile, sendPathRequestResource)
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

func parseLogFile(logFilePath string, sendPathRequestsResource string) {
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
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, pathRequestSubstring) {
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

func reducePathRequest(line string) string {
	if len(line) == 0 {
		return line
	}
	i := strings.LastIndex(line, ", ")
	j := strings.LastIndex(line, " [")

	if j < i {
		j = len(line) - 1
	}
	return line[i+2: j]
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
