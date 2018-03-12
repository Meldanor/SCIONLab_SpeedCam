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
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type SpeedCam struct {
	isdAs    *addr.ISD_AS
	duration time.Duration
	start    time.Time

	prometheusUrl string
}

func CreateSpeedCam(isdAs *addr.ISD_AS, duration time.Duration) *SpeedCam {
	return &SpeedCam{isdAs: isdAs, duration: duration}
}

func (cam *SpeedCam) Start(prometheusUrl string, pollInterval time.Duration) ([]SpeedCamResult, error) {

	cam.prometheusUrl = prometheusUrl
	cam.start = time.Now()

	resultChannel := make(chan SpeedCamResult)

	go collectData(cam, pollInterval, resultChannel)

	results := make([]SpeedCamResult, 0)

	for e := range resultChannel {
		results = append(results, e)
	}

	results, err := differentiateResults(results)

	return results, err
}

func differentiateResults(results []SpeedCamResult) ([]SpeedCamResult, error) {

	size := len(results)
	if size <= 1 {
		return results, errors.New(fmt.Sprintf("Too few elements to differentiate (needs 2 or more): %v", size))
	}

	diffResults := make([]SpeedCamResult, size-1)

	for i := 0; i < size-1; i++ {
		diffResults[i] = differentiateResult(results[i], results[i+1])
	}

	return diffResults, nil
}

func differentiateResult(resultStart SpeedCamResult, resultEnd SpeedCamResult) SpeedCamResult {

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

	unixTime := (resultEnd.Timestamp.Unix() + resultStart.Timestamp.Unix()) / 2
	timeStamp := time.Unix(unixTime, 0)

	return SpeedCamResult{Timestamp: timeStamp, BandwidthOut: output, BandwidthIn: input}
}

func collectData(cam *SpeedCam, pollInterval time.Duration, resultChannel chan SpeedCamResult) {
	end := cam.start.Add(cam.duration)
	for {

		e, result := cam.pollData()

		if e != nil {
			fmt.Errorf("error polling data. speedcam: %v, url: %v", cam.isdAs, cam.prometheusUrl)
		}

		resultChannel <- result
		if time.Now().After(end) {
			break
		}

		time.Sleep(pollInterval)
	}
	close(resultChannel)
}

func (cam *SpeedCam) pollData() (error, SpeedCamResult) {

	var result SpeedCamResult
	client := http.Client{
		Timeout: time.Second * 2, // Maximum of 2 secs
	}

	req, err := http.NewRequest(http.MethodGet, cam.prometheusUrl, nil)
	if err != nil {
		return err, result
	}

	req.Header.Set("User-Agent", "speedcam-inspector")

	res, getErr := client.Do(req)
	if getErr != nil {
		return getErr, result
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return readErr, result
	}

	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	result = SpeedCamResult{Timestamp: time.Now(), BandwidthIn: 0, BandwidthOut: 0}
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

	return nil, result
}

func parseValue(line string) uint64 {
	i := strings.LastIndex(line, " ") + 1
	numberString := line[i:]
	v, _ := strconv.ParseUint(numberString, 10, 64)
	return v
}

type SpeedCamResult struct {
	Timestamp    time.Time
	BandwidthIn  datasize.ByteSize
	BandwidthOut datasize.ByteSize
}
