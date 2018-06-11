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
	"fmt"
	"math"
)

// Configuration for the SpeedCam algorithm
type SpeedCamConfig struct {
	// Amount of previous episodes to store per node
	Episodes int
	// The importance for node's degree to be selected
	WeightDegree float64
	// The importance for node's capacity to be selected
	WeightCapacity float64
	// The importance for node's success rate to be selected
	WeightSuccess float64
	// The importance for node's activity rate to be selected
	WeightActivity float64
	// Additional or fewer SpeedCams to be selected
	SpeedCamDiff int
	// If enabled, there will be additional console output
	Verbose bool
	// If it is an non empty string, the inspector will write the results to this dir as JSON files
	ResultDir string
	// Maximum amount of files before deleting old files. Zero or negative stands for infinity.
	MaxResults int
	// Currently supported are 'const', 'linear' and 'log'
	ScaleType string
	// The factor for the scale. For 'log' this is the base for the logarithmic, for 'linear' it is the factor and
	// for 'const' it is the constant itself
	ScaleParam float64
	// The strategy to wait till next inspection. Currently supported are 'fixed','random','experience'
	IntervalStrategy string
	// Seconds to wait at minimum till next inspection.
	IntervalWaitMin uint
	// Seconds to wait at maximum till next inspection.
	IntervalWaitMax uint
}

// Default values for the algorithm.
func Default() *SpeedCamConfig {
	config := new(SpeedCamConfig)
	config.Episodes = 6
	config.WeightDegree = 1.0
	config.WeightCapacity = 1.0
	config.WeightSuccess = 1.0
	config.WeightActivity = 1.0
	config.SpeedCamDiff = 0
	config.Verbose = true
	config.ResultDir = ""
	config.MaxResults = -1
	config.ScaleType = "linear"
	config.ScaleParam = 0.2
	config.IntervalStrategy = "fixed"
	config.IntervalWaitMin = 10   // 10 seconds
	config.IntervalWaitMax = 3600 // 1 hour
	return config
}

func (config *SpeedCamConfig) String() string {
	return fmt.Sprintf("{Episodes: %v, wDegree: %v, wCapacity: %v, wSuccess: %v, wActivity: %v, "+
		"SpeedCamDiff: %v, Verbose: %v, ResultDir: %v, ScaleType: %v, ScaleParam: %3.3f, "+
		"IntervalStrategy: %v, Interval: [%v - %v]}",
		config.Episodes, config.WeightDegree, config.WeightCapacity, config.WeightSuccess, config.WeightActivity,
		config.SpeedCamDiff, config.Verbose, config.ResultDir, config.ScaleType, config.ScaleParam,
		config.IntervalStrategy, config.IntervalWaitMin, config.IntervalWaitMax)
}

func (config *SpeedCamConfig) Scale(n int) int {
	if config.ScaleParam < 0 {
		MyLogger.Panicf("Param for scale %3.3f cannot be negative!", config.ScaleParam)
	}
	switch config.ScaleType {
	case "const":
		return int(config.ScaleParam)
	case "linear":
		size := float64(n) * config.ScaleParam
		return int(size)
	case "log":
		if config.ScaleParam == 1 {
			MyLogger.Panicf("Invalid base of 1 for log scale!", config.ScaleParam)
		}
		size := float64(n)
		result := math.Ceil(math.Log(size) / math.Log(config.ScaleParam))
		return int(result)
	default:
		MyLogger.Panicf("Unsupported scale type '%v'", config.ScaleType)
		return -1
	}
}

func (config *SpeedCamConfig) StoreInfiniteFiles() bool {
	return config.MaxResults <= 0
}
