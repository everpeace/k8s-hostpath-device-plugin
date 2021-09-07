package config

import (
	"github.com/go-playground/validator/v10"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Validation for HostPathDevicePluginConfig", func() {
	GetStructNamespace := func(e validator.FieldError) string { return e.StructNamespace() }
	GetTag := func(e validator.FieldError) string { return e.Tag() }

	When("valid config", func() {
		It("should succeed", func() {
			Expect(validate.Struct(&HostPathDevicePluginConfig{
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
			})).ShouldNot(HaveOccurred())
		})
	})
	When("require field missing", func() {
		It("should raise validation error", func() {
			err := validate.Struct(&HostPathDevicePluginConfig{})
			Expect(err).Should(HaveOccurred())
			Expect(err).To(MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": SatisfyAll(
					WithTransform(GetStructNamespace, Equal("HostPathDevicePluginConfig.ResourceName")),
					WithTransform(GetTag, Equal("required")),
				),
				"1": SatisfyAll(
					WithTransform(GetStructNamespace, Equal("HostPathDevicePluginConfig.SocketName")),
					WithTransform(GetTag, Equal("required")),
				),
				"2": SatisfyAll(
					WithTransform(GetStructNamespace, Equal("HostPathDevicePluginConfig.HostPath.Path")),
					WithTransform(GetTag, Equal("required")),
				),
				"3": SatisfyAll(
					WithTransform(GetStructNamespace, Equal("HostPathDevicePluginConfig.VolumeMount.MountPath")),
					WithTransform(GetTag, Equal("required")),
				),
				"4": SatisfyAll(
					WithTransform(GetStructNamespace, Equal("HostPathDevicePluginConfig.NumDevices")),
					WithTransform(GetTag, Equal("min")),
				),
			}))
		})
	})
})
