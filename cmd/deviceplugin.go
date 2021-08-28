package cmd

import (
	"os"
	"syscall"

	dp "github.com/everpeace/k8s-hostpath-device-plugin/pkg/deviceplugin"
	"github.com/everpeace/k8s-hostpath-device-plugin/pkg/watcher"
	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

var (
	configFilePath string
)

// devicepluginCmd represents the deviceplugin command
var devicepluginCmd = &cobra.Command{
	Use:   "deviceplugin",
	Short: "start device plugin",
	Run: func(cmd *cobra.Command, args []string) {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		if debug {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}
		if logPretty {
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		}

		log.Info().Msg("Starging K8s HostPath Device Plugin")
		log.Info().Str("Path", pluginapi.DevicePluginPath).Msg("Starting filesystem watcher.")
		fsWatcher, err := watcher.NewFSWatcher(pluginapi.DevicePluginPath)
		if err != nil {
			log.Fatal().Str("Path", pluginapi.DevicePluginPath).Err(err).Msg("Failed to create filesystem watcher")
		}
		defer fsWatcher.Close()

		log.Info().Msg("Starting signal watcher.")
		sigs := watcher.NewSignalWatcher(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

		config := dp.MustLoadConfig(configFilePath)
		restart := true
		var devicePlugin *dp.HostPathDevicePlugin
		for {
			if restart {
				if devicePlugin != nil {
					devicePlugin.Stop()
				}

				devicePlugin, err = dp.NewHostPathDevicePlugin(config)
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
			case event := <-fsWatcher.Events:
				if event.Name == pluginapi.KubeletSocket && event.Op&fsnotify.Create == fsnotify.Create {
					log.Info().
						Str("KubeletSocket", pluginapi.KubeletSocket).
						Msg("inotify: detected KubeletSocket created.  Restarting K8s HostPath Device Plugin")
					log.Printf("inotify: %s created, restarting.", pluginapi.KubeletSocket)
					restart = true
				}

			case err := <-fsWatcher.Errors:
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
	},
}

func init() {
	rootCmd.AddCommand(devicepluginCmd)
	devicepluginCmd.PersistentFlags().StringVar(&configFilePath, "config", "/k8s-hostpath-device-plugin/config.yaml", "config file path")
}
