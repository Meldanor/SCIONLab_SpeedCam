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
package main

import (
	"flag"
	sc "github.com/Meldanor/SCIONLab_SpeedCam/speed_cam"
)

var (
	defaultConfig = sc.Default()

	psRequestFetchUrlFlag    = flag.String("psUrl", "", "Url to fetch path server requests from")
	borderRouterFetchUrlFlag = flag.String("brUrl", "", "Url to fetch information about border router")

	episodesFlag     = flag.Int("cEpisodes", defaultConfig.Episodes, "The amount of past episodes to save")
	wDegreeFlag      = flag.Float64("cWDegree", defaultConfig.WeightDegree, "The weight for the degree")
	wCapacityFlag    = flag.Float64("cWCapacity", defaultConfig.WeightCapacity, "The weight for the capacity")
	wSuccessFlag     = flag.Float64("cWSuccess", defaultConfig.WeightSuccess, "The weight for the success")
	wActivityFlag    = flag.Float64("cWActivity", defaultConfig.WeightActivity, "The weight for the activity")
	speedCamDiffFlag = flag.Int("cSpeedCamDiff", defaultConfig.SpeedCamDiff, "Additional or fewer speed cams per episode")
	verboseFlag      = flag.Bool("verbose", defaultConfig.Verbose, "Additional output")
	resultDirFlag    = flag.String("resultDir", defaultConfig.ResultDir, "Write inspection results to that dir")
	scaleTypeFlag    = flag.String("scaleType", defaultConfig.ScaleType, "How many SpeedCams should be selected? Supported: const, log and linear")
	scaleParamFlag   = flag.Float64("scaleParam", defaultConfig.ScaleParam, "The parameter for the scale func. Base for log, factor for linear and the const for const")
)

func main() {

	flag.Parse()

	if len(*psRequestFetchUrlFlag) == 0 {
		flag.Usage()
		sc.MyLogger.Criticalf("missing '-psUrl' parameter\n")
		return
	}
	if len(*borderRouterFetchUrlFlag) == 0 {
		flag.Usage()
		sc.MyLogger.Criticalf("missing '-brUrl' parameter\n")
		return
	}
	config := getConfig()
	sc.MyLogger.Debugf("Config: %v\n", config)
	sc.RunProgram(config, *psRequestFetchUrlFlag, *borderRouterFetchUrlFlag)
}

func getConfig() *sc.SpeedCamConfig {
	return &sc.SpeedCamConfig{
		Episodes:       *episodesFlag,
		WeightDegree:   *wDegreeFlag,
		WeightCapacity: *wCapacityFlag,
		WeightSuccess:  *wSuccessFlag,
		WeightActivity: *wActivityFlag,
		SpeedCamDiff:   *speedCamDiffFlag,
		Verbose:        *verboseFlag,
		ResultDir:      *resultDirFlag,
		ScaleType:      *scaleTypeFlag,
		ScaleParam:     *scaleParamFlag}
}
