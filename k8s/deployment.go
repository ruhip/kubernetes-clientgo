package k8s

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/client-go/pkg/api/v1" old
	apps_v1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	//"k8s.io/client-go/pkg/api/v1"
	"fmt"
	"gw.com.cn/dzhyun/yunconsole2.git/context"
	"gw.com.cn/dzhyun/yunconsole2.git/db"
	//v1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	//v1beta1 "k8s.io/api/apps/v1beta1"
	//"k8s.io/apimachinery/pkg/api/resource"
	"strings"
	"time"
)

type KAppInfo struct {
	Type                string
	Kind                string
	Name                string
	Namespace           string
	Replicas            int32
	RepicasNow          int32
	AvailableReplicas   int32
	UnavailableReplicas int32 `json:"-"`
	Status              string
	Labels              map[string]string `json:"-"`
	Images              string
	CreateTime          time.Time
	UpdateTime          time.Time
	Interface           string
	PortMap             string
	Env                 map[string]string
	Vol                 map[string]string
	Path                string
	ServiceId           string
	ImageName           string  `json:"-"`
	ImageVer            string  `json:"-"`
	NodeName            string  `json:"-"`
	GroupId             string  `json:"-"`
	HostNetwork         int32   `json:"-"`
	Cpu                 float32 `json:"-"`
	Cpulimit            float32 `json:"-"`
	Memory              int     `json:"-"`
	Memorylimit         int     `json:"-"`
}
type DeploymentCtrl struct {
	client K8sClient
}

func (p *DeploymentCtrl) List(namespace string) ([]*KAppInfo, error) {
	deploymentList, err := p.client.ClientSet.AppsV1().Deployments(namespace).List(metav1.ListOptions{})

	//deploymentList, err := p.client.ClientSet.ExtensionsV1beta1().Deployments(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	//result *v1.DeploymentList
	deploylist := make([]*KAppInfo, 0, len(deploymentList.Items))
	for _, item := range deploymentList.Items {
		deployinfo := new(KAppInfo)
		deployinfo.Type = ""
		deployinfo.Kind = "Deployment"
		deployinfo.Name = item.Namespace + "/" + item.Name
		deployinfo.Namespace = item.Namespace
		deployinfo.Replicas = *(item.Spec.Replicas)
		deployinfo.RepicasNow = item.Status.Replicas
		deployinfo.AvailableReplicas = item.Status.AvailableReplicas
		deployinfo.UnavailableReplicas = item.Status.UnavailableReplicas
		deployinfo.Labels = item.Spec.Selector.MatchLabels
		deployinfo.CreateTime = item.CreationTimestamp.Time
		deployinfo.Status = "abnormal"

		for _, v := range item.Status.Conditions {
			if v.Type == apps_v1.DeploymentAvailable {
				if v.Status == v1.ConditionTrue {
					deployinfo.Status = "running"
				}
				break
			}
		}
		PodTemplateSpecToApp(deployinfo, item.Spec.Template)

		deploylist = append(deploylist, deployinfo)

	}
	return deploylist, nil
}

func (p *DeploymentCtrl) CreateDeploy(reqdeploy *db.DeployInfo) error {
	deploy, err := p.createDeploymentInfo(reqdeploy)
	if err != nil {
		return err
	}
	_, err = p.client.ClientSet.AppsV1().Deployments(reqdeploy.Namespace).Create(deploy)
	return err
}

func (p *DeploymentCtrl) UpdateDeploy(reqdeploy *db.DeployInfo) error {
	deploy, err := p.createDeploymentInfo(reqdeploy)
	if err != nil {
		return err
	}
	_, err = p.client.ClientSet.AppsV1().Deployments(reqdeploy.Namespace).Update(deploy)
	if err != nil {
		serr := fmt.Sprintf("err:%s", err)
		if strings.Contains(serr, "not found") {
			_, err = p.client.ClientSet.AppsV1().Deployments(reqdeploy.Namespace).Create(deploy)
		}
	}
	return err
}

