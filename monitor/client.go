package monitor

import (
	client "github.com/google/cadvisor/client/v2"
	"sync"
)

var (
	//cadvisorClient *client.Client
	clientLock = new(sync.RWMutex)
	monitorset map[string]*Monitor
)

//"10.15.144.51" + ":" + "4194"
func CreateMonitor(nodename, addr string) {
	clientLock.Lock()
	defer clientLock.Unlock()
	if monitorset == nil {
		monitorset = make(map[string]*Monitor)
	}
	if _, ok := monitorset[nodename]; ok {
		return
	}
	client, err := client.NewClient("http://" + addr)
	if err != nil {
	}
	monitor := &Monitor{
		Nodename:       nodename,
		Addr:           addr,
		CadvisorClient: client,
	}
	monitorset[nodename] = monitor
	monitor.Start()
}

func GetMonitor(nodename string) *Monitor {
	clientLock.Lock()
	defer clientLock.Unlock()
	return monitorset[nodename]
}

func RemoveMonitor(nodename string) error {
	clientLock.Lock()
	defer clientLock.Unlock()
	if monitor := monitorset[nodename]; monitor != nil {
		monitor.Stop()
	}
	delete(monitorset, nodename)
	return nil
}
