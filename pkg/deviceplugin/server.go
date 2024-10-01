package deviceplugin

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/everpeace/k8s-hostpath-device-plugin/pkg/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

var (
	_ pluginapi.DevicePluginServer = &HostPathDevicePlugin{}
)

// NewHostPathDevicePlugin implements the Kubernetes device plugin API
type HostPathDevicePlugin struct {
	config config.HostPathDevicePluginConfig
	devs   []*pluginapi.Device
	stop   chan interface{}
	health chan string
	server *grpc.Server
	logger zerolog.Logger
}

// NewHostPathDevicePlugin returns an initialized NewHostPathDevicePlugin
func NewHostPathDevicePlugin(cfg config.HostPathDevicePluginConfig) (*HostPathDevicePlugin, error) {
	dp := &HostPathDevicePlugin{
		config: cfg,
		devs:   make([]*pluginapi.Device, cfg.NumDevices),
		stop:   make(chan interface{}),
		health: make(chan string),
		logger: log.With().Str("ResourceName", cfg.ResourceName).Logger(),
	}

	health := dp.getHostPathHealth()
	for i := range dp.devs {
		dp.devs[i] = &pluginapi.Device{
			ID:     fmt.Sprint(i),
			Health: health,
		}
	}

	return dp, nil
}

// dial establishes the gRPC communication with the registered device plugin.
func dial(unixSocketPath string, timeout time.Duration) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// SA1019
	// nolint:staticcheck
	c, err := grpc.DialContext(ctx, unixSocketPath, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock(),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", addr)
		}),
	)

	if err != nil {
		return nil, err
	}

	return c, nil
}

func (m *HostPathDevicePlugin) getHostPathHealth() string {
	health := pluginapi.Healthy
	if _, err := os.Stat(m.config.HostPath.Path); os.IsNotExist(err) {
		health = pluginapi.Unhealthy
		log.Warn().Str("HostPath", m.config.HostPath.Path).Msg("HostPath not found")
	}
	return health
}

// Start starts the gRPC server of the device plugin
func (m *HostPathDevicePlugin) Start() error {
	err := m.cleanup()
	if err != nil {
		return err
	}

	socket, err := net.Listen("unix", m.config.Socket())
	if err != nil {
		return err
	}

	m.server = grpc.NewServer([]grpc.ServerOption{}...)
	pluginapi.RegisterDevicePluginServer(m.server, m)

	go func() {
		_ = m.server.Serve(socket)
	}()

	// Wait for server to start by launching a blocking connexion
	conn, err := dial(m.config.Socket(), 5*time.Second)
	if err != nil {
		return err
	}
	conn.Close()

	go m.healthCheck()

	return nil
}

// Stop stops the gRPC server
func (m *HostPathDevicePlugin) Stop() error {
	if m.server == nil {
		return nil
	}

	m.server.Stop()
	m.server = nil
	close(m.stop)

	return m.cleanup()
}

// Register registers the device plugin for the given resourceName with Kubelet.
func (m *HostPathDevicePlugin) Register(kubeletEndpoint, resourceName string) error {
	conn, err := dial(kubeletEndpoint, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pluginapi.NewRegistrationClient(conn)
	reqt := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     m.config.SocketName,
		ResourceName: resourceName,
	}

	_, err = client.Register(context.Background(), reqt)
	if err != nil {
		return err
	}
	return nil
}

// ListAndWatch lists devices and update that list according to the health status
func (m *HostPathDevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	for {
		select {
		case <-m.stop:
			return nil
		case health := <-m.health:
			// Update health of devices only in this thread.
			for _, dev := range m.devs {
				dev.Health = health
			}
			log.Info().Interface("Devices", m.devs).Msg("Exposing devices")
			if err := s.Send(&pluginapi.ListAndWatchResponse{Devices: m.devs}); err != nil {
				m.logger.Error().Err(err).Str("Method", "ListAndWatch").Msg("Failed to send device list")
				return err
			}
		}
	}
}

func (m *HostPathDevicePlugin) healthCheck() {
	log.Info().Dur("Interval", m.config.HealthCheckInterval).Msg("Starting health check")
	ticker := time.NewTicker(m.config.HealthCheckInterval)
	lastHealth := "Unknown"
	for {
		select {
		case <-ticker.C:
			health := m.getHostPathHealth()
			if lastHealth != health {
				log.Info().
					Str("HostPath", m.config.HostPath.Path).
					Str("LastHealth", lastHealth).
					Str("Health", health).Msg("Health is changed")
				m.health <- health
			}
			lastHealth = health
		case <-m.stop:
			ticker.Stop()
			return
		}
	}
}

// Allocate which return list of devices.
func (m *HostPathDevicePlugin) Allocate(ctx context.Context, request *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	log.Debug().Interface("AllocateRequest", request).Msg("Start Allocate()")

	containerResponses := make([]*pluginapi.ContainerAllocateResponse, len(request.GetContainerRequests()))
	for i := range request.GetContainerRequests() {
		// this returns empty container allocate response
		// because webhook declares hostPath volume and volumeMounts to the Pods
		containerResponses[i] = &pluginapi.ContainerAllocateResponse{}
	}

	response := pluginapi.AllocateResponse{
		ContainerResponses: containerResponses,
	}

	log.Debug().
		Interface("AllocateRequest", request).
		Interface("AllocateResponse", response).
		Msg("Finish Allocate()")
	return &response, nil
}

func (m *HostPathDevicePlugin) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{
		PreStartRequired: false,
	}, nil
}

func (m *HostPathDevicePlugin) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (m *HostPathDevicePlugin) GetPreferredAllocation(context.Context, *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	return &pluginapi.PreferredAllocationResponse{}, nil
}

func (m *HostPathDevicePlugin) cleanup() error {
	if err := os.Remove(m.config.Socket()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Serve starts the gRPC server and register the device plugin to Kubelet
func (m *HostPathDevicePlugin) Serve() error {
	err := m.Start()
	if err != nil {
		log.Error().Err(err).Msg("Could not start device plugin")
		return err
	}

	m.logger.Info().Str("Socket", m.config.Socket()).Msg("Starting to serve on")

	err = m.Register(pluginapi.KubeletSocket, m.config.ResourceName)
	if err != nil {
		m.logger.Error().Err(err).Msg("Failed to register device plugin")
		_ = m.Stop()
		return err
	}

	m.logger.Info().Msg("Registered device plugin with Kubelet")
	return nil
}
