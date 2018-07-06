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
	"bufio"
	"errors"
	"fmt"
	"github.com/c2h5oh/datasize"
	"github.com/scionproto/scion/go/lib/addr"
	"strconv"
	"strings"
	"time"
)

type SpeedCam struct {
	isdAs    addr.IA
	duration time.Duration
	start    time.Time
}

func CreateSpeedCam(isdAs addr.IA, duration time.Duration) *SpeedCam {
	return &SpeedCam{isdAs: isdAs, duration: duration}
}

func (cam *SpeedCam) Measure(measurementPoints []PrometheusClientInfo, pollInterval time.Duration) map[addr.IA][]SpeedCamResult {

	cam.start = time.Now()

	resultChannel := make(chan Result, len(measurementPoints))
	defer close(resultChannel)

	for _, v := range measurementPoints {
		go cam.measureData(v, pollInterval, resultChannel)
	}

	resultMap := make(map[addr.IA][]SpeedCamResult)
	for i := 0; i < len(measurementPoints); i++ {
		result := <-resultChannel
		if result.err != nil {
			MyLogger.Criticalf("error: %v\n", result.err)
			continue
		}
		resultsPerBr := result.results
		brId := resultsPerBr[0].Neighbor
		resultMap[brId] = resultsPerBr
	}
	return resultMap
}

func (cam *SpeedCam) measureData(measurementPoint PrometheusClientInfo, pollInterval time.Duration, c chan Result) {

	results, err := collectData(cam, measurementPoint, pollInterval)
	if err != nil {
		c <- Result{results: results, err: err}
		return
	}
	result := differentiateResults(results)
	c <- result
}

func differentiateResults(results []SpeedCamResult) Result {

	size := len(results)
	if size <= 1 {
		err := errors.New(fmt.Sprintf("Too few elements to differentiate (needs 2 or more): %v", size))
		return Result{results: results, err: err}
	}

	diffResults := make([]SpeedCamResult, size-1)

	for i := 0; i < size-1; i++ {
		diffResults[i] = differentiateResult(results[i], results[i+1])
	}
	return Result{results: diffResults, err: nil}
}

func differentiateResult(resultStart SpeedCamResult, resultEnd SpeedCamResult) SpeedCamResult {

	result := SpeedCamResult{Neighbor: resultStart.Neighbor, Source: resultStart.Source}
	// Prevent underflow
	output := datasize.B
	if resultStart.BandwidthOut > resultEnd.BandwidthOut {
		output = 0
	} else {
		output = resultEnd.BandwidthOut - resultStart.BandwidthOut
	}

	// Prevent underflow
	input := datasize.B
	if resultStart.BandwidthIn > resultEnd.BandwidthIn {
		input = 0
	} else {
		input = resultEnd.BandwidthIn - resultStart.BandwidthIn
	}

	duration := resultEnd.Timestamp.Sub(resultStart.Timestamp)
	unixTime := (resultEnd.Timestamp.Unix() + resultStart.Timestamp.Unix()) / 2
	timeStamp := time.Unix(unixTime, 0)
	result.BandwidthOut = output / datasize.ByteSize(duration.Seconds())
	result.BandwidthIn = input / datasize.ByteSize(duration.Seconds())
	result.Timestamp = timeStamp

	return result
}

func collectData(cam *SpeedCam, measurementPoint PrometheusClientInfo, pollInterval time.Duration) ([]SpeedCamResult, error) {
	end := cam.start.Add(cam.duration)
	results := make([]SpeedCamResult, 0)
	var err error = nil
	for {
		url := measurementPoint.URL()

		result := SpeedCamResult{Timestamp: time.Now(), BandwidthIn: 0, BandwidthOut: 0, Source: cam.isdAs, Neighbor: measurementPoint.TargetIsdAs}
		pollErr := cam.pollData(url, &result)

		if pollErr != nil {
			err = errors.New(fmt.Sprintf("error polling data. speed cam: %v, url: %v\n", cam.isdAs, url))
			break
		}

		results = append(results, result)
		if time.Now().After(end) {
			break
		}

		time.Sleep(pollInterval)
	}

	return results, err
}

func (cam *SpeedCam) pollData(prometheusUrl string, result *SpeedCamResult) error {

	readBytes, err := FetchData(prometheusUrl + "/metrics")
	if err != nil {
		MyLogger.Criticalf("error polling data, err: %v\n", err)
		return err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(readBytes)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "border_input_bytes_total") {
			v := parseValue(line)
			result.BandwidthIn = datasize.ByteSize(v)
		} else if strings.HasPrefix(line, "border_output_bytes_total") {
			v := parseValue(line)
			result.BandwidthOut = datasize.ByteSize(v)
		}
	}

	return nil
}

func parseValue(line string) uint64 {
	i := strings.LastIndex(line, " ") + 1
	numberString := line[i:]
	v, _ := strconv.ParseFloat(numberString, 64)
	return uint64(v)
}

type SpeedCamResult struct {
	Timestamp    time.Time
	BandwidthIn  datasize.ByteSize
	BandwidthOut datasize.ByteSize
	Source       addr.IA
	Neighbor     addr.IA
}

type Result struct {
	results []SpeedCamResult
	err     error
}