func (p *DeploymentCtrl) createDeploymentInfo(reqdeploy *db.DeployInfo) (*apps_v1.Deployment, error) {

	podTemplateSpec, err := createPodTemplateSpec(reqdeploy)
	if err != nil {
		return nil, err
	}
	deploy := &apps_v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: reqdeploy.Name, Namespace: reqdeploy.Namespace},
		Spec: apps_v1.DeploymentSpec{
			Replicas: &reqdeploy.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"dzhyunapp": reqdeploy.Name,
				},
			},
			Template: *podTemplateSpec,
		},
	}

	return deploy, nil

}

func (p *DeploymentCtrl) DeleteDeploy(namespace, name string) error {
	name = strings.Replace(name, ".", "-", -1)
	var graceperiodseconds int64 = 0
	propaationpolicy := metav1.DeletePropagationBackground
	options := &metav1.DeleteOptions{
		GracePeriodSeconds: &graceperiodseconds,
		PropagationPolicy:  &propaationpolicy,
	}
	err := p.client.ClientSet.AppsV1().Deployments(namespace).Delete(name, options)
	return err
}

func (p *DeploymentCtrl) Delete(name string) error {
	name = strings.Replace(name, ".", "-", -1)
	var graceperiodseconds int64 = 0
	propaationpolicy := metav1.DeletePropagationBackground
	options := &metav1.DeleteOptions{
		GracePeriodSeconds: &graceperiodseconds,
		PropagationPolicy:  &propaationpolicy,
	}
	err := p.client.ClientSet.AppsV1().Deployments("default").Delete(name, options)
	return err
}

/*
func (p *DeploymentCtrl) DeleteDeployment(name string) error {
	return p.client.ClientSet.Extensions().Deployments("default").Delete(name, &v1.DeleteOptions{})
}
*/

func (p *DeploymentCtrl) Deleters(name, namespaces string) error {
	err := p.client.ClientSet.CoreV1().ReplicationControllers(namespaces).Delete(name, &metav1.DeleteOptions{})
	return err
}

func createPodTemplateSpec(reqdeploy *db.DeployInfo) (*v1.PodTemplateSpec, error) {
	//deploy := new(apps_v1.Deployment)
	portInfo, err := ParsePortInfo(reqdeploy.Portmap)
	if err != nil {
		return nil, err
	}
	containerports := []v1.ContainerPort{}
	for _, v := range portInfo {
		containerports = append(containerports, v1.ContainerPort{
			ContainerPort: v.InnerPort,
			HostPort:      v.OuterPort,
		})
	}
	hostnetwork := false
	if reqdeploy.HostNetwork == 1 {
		hostnetwork = true
	}
	containerresource := GetResouce(reqdeploy)
	image := context.GetConfig().Docker.RegistryAddr + "/" + reqdeploy.Image + ":" + reqdeploy.ImageVer
	env := []v1.EnvVar{}
	for k, v := range reqdeploy.Env {
		env = append(env, v1.EnvVar{Name: k, Value: v})
	}
	if reqdeploy.NodeName != "" {
		env = append(env, v1.EnvVar{Name: "dzhyunNodeName", Value: reqdeploy.NodeName})
	} else if reqdeploy.GroupId != "" {
		env = append(env, v1.EnvVar{Name: "dzhyunNodeSelect", Value: reqdeploy.GroupId})
	}
	bindenv, _ := RegistEnvVaria(reqdeploy.Name, reqdeploy.Path, reqdeploy.Interface, reqdeploy.Type)
	env = append(bindenv, env...)
	//vm := []v1.VolumeMount{}
	//volumes := []v1.Volume{}
	vm, volumes := SetDefatultVolume(reqdeploy.Name)
	index := 0
	vmname := ""
	for k, v := range reqdeploy.Volmap {
		hostpath := strings.Replace(k, "{$appname}", reqdeploy.Name, -1)
		index++
		vmname = fmt.Sprintf("vmname%d", index)
		vm = append(vm, v1.VolumeMount{Name: vmname, MountPath: v})
		volumes = append(volumes, v1.Volume{
			Name: vmname,
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{Path: hostpath},
			}})
	}
	/*
		deploy.ObjectMeta = metav1.ObjectMeta{Name: reqdeploy.Name, Namespace: reqdeploy.Namespace}
		deploy.Spec = apps_v1.DeploymentSpec{
			Replicas: &reqdeploy.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"dzhyunapp": reqdeploy.Name,
				},
			},
	*/
	//RestartPolicy: v1.RestartPolicyAlways,
	podTemplate := v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"dzhyunapp": reqdeploy.Name,
			},
		},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyAlways,
			Containers: []v1.Container{
				v1.Container{
					Name:            reqdeploy.Name,
					Image:           image,
					Ports:           containerports,
					ImagePullPolicy: v1.PullIfNotPresent,
					Env:             env,
					VolumeMounts:    vm,
					Resources:       containerresource,
				},
			},
			Volumes:     volumes,
			HostNetwork: hostnetwork,
		},
	}
	if reqdeploy.NodeName != "" {
		podTemplate.Spec.NodeName = reqdeploy.NodeName
	} else if reqdeploy.GroupId != "" {
		nodeselect := make(map[string]string)
		nodeselect[reqdeploy.GroupId] = "dzhyungroup"
		podTemplate.Spec.NodeSelector = nodeselect
	}

	return &podTemplate, nil

}

