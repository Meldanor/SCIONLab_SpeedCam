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

import "time"

var ProgramRunning = true

func RunProgram(config *SpeedCamConfig, requestFetchUrl string, borderRouterFetchUrl string) {
	// Initiate the speed cam algorithm
	inspector := CreateEmptyGraph(config)
	requestRestFetcher := PathRequestRestFetcher{FetchUrl: requestFetchUrl}
	borderRouterInfoFetcher := PrometheusClientFetcher{FetcherResource: borderRouterFetchUrl}

	//Start speed cam algorithm
	go inspector.Start(requestRestFetcher, borderRouterInfoFetcher)

	MyLogger.Debug("Wait 2 seconds before starting the inspection...")
	time.Sleep(2 * time.Second)

	MyLogger.Debug("Starting inspection loop...")
	for ; ProgramRunning; {
		inspector.StartInspection()
		time.Sleep(10 * time.Second)
	}
	MyLogger.Debug("Finished loop!")
}
