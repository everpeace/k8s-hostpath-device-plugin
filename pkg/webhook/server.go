package webhook

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/everpeace/k8s-hostpath-device-plugin/pkg/config"
	"github.com/rs/zerolog/log"
	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
)

type ServerConfig struct {
	CertFile                string
	KeyFile                 string
	Listen                  string
	GracefulShutdownTimeout time.Duration
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

func (s *Server) Start(ctx context.Context) error {
	// Initialize a new cert watcher with cert/key pair
	watcher, err := certwatcher.New(s.whCfg.CertFile, s.whCfg.KeyFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize certWatcher")
	}

	// Setup TLS listener using GetCertficate for fetching the cert when changes
	ln, err := tls.Listen("tcp", s.whCfg.Listen, &tls.Config{
		GetCertificate: watcher.GetCertificate,
	})
	if err != nil {
		log.Fatal().Err(err).Str("Listen", s.whCfg.Listen).Msg("Failed to initialize listener")
	}

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

	eg := errgroup.Group{}
	// Start goroutine with certwatcher running fsnotify against supplied certdir
	eg.Go(func() error { return watcher.Start(ctx) })
	// Start webhook server
	eg.Go(func() error {
		mux := http.NewServeMux()
		srv := &http.Server{Handler: mux}

		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		mux.Handle("/mutating", kwhhttp.MustHandlerFor(kwhhttp.HandlerConfig{Webhook: wh, Logger: kwhLogger}))

		go func() {
			log.Info().Str("Listen", s.whCfg.Listen).Msg("Start listening")
			if err := srv.Serve(ln); err != http.ErrServerClosed {
				log.Fatal().Err(err).Msg("Failed to close server")
			}
		}()
		<-ctx.Done()

		shutDownCtx, shutDownCancel := context.WithTimeout(context.Background(), s.whCfg.GracefulShutdownTimeout)
		defer shutDownCancel()
		if err := srv.Shutdown(shutDownCtx); err != nil {
			log.Error().Err(err).Msg("Failed to gracefully shutdown")
			return err
		}
		return nil
	})

	return eg.Wait()
}
