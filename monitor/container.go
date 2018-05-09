package monitor

import (
	//"//glog"
	client "github.com/google/cadvisor/client/v2"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/info/v2"
	"gw.com.cn/dzhyun/yunconsole2.git/db"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	containerHistoryCount int = 2
	machineHistoryCount   int = 2
)

type StatInfo struct {
	Name     string
	CpuRatio string
	MemUsed  uint64
	MemTotal uint64
}

type Monitor struct {
	currentContainers     []string
	currentContainersLock *sync.RWMutex
	containerStatHistory  map[string][containerHistoryCount]*info.ContainerInfo
	//machineStatHistory    map[string][machineHistoryCount]*v2.MachineStats
	csLock              *sync.RWMutex
	Nodename            string
	Addr                string
	CadvisorClient      *client.Client
	minfo               *info.MachineInfo
	machinestats        []v2.MachineStats
	containerstats      map[string]*StatInfo
	machinestatDuration int64
	machinecpu          float64
	monitorExitCh       chan bool
	wg                  sync.WaitGroup
	OrdinaryContainers  map[string]*db.ContainerInfo
	//StoppedContainers   map[string]*db.ContainerInfo
	CurrentPodSandboxs map[string]*db.ContainerInfo
	//StoppedPodsandboxs  map[string]*db.ContainerInfo
	//machinememused      string
	//machinetotal        string
}

func (this *Monitor) Start() {
	this.currentContainers = make([]string, 0, 0)
	this.currentContainersLock = new(sync.RWMutex)
	this.containerStatHistory = make(map[string][containerHistoryCount]*info.ContainerInfo)
	this.csLock = new(sync.RWMutex)
	this.containerstats = make(map[string]*StatInfo)
	this.monitorExitCh = make(chan bool)
	this.OrdinaryContainers = make(map[string]*db.ContainerInfo)
	//this.StoppedContainers = make(map[string]*db.ContainerInfo)
	this.CurrentPodSandboxs = make(map[string]*db.ContainerInfo)
	//this.StoppedPodsandboxs = make(map[string]*db.ContainerInfo)
	this.InitHistoryData()
	this.wg.Add(1)
	go this.InitDataHistory()
	this.wg.Add(1)
	go this.collect()
}

func (this *Monitor) Stop() {
	close(this.monitorExitCh)
	this.wg.Wait()
}

func (this *Monitor) CurrentContainers() []string {
	this.currentContainersLock.RLock()
	defer this.currentContainersLock.RUnlock()
	return this.currentContainers
}

func (this *Monitor) SetCurrentContainers(containers []string) {
	this.currentContainersLock.Lock()
	defer this.currentContainersLock.Unlock()
	this.currentContainers = containers
}

func (this *Monitor) SetMachineStats(stats []v2.MachineStats) {
	this.currentContainersLock.Lock()
	defer this.currentContainersLock.Unlock()
	this.machinestats = stats
}

func (this *Monitor) GetMachineStats() []v2.MachineStats {
	this.currentContainersLock.Lock()
	defer this.currentContainersLock.Unlock()
	return this.machinestats
}

func (this *Monitor) GetContainerStat(name string) *StatInfo {
	this.currentContainersLock.Lock()
	defer this.currentContainersLock.Unlock()
	return this.containerstats[name]
}

func (this *Monitor) GetMachineCpu() float64 {
	this.currentContainersLock.Lock()
	defer this.currentContainersLock.Unlock()
	return this.machinecpu
}

func (this *Monitor) InitDataHistory() {
	defer this.wg.Done()
	if machineInfo, err := this.CadvisorClient.MachineInfo(); err == nil {
		this.minfo = machineInfo
	}
	updateStatsCh := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-this.monitorExitCh:
			return
		case <-updateStatsCh.C:
			this.UpdateMachineStats()
			this.UpdateContainerStat()
		}
	}
}

func (this *Monitor) UpdateMachineStats() {
	machinestats, err := this.CadvisorClient.MachineStats()
	if err != nil {
		return
	}
	this.SetMachineStats(machinestats)
}

