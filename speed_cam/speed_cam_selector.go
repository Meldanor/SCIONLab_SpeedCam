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
	"math"
	"math/rand"
	"sort"
)

type SpeedCamSelector struct {
	config *SpeedCamConfig
}

func Create(config *SpeedCamConfig) *SpeedCamSelector {
	selector := new(SpeedCamSelector)
	selector.config = config
	return selector
}

type speedCamCandidate struct {
	score float64
	node  networkNode
}

func (selector *SpeedCamSelector) SelectUsableSpeedCams(nodes map[addr.ISD_AS]networkNode) []networkNode {
	candidates := make(map[addr.ISD_AS]*speedCamCandidate)
	for k, v := range nodes {
		candidates[k] = selector.calculateScore(v)
	}

	selector.normalizeScores(candidates)

	return selector.selectCams(candidates)
}

func (selector *SpeedCamSelector) calculateScore(node networkNode) *speedCamCandidate {
	var score = 0.0
	info := node.info
	score += float64(info.degree) * selector.config.WeightDegree
	score += float64(info.capacity) * selector.config.WeightCapacity
	score += float64(info.GetActivity()) * selector.config.WeightActivity
	score += float64(info.SuccessRate()) * selector.config.WeightSuccess

	candidate := new(speedCamCandidate)
	candidate.score = score
	candidate.node = node
	return candidate
}

func (selector *SpeedCamSelector) normalizeScores(candidates map[addr.ISD_AS]*speedCamCandidate) {
	maxScore := -1.0

	for _, v := range candidates {
		maxScore = math.Max(maxScore, v.score)
	}

	for _, v := range candidates {
		v.score = v.score / maxScore
	}
}

func (selector *SpeedCamSelector) selectCams(candidates map[addr.ISD_AS]*speedCamCandidate) []networkNode {

	count := selector.config.Scale(len(candidates)) + selector.config.SpeedCamDiff
	MyLogger.Debugf("Candidates: %v, SpeedCam count: %v", len(candidates), count)
	var i = 0

	selectedCams := make(map[addr.ISD_AS]*speedCamCandidate)

	for k, v := range candidates {
		MyLogger.Debugf("Candidate: %v, chance: %.4f", k, v.score)
	}
	for k, v := range candidates {
		// Is the speedCam selected?
		chance := rand.Float64()
		if chance <= v.score {
			// Add speedCam to selection
			selectedCams[k] = v
			i += 1
			// Are there enough speedCams?
			if i == count {
				break
			}
		}
	}

	// Are not enough speedCams selected -> select highest chance
	if i != count {
		var notSelectedCams []speedCamCandidate
		for k, v := range candidates {
			_, ok := selectedCams[k]
			if !ok {
				notSelectedCams = append(notSelectedCams, *v)
			}
		}

		// Sort in descending order
		sort.Slice(notSelectedCams, func(i, j int) bool {
			return notSelectedCams[i].score >= notSelectedCams[j].score
		})

		for j := 0; i < count && j < len(notSelectedCams); i, j = i+1, j+1 {
			element := notSelectedCams[j]
			selectedCams[element.node.IsdAs] = &element
		}
	}

	var result []networkNode
	for _, v := range selectedCams {
		result = append(result, v.node)
	}
	return result
}
