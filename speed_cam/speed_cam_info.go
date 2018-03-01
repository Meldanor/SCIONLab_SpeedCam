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
	"container/ring"
	"github.com/c2h5oh/datasize"
	"github.com/scionproto/scion/go/lib/addr"
	"time"
)

// Information about a single AS necessary for the SpeedCam algorithm.
// Contains previous episode results and topology information.
type speedCamInfo struct {
	isdAs      addr.ISD_AS
	degree     uint
	successes  *ring.Ring
	activities *ring.Ring
	capacity   datasize.ByteSize
}

// Constructs a new information with assigned ISD-AS and initialized ring buffers.
func NewInfo(isdAs addr.ISD_AS, config *SpeedCamConfig) *speedCamInfo {
	info := new(speedCamInfo)
	info.isdAs = isdAs
	info.successes = ring.New(config.Episodes)
	info.activities = ring.New(config.Episodes)
	// Initialize rings
	for i := 0; i < config.Episodes; i++ {
		info.successes.Value = false
		info.successes = info.successes.Next()

		info.activities.Value = nil
		info.activities = info.activities.Next()
	}
	return info
}

// Add the result of SpeedCam to the history of this AS.
func (scInfo *speedCamInfo) AddDetectionResult(isSuccess bool) {
	scInfo.successes = scInfo.successes.Prev()
	scInfo.successes.Value = isSuccess
}

// Calculates the success rate to detect a congestion of this AS.
// Returns a number between 0.0 and 1.0 (both inclusive)
func (scInfo *speedCamInfo) SuccessRate() float64 {
	result := 0.0
	i := 1.0
	scInfo.successes.Do(func(x interface{}) {
		var num float64
		if x.(bool) {
			num = 1.0
		} else {
			num = 0.0
		}
		result += num / i
		i++
	})
	return result
}

type activity struct {
	start     time.Time
	duration  time.Duration
	bandwidth datasize.ByteSize
}

// Add the actual bandwidth in a certain time span to this AS.
func (scInfo *speedCamInfo) AddActivity(start time.Time, duration time.Duration, bandwidth datasize.ByteSize) {
	scInfo.activities = scInfo.activities.Prev()

	activity := new(activity)
	activity.start = start
	activity.duration = duration
	activity.bandwidth = bandwidth
	scInfo.activities.Value = *activity
}

func (scInfo *speedCamInfo) GetActivity() float64 {
	var sum datasize.ByteSize
	var totalCapacity datasize.ByteSize
	scInfo.activities.Do(func(x interface{}) {
		if x == nil {
			return
		}
		activity := x.(activity)
		sum += activity.bandwidth
		totalCapacity += datasize.ByteSize(scInfo.capacity)
	})

	// Avoid "divided by zero"
	if totalCapacity == 0 {
		return 0.0
	}

	return float64(sum) / float64(totalCapacity)
}
