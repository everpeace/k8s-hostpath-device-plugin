package config

import (
	"os"
	"regexp"
	"time"

	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	defaultHealthCheckInterval = time.Duration(10) * time.Second
)

var (
	validate *validator.Validate
)

// HostPathDevicePluginConfig holds a config for HostPathDevicePlugin
type HostPathDevicePluginConfig struct {
	// ResourceName defines a extended resource name which the device plugin serves
	ResourceName string `yaml:"resourceName" validate:"required"`
	// SocketName defines a filename of unix socket to be created that the device plugin listens
	SocketName string `yaml:"socketName" validate:"required"`
	// HostPath specifies the host path volume that the plugin serves as a extended resource
	HostPath corev1.HostPathVolumeSource `yaml:"hostPath" validate:"-"`
	// VolumeMount specifies how the extended resource mounts the HostPath to containers.  Name field will be ignored.
	VolumeMount corev1.VolumeMount `yaml:"volumeMount" validate:"-"`
	// NumDevices specifies how many extended resource the device plugin serves
	NumDevices int `yaml:"numDevices" validate:"min=1"`
	// HealthCheckInterval specifies the healthcheck interval of the Spec.HostPath
	HealthCheckInterval time.Duration `yaml:"healthCheckInterval"`
}

func (c HostPathDevicePluginConfig) Socket() string {
	return pluginapi.DevicePluginPath + c.SocketName
}

func (c HostPathDevicePluginConfig) HostPathVolumeName() string {
	return "hostpath-device-volume-" + regexp.MustCompile(`[./]`).ReplaceAllString(string(c.ResourceName), "-")
}

func MustLoadConfig(configPath string) HostPathDevicePluginConfig {
	logger := log.With().Str("ConfigFile", configPath).Logger()

	f, err := os.Open(configPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to open config file")
	}

	var config HostPathDevicePluginConfig
	decoder := yaml.NewYAMLOrJSONDecoder(f, 256)
	if err := decoder.Decode(&config); err != nil {
		logger.Fatal().Err(err).Msg("Failed to parse config file")
	}

	if err := validate.Struct(&config); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		logger.Fatal().Err(validationErrors).Msg("Failed to validate config")
	}

	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = defaultHealthCheckInterval
	}

	logger.Info().Interface("Config", config).Msg("Config loaded")
	return config
}

func HostPathVolumeValidation(sl validator.StructLevel) {
	hpv := sl.Current().Interface().(corev1.HostPathVolumeSource)

	if len(hpv.Path) == 0 {
		sl.ReportError(hpv.Path, "path", "Path", "required", "")
	}
}

func VolumeMountValidation(sl validator.StructLevel) {
	hpv := sl.Current().Interface().(corev1.VolumeMount)

	if len(hpv.MountPath) == 0 {
		sl.ReportError(hpv.MountPath, "mountPath", "MountPath", "required", "")
	}
}

func init() {
	validate = validator.New()
	validate.RegisterStructValidation(HostPathVolumeValidation, corev1.HostPathVolumeSource{})
	validate.RegisterStructValidation(VolumeMountValidation, corev1.VolumeMount{})
}