func (this *Monitor) collect() {
	defer this.wg.Done()
	t := time.NewTicker(time.Second * time.Duration(10)).C

	for {
		select {
		case <-this.monitorExitCh:
			return
		case <-t:
			this.ContainerMetrics()
		}
	}
}
func (this *Monitor) ContainerMetrics() {

	containers := this.CurrentContainers()
	machinestats := this.GetMachineStats()
	if len(machinestats) != 2 {
		return
	}
	stat0 := machinestats[0]
	stat1 := machinestats[1]
	this.machinestatDuration = stat1.Timestamp.Sub(stat0.Timestamp).Nanoseconds()
	busy := (stat1.Cpu.Usage.User - stat0.Cpu.Usage.User) + (stat1.Cpu.Usage.System - stat0.Cpu.Usage.System)
	total := stat1.Cpu.Usage.Total - stat0.Cpu.Usage.Total
	this.machinecpu = float64(busy * 100 / (uint64(this.minfo.NumCores) * total))

	for _, container := range containers {

		if !this.ContainerPrepared(container) {
			continue
		}
		cntinfo := this.containerStatHistory[container]
		cntduration := cntinfo[0].Stats[0].Timestamp.Sub(cntinfo[1].Stats[0].Timestamp).Nanoseconds()
		if cntduration < 1 {
			continue
		}
		cpubusy := float64(cntinfo[0].Stats[0].Cpu.Usage.Total-cntinfo[1].Stats[0].Cpu.Usage.Total) / float64(total)
		cpubusy = cpubusy * float64(this.machinestatDuration) / float64(cntduration)
		if cpubusy > 100 {
			cpubusy = 100.00
		}
		//mempercent := float64(cntinfo[0].Stats[0].Memory.Usage) / float64(this.minfo.MemoryCapacity) * 100.0
		////glog.Infof("contaier cpubusy:%+v,mempercent:%+v,cntduration:%d", cpubusy, mempercent, cntduration)
		cntstat := &StatInfo{
			Name:     container,
			CpuRatio: strconv.FormatFloat(cpubusy, 'f', 2, 64),
			MemUsed:  cntinfo[0].Stats[0].Memory.Usage,
			MemTotal: this.minfo.MemoryCapacity,
		}
		this.containerstats[container] = cntstat
	}
	return
}

func (this *Monitor) InitHistoryData() {
	containerlist, err := db.GetContainerLifeCycleHisory(this.Nodename, this.Addr)
	if err != nil {

	}
	for _, cntinfo := range containerlist {
		if strings.Compare(cntinfo.DockerType, "podsandbox") == 0 {
			if v, ok := this.CurrentPodSandboxs[cntinfo.PodName]; !ok {
				this.CurrentPodSandboxs[cntinfo.PodName] = cntinfo
			} else {
				exceptioncontainers := make(map[string]*db.ContainerInfo)
				v.Status = "stopped"
				exceptioncontainers[cntinfo.Id] = v
				db.UpdateContainerInfo(exceptioncontainers)
				cntinfo.Status = "stopped"
				exceptioncontainers[cntinfo.Id] = cntinfo
				db.UpdateContainerInfo(exceptioncontainers)
			}
		} else if strings.Compare(cntinfo.DockerType, "container") == 0 {
			this.OrdinaryContainers[cntinfo.Id] = cntinfo
		}
	}

}

