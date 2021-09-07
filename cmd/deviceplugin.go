package cmd

import (
	dp "github.com/everpeace/k8s-hostpath-device-plugin/pkg/deviceplugin"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	configFilePath string
)

// devicepluginCmd represents the deviceplugin command
var devicepluginCmd = &cobra.Command{
	Use:   "deviceplugin",
	Short: "start device plugin",
	Run: func(cmd *cobra.Command, args []string) {
		mustLoadConfig()
		log.Info().Msg("Starging K8s HostPath Device Plugin")
		dp.MustNewRunner(cfg).Run()
	},
}

func init() {
	rootCmd.AddCommand(devicepluginCmd)
	devicepluginCmd.PersistentFlags().StringVar(&configFilePath, "config", "/k8s-hostpath-device-plugin/config.yaml", "config file path")
}
