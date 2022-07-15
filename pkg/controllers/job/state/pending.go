/*
Copyright 2017 The Volcano Authors.

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

type pendingState struct {
	job *apis.JobInfo
}

func (ps *pendingState) Execute(action vcbusv1.Action) error {
	switch action {
	case vcbusv1.RestartJobAction:
		return KillJob(ps.job, PodRetainPhaseNone, func(status *vcbatchv1.JobStatus) bool {
			status.RetryCount++
			status.State.Phase = vcbatchv1.Restarting
			return true
		})

	case vcbusv1.AbortJobAction:
		return KillJob(ps.job, PodRetainPhaseSoft, func(status *vcbatchv1.JobStatus) bool {
			status.State.Phase = vcbatchv1.Aborting
			return true
		})
	case vcbusv1.CompleteJobAction:
		return KillJob(ps.job, PodRetainPhaseSoft, func(status *vcbatchv1.JobStatus) bool {
			status.State.Phase = vcbatchv1.Completing
			return true
		})
	case vcbusv1.TerminateJobAction:
		return KillJob(ps.job, PodRetainPhaseSoft, func(status *vcbatchv1.JobStatus) bool {
			status.State.Phase = vcbatchv1.Terminating
			return true
		})
	default:
		return SyncJob(ps.job, func(status *vcbatchv1.JobStatus) bool {
			if ps.job.Job.Spec.MinAvailable <= status.Running+status.Succeeded+status.Failed {
				status.State.Phase = vcbatchv1.Running
				return true
			}
			return false
		})
	}
}