func (this *Monitor) UpdateCurrentContainers() {
	query := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	dockerContainers, err := this.CadvisorClient.AllDockerContainers(query)
	if err != nil {
		return
	}
	containers := make([]string, 0)
	ordinarycontainers := make(map[string]*db.ContainerInfo)
	currentpodsanboxs := make(map[string]*db.ContainerInfo)
	allrunningdocker := make([]*db.ContainerInfo, 0, len(dockerContainers))
	for _, container := range dockerContainers {
		containers = append(containers, container.Id)
		cntinfo := new(db.ContainerInfo)
		cntinfo.Id = container.Id
		if len(container.Aliases) > 1 {
			cntinfo.AliaseNameOne = container.Aliases[0]
			cntinfo.AliaseNameTwo = container.Aliases[1]
		}
		cntinfo.NodeName = this.Nodename
		cntinfo.Addr = this.Addr
		cntinfo.CreateAt = container.Spec.CreationTime
		cntinfo.Namespace = container.Labels["io.kubernetes.pod.namespace"]
		cntinfo.AppName = container.Labels["dzhyunapp"]
		cntinfo.ContainerName = container.Labels["io.kubernetes.container.name"]
		cntinfo.DockerType = container.Labels["io.kubernetes.docker.type"]
		cntinfo.PodName = container.Labels["io.kubernetes.pod.name"]
		cntinfo.Restart = container.Labels["annotation.io.kubernetes.container.restartCount"]
		cntinfo.Status = "running"
		cntinfo.UnMonitorCount = 0

		allrunningdocker = append(allrunningdocker, cntinfo)

		if strings.Compare(cntinfo.DockerType, "podsandbox") == 0 {
			cntinfo.Inspected = false
			currentpodsanboxs[cntinfo.PodName] = cntinfo
		} else if strings.Compare(cntinfo.DockerType, "container") == 0 {
			cntinfo.Inspected = true
			ordinarycontainers[container.Id] = cntinfo
		}
	}
	//如果当前pod对应的容器也在运行，那么将该pod设置为已被监控到
	for _, v := range ordinarycontainers {
		if _, ok := currentpodsanboxs[v.PodName]; ok {
			currentpodsanboxs[v.PodName].Inspected = true
		}
	}

	//找出新增的容器
	addedcontainers := make(map[string]*db.ContainerInfo)
	for k, v := range ordinarycontainers {
		if _, ok := this.OrdinaryContainers[k]; !ok {
			addedcontainers[k] = v
		}
	}
	//找出新增的pod
	for k, v := range currentpodsanboxs {
		if _, ok := this.CurrentPodSandboxs[k]; !ok {
			addedcontainers[v.Id] = v
		}
	}

	db.UpdateContainerInfo(addedcontainers)

	updatedcontainers := make(map[string]*db.ContainerInfo)
	for k, v := range this.CurrentPodSandboxs {
		if sandbox, ok := currentpodsanboxs[k]; ok {
			//找出修改的pod
			if v.Id != sandbox.Id {
				v.Status = "stopped"
				updatedcontainers[v.Id] = v
				sandbox.UnMonitorCount = v.UnMonitorCount
				updatedcontainers[sandbox.Id] = sandbox
				continue
			}
			if !sandbox.Inspected {
				sandbox.UnMonitorCount = v.UnMonitorCount + 1
				updatedcontainers[sandbox.Id] = sandbox
			} else if !v.Inspected {
				sandbox.UnMonitorCount = v.UnMonitorCount
				updatedcontainers[sandbox.Id] = sandbox
			} else {
				sandbox.UnMonitorCount = v.UnMonitorCount
			}
		} else {
			v.Status = "stopped"
			updatedcontainers[v.Id] = v
		}
	}

	for k, v := range this.OrdinaryContainers {
		if _, ok := ordinarycontainers[k]; !ok {
			v.Status = "stopped"
			updatedcontainers[k] = v
		}
	}
	db.UpdateContainerInfo(updatedcontainers)
	this.OrdinaryContainers = ordinarycontainers
	this.CurrentPodSandboxs = currentpodsanboxs

	this.SetCurrentContainers(containers)

}
func (this *Monitor) UpdateContainerStat() error {

	this.UpdateCurrentContainers()
	containers := this.CurrentContainers()
	reqParams := &info.ContainerInfoRequest{
		NumStats: 1,
	}

	this.csLock.Lock()
	defer this.csLock.Unlock()

	for _, container := range containers {

		containerInfo, err := this.CadvisorClient.DockerContainer(container, reqParams)
		if err != nil {
			continue
		}

		containerStatHistoryCopy := this.containerStatHistory[container]
		for i := containerHistoryCount - 1; i > 0; i-- {
			containerStatHistoryCopy[i] = containerStatHistoryCopy[i-1]
		}
		containerStatHistoryCopy[0] = &containerInfo
		this.containerStatHistory[container] = containerStatHistoryCopy
	}

	return nil
}

func (this *Monitor) ContainerPrepared(container string) bool {
	this.csLock.RLock()
	defer this.csLock.RUnlock()
	if _, ok := this.containerStatHistory[container]; !ok {
		return false
	}
	return this.containerStatHistory[container][1] != nil
}
