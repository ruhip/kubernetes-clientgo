package k8s

import (
	"flag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeconfig *string

func init() {

	kubeconfig = flag.String("kubeconfig", "./rootfs/k8s.config", "path to the kubeconfig file")

}

type K8sClient struct {
	ClientSet *kubernetes.Clientset
}

var k8s K8sClient

func InitK8sClinet() error {
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return err
	}
	k8s.ClientSet, err = kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	return nil

}
