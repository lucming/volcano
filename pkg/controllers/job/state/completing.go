/*
Copyright 2019 The Volcano Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package state

import (
	vcbatchv1 "volcano.sh/apis/pkg/apis/batch/v1"
	vcbusv1 "volcano.sh/apis/pkg/apis/bus/v1"
	"volcano.sh/volcano/pkg/controllers/apis"
)

type completingState struct {
	job *apis.JobInfo
}

func (ps *completingState) Execute(action vcbusv1.Action) error {
	return KillJob(ps.job, PodRetainPhaseSoft, func(status *vcbatchv1.JobStatus) bool {
		// If any "alive" pods, still in Completing phase
		if status.Terminating != 0 || status.Pending != 0 || status.Running != 0 {
			return false
		}
		status.State.Phase = vcbatchv1.Completed
		return true
	})
}
