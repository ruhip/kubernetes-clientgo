package k8s

import (
	"errors"
	"gw.com.cn/dzhyun/yunconsole2.git/db"
	//"k8s.io/client-go/pkg/api/v1"
)

type K8sManager struct {
	node       *NodeCtrl
	pod        *PodCtrl
	deployment *DeploymentCtrl
	namespace  *NamespaceCtrl
	event      *EventCtrl
	daemonset  *DaemonSetCtrl
}

func NewManager() (*K8sManager, error) {
	if k8s.ClientSet == nil {
		return nil, errors.New("k8s.ClientSet==nil")
	}
	k8sman := new(K8sManager)
	k8sman.node = &NodeCtrl{client: k8s}
	k8sman.pod = &PodCtrl{client: k8s}
	k8sman.deployment = &DeploymentCtrl{client: k8s}
	k8sman.namespace = &NamespaceCtrl{client: k8s}
	k8sman.event = &EventCtrl{client: k8s}
	k8sman.daemonset = &DaemonSetCtrl{client: k8s}

	go startPodController(k8s)
	go k8sman.nodesMonitorStart()
	return k8sman, nil
}

func (m *K8sManager) ListAllNamespacesImpl() ([]*NamespaceInfo, error) {
	return m.namespace.ListAllNamespaces()
}

func (m *K8sManager) AddNamespaceImpl(namespace string) error {
	return m.namespace.AddNamespace(namespace)
}
func (m *K8sManager) DeleteNamespaceImpl(namespace string) error {
	return m.namespace.DeleteNamespaces(namespace)
}

func (m *K8sManager) nodesMonitorStart() error {
	return m.node.NodesMonitor()
}

func (m *K8sManager) GetNodesImpl(groupid string) ([]*NodeInfo, error) {
	return m.node.ListNodes(groupid)
}

func (m *K8sManager) GetK8sNodesFromLabelImpl(label string) ([]*LabelNodeInfo, error) {
	return m.node.LabelNodes(label)
}

func (m *K8sManager) GetDeploymentesImpl(namespace string) ([]*KAppInfo, error) {
	return m.deployment.List(namespace)
}

func (m *K8sManager) CreateDeploymentImpl(reqdeploy *db.DeployInfo) error {
	return m.deployment.CreateDeploy(reqdeploy)
}

func (m *K8sManager) UpdateDeploymentImpl(reqdeploy *db.DeployInfo) error {
	return m.deployment.UpdateDeploy(reqdeploy)
}

func (m *K8sManager) DeleteDeploymentImpl(namespace, name string) error {
	return m.deployment.DeleteDeploy(namespace, name)
}

func (m *K8sManager) DeletePodImpl(namespace, name string) error {
	return m.pod.DeletePod(namespace, name)
}

func (m *K8sManager) ListPodImpl(namespace, label string, all bool) ([]*PodInfo, error) {
	return m.pod.ListPod(namespace, label, all)
}

func (m *K8sManager) DeleteDeploymentesImpl(name string) error {
	return m.deployment.Delete(name)
}

func (m *K8sManager) AddNodeslabelImpl(labenodes *AddLabelNodes) error {
	return m.node.AddlabelToNodes(labenodes)
}

func (m *K8sManager) UpdateNodeslabelImpl(labenodes *AddLabelNodes) error {
	return m.node.UpdateNodeslabel(labenodes)
}

func (m *K8sManager) DeleteNodelabelImpl(labenodes *AddLabelNodes) error {
	return m.node.RemoveNodeLabel(labenodes)
}

func (m *K8sManager) DeleteGroupImpl(grupid string) error {
	return m.node.DeleteGroup(grupid)
}

func (m *K8sManager) ListLabelEventsImpl(namespace, label string) (interface{}, error) {
	return m.event.ListEventByLabels(namespace, label)
}

func (m *K8sManager) ListEventsImpl(namespace string) (interface{}, error) {
	return m.event.ListEvent(namespace)
}

func (m *K8sManager) ListDaemonsetImpl(namespace string) ([]*KAppInfo, error) {
	return m.daemonset.ListDaemonSet(namespace)
}

func (m *K8sManager) DaemonsetCreateImpl(reqdaemonset *db.DeployInfo) error {
	return m.daemonset.CreateDaemonSet(reqdaemonset)
}

func (m *K8sManager) DaemonsetUpdateImpl(reqdaemonset *db.DeployInfo) error {
	return m.daemonset.UpdateDaemonSet(reqdaemonset)
}

func (m *K8sManager) DeleteDaemonsetImpl(namespace, name string) error {
	return m.daemonset.DeleteDaemonSet(namespace, name)
}

func startPodController(k8s K8sClient) {
	stopCh := make(chan struct{})
	pc := NewPodController(k8s.ClientSet, 0)
	pc.Run(stopCh)
}
