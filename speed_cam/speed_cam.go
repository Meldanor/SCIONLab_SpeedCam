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
	"bytes"
	"fmt"
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

	// ToDo: Replace the mechanism with another way to fetch data
	prometheusUrl string
}

func CreateSpeedCam(isdAs *addr.ISD_AS, duration time.Duration) *SpeedCam {
	return &SpeedCam{isdAs: isdAs, duration: duration}
}

func (cam *SpeedCam) Start(prometheusUrl string) error {

	cam.prometheusUrl = prometheusUrl
	cam.start = time.Now()

	// ToDo: Implement multithreading poll mechanism

	return nil
}

func (cam *SpeedCam) pollData() (error, SpeedCamResult) {

	var result SpeedCamResult
	spaceClient := http.Client{
		Timeout: time.Second * 2, // Maximum of 2 secs
	}

	req, err := http.NewRequest(http.MethodGet, cam.prometheusUrl, nil)
	if err != nil {
		return err, result
	}

	req.Header.Set("User-Agent", "speedcam-inspector")

	res, getErr := spaceClient.Do(req)
	if getErr != nil {
		return getErr, result
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return readErr, result
	}

	n := bytes.IndexByte(body, 0)
	scanner := bufio.NewScanner(strings.NewReader(string(body[:n])))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "border_input_bytes_total") {
			i := strings.LastIndex(line, " ") + 1
			numberString := line[i:]

			v, _ := strconv.ParseUint(numberString, 10, 64)
			result = SpeedCamResult{timestamp: time.Now(), bandwidth: v}
		}
		fmt.Println(scanner.Text())
	}

	return nil, result
}

type SpeedCamResult struct {
	timestamp time.Time
	bandwidth uint64
}
