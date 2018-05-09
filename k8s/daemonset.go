package k8s

import (
	"fmt"
	"gw.com.cn/dzhyun/yunconsole2.git/db"
	apps_v1 "k8s.io/api/apps/v1"
	//"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

type DaemonSetCtrl struct {
	client K8sClient
}

func (this *DaemonSetCtrl) ListDaemonSet(namespace string) ([]*KAppInfo, error) {
	daemonSetList, err := this.client.ClientSet.AppsV1().DaemonSets(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	//result *v1.DaemonSetList
	dslist := make([]*KAppInfo, 0, len(daemonSetList.Items))
	for _, item := range daemonSetList.Items {
		ds := new(KAppInfo)
		ds.Type = ""
		ds.Kind = "DaemonSet"
		ds.Name = item.Namespace + "/" + item.Name
		ds.Namespace = item.Namespace
		ds.Replicas = item.Status.DesiredNumberScheduled
		ds.AvailableReplicas = item.Status.NumberReady
		ds.RepicasNow = item.Status.NumberReady
		ds.Labels = item.Spec.Selector.MatchLabels
		ds.CreateTime = item.CreationTimestamp.Time
		ds.Status = "abnormal"
		if item.Status.NumberAvailable > 0 && item.Status.NumberAvailable == item.Status.DesiredNumberScheduled {
			ds.Status = "running"
		}
		PodTemplateSpecToApp(ds, item.Spec.Template)

		dslist = append(dslist, ds)
	}
	return dslist, nil
}

func (this *DaemonSetCtrl) ListDaemonSetByLabels(namespace, labels string) ([]apps_v1.DaemonSet, error) {
	daemonSetList, err := this.client.ClientSet.AppsV1().DaemonSets(namespace).List(metav1.ListOptions{LabelSelector: labels})

	//daemonSetList, err := this.client.ClientSet.Extensions().DaemonSets(namespace).List(metav1.ListOptions{LabelSelector: labels})
	if err != nil {
		return nil, err
	}
	return daemonSetList.Items, nil
}

func (this *DaemonSetCtrl) CreateDaemonSet(reqdaemonset *db.DeployInfo) error {
	daemonSet, err := this.createDaemonSetInfo(reqdaemonset)
	if err != nil {
		return err
	}
	_, err = this.client.ClientSet.AppsV1().DaemonSets(reqdaemonset.Namespace).Create(daemonSet)
	return err
}

func (this *DaemonSetCtrl) UpdateDaemonSet(reqdaemonset *db.DeployInfo) error {
	daemonSet, err := this.createDaemonSetInfo(reqdaemonset)
	if err != nil {
		return err
	}
	_, err = this.client.ClientSet.AppsV1().DaemonSets(reqdaemonset.Namespace).Update(daemonSet)
	if err != nil {
		serr := fmt.Sprintf("err:%s", err)
		if strings.Contains(serr, "not found") {
			_, err = this.client.ClientSet.AppsV1().DaemonSets(reqdaemonset.Namespace).Create(daemonSet)
		}
	}
	return err
}

func (this *DaemonSetCtrl) GetDaemonSet(namespace, name string) (*apps_v1.DaemonSet, error) {
	return this.client.ClientSet.AppsV1().DaemonSets(namespace).Get(name, metav1.GetOptions{})
}

func (this *DaemonSetCtrl) DeleteDaemonSet(namespace, name string) error {
	return this.client.ClientSet.AppsV1().DaemonSets(namespace).Delete(name, &metav1.DeleteOptions{})
}

func (this *DaemonSetCtrl) createDaemonSetInfo(reqdaemonset *db.DeployInfo) (*apps_v1.DaemonSet, error) {
	podTemplateSpec, err := createPodTemplateSpec(reqdaemonset)
	if err != nil {
		return nil, err
	}
	daemonSet := &apps_v1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: reqdaemonset.Name, Namespace: reqdaemonset.Namespace},
		Spec: apps_v1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"dzhyunapp": reqdaemonset.Name,
				},
			},
			Template: *podTemplateSpec,
		},
	}
	return daemonSet, nil
}