func PodTemplateSpecToApp(deployinfo *KAppInfo, Template v1.PodTemplateSpec) error {
	vol := make(map[string]string)
	for _, v := range Template.Spec.Containers {
		deployinfo.Images = strings.TrimLeft(v.Image, context.GetConfig().Docker.RegistryAddr+"/")
		imagesplit := strings.Split(deployinfo.Images, ":")
		if len(imagesplit) == 2 {
			deployinfo.ImageName = imagesplit[0]
			deployinfo.ImageVer = imagesplit[1]
		}
		deployinfo.Env = make(map[string]string)
		for _, env := range v.Env {
			if env.Name == "SERVICE_ID" {
				deployinfo.ServiceId = env.Value
				deployinfo.Path = env.Value
			} else if env.Name == "INTERFACE" {
				deployinfo.Interface = env.Value
			} else if env.Name == "INSTALLTYPE" {
				deployinfo.Type = env.Value
				continue
			} else if env.Name == "dzhyunNodeName" {
				deployinfo.NodeName = env.Value
				continue
			} else if env.Name == "dzhyunNodeSelect" {
				deployinfo.GroupId = env.Value
				continue
			}
			deployinfo.Env[env.Name] = env.Value
		}
		for _, portitem := range v.Ports {
			portfmt := fmt.Sprintf("%s:%d:%d", string(portitem.Protocol), portitem.HostPort, portitem.ContainerPort)
			deployinfo.PortMap = deployinfo.PortMap + portfmt
		}
		for _, vm := range v.VolumeMounts {
			vol[vm.Name] = vm.MountPath
		}
		if v.Resources.Limits != nil {
			if cpu, ok := v.Resources.Limits[v1.ResourceCPU]; ok {
				deployinfo.Cpulimit = convertCputofloat32((&cpu).String())
			}
			if mem, ok := v.Resources.Limits[v1.ResourceMemory]; ok {
				deployinfo.Memorylimit = convertMemtoInt((&mem).String())
			}

		}
		if v.Resources.Requests != nil {
			if cpu, ok := v.Resources.Requests[v1.ResourceCPU]; ok {
				deployinfo.Cpu = convertCputofloat32((&cpu).String())
			}
			if mem, ok := v.Resources.Requests[v1.ResourceMemory]; ok {
				deployinfo.Memory = convertMemtoInt((&mem).String())
			}

		}

	}
	deployinfo.Vol = make(map[string]string)

	for _, vm := range Template.Spec.Volumes {
		deployinfo.Vol[vm.HostPath.Path] = vol[vm.Name]
	}
	if Template.Spec.HostNetwork {
		deployinfo.HostNetwork = 1
	} else {
		deployinfo.HostNetwork = 0
	}
	return nil
}
