package deviceplugin

import (
	"os"
	"syscall"

	"github.com/everpeace/k8s-hostpath-device-plugin/pkg/config"
	"github.com/everpeace/k8s-hostpath-device-plugin/pkg/watcher"
	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

type Runner struct {
	cfg       config.HostPathDevicePluginConfig
	fsWatcher *fsnotify.Watcher
	sigCh     chan os.Signal
}

func MustNewRunner(
	cfg config.HostPathDevicePluginConfig,
) *Runner {
	log.Info().Str("Path", pluginapi.DevicePluginPath).Msg("Starting filesystem watcher.")
	fsWatcher, err := watcher.NewFSWatcher(pluginapi.DevicePluginPath)
	if err != nil {
		log.Fatal().Str("Path", pluginapi.DevicePluginPath).Err(err).Msg("Failed to create filesystem watcher")
	}

	log.Info().Msg("Starting signal watcher.")
	sigCh := watcher.NewSignalWatcher(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	return &Runner{
		cfg:       cfg,
		fsWatcher: fsWatcher,
		sigCh:     sigCh,
	}
}

func (r *Runner) Run() {
	var devicePlugin *HostPathDevicePlugin
	restart := true
	for {
		if restart {
			if devicePlugin != nil {
				if err := devicePlugin.Stop(); err != nil {
					log.Fatal().Err(err).Msg("Failed to stop HostPath device plugin")
				}
			}

			devicePlugin, err := NewHostPathDevicePlugin(r.cfg)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to initialize HostPath device plugin")
			}

			if err := devicePlugin.Serve(); err != nil {
				log.Error().Err(err).Msg("Failed to start HostPath device plugin")
			} else {
				restart = false
			}
		}

		select {
		case event, ok := <-r.fsWatcher.Events:
			if ok {
				if event.Name == pluginapi.KubeletSocket && event.Op&fsnotify.Create == fsnotify.Create {
					log.Info().
						Str("KubeletSocket", pluginapi.KubeletSocket).
						Msg("inotify: detected KubeletSocket created.  Restarting K8s HostPath Device Plugin")
					restart = true
				}
			}

		case err, ok := <-r.fsWatcher.Errors:
			if ok {
				log.Error().Err(err).Msg("inotify: got error")
			}

		case s := <-r.sigCh:
			switch s {
			case syscall.SIGHUP:
				log.Info().Str("Signal", s.String()).Msg("Received Signal.  Restarting K8s HostPath Device Plugin")
				restart = true
			default:
				log.Info().Str("Signal", s.String()).Msg("Received signal, shutting down")
				if err := devicePlugin.Stop(); err != nil {
					log.Fatal().Err(err).Msg("Failed to shutdown")
				}
				r.fsWatcher.Close()
				log.Info().Msg("Shutdown successfully")
				os.Exit(0)
			}
		}
	}
}
