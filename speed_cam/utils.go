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
	"io/ioutil"
	"net/http"
	"time"
)

func FetchData(restResourceUrl string) ([]byte, error) {
	var body []byte

	client := http.Client{
		Timeout: time.Second * 2, // Maximum of 2 secs
	}

	req, err := http.NewRequest(http.MethodGet, restResourceUrl, nil)
	if err != nil {
		return body, err
	}

	req.Header.Set("User-Agent", "speedcam-inspector")

	res, err := client.Do(req)
	if err != nil {
		return body, err
	}

	body, err = ioutil.ReadAll(res.Body)
	return body, err
}

func FetchJsonData(restResourceUrl string, data interface{}) error {

	readBytes, err := FetchData(restResourceUrl)

	if err != nil {
		return err
	}

	err = json.Unmarshal(readBytes, data)
	return err
}
