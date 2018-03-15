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
	"github.com/scionproto/scion/go/lib/addr"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	counter = 1
)

func servePrometheusResults(w http.ResponseWriter, r *http.Request) {

	w.Write(loadPrometheusResult())
	counter++
}

func loadPrometheusResult() []byte {

	bytes, err := ioutil.ReadFile("../test_resources/prometheus_result_" + strconv.Itoa(counter) + ".txt")
	if err != nil {
		return []byte{0}
	}
	return bytes
}

func TestFetchResult(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(servePrometheusResults))
	defer ts.Close()

	sourceIsdAs, _ := addr.IAFromString("1-10")
	targetIsdAs, _ := addr.IAFromString("1-11")
	cam := CreateSpeedCam(*sourceIsdAs, 9*time.Second)

	index := strings.LastIndex(ts.URL, ":")
	ip := ts.URL[:index]
	port, _ := strconv.ParseInt(ts.URL[index+1:], 10, 32)
	measurementPoints := []PrometheusClientInfo{
		{Ip: ip, Port: int(port), BrId: "1-10-1", SourceIsdAs: *sourceIsdAs, TargetIsdAs: *targetIsdAs},
	}
	resultMap := cam.Measure(measurementPoints, 3*time.Second)

	expectedSpeedCamResults := []SpeedCamResult{
		{BandwidthIn: 20619, BandwidthOut: 14278},
		{BandwidthIn: 4474, BandwidthOut: 3894},
		{BandwidthIn: 0, BandwidthOut: 0}}

	results := resultMap[*targetIsdAs]
	for i := 0; i < 3; i++ {
		result := expectedSpeedCamResults[i]
		if result.BandwidthIn != results[i].BandwidthIn || result.BandwidthOut != results[i].BandwidthOut {
			t.Errorf("Expected %v, but was %v\n", result, results[i])
		}
	}
}
