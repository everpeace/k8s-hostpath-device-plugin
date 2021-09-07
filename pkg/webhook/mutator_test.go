package webhook_test

import (
	"context"
	"fmt"

	"github.com/everpeace/k8s-hostpath-device-plugin/pkg/config"
	"github.com/everpeace/k8s-hostpath-device-plugin/pkg/webhook"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/slok/kubewebhook/v2/pkg/model"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var _ = Describe("Mutator", func() {
	ctx := context.Background()
	var cfg = config.HostPathDevicePluginConfig{
		ResourceName: "test.org/test-resource",
		SocketName:   "test-resource",
		HostPath: corev1.HostPathVolumeSource{
			Path: "/mnt/hostpath",
		},
		VolumeMount: corev1.VolumeMount{
			MountPath: "/mnt/hostpath",
		},
		NumDevices:          100,
		HealthCheckInterval: 0,
	}
	var mutator kwhmutating.Mutator

	BeforeEach(func() {
		mutator = webhook.NewMutator(cfg)
	})
	Context("Non-Pod Input", func() {
		It("shoudl return empty response", func() {
			res, err := mutator.Mutate(ctx, nil, &corev1.ConfigMap{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(res).Should(BeEquivalentTo(&kwhmutating.MutatorResult{}))
		})
	})

	Context("Pod Input And Operations other than Create", func() {
		operations := []model.AdmissionReviewOp{
			model.OperationUnknown, model.OperationUpdate, model.OperationDelete, model.OperationConnect,
		}
		It("shoudl return empty response", func() {
			for _, op := range operations {
				By(string(op))
				res, err := mutator.Mutate(ctx, &model.AdmissionReview{Operation: op}, &corev1.Pod{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(res).Should(BeEquivalentTo(&kwhmutating.MutatorResult{}))
			}
		})
	})

	Context("Pod Input And Create operation", func() {
		createReview := &model.AdmissionReview{Operation: model.OperationCreate}
		When("Pod has user-defined target hostpath volume", func() {
			expectedError := fmt.Sprintf(
				"Forbid to declare a volume with hostPath.path=%s. Request %s resource instead",
				cfg.HostPath.Path, cfg.ResourceName,
			)
			It("should return error", func() {
				var err error
				By("exactly equal to target hostpath")
				_, err = mutator.Mutate(ctx, createReview, &corev1.Pod{
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{{
							Name: "user-defined-target-host-path",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: cfg.HostPath.Path,
								},
							},
						}},
					},
				})
				Expect(err).Should(MatchError(expectedError))

				By("requesting hostpath is subpath")
				_, err = mutator.Mutate(ctx, createReview, &corev1.Pod{
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{{
							Name: "user-defined-target-host-path",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: cfg.HostPath.Path + "/subpath",
								},
							},
						}},
					},
				})
				Expect(err).Should(MatchError(expectedError))
			})
		})
		When("Pod has no user-defined target hostpath volume", func() {
			When("requesting no target hostpath resource", func() {
				It("should just return the input", func() {
					pod := &corev1.Pod{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:  "ctr",
								Image: "busybox",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU: resource.MustParse("1"),
									},
								},
							}},
						},
					}
					res, _ := mutator.Mutate(ctx, createReview, pod)
					Expect(res.MutatedObject.(*corev1.Pod)).Should(BeEquivalentTo(pod))
				})
			})
			When("requesting target hostpath resource", func() {
				It("should just return the input", func() {
					pod := &corev1.Pod{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:  "ctr",
								Image: "busybox",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:                    resource.MustParse("1"),
										corev1.ResourceName(cfg.ResourceName): resource.MustParse("1"),
									},
								},
							}},
						},
					}
					res, _ := mutator.Mutate(ctx, createReview, pod)

					expected := pod.DeepCopy()
					expected.Spec.Volumes = []corev1.Volume{{
						Name: cfg.HostPathVolumeName(),
						VolumeSource: corev1.VolumeSource{
							HostPath: cfg.HostPath.DeepCopy(),
						},
					}}
					expectedVolumeMount := cfg.VolumeMount.DeepCopy()
					expectedVolumeMount.Name = cfg.HostPathVolumeName()
					expected.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{*expectedVolumeMount}
					Expect(res.MutatedObject.(*corev1.Pod)).Should(BeEquivalentTo(expected))
				})
			})
		})
	})
})
