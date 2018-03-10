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

import "io/ioutil"
import (
	"github.com/c2h5oh/datasize"
	"github.com/scionproto/scion/go/lib/addr"
	"gopkg.in/yaml.v2"
)

func CreateGraphFromTopology(topologyFile string, config *SpeedCamConfig) (*NetworkGraph, error) {

	graph := CreateEmpty(config)

	bytes, err := ioutil.ReadFile(topologyFile)

	if err != nil {
		return graph, err
	}

	topologyConfig := make(map[string]interface{})

	err = yaml.Unmarshal(bytes, &topologyConfig)

	if err != nil {
		return graph, err
	}

	isdAses := topologyConfig["ASes"]
	for k := range isdAses.(map[interface{}]interface{}) {
		isdAs, _ := addr.IAFromString(k.(string))
		graph.AddIsdAs(*isdAs)
	}

	links := topologyConfig["links"]

	for _, v := range links.([]interface{}) {
		link := v.(map[interface{}]interface{})
		source := link["a"].(string)
		target := link["b"].(string)

		capacity, exists := link["bw"].(int)

		sourceIA, _ := addr.IAFromString(source)
		targetIA, _ := addr.IAFromString(target)

		graph.ConnectIsdAses(*sourceIA, *targetIA)

		if exists {
			sourceInfo := graph.nodes[*sourceIA].info
			targetInfo := graph.nodes[*targetIA].info

			sourceInfo.capacity += datasize.KB * datasize.ByteSize(capacity)
			targetInfo.capacity += datasize.KB * datasize.ByteSize(capacity)
		}
	}

	return graph, nil
}
