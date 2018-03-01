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
	"github.com/c2h5oh/datasize"
	"github.com/scionproto/scion/go/lib/addr"
	"testing"
	"time"
)

func TestNewInfo(t *testing.T) {
	as17, _ := addr.IAFromString("1-7")
	config := Default()
	info := NewInfo(*as17, config)

	if info.isdAs != *as17 {
		t.Errorf("Info contains wrong ISD-AS %v, but should be %v ", info.isdAs, as17)
	}
	if info.capacity != 0 {
		t.Errorf("Info should have capacity 0, but it was %v", info.capacity)
	}
	if info.degree != 0 {
		t.Errorf("Info should have degree 0, but it was %v", info.degree)
	}
}

// Test the success rate calculation using 4 previous episodes
// newer episode -> older episode
// 2 successes, 1 failure, 1 success
func TestSuccessRate(t *testing.T) {
	as17, _ := addr.IAFromString("1-7")
	config := Default()
	info := NewInfo(*as17, config)

	// oldest episode first to add
	info.AddDetectionResult(true)
	info.AddDetectionResult(false)
	info.AddDetectionResult(true)
	// newest episode last to add
	info.AddDetectionResult(true)

	expectedValue := 1.75
	rate := info.SuccessRate()
	if rate != expectedValue {
		t.Errorf("Info success rate should be %v, but it was %v", expectedValue, rate)
	}
}

// Test activity rate calculation using 3 previous episodes
// Episodes: 4 GB, 5 GB, 6 GB
// Total capacity : 10 GB
func TestActivityRate(t *testing.T) {
	as17, _ := addr.IAFromString("1-7")
	config := Default()
	info := NewInfo(*as17, config)
	// 10 GBytes/s
	info.capacity = 10 * datasize.GB

	// Arbitrary values
	date := time.Date(2018, 02, 23, 10, 0, 0, 0, time.Local)
	duration := time.Second * 30

	info.AddActivity(date, duration, 4*datasize.GB)
	info.AddActivity(date, duration, 5*datasize.GB)
	info.AddActivity(date, duration, 6*datasize.GB)

	expectedValue := 0.5
	activity := info.GetActivity()
	if activity != expectedValue {
		t.Errorf("Info activity rate should be %v, but it was %v", expectedValue, activity)
	}
}
