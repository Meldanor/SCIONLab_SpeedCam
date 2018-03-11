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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
)

var (
	expectedRequests = []string{"2-21 34>67 1-11 40>28 1-13 82>62 1-12", "1-12 21>99 2-22", "1-13 32>18 1-16"}
)

func handler(w http.ResponseWriter, r *http.Request) {

	bytes, _ := json.Marshal(expectedRequests)
	w.Write(bytes)
}

func TestSCPBFetch(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	fetcher := PathRequestRestFetcher{
		FetchUrl: ts.URL + "/pathServerRequests",
	}
	requests, err := fetcher.FetchPathRequests()

	if err != nil {
		t.Errorf("error: %v", err)
	}

	sort.Strings(expectedRequests)
	sort.Strings(requests)

	for i := 0; i < len(expectedRequests); i++ {
		if !strings.EqualFold(expectedRequests[i], requests[i]) {
			t.Errorf("Received and expected results are not equals! Received: %v, expected: %v", requests, expectedRequests)
			break
		}
	}
}
