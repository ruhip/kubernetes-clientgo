package k8s

import (
	"errors"
	"fmt"
	"gw.com.cn/dzhyun/yunconsole2.git/metricmgr"
	"gw.com.cn/dzhyun/yunconsole2.git/monitor"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"
)

type NodePodsInfo struct {
	Total   int
	Running int
	Paused  int
	Stopped int
}

type ReservedCpus struct {
	UsedCpus  int
	TotalCpus int
}

type ReservedMems struct {
	UsedMemory  int64
	TotalMemory int64
}

type NodeInfo struct {
	Name          string
	Addr          string
	Ready         bool
	DockerVersion string
	Cpu           float32
	MemUsed       string
	MemTotal      string
	DiskUsed      string
	DiskTotal     string
	Total         int          //容器总数
	Running       int          //运行容器数
	Paused        int          //暂停容器数
	Stopped       int          //停止容器数
	Pods          NodePodsInfo `json:"-"`
	Cpus          ReservedCpus `json:"-"`
	Mems          ReservedMems `json:"-"`
	Groups        []string
	Status        int `json:"Status"`
	ServerVersion string
	Labels        map[string]string `json:"-"`
}

type AddLabelNodes struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Desc      string   `json:"desc"`
	NodesList []string `json:"nodes"`
}
type LabelNodeInfo struct {
	NodeName string
	NodeIp   string
	Ready    bool
}
type NodeCtrl struct {
	client K8sClient
}

func (p *NodeCtrl) ListNodes(groupid string) ([]*NodeInfo, error) {
	listoption := metav1.ListOptions{}
	if len(groupid) > 0 {
		listoption.LabelSelector = groupid + "=dzhyungroup"
	}
	result, err := p.client.ClientSet.CoreV1().Nodes().List(listoption)
	if err != nil {
		return nil, err
	}
	nodesmap := make(map[string]*NodeInfo)
	nodesinfo := make([]*NodeInfo, 0, len(result.Items))
	for _, node := range result.Items {

		ni := new(NodeInfo)
		ni.Name = node.Name
		nodesmap[node.Name] = ni
		ni.Groups = make([]string, 0, len(node.Labels))
		for k, _ := range node.Labels {
			if strings.HasPrefix(k, "dzhgroup_") {
				ni.Groups = append(ni.Groups, k)
			}
		}

		for _, j := range node.Status.Addresses {
			if j.Type == v1.NodeInternalIP {
				ni.Addr = j.Address
				break
			}
		}
		ni.Status = 1
		ni.Ready = false
		for _, v := range node.Status.Conditions {
			if v.Type == v1.NodeReady {
				if v.Status == v1.ConditionTrue {
					ni.Status = 0
					ni.Ready = true
				}
				break
			}

		}
		ni.ServerVersion = node.Status.NodeInfo.ContainerRuntimeVersion
		capcpu := node.Status.Capacity[v1.ResourceCPU]
		if vcpu, b := capcpu.AsInt64(); b {
			ni.Cpus.TotalCpus = int(vcpu)
		}
		allocatablecpu := node.Status.Allocatable[v1.ResourceCPU]
		if vcpu, b := allocatablecpu.AsInt64(); b {
			ni.Cpus.UsedCpus = ni.Cpus.TotalCpus - int(vcpu)
		}
		capmem := node.Status.Capacity[v1.ResourceMemory]
		if vmem, b := capmem.AsInt64(); b {
			ni.Mems.TotalMemory = vmem
		}
		allocatablemem := node.Status.Allocatable[v1.ResourceMemory]
		if vmem, b := allocatablemem.AsInt64(); b {
			ni.Mems.UsedMemory = ni.Mems.TotalMemory - vmem
		}
		ni.DiskUsed = "0 GB"
		ni.DiskTotal = "0 GB"
		ni.MemUsed = ParseStat(uint(ni.Mems.UsedMemory))
		ni.MemTotal = ParseStat(uint(ni.Mems.TotalMemory))
		ni.Cpu = (100.0 * float32(ni.Cpus.UsedCpus)) / float32(ni.Cpus.TotalCpus)

		if mt := monitor.GetMonitor(ni.Name); mt != nil {
			if machincpu := mt.GetMachineCpu(); machincpu > 0 {
				ni.Cpu = float32(machincpu)
			}
		}

		if mt := monitor.GetMonitor(ni.Name); mt != nil {
			p.FillSysStat(ni)
		}
		nodesinfo = append(nodesinfo, ni)

	}
	//*v1.PodList
	//podslist, err := DefaultK8sMgr.GetDeploymentPodsImpl("")
	podslist, err := DefaultK8sMgr.ListPodImpl("", "", true)
	if err != nil {
		return nodesinfo, nil
	}
	for _, pod := range podslist {
		ni, ok := nodesmap[pod.NodeName]
		if !ok {
			continue
		}
		ni.Total++
		if pod.Status == "ready" {
			ni.Running++
		}
	}

	return nodesinfo, nil

}

