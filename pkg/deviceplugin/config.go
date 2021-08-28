package deviceplugin

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
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
	// Spec specifies mount specification for the extended resource
	Spec HostPathMountSpec `yaml:"spec" validate:"required"`
	// NumDevices specifies how many extended resource the device plugin serves
	NumDevices int `yaml:"numDevices" validate:"min=1"`
	// HealthCheckInterval specifies the healthcheck interval of the Spec.HostPath
	HealthCheckInterval time.Duration `yaml:"healthCheckInterval"`
}

type HostPathMountSpec struct {
	HostPath      string `yaml:"hostPath" validate:"required"`
	ContainerPath string `yaml:"containerPath" validate:"required"`
	ReadOnly      bool   `yaml:"readOnly,omitempty"`
}

func (c HostPathDevicePluginConfig) Socket() string {
	return pluginapi.DevicePluginPath + c.SocketName
}

func (c HostPathMountSpec) Mount() pluginapi.Mount {
	return pluginapi.Mount{
		HostPath:      c.HostPath,
		ContainerPath: c.ContainerPath,
		ReadOnly:      c.ReadOnly,
	}
}

func MustLoadConfig(configPath string) HostPathDevicePluginConfig {
	logger := log.With().Str("ConfigFile", configPath).Logger()

	raw, err := ioutil.ReadFile(configPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to read config file")

		fmt.Println(err.Error())
		os.Exit(1)
	}

	var config HostPathDevicePluginConfig
	if err := yaml.Unmarshal(raw, &config); err != nil {
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

func MountValidation(sl validator.StructLevel) {
	mount := sl.Current().Interface().(pluginapi.Mount)

	if len(mount.HostPath) == 0 {
		sl.ReportError(mount.HostPath, "hostPath", "HostPath", "requiredHostPath", "")
	}

	if len(mount.ContainerPath) == 0 {
		sl.ReportError(mount.ContainerPath, "containerPath", "ContainerPath", "requiredContainerPath", "")
	}
}

func init() {
	validate = validator.New()
	validate.RegisterStructValidation(MountValidation, pluginapi.Mount{})
}
