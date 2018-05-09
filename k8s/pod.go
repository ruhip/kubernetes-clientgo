package k8s

import (
	//"//glog"
	"fmt"
	"gw.com.cn/dzhyun/yunconsole2.git/monitor"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

type PodCtrl struct {
	client K8sClient
}

type ContainerInfo struct {
	Env      map[string]string
	Images   string
	Portsmap string
}

type ContainerStatus struct {
	ContainerID string
	Ready       bool
}
type PodInfo struct {
	Name              string
	Namespace         string
	Status            string
	Message           string    `json:"message"`
	CreateTimeStamp   time.Time `json:"Created"`
	UpdateTimeStamp   time.Time `json:"Updated"`
	NodeName          string
	HostIp            string
	PodIp             string
	RestartCount      int32 `json:"restart"`
	Labels            map[string]string
	Images            string
	ImgName           string
	ImgVer            string
	Type              string
	Interface         string
	PortMap           string
	Env               map[string]string
	Vol               map[string]string
	Path              string
	Cpu               string
	MemUsed           string
	MemTotal          string
	ServiceId         string
	HostNetwork       bool
	containerList     []*ContainerInfo
	containerStatuses []*ContainerStatus
	ContainerName     string
}

func (this *PodCtrl) ListPod(namespace, label string, all bool) ([]*PodInfo, error) {
	//label = "app:test1,version:latest"
	//FieldSelector: "show-all=false" "field label not supported: show-all"
	//IncludeUninitialized: false
	//status.phase!=terminated
	//listoption := metav1.ListOptions{FieldSelector: "status.phase!=Failed"}
	listoption := metav1.ListOptions{}
	if !all {
		listoption.FieldSelector = "status.phase!=Failed"
	}
	if len(label) > 0 {
		listoption.LabelSelector = label
	}
	result, err := this.client.ClientSet.CoreV1().Pods("").List(listoption)
	if err != nil {
		return nil, err
	}
	podinfolist := make([]*PodInfo, 0, len(result.Items))
	for _, item := range result.Items {
		if len(namespace) > 0 && item.Namespace != namespace {
			continue
		}
		ni := new(PodInfo)
		ni.Name = item.ObjectMeta.Name
		ni.Namespace = item.ObjectMeta.Namespace
		ni.CreateTimeStamp = item.ObjectMeta.CreationTimestamp.Time
		ni.NodeName = item.Spec.NodeName
		ni.Labels = item.ObjectMeta.Labels
		ni.HostNetwork = item.Spec.HostNetwork
		ni.containerList = make([]*ContainerInfo, 0, len(item.Spec.Containers))
		vol := make(map[string]string)
		for _, container := range item.Spec.Containers {
			cn := new(ContainerInfo)
			cn.Env = make(map[string]string)
			//cn.containerID = container.con
			for _, env := range container.Env {
				cn.Env[env.Name] = env.Value
				if env.Name == "SERVICE_ID" {
					ni.ServiceId = env.Value
					ni.Path = env.Value
				} else if env.Name == "INTERFACE" {
					ni.Interface = env.Value
				} else if env.Name == "INSTALLTYPE" {
					ni.Type = env.Value
				}
				cn.Env[env.Name] = env.Value
			}
			cn.Images = container.Image
			portsvalue := ""
			for _, portitem := range container.Ports {
				portsfmt := fmt.Sprintf("%s:%d:%d", string(portitem.Protocol), portitem.HostPort, portitem.ContainerPort)
				portsvalue += " " + portsfmt
			}
			cn.Portsmap = portsvalue
			ni.containerList = append(ni.containerList, cn)
			ni.Images = cn.Images
			ni.Env = cn.Env
			ni.PortMap = portsvalue

			for _, vm := range container.VolumeMounts {
				vol[vm.Name] = vm.MountPath
			}
		}
		ni.Vol = make(map[string]string)

		for _, vm := range item.Spec.Volumes {
			if vm.HostPath != nil {
				ni.Vol[vm.HostPath.Path] = vol[vm.Name]
			}
		}
		ni.Status = "pending"
		ni.containerStatuses = make([]*ContainerStatus, 0, len(item.Status.ContainerStatuses))
		for _, containerstatus := range item.Status.ContainerStatuses {
			status := new(ContainerStatus)
			status.ContainerID = containerstatus.ContainerID
			if len(containerstatus.ContainerID) > 10 {
				ni.ContainerName = containerstatus.ContainerID[9:]
			}
			ni.RestartCount = containerstatus.RestartCount
			status.Ready = containerstatus.Ready
			if status.Ready {
				ni.Status = "ready"
			}
			ni.containerStatuses = append(ni.containerStatuses, status)
			if containerstatus.State.Terminated != nil {
				ni.Message = containerstatus.State.Terminated.Message
			} else if containerstatus.State.Waiting != nil {
				ni.Message = containerstatus.State.Waiting.Message
			}
		}
		if len(ni.Message) == 0 {
			ni.Message = item.Status.Message
		}
		conditonMsg := ""
		for _, condtion := range item.Status.Conditions {
			if condtion.Type == v1.PodScheduled && condtion.Status == v1.ConditionFalse {
				conditonMsg = condtion.Reason + ";" + condtion.Message
			}
		}
		if len(conditonMsg) > 0 {
			ni.Message = conditonMsg
		}

		//ni.Status = string(item.Status.Phase)
		ni.HostIp = item.Status.HostIP
		ni.PodIp = item.Status.PodIP

		if mt := monitor.GetMonitor(ni.NodeName); mt != nil {
			if stat := mt.GetContainerStat(ni.ContainerName); stat != nil {
				ni.Cpu = stat.CpuRatio
				ni.MemUsed = fmt.Sprintf("%d", stat.MemUsed)
				ni.MemTotal = fmt.Sprintf("%d", stat.MemTotal)
			}
		}
		podinfolist = append(podinfolist, ni)
	}
	return podinfolist, err

}

func (this *PodCtrl) DeletePod(namespace, podName string) error {
	return this.client.ClientSet.Core().Pods(namespace).Delete(podName, &metav1.DeleteOptions{})
}
