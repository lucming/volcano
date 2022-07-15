/*
Copyright 2021 The Volcano Authors.

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

package jobseq

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	vcbatchv1 "volcano.sh/apis/pkg/apis/batch/v1"
	vcbusv1 "volcano.sh/apis/pkg/apis/bus/v1"

	e2eutil "volcano.sh/volcano/test/e2e/util"
)

var _ = Describe("TensorFlow Plugin E2E Test", func() {
	It("Will Start in pending state and goes through other phases to get complete phase", func() {
		ctx := e2eutil.InitTestContext(e2eutil.Options{})
		defer e2eutil.CleanupTestContext(ctx)

		jobName := "tensorflow-dist-mnist"

		job := &vcbatchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name: jobName,
			},
			Spec: vcbatchv1.JobSpec{
				MinAvailable:  int32(3),
				SchedulerName: e2eutil.SchedulerName,
				Plugins: map[string][]string{
					"tensorflow": {"--ps=ps", "--worker=worker", "--port=2222"},
				},
				Policies: []vcbatchv1.LifecyclePolicy{
					{
						Event:  vcbusv1.PodEvictedEvent,
						Action: vcbusv1.RestartJobAction,
					},
				},
				Tasks: []vcbatchv1.TaskSpec{
					{
						Replicas: int32(1),
						Name:     "ps",
						Template: v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								RestartPolicy: v1.RestartPolicyNever,
								Containers: []v1.Container{
									{
										Command: []string{
											"sh",
											"-c",
											"python /var/tf_dist_mnist/dist_mnist.py --train_steps 1000",
										},
										Image: e2eutil.DefaultTFImage,
										Name:  "tensorflow",
										Ports: []v1.ContainerPort{
											{
												Name:          "tfjob-port",
												ContainerPort: int32(2222),
											},
										},
									},
								},
							},
						},
					},
					{
						Replicas: int32(2),
						Name:     "worker",
						Policies: []vcbatchv1.LifecyclePolicy{
							{
								Event:  vcbusv1.TaskCompletedEvent,
								Action: vcbusv1.CompleteJobAction,
							},
						},
						Template: v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								RestartPolicy: v1.RestartPolicyNever,
								Containers: []v1.Container{
									{
										Command: []string{
											"sh",
											"-c",
											"python /var/tf_dist_mnist/dist_mnist.py --train_steps 1000",
										},
										Image: e2eutil.DefaultTFImage,
										Name:  "tensorflow",
										Ports: []v1.ContainerPort{
											{
												Name:          "tfjob-port",
												ContainerPort: int32(2222),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		created, err := ctx.Vcclient.BatchV1().Jobs(ctx.Namespace).Create(context.TODO(), job, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		err = e2eutil.WaitJobStates(ctx, created, []vcbatchv1.JobPhase{vcbatchv1.Pending, vcbatchv1.Running, vcbatchv1.Completed}, e2eutil.FiveMinute)
		Expect(err).NotTo(HaveOccurred())
	})

})
