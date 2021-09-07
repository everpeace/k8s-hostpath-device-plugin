package cmd

import (
	"os"

	"github.com/everpeace/k8s-hostpath-device-plugin/pkg/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	debug     bool
	logPretty bool
)

var (
	cfg config.HostPathDevicePluginConfig
)

var rootCmd = &cobra.Command{
	Use:   "k8s-hostpath-device-plugin",
	Short: "Kubernetes hostPath device plugin",
	Long:  `This is a very thin device plugin which just exposed a host path to a container.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		if debug {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}
		if logPretty {
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		}
	},
}

func mustLoadConfig() {
	cfg = config.MustLoadConfig(configFilePath)
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "sets log level to debug")
	rootCmd.PersistentFlags().BoolVar(&logPretty, "log-pretty", true, "set pretty logging(human-friendly & colorized output), json logging if false")
}
