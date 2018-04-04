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
	"math/rand"
	"sort"
	"time"
)

func getWaitTime(inspector *Inspector) time.Duration {
	config := inspector.config

	switch config.IntervalStrategy {
	case "fixed":
		return time.Duration(config.IntervalWaitMin) * time.Second
	case "random":
		return calculateRandomWaitTime(config)
	case "experience":
		return calculateExperiencedWaitTime(inspector)
	default:
		MyLogger.Panicf("Unknown wait strategy %v!", config.IntervalStrategy)
		return -1
	}
}

func calculateRandomWaitTime(config *SpeedCamConfig) time.Duration {

	sleepTime := rand.Int63n(int64(config.IntervalWaitMax-config.IntervalWaitMin)) + int64(config.IntervalWaitMin)
	return time.Duration(time.Duration(sleepTime) * time.Second)
}

func calculateExperiencedWaitTime(inspector *Inspector) time.Duration {
	timeSlots := make([]activityPerTimeSlot, 1440)
	for i := 0; i < 1440; i++ {
		timeSlots[i] = activityPerTimeSlot{hour: i / 60, minute: i % 60, activity: 0}
	}

	// Calculate activity per timestamp for the complete day
	for _, v := range inspector.graph.nodes {
		v.info.activities.Do(func(x interface{}) {
			if x == nil {
				return
			}
			activity := x.(activity)
			timeStart := activity.start
			timeEnd := timeStart.Add(activity.duration)
			startIndex := timeStart.Minute() + timeStart.Hour()*60
			endIndex := timeEnd.Minute() + timeEnd.Hour()*60

			for i := startIndex; i <= endIndex; i++ {
				timeSlots[i].activity += activity.bandwidth
			}
		})
	}

	// Check if there is a long enough to consider history
	activeSlots := 0
	for _, v := range timeSlots {
		if v.activity != 0 {
			activeSlots++
		}
	}

	if activeSlots < 5 {
		MyLogger.Warningf("Only '%v' active time slots! Using random wait time instead until at least 5 slots.", activeSlots)
		return calculateRandomWaitTime(inspector.config)
	}

	// Sort by activity
	sort.Slice(timeSlots, func(i, j int) bool {
		return timeSlots[i].activity < timeSlots[j].activity
	})

	// Select the best 5 % slots
	count := 1440 / 20
	// And I though date-time handling was hard in pre Java 8... Even COBOL in the 80s had better support.
	timeSlotCandidates := make([]time.Time, count)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	for i := 0; i < count; i++ {
		dur := time.Duration(timeSlots[i].hour)*time.Hour + time.Duration(timeSlots[i].minute)*time.Minute
		timeSlotCandidates[i] = today.Add(dur)
	}

	sort.Slice(timeSlotCandidates, func(i, j int) bool {
		return timeSlotCandidates[i].Before(timeSlotCandidates[j])
	})

	// Select the next best point
	for _, v := range timeSlotCandidates {
		if v.After(now) {
			return v.Sub(now)
		}
	}

	MyLogger.Warningf("No next best point found. Sleep for today")
	return today.AddDate(0, 0, 1).Sub(now)
}

type activityPerTimeSlot struct {
	hour     int
	minute   int
	activity datasize.ByteSize
}
