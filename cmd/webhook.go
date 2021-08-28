package cmd

import (
	"github.com/everpeace/k8s-hostpath-device-plugin/pkg/webhook"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	whCfg = webhook.ServerConfig{
		CertFile: "/cert/tls.crt",
		KeyFile:  "/cert/tls.key",
		Listen:   ":8443",
	}
)

// webhookCmd represents the webhook command
var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "start webhook",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Interface("Config", whCfg).Msg("Loaded webhook server config")
		server := webhook.NewServer(cfg, whCfg)
		if err := server.Start(); err != nil {
			log.Fatal().Str("Listen", whCfg.Listen).Err(err).Msg("Failed to listen")
		}
	},
}

func init() {
	rootCmd.AddCommand(webhookCmd)
	webhookCmd.PersistentFlags().StringVar(&whCfg.CertFile, "tls-cert-file", whCfg.CertFile, ""+
		"File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated "+
		"after server cert).")
	webhookCmd.PersistentFlags().StringVar(&whCfg.KeyFile, "tls-private-key-file", whCfg.KeyFile, ""+
		"File containing the default x509 private key matching --tls-cert-file.")
	webhookCmd.PersistentFlags().StringVar(&whCfg.Listen, "listen", whCfg.Listen, "listen address")
}
