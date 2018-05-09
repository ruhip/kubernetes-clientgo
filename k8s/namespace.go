package k8s

import (
	//"//glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/client-go/pkg/api/v1" old
	"k8s.io/api/core/v1"
	"strings"
)

type NamespaceCtrl struct {
	client K8sClient
}

type NamespaceInfo struct {
	Name   string
	Status string
}

func (this *NamespaceCtrl) ListAllNamespaces() ([]*NamespaceInfo, error) {

	result, err := this.client.ClientSet.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	namespacelist := make([]*NamespaceInfo, 0, len(result.Items))
	for _, item := range result.Items {
		if strings.Contains(item.ObjectMeta.Name, "kube") {
			continue
		}
		space := new(NamespaceInfo)
		space.Name = item.ObjectMeta.Name
		space.Status = string(item.Status.Phase)
		namespacelist = append(namespacelist, space)
	}
	return namespacelist, err
}
func (this *NamespaceCtrl) AddNamespace(name string) error {
	namespace := &v1.Namespace{}
	objectMeta := metav1.ObjectMeta{}
	objectMeta.Name = name
	namespace.ObjectMeta = objectMeta
	_, err := this.client.ClientSet.CoreV1().Namespaces().Create(namespace)
	return err
}

/*
func (this *NamespaceCtrl) ListNamespacesByLabels(labels map[string]string) ([]v1.Namespace, error) {
	labelSelector := this.buildLabelSelector(labels)
	result, err := this.client.ClientSet.CoreV1().Namespaces().List(v1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, err
	}
	return result.Items, err
}
*/
func (this *NamespaceCtrl) DeleteNamespaces(name string) error {
	return this.client.ClientSet.CoreV1().Namespaces().Delete(name, &metav1.DeleteOptions{})
}
