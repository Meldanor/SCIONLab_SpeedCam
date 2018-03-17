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

import "fmt"

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

	return config
}

func (config *SpeedCamConfig) String() string {
	return fmt.Sprintf("{Episodes: %v, wDegree: %v, wCapacity: %v, wSuccess: %v, wActivity: %v, SpeedCamDiff: %v}",
		config.Episodes, config.WeightDegree, config.WeightCapacity, config.WeightSuccess, config.WeightActivity,
		config.SpeedCamDiff)
}
