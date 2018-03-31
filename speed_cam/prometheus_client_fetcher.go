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
	"strconv"
)

type PrometheusClientFetcher struct {
	FetcherResource string
	Info            []PrometheusClientInfo
}

func (fetcher *PrometheusClientFetcher) PollData() error {

	err := FetchJsonData(fetcher.FetcherResource, &fetcher.Info)
	return err
}

type PrometheusClientInfo struct {
	Ip          string
	Port        int
	BrId        string
	SourceIsdAs addr.ISD_AS
	TargetIsdAs addr.ISD_AS
}

func (info *PrometheusClientInfo) URL() string {
	return "http://" + info.Ip + ":" + strconv.FormatInt(int64(info.Port), 10)
}
