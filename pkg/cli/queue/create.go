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

package queue

import (
	"context"

	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	vcschedulingv1 "volcano.sh/apis/pkg/apis/scheduling/v1"
	"volcano.sh/apis/pkg/client/clientset/versioned"
)

type createFlags struct {
	commonFlags

	Name   string
	Weight int32
	// State is state of Queue
	State string
}

var createQueueFlags = &createFlags{}

// InitCreateFlags is used to init all flags during queue creating.
func InitCreateFlags(cmd *cobra.Command) {
	initFlags(cmd, &createQueueFlags.commonFlags)

	cmd.Flags().StringVarP(&createQueueFlags.Name, "name", "n", "test", "the name of queue")
	cmd.Flags().Int32VarP(&createQueueFlags.Weight, "weight", "w", 1, "the weight of the queue")

	cmd.Flags().StringVarP(&createQueueFlags.State, "state", "S", "Open", "the state of queue")
}

// CreateQueue create queue.
func CreateQueue() error {
	config, err := buildConfig(createQueueFlags.Master, createQueueFlags.Kubeconfig)
	if err != nil {
		return err
	}

	queue := &vcschedulingv1.Queue{
		ObjectMeta: metav1.ObjectMeta{
			Name: createQueueFlags.Name,
		},
		Spec: vcschedulingv1.QueueSpec{
			Weight: createQueueFlags.Weight,
		},
		Status: vcschedulingv1.QueueStatus{
			State: vcschedulingv1.QueueState(createQueueFlags.State),
		},
	}

	queueClient := versioned.NewForConfigOrDie(config)
	if _, err := queueClient.SchedulingV1().Queues().Create(context.TODO(), queue, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}
