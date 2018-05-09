package k8s

import (
	"fmt"
	"gw.com.cn/dzhyun/yunconsole2.git/db"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"os"
	"strconv"
	"strings"
)

type PortInfo struct {
	Protocol  string
	InnerPort int32
	OuterPort int32
}

//docker option
func ParsePortInfo(portMapping string) ([]PortInfo, error) {
	//format: tcp:10800:10600,tcp:1:2
	dpMap := make([]PortInfo, 0)
	if len(portMapping) == 0 {
		return dpMap, nil
	}
	portArr := strings.Split(portMapping, ",")
	for _, port := range portArr {
		portinfo := strings.Split(port, ":")
		if len(portinfo) != 3 {
			return dpMap, fmt.Errorf("invalid mapping port info[%s]", port)
		}
		hostPort, err := strconv.Atoi(portinfo[1])
		if err != nil {
			return dpMap, fmt.Errorf("invalid hostport[%s] error[%s]", portinfo[1], err)
		}
		dockerPort, err := strconv.Atoi(portinfo[2])
		if err != nil {
			return dpMap, fmt.Errorf("invalid dockerPort[%s] error[%s]", portinfo[2], err)
		}
		portmap := PortInfo{
			Protocol:  portinfo[0],
			OuterPort: int32(hostPort),
			InnerPort: int32(dockerPort)}
		dpMap = append(dpMap, portmap)
	}
	return dpMap, nil
}

func RegistEnvVaria(appname, businesspath, inter, itype string) ([]v1.EnvVar, error) {
	poccAddr, _ := os.LookupEnv("POCCADDR")
	envs := make([]v1.EnvVar, 0, 4)
	envs = append(envs, v1.EnvVar{Name: "POCCADDR", Value: poccAddr})
	serviceID := businesspath + "/" + appname
	envs = append(envs, v1.EnvVar{Name: "SERVICE_ID", Value: serviceID})
	//envs = append(envs, v1.EnvVar{"HOSTIP", nodeName})
	envs = append(envs, v1.EnvVar{Name: "INTERFACE", Value: inter})
	//envs = append(envs, v1.EnvVar{Name: "INSTALLTYPE", Value: itype})
	return envs, nil
}

func SetDefatultVolume(appname string) ([]v1.VolumeMount, []v1.Volume) {
	vm := []v1.VolumeMount{}
	volumes := []v1.Volume{}
	vm = append(vm, v1.VolumeMount{Name: "vmnamedefatult1", MountPath: "/opt/app/data"})
	vm = append(vm, v1.VolumeMount{Name: "vmnamedefatult2", MountPath: "/opt/app/log"})
	vm = append(vm, v1.VolumeMount{Name: "vmnamedefatult3", MountPath: "/opt/app/trace"})
	suffix := fmt.Sprintf("/opt/dzhyun/%s/", appname)
	volumes = append(volumes, v1.Volume{
		Name: "vmnamedefatult1",
		VolumeSource: v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{Path: suffix + "data"},
		}})
	volumes = append(volumes, v1.Volume{
		Name: "vmnamedefatult2",
		VolumeSource: v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{Path: suffix + "log"},
		}})
	volumes = append(volumes, v1.Volume{
		Name: "vmnamedefatult3",
		VolumeSource: v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{Path: suffix + "trace"},
		}})
	return vm, volumes
}

func GetResouce(reqdeploy *db.DeployInfo) v1.ResourceRequirements {
	containerresource := v1.ResourceRequirements{
		Limits:   make(map[v1.ResourceName]resource.Quantity, 2),
		Requests: make(map[v1.ResourceName]resource.Quantity, 2),
	}
	if reqdeploy.Cpu < 0.01 && reqdeploy.Cpulimit < 0.01 && reqdeploy.Memory < 1 && reqdeploy.Memorylimit < 1 {
		return containerresource
	}
	//1core = 1000m (millicores)
	if reqdeploy.Cpu > 0.01 {
		cpurequire := fmt.Sprintf("%dm", int(reqdeploy.Cpu*1000))
		if cpuvalue, err := resource.ParseQuantity(cpurequire); err == nil {
			containerresource.Requests[v1.ResourceCPU] = cpuvalue
		}
	}
	if reqdeploy.Cpulimit > 0.01 {
		cpulimit := fmt.Sprintf("%dm", int(reqdeploy.Cpulimit*1000))
		if cpulimitvalue, err := resource.ParseQuantity(cpulimit); err == nil {
			containerresource.Limits[v1.ResourceCPU] = cpulimitvalue
		}

	}
	if reqdeploy.Memory > 1 {
		memrequire := strconv.Itoa(reqdeploy.Memory * 1024 * 1024)
		if memvalue, err := resource.ParseQuantity(memrequire); err == nil {
			containerresource.Requests[v1.ResourceMemory] = memvalue
		}
	}

	if reqdeploy.Memorylimit > 1 {
		memlimit := strconv.Itoa(reqdeploy.Memorylimit * 1024 * 1024)
		if memlimitvalue, err := resource.ParseQuantity(memlimit); err == nil {
			containerresource.Limits[v1.ResourceMemory] = memlimitvalue
		}
	}

	return containerresource
}

func ParseStat(data uint) string {
	if data > 1024*1024*1024 {
		d := float64(data) / (1024 * 1024 * 1024)
		number := fmt.Sprintf("%.2f GB", d)
		return number
	}
	if data > 1024*1024 {
		d := float64(data) / (1024 * 1024)
		number := fmt.Sprintf("%.2f MB", d)
		return number
	}
	if data > 1024 {
		d := float64(data) / 1024
		number := fmt.Sprintf("%.2f KB", d)
		return number
	}
	number := fmt.Sprintf("%.2f  B", data)
	return number
}

func convertCputofloat32(quantity string) (cpu float32) {
	var val int
	if strings.Contains(quantity, "m") {
		fmt.Sscanf(quantity, "%dm", &val)
		cpu = float32(val) / 1000.0
	} else {
		val, _ = strconv.Atoi(quantity)
		cpu = float32(val)
	}
	return
}

func convertMemtoInt(quantity string) int {
	var val int
	val, _ = strconv.Atoi(quantity)
	return val / (1024 * 1024)
}
