/*
Copyright 2018 The Volcano Authors.

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

package validate

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/kubernetes/pkg/apis/core/validation"

	vcbatchv1 "volcano.sh/apis/pkg/apis/batch/v1"
	vcbusv1 "volcano.sh/apis/pkg/apis/bus/v1"
)

// policyEventMap defines all policy events and whether to allow external use.
var policyEventMap = map[vcbusv1.Event]bool{
	vcbusv1.AnyEvent:           true,
	vcbusv1.PodFailedEvent:     true,
	vcbusv1.PodEvictedEvent:    true,
	vcbusv1.JobUnknownEvent:    true,
	vcbusv1.TaskCompletedEvent: true,
	vcbusv1.TaskFailedEvent:    true,
	vcbusv1.OutOfSyncEvent:     false,
	vcbusv1.CommandIssuedEvent: false,
	vcbusv1.JobUpdatedEvent:    true,
}

// policyActionMap defines all policy actions and whether to allow external use.
var policyActionMap = map[vcbusv1.Action]bool{
	vcbusv1.AbortJobAction:     true,
	vcbusv1.RestartJobAction:   true,
	vcbusv1.RestartTaskAction:  true,
	vcbusv1.TerminateJobAction: true,
	vcbusv1.CompleteJobAction:  true,
	vcbusv1.ResumeJobAction:    true,
	vcbusv1.SyncJobAction:      false,
	vcbusv1.EnqueueAction:      false,
	vcbusv1.SyncQueueAction:    false,
	vcbusv1.OpenQueueAction:    false,
	vcbusv1.CloseQueueAction:   false,
}

func validatePolicies(policies []vcbatchv1.LifecyclePolicy, fldPath *field.Path) error {
	var err error
	policyEvents := map[vcbusv1.Event]struct{}{}
	exitCodes := map[int32]struct{}{}

	for _, policy := range policies {
		if (policy.Event != "" || len(policy.Events) != 0) && policy.ExitCode != nil {
			err = multierror.Append(err, fmt.Errorf("must not specify event and exitCode simultaneously"))
			break
		}

		if policy.Event == "" && len(policy.Events) == 0 && policy.ExitCode == nil {
			err = multierror.Append(err, fmt.Errorf("either event and exitCode should be specified"))
			break
		}

		if len(policy.Event) != 0 || len(policy.Events) != 0 {
			bFlag := false
			policyEventsList := getEventList(policy)
			for _, event := range policyEventsList {
				if allow, ok := policyEventMap[event]; !ok || !allow {
					err = multierror.Append(err, field.Invalid(fldPath, event, "invalid policy event"))
					bFlag = true
					break
				}

				if allow, ok := policyActionMap[policy.Action]; !ok || !allow {
					err = multierror.Append(err, field.Invalid(fldPath, policy.Action, "invalid policy action"))
					bFlag = true
					break
				}
				if _, found := policyEvents[event]; found {
					err = multierror.Append(err, fmt.Errorf("duplicate event %v  across different policy", event))
					bFlag = true
					break
				} else {
					policyEvents[event] = struct{}{}
				}
			}
			if bFlag {
				break
			}
		} else {
			if *policy.ExitCode == 0 {
				err = multierror.Append(err, fmt.Errorf("0 is not a valid error code"))
				break
			}
			if _, found := exitCodes[*policy.ExitCode]; found {
				err = multierror.Append(err, fmt.Errorf("duplicate exitCode %v", *policy.ExitCode))
				break
			} else {
				exitCodes[*policy.ExitCode] = struct{}{}
			}
		}
	}

	if _, found := policyEvents[vcbusv1.AnyEvent]; found && len(policyEvents) > 1 {
		err = multierror.Append(err, fmt.Errorf("if there's * here, no other policy should be here"))
	}

	return err
}

func getEventList(policy vcbatchv1.LifecyclePolicy) []vcbusv1.Event {
	policyEventsList := policy.Events
	if len(policy.Event) > 0 {
		policyEventsList = append(policyEventsList, policy.Event)
	}
	uniquePolicyEventlist := removeDuplicates(policyEventsList)
	return uniquePolicyEventlist
}

func removeDuplicates(eventList []vcbusv1.Event) []vcbusv1.Event {
	keys := make(map[vcbusv1.Event]bool)
	list := []vcbusv1.Event{}
	for _, val := range eventList {
		if _, value := keys[val]; !value {
			keys[val] = true
			list = append(list, val)
		}
	}
	return list
}

func getValidEvents() []vcbusv1.Event {
	var events []vcbusv1.Event
	for e, allow := range policyEventMap {
		if allow {
			events = append(events, e)
		}
	}

	return events
}

func getValidActions() []vcbusv1.Action {
	var actions []vcbusv1.Action
	for a, allow := range policyActionMap {
		if allow {
			actions = append(actions, a)
		}
	}

	return actions
}

// validateIO validates IO configuration.
func validateIO(volumes []vcbatchv1.VolumeSpec) error {
	volumeMap := map[string]bool{}
	for _, volume := range volumes {
		if len(volume.MountPath) == 0 {
			return fmt.Errorf(" mountPath is required;")
		}
		if _, found := volumeMap[volume.MountPath]; found {
			return fmt.Errorf(" duplicated mountPath: %s;", volume.MountPath)
		}
		if volume.VolumeClaim == nil && volume.VolumeClaimName == "" {
			return fmt.Errorf(" either VolumeClaim or VolumeClaimName must be specified;")
		}
		if len(volume.VolumeClaimName) != 0 {
			if volume.VolumeClaim != nil {
				return fmt.Errorf("conflict: If you want to use an existing PVC, just specify VolumeClaimName." +
					"If you want to create a new PVC, you do not need to specify VolumeClaimName")
			}
			if errMsgs := validation.ValidatePersistentVolumeName(volume.VolumeClaimName, false); len(errMsgs) > 0 {
				return fmt.Errorf("invalid VolumeClaimName %s : %v", volume.VolumeClaimName, errMsgs)
			}
		}

		volumeMap[volume.MountPath] = true
	}
	return nil
}

// topoSort uses topo sort to sort job tasks based on dependsOn field
// it will return an array contains all sorted task names and a bool which indicates whether it's a valid dag
func topoSort(job *vcbatchv1.Job) ([]string, bool) {
	graph, inDegree, taskList := makeGraph(job)
	var taskStack []string
	for task, degree := range inDegree {
		if degree == 0 {
			taskStack = append(taskStack, task)
		}
	}

	sortedTasks := make([]string, 0)
	for len(taskStack) > 0 {
		length := len(taskStack)
		var out string
		out, taskStack = taskStack[length-1], taskStack[:length-1]
		sortedTasks = append(sortedTasks, out)
		for in, connected := range graph[out] {
			if connected {
				graph[out][in] = false
				inDegree[in]--
				if inDegree[in] == 0 {
					taskStack = append(taskStack, in)
				}
			}
		}
	}

	isDag := len(sortedTasks) == len(taskList)
	if !isDag {
		return nil, false
	}

	return sortedTasks, isDag
}

func makeGraph(job *vcbatchv1.Job) (map[string]map[string]bool, map[string]int, []string) {
	graph := make(map[string]map[string]bool)
	inDegree := make(map[string]int)
	taskList := make([]string, 0)

	for _, task := range job.Spec.Tasks {
		taskList = append(taskList, task.Name)
		inDegree[task.Name] = 0
		if task.DependsOn != nil {
			for _, dependOnTask := range task.DependsOn.Name {
				if graph[dependOnTask] == nil {
					graph[dependOnTask] = make(map[string]bool)
				}

				graph[dependOnTask][task.Name] = true
				inDegree[task.Name]++
			}
		}
	}

	return graph, inDegree, taskList
}
