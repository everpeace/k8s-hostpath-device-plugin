package main

import (
	"flag"
	"os"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	ConfigFilePath = "/k8s-hostpath-device-plugin/config.yaml"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	debug := flag.Bool("debug", false, "sets log level to debug")
	logPretty := flag.Bool("log-pretty", true, "set pretty logging(human-friendly & colorized output), json logging if false")
	flag.Parse()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	if *logPretty {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	log.Info().Msg("Starging K8s HostPath Device Plugin")
	log.Info().Msg("Starting FS watcher.")
	watcher, err := newFSWatcher(pluginapi.DevicePluginPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to created FS watcher")
	}
	defer watcher.Close()

	log.Info().Msg("Starting OS watcher.")
	sigs := newOSWatcher(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	config := mustLoadConfig(ConfigFilePath)

	restart := true
	var devicePlugin *HostPathDevicePlugin
	for {
		if restart {
			if devicePlugin != nil {
				devicePlugin.Stop()
			}

			devicePlugin, err = NewHostPathDevicePlugin(config)
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
		case event := <-watcher.Events:
			if event.Name == pluginapi.KubeletSocket && event.Op&fsnotify.Create == fsnotify.Create {
				log.Info().
					Str("KubeletSocket", pluginapi.KubeletSocket).
					Msg("inotify: detected KubeletSocket created.  Restarting K8s HostPath Device Plugin")
				log.Printf("inotify: %s created, restarting.", pluginapi.KubeletSocket)
				restart = true
			}

		case err := <-watcher.Errors:
			log.Error().Err(err).Msg("inotify: got error")

		case s := <-sigs:
			switch s {
			case syscall.SIGHUP:
				log.Info().Str("Signal", s.String()).Msg("Received Signal.  Restarting K8s HostPath Device Plugin")
				restart = true
			default:
				log.Info().Str("Signal", s.String()).Msg("Received signal, shutting down")
				if err := devicePlugin.Stop(); err != nil {
					log.Fatal().Err(err).Msg("Failed to shutdown")
				}
				log.Info().Msg("Shutdown successfully")
				os.Exit(0)
			}
		}
	}
}