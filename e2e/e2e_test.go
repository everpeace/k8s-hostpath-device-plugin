package e2e

import (
	"context"
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("E2ETest", func() {
	ctx := context.Background()
	BeforeEach(func() {
		Expect(
			k8sClient.CoreV1().Pods(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{}),
		).NotTo(HaveOccurred())
		Eventually(func() (int, error) {
			podList, err := k8sClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				return 0, err
			}
			return len(podList.Items), nil
		}, "30s").Should(BeZero())
	})
	When("Pod has user-defined target hostpath volume", func() {
		It("should not create pod", func() {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "fail"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "ctr",
						Image: "busybox",
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "user-defined",
							MountPath: filepath.Join("/host", dpCfg.HostPath.Path),
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "user-defined",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: dpCfg.HostPath.Path,
							},
						},
					}},
				},
			}
			var err error
			_, err = k8sClient.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
			Expect(err.Error()).Should(ContainSubstring(
				fmt.Sprintf("Forbid to declare a volume with hostPath.path=%s", dpCfg.HostPath.Path),
			))
			Expect(err.Error()).Should(ContainSubstring(
				fmt.Sprintf("Request %s resource instead", dpCfg.ResourceName),
			))
		})
	})
	When("Pod requests target resource", func() {
		It("should add volume and volumemounts and the pod succeed", func() {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "succeed"},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Name:  "ctr",
						Image: "busybox",
						Command: []string{
							"sh", "-c",
							"set -ex; ls -al /sample; cat /sample/hello",
						},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceName(dpCfg.ResourceName): resource.MustParse("1"),
							},
						},
					}},
				},
			}
			created, err := k8sClient.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(created.Spec.Volumes).Should(ContainElement(corev1.Volume{
				Name: dpCfg.HostPathVolumeName(),
				VolumeSource: corev1.VolumeSource{
					HostPath: dpCfg.HostPath.DeepCopy(),
				},
			}))
			expectedVolumeMount := dpCfg.VolumeMount.DeepCopy()
			expectedVolumeMount.Name = dpCfg.HostPathVolumeName()
			Expect(created.Spec.Containers[0].VolumeMounts).Should(ContainElement(*expectedVolumeMount))
			Eventually(func() (corev1.PodPhase, error) {
				p, err := k8sClient.CoreV1().Pods(namespace).Get(ctx, pod.Name, metav1.GetOptions{})
				if err != nil {
					return corev1.PodPhase(""), err
				}
				return p.Status.Phase, nil
			}, "30s").Should(Equal(corev1.PodSucceeded))
		})
	})
})
