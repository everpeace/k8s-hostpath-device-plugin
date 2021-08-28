package webhook

import (
	"context"

	"github.com/everpeace/k8s-hostpath-device-plugin/pkg/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type hostPathMutator struct {
	cfg config.HostPathDevicePluginConfig
}

func NewMutator(cfg config.HostPathDevicePluginConfig) kwhmutating.Mutator {
	return &hostPathMutator{cfg: cfg}
}

func (m *hostPathMutator) Mutate(_ context.Context, r *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhmutating.MutatorResult, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return &kwhmutating.MutatorResult{}, nil
	}
	if r.Operation != kwhmodel.OperationCreate {
		return &kwhmutating.MutatorResult{}, nil
	}
	logger := log.With().Str("Pod", pod.Namespace+"/"+pod.Name).Logger()

	volumeName := m.cfg.HostPathVolumeName()
	found := false
	mutateHostPathDeviceVolumeIfRequested := func(c *corev1.Container, l zerolog.Logger) {
		if m.isContainerRequestHostPathDevice(*c) {
			found = true
			vm := m.cfg.VolumeMount.DeepCopy()
			vm.Name = volumeName
			c.VolumeMounts = append(c.VolumeMounts, *vm)
			l.Info().Interface("VolumeMount", vm).Msg("VolumeMount added")
		}
	}
	for i, c := range pod.Spec.InitContainers {
		mutateHostPathDeviceVolumeIfRequested(&c, logger.With().Str("InitContainer", c.Name).Logger())
		pod.Spec.InitContainers[i] = c
	}
	for i, c := range pod.Spec.Containers {
		mutateHostPathDeviceVolumeIfRequested(&c, logger.With().Str("Container", c.Name).Logger())
		pod.Spec.Containers[i] = c
	}
	if found {
		volume := corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				HostPath: m.cfg.HostPath.DeepCopy(),
			},
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
		logger.Info().Interface("Volume", volume).Msg("Volume added")
	}
	return &kwhmutating.MutatorResult{MutatedObject: obj}, nil
}

func (m *hostPathMutator) isContainerRequestHostPathDevice(c corev1.Container) bool {
	checkResourceList := func(rl corev1.ResourceList) bool {
		if rl != nil {
			q, ok := rl[corev1.ResourceName(m.cfg.ResourceName)]
			if ok && !q.IsZero() {
				return true
			}
		}
		return false
	}
	return checkResourceList(c.Resources.Requests) || checkResourceList(c.Resources.Limits)
}
