package webhook

import (
	"net/http"

	"github.com/everpeace/k8s-hostpath-device-plugin/pkg/config"
	"github.com/rs/zerolog/log"
	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	corev1 "k8s.io/api/core/v1"
)

type ServerConfig struct {
	CertFile string
	KeyFile  string
	Listen   string
}
type Server struct {
	cfg   config.HostPathDevicePluginConfig
	whCfg ServerConfig
}

func NewServer(
	cfg config.HostPathDevicePluginConfig,
	whCfg ServerConfig,
) *Server {
	return &Server{cfg: cfg, whCfg: whCfg}
}

func (s *Server) Start() error {
	kwhLogger := newZerologKubeWebhookLogger(log.Logger)
	wh, err := kwhmutating.NewWebhook(kwhmutating.WebhookConfig{
		ID:      "hostPathDevice",
		Obj:     &corev1.Pod{},
		Mutator: NewMutator(s.cfg),
		Logger:  kwhLogger,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize webhook config")
	}

	mux := http.NewServeMux()
	respondOK := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	mux.HandleFunc("/", respondOK)
	mux.Handle("/mutating", kwhhttp.MustHandlerFor(kwhhttp.HandlerConfig{Webhook: wh, Logger: kwhLogger}))
	log.Info().Str("Listen", s.whCfg.Listen).Msg("Start listening")
	return http.ListenAndServeTLS(s.whCfg.Listen, s.whCfg.CertFile, s.whCfg.KeyFile, mux)
}
