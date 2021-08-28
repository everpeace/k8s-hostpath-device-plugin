package cmd

import (
	"net/http"

	"github.com/everpeace/k8s-hostpath-device-plugin/pkg/webhook"
	"github.com/rs/zerolog/log"
	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

var (
	certFile string = "/cert/tls.crt"
	keyFile  string = "/cert/tls.key"
	listen   string = ":8443"
)

// webhookCmd represents the webhook command
var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "start webhook",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		kwhLogger := newZerologKubeWebhookLogger(log.Logger)
		mcfg := kwhmutating.WebhookConfig{
			ID:      "hostPathDevice",
			Obj:     &corev1.Pod{},
			Mutator: webhook.NewMutator(cfg),
			Logger:  kwhLogger,
		}
		wh, err := kwhmutating.NewWebhook(mcfg)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize webhook config")
		}

		mux := http.NewServeMux()
		respondOK := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			return
		}
		mux.HandleFunc("/", respondOK)
		mux.Handle("/mutating", kwhhttp.MustHandlerFor(kwhhttp.HandlerConfig{Webhook: wh, Logger: kwhLogger}))
		log.Info().Str("Listen", listen).Msg("Start listening")

		err = http.ListenAndServeTLS(listen, certFile, keyFile, mux)
		if err != nil {
			log.Fatal().Str("Listen", listen).Err(err).Msg("Failed to start listen")
		}
	},
}

func init() {
	rootCmd.AddCommand(webhookCmd)
	webhookCmd.PersistentFlags().StringVar(&certFile, "tls-cert-file", certFile, ""+
		"File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated "+
		"after server cert).")
	webhookCmd.PersistentFlags().StringVar(&keyFile, "tls-private-key-file", keyFile, ""+
		"File containing the default x509 private key matching --tls-cert-file.")
	webhookCmd.PersistentFlags().StringVar(&listen, "listen", listen, "listen address")
}
