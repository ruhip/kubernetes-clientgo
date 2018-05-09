package k8s

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/client-go/pkg/api/v1" old
	"k8s.io/api/core/v1"
)

type EventCtrl struct {
	client K8sClient
}

func (this *EventCtrl) ListEventByLabels(namespace, label string) ([]v1.Event, error) {

	resultList, err := this.client.ClientSet.Core().Events(namespace).List(metav1.ListOptions{LabelSelector: label})
	if err != nil {
		return nil, err
	}
	return resultList.Items, nil
}
func (this *EventCtrl) ListEvent(namespace string) ([]v1.Event, error) {
	resultList, err := this.client.ClientSet.Core().Events(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return resultList.Items, nil
}
