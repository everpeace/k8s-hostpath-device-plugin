package e2e

import (
	"flag"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/everpeace/k8s-hostpath-device-plugin/pkg/config"
	dpconfig "github.com/everpeace/k8s-hostpath-device-plugin/pkg/config"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfigPath   string
	pluginconfigPath string

	k8sClient *kubernetes.Clientset
	namespace string
	dpCfg     config.HostPathDevicePluginConfig
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2e Suite")
}

var _ = BeforeSuite(func() {
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		nil,
	)
	var err error
	namespace, _, err = kubeConfig.Namespace()
	Expect(err).ShouldNot(HaveOccurred())

	clientConfig, err := kubeConfig.ClientConfig()
	Expect(err).ShouldNot(HaveOccurred())

	k8sClient, err = kubernetes.NewForConfig(clientConfig)
	Expect(err).ShouldNot(HaveOccurred())

	dpCfg = dpconfig.MustLoadConfig(pluginconfigPath)
})

func init() {
	flag.StringVar(&kubeconfigPath, "kubeconfig", "", "absolute path to the kubeconfig")
	flag.StringVar(&pluginconfigPath, "pluginconfig", "", "absolute path to the k8s-hostpath-device-plugin config")
}