func (this *NodeCtrl) FillSysStat(nodeInfo *NodeInfo) {

	metric, err := metricmgr.GetMetric(nodeInfo.Addr)
	if err != nil {
		return
	}

	if metric == nil {
		return
	}

	if metric.Cpu != nil {
		nodeInfo.Cpu = metric.Cpu.System.Pct * 100
	}

	if metric.Memory != nil {
		memUsed := metric.Memory.Actual.Used.Bytes
		memTotal := (metric.Memory.Actual.Used.Bytes + metric.Memory.Actual.Free)
		nodeInfo.MemUsed = ParseStat(memUsed)
		nodeInfo.MemTotal = ParseStat(memTotal)
	}

	//this.LogDebug("%s fs metric: %+v", nodeInfo.Name, metric.Fs)
	if metric.Fs != nil {
		diskUsed := metric.Fs.Used.Bytes
		diskTotal := metric.Fs.Total
		nodeInfo.DiskUsed = ParseStat(diskUsed)
		nodeInfo.DiskTotal = ParseStat(uint(diskTotal))
	}
}

func (p *NodeCtrl) DeleteGroup(groupid string) error {
	listoption := metav1.ListOptions{}
	if len(groupid) > 0 {
		listoption.LabelSelector = groupid + "=dzhyungroup"
	}
	result, err := p.client.ClientSet.CoreV1().Nodes().List(listoption)
	if err != nil {
		return err
	}
	for _, node := range result.Items {
		if node.ObjectMeta.Labels == nil {
			continue
		}
		if _, ok := node.ObjectMeta.Labels[groupid]; ok {
			delete(node.ObjectMeta.Labels, groupid)
			_, err = p.client.ClientSet.CoreV1().Nodes().Update(&node)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *NodeCtrl) LabelNodes(label string) ([]*LabelNodeInfo, error) {
	option := metav1.ListOptions{LabelSelector: label}
	result, err := p.client.ClientSet.CoreV1().Nodes().List(option)
	if err != nil {
		return nil, err
	}
	labelnodeslist := make([]*LabelNodeInfo, 0, len(result.Items))
	for _, item := range result.Items {
		nodeinfo := new(LabelNodeInfo)
		nodeinfo.NodeName = item.Name
		for _, addr := range item.Status.Addresses {
			if addr.Type == v1.NodeInternalIP {
				nodeinfo.NodeIp = addr.Address
				break
			}
		}
		nodeinfo.Ready = false
		for _, con := range item.Status.Conditions {
			if con.Type == v1.NodeReady {
				if con.Status == v1.ConditionTrue {
					nodeinfo.Ready = true
				}
				break
			}

		}
		labelnodeslist = append(labelnodeslist, nodeinfo)
	}
	return labelnodeslist, nil
}

func (p *NodeCtrl) AddlabelToNodes(labelnodes *AddLabelNodes) error {
	var err error
	for _, nodename := range labelnodes.NodesList {
		var node *v1.Node
		node, err = p.client.ClientSet.CoreV1().Nodes().Get(nodename, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if node.ObjectMeta.Labels == nil {
			node.ObjectMeta.Labels = make(map[string]string)

		}
		node.ObjectMeta.Labels[labelnodes.ID] = "dzhyungroup"
		_, err = p.client.ClientSet.CoreV1().Nodes().Update(node)
	}
	return err

}

func (p *NodeCtrl) NodesNames(labelnodes *AddLabelNodes, nl []*NodeInfo) ([]string, []string) {
	addnodes := make([]string, 0, 0)
	removenodes := make([]string, 0, 0)

	for _, nodename := range labelnodes.NodesList {
		bOk := false
		for _, item := range nl {
			if nodename == item.Name {
				bOk = true
				break
			}
		}
		if bOk {
			continue
		}
		addnodes = append(addnodes, nodename)
	}
	for _, item := range nl {
		bOk := false
		for _, nodename := range labelnodes.NodesList {
			if nodename == item.Name {
				bOk = true
				break
			}
		}
		if bOk {
			continue
		}
		removenodes = append(removenodes, item.Name)
	}
	return addnodes, removenodes
}
func (p *NodeCtrl) UpdateNodeslabel(labelnodes *AddLabelNodes) error {
	nl, err := p.ListNodes(labelnodes.ID)
	if err != nil {
		return nil
	}
	addnodes, removendoes := p.NodesNames(labelnodes, nl)

	ln := new(AddLabelNodes)
	ln.ID = labelnodes.ID
	ln.NodesList = addnodes
	err = p.AddlabelToNodes(ln)
	if err != nil {
		return err
	}
	ln = new(AddLabelNodes)
	ln.ID = labelnodes.ID
	ln.NodesList = removendoes
	err = p.RemoveNodeLabel(ln)
	if err != nil {
		return err
	}
	return nil

}

func (p *NodeCtrl) RemoveNodeLabel(labelnodes *AddLabelNodes) error {
	var node *v1.Node
	var err error
	for _, nodename := range labelnodes.NodesList {
		node, err = p.client.ClientSet.CoreV1().Nodes().Get(nodename, metav1.GetOptions{})

		if err != nil {
			return err
		}
		if node.ObjectMeta.Labels == nil {
			continue
		}
		if _, ok := node.ObjectMeta.Labels[labelnodes.ID]; ok {
			delete(node.ObjectMeta.Labels, labelnodes.ID)
			_, err = p.client.ClientSet.CoreV1().Nodes().Update(node)
			if err != nil {
				return err
			}
		}
	}

	return nil

}

func (p *NodeCtrl) RemoveNodeLabel2(label, nodename string) error {
	var node *v1.Node
	node, err := p.client.ClientSet.CoreV1().Nodes().Get(nodename, metav1.GetOptions{})

	if err != nil {
		return err
	}
	notfound := true
	for k, _ := range node.ObjectMeta.Labels {
		if k == "dzhgroup_"+label {
			delete(node.ObjectMeta.Labels, k)
			notfound = false
		}
	}
	if notfound {
		return errors.New("label notfound")

	}
	_, err = p.client.ClientSet.CoreV1().Nodes().Update(node)
	if err != nil {
		return err
	}
	return nil

}

func (p *NodeCtrl) NodesMonitor() error {

	t := time.NewTicker(time.Second * time.Duration(5)).C

	for {
		p.nodeslist()
		<-t
	}

	return nil
}

func (p *NodeCtrl) nodeslist() error {
	nodelist, err := p.client.ClientSet.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, node := range nodelist.Items {
		nodename := node.Name
		nodeip := ""
		ready := false
		for _, addr := range node.Status.Addresses {
			if addr.Type == v1.NodeInternalIP {
				nodeip = addr.Address
				break
			}
		}
		for _, con := range node.Status.Conditions {
			if con.Type == v1.NodeReady {
				if con.Status == v1.ConditionTrue {
					ready = true
				}
				break
			}
		}
		if len(nodeip) > 0 && ready {
			monitor.CreateMonitor(nodename, fmt.Sprintf("%s:%d", nodeip, 4194))
		}
		if !ready {
			if mt := monitor.GetMonitor(nodename); mt != nil {
				//glog.Infof("ready RemoveMonitor->nodename:%s", nodename)
				monitor.RemoveMonitor(nodename)
			}
		}

	}
	return nil
}
